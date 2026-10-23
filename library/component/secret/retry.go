package secret

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cst"
)

// 重试策略：按错误分类，只对可重试错误做指数退避 + 全抖动（第 i 次等待 = rand(0, min(cap, base<<i))），
// 以整体 deadline 收口而不按次数——窗口要覆盖 Pod 内 sidecar 就绪前出站被拦、Pod Identity 端点短暂不可用这类瞬时失败。
// 每次尝试另有单次超时（默认 5s），且不超过 deadline 剩余时间：连接建立后响应停滞这类黑洞故障也按超时失败并进入重试，
// 一次 retry 的总耗时因此以 deadline 为上界（deadline 为 0 时只尝试一次，上界为单次超时）。
// deadline 不能超过探针预算：Deployment 未配 startupProbe 时 liveness 约 40s 后杀容器，前面还有 sidecar 探测在分摊，
// 默认 20s；环境变量 SECRET_RETRY_DEADLINE_SECONDS 可覆盖，0 表示不重试。deadline 按凭据名计算，
// 启动预算的推导以"每个服务只引用一个凭据名"为前提（同名凭据第二次起命中缓存），引用多个凭据名时最坏耗时按个数叠加。
// 全抖动把同时重启的一批 Pod 的重试打散，避免对 DEW 的同步冲击
var (
	retryBase           = time.Second
	retryCap            = 5 * time.Second
	retryDeadline       = retryDeadlineFromEnv()
	retryAttemptTimeout = 5 * time.Second
	// 测试钩子：替换随机源与休眠
	retryRand  = rand.Float64
	retrySleep = sleepCtx
)

const defaultRetryDeadline = 20 * time.Second

// retryDeadlineFromEnv 读取 deadline 配置；未设置或非法值回落默认，负数按 0（不重试）
func retryDeadlineFromEnv() time.Duration {
	v := os.Getenv(cst.ENV_SECRET_RETRY_DEADLINE_SECONDS)
	if v == "" {
		return defaultRetryDeadline
	}
	sec, err := strconv.Atoi(v)
	if err != nil {
		return defaultRetryDeadline
	}
	if sec < 0 {
		sec = 0
	}
	return time.Duration(sec) * time.Second
}

// retry 执行 fn；每次尝试传入带单次超时的 ctx（min(单次超时, deadline 剩余)），可重试错误在 deadline 内按指数退避 + 全抖动重试，
// 每次重试打 WARN（只带引用名，不带值）；不可重试错误、ctx 取消、deadline 耗尽立即返回。desc 为日志里的凭据标识（scheme://name）。
// fn 必须遵守传入 ctx 的截止时间（不接 ctx 的 SDK 由 Provider 自行在 ctx 结束时返回），否则单次超时无效
func retry(ctx context.Context, desc string, fn func(ctx context.Context) error) error {
	start := time.Now()
	var err error
	for attempt := 0; ; attempt++ {
		budget := retryAttemptTimeout
		if retryDeadline > 0 {
			remaining := retryDeadline - time.Since(start)
			if remaining <= 0 {
				log.Warn("[secret] 拉取凭据重试已达 deadline，放弃", zap.String("ref", desc),
					zap.Int("attempts", attempt), zap.Duration("elapsed", time.Since(start)), zap.Duration("deadline", retryDeadline), zap.Error(err))
				return err
			}
			if remaining < budget {
				budget = remaining
			}
		}
		err = attemptWithTimeout(ctx, budget, fn)
		if err == nil || !IsRetryable(err) {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		wait := backoff(attempt)
		elapsed := time.Since(start)
		if retryDeadline <= 0 || elapsed+wait >= retryDeadline {
			log.Warn("[secret] 拉取凭据重试已达 deadline，放弃", zap.String("ref", desc),
				zap.Int("attempts", attempt+1), zap.Duration("elapsed", elapsed), zap.Duration("deadline", retryDeadline), zap.Error(err))
			return err
		}
		log.Warn("[secret] 拉取凭据失败，退避后重试", zap.String("ref", desc),
			zap.Int("attempt", attempt+1), zap.Duration("elapsed", elapsed), zap.Duration("wait", wait), zap.Error(err))
		if e := retrySleep(ctx, wait); e != nil {
			return e
		}
	}
}

// attemptWithTimeout 以单次预算执行一次尝试；可重试错误恰逢单次超时到点时统一返回 ErrAttemptTimeout（不可重试错误原样保留），
// 避免各 SDK 的超时错误形态不一（context.DeadlineExceeded / net 超时 / SDK 自定义错误）影响分类
func attemptWithTimeout(ctx context.Context, budget time.Duration, fn func(ctx context.Context) error) error {
	actx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	err := fn(actx)
	if err != nil && IsRetryable(err) && ctx.Err() == nil && actx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("%w (%s): %v", ErrAttemptTimeout, budget.Round(time.Millisecond), err)
	}
	return err
}

// backoff 第 attempt 次失败后的等待：全抖动，上界 min(cap, base<<attempt)，attempt 过大时上界钉在 cap 防溢出
func backoff(attempt int) time.Duration {
	upper := retryCap
	if attempt < 30 {
		if b := retryBase << uint(attempt); b < upper {
			upper = b
		}
	}
	return time.Duration(retryRand() * float64(upper))
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
