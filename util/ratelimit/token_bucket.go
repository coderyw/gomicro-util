// Package ratelimit 提供基于 Redis Lua 令牌桶的分布式限流工具。
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// RateLimiter 定义限流器接口，用于在测试中替换实现。
type RateLimiter interface {
	// WaitN 阻塞直到可以消耗 n 个令牌后返回。
	// ctx 取消或超时时返回对应的 ctx 错误。
	WaitN(ctx context.Context, n int) error
}

// tokenBucketLua 是分布式令牌桶的 Redis Lua 脚本。
// 按时间差原子地补充令牌并消耗，避免多实例并发写入时的竞态条件。
//
// 入参：
//
//	KEYS[1]        令牌桶 Hash key
//	ARGV[1] now_ms 当前毫秒时间戳（由客户端传入，避免使用 Redis TIME 命令）
//	ARGV[2] n      本次申请消耗的令牌数量
//	ARGV[3] cap    桶的容量上限，即 ratePerSec（每段单次不得超过此值）
//	ARGV[4] fill   每毫秒补充令牌数（浮点字符串，如 "0.150000" 表示 150/s）
//
// 返回值：
//
//	1 令牌充足，已消耗 n 个令牌
//	0 令牌不足，未消耗，调用方需等待后重试
const tokenBucketLua = `
local key   = KEYS[1]
local now   = tonumber(ARGV[1])
local n     = tonumber(ARGV[2])
local cap   = tonumber(ARGV[3])
local fill  = tonumber(ARGV[4])

local last   = tonumber(redis.call('HGET', key, 'last_ms'))
local tokens = tonumber(redis.call('HGET', key, 'tokens'))

-- 首次访问：令牌桶满载初始化
if not last then
  last   = now
  tokens = cap
end

-- 按时间差补充令牌，不超过桶容量
local elapsed = math.max(0, now - last)
tokens = math.min(cap, tokens + elapsed * fill)

if tokens >= n then
  tokens = tokens - n
  redis.call('HSET', key, 'tokens', tokens, 'last_ms', now)
  redis.call('PEXPIRE', key, 10000)
  return 1
else
  redis.call('HSET', key, 'tokens', tokens, 'last_ms', now)
  redis.call('PEXPIRE', key, 10000)
  return 0
end
`

// TokenBucketLimiter 是基于 Redis Lua 脚本的分布式令牌桶限流器。
//
// 多服务实例使用相同的 Redis key，令牌池全局有效，不受副本数影响。
// 线程安全：令牌的补充与消耗通过 Lua 脚本原子执行，可被多 goroutine 并发调用。
type TokenBucketLimiter struct {
	// key 是 Redis 存储令牌桶状态的 Hash key。
	// 同一业务的所有服务实例应使用相同的 key，以共享全局令牌池。
	key string

	// ratePerSec 是令牌补充速率（个/秒），同时也是桶的容量上限。
	// 桶容量限制了单次 tryAcquire 可申请的最大令牌数；
	// WaitN 通过分段消耗绕过此限制，支持任意大小的 n。
	ratePerSec int
}

// NewTokenBucketLimiter 创建分布式令牌桶限流器。
//
//   - key：Redis 存储令牌桶的 Hash key。多实例共享同一 key 实现全局限速。
//   - ratePerSec：每秒补充令牌数（个），同时也是桶的容量上限。
//     例：ratePerSec=150 表示所有实例合计每秒最多消耗 150 个令牌。
func NewTokenBucketLimiter(key string, ratePerSec int) *TokenBucketLimiter {
	return &TokenBucketLimiter{key: key, ratePerSec: ratePerSec}
}

// 编译期检查：确保 TokenBucketLimiter 实现 RateLimiter 接口
var _ RateLimiter = (*TokenBucketLimiter)(nil)

// tryAcquire 尝试原子地从令牌桶中消耗 n 个令牌。n 不得超过 ratePerSec。
//
// 返回值：
//   - acquired=true：令牌充足，已成功消耗 n 个令牌。
//   - acquired=false, err=nil：令牌不足，未消耗，调用方应等待后重试。
//   - acquired=false, err!=nil：Redis 调用失败（含 ctx 取消/超时错误）。
func (r *TokenBucketLimiter) tryAcquire(ctx context.Context, n int) (acquired bool, err error) {
	nowMs := time.Now().UnixMilli()
	fillPerMs := fmt.Sprintf("%.6f", float64(r.ratePerSec)/1000.0)
	res, err := redis.GetRedis(ctx).Eval(
		ctx,
		tokenBucketLua,
		[]string{r.key},
		nowMs, n, r.ratePerSec, fillPerMs,
	)
	if err != nil {
		return false, err
	}
	ok, isInt64 := res.(int64)
	if !isInt64 {
		// Lua 脚本返回值始终为整数，若类型不符说明 Redis 客户端行为异常
		return false, fmt.Errorf("token_bucket: Eval 返回类型非预期: %T", res)
	}
	return ok == 1, nil
}

// WaitN 阻塞直到可以消耗 n 个令牌后返回。
//
// 行为说明：
//   - n=0 时立即返回 nil，不访问 Redis。
//   - n > ratePerSec（桶容量）时自动分段消耗，每段申请 ≤ ratePerSec 个令牌，
//     总等待时长约为 n/ratePerSec 秒。分段保证 tokens >= chunk 可以被满足，
//   - ctx 取消或超时时立即返回对应的 ctx 错误，不触发降级。
//   - Redis 不可用（非 ctx 错误）时降级放行，返回 nil 并记录告警日志；
//     限速失效优先于服务中断，避免限流器成为单点故障。
func (r *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	remaining := n
	for remaining > 0 {
		// 分段：每段申请的令牌数不超过桶容量，确保 tokens >= chunk 最终可被满足
		chunk := remaining
		if chunk > r.ratePerSec {
			chunk = r.ratePerSec
		}
		for {
			ok, err := r.tryAcquire(ctx, chunk)
			if err != nil {
				// ctx 取消/超时：向上透传，不触发降级；调用方应处理该错误
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				// Redis 不可用：降级放行，记录告警；限速失效优先于服务中断
				log.WarnCtx(ctx, "[token_bucket] Redis 不可用，跳过限速",
					zap.String("key", r.key), zap.Error(err))
				return nil
			}
			if ok {
				remaining -= chunk
				break
			}
			// 令牌不足：估算等待 chunk 个令牌补充所需的时长后重试，
			// 避免对 Redis 产生紧密轮询（+1ms 防止浮点截断导致连续失败）
			waitDur := time.Duration(float64(chunk)/float64(r.ratePerSec)*float64(time.Second)) + time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitDur):
			}
		}
	}
	return nil
}
