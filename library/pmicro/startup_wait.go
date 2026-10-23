package pmicro

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cst"
)

// Istio 经典注入下业务容器与 istio-proxy 同时启动：iptables 已把出站劫持到 envoy，envoy 却要几秒才就绪，
// 启动最早的出站（取 Pod Identity 凭证、拉凭据、登录 Nacos）会被拒或超时。
// 这里在加载配置前轮询 pilot-agent 的就绪端点：200 = envoy 就绪；503 = sidecar 在但 envoy 未好，继续等；
// 连接被拒持续一段时间 = 没注入 sidecar（CronJob Pod 即此形态），直接放行。到上限仍未就绪只告警不阻塞，
// 后面的凭据拉取还有分类重试兜底。等待上限受探针预算约束，默认 10s，STARTUP_PROXY_WAIT_SECONDS 可覆盖，0 为关闭
var (
	sidecarReadyURL     = "http://127.0.0.1:15021/healthz/ready"
	sidecarPollInterval = 500 * time.Millisecond
	sidecarProbeTimeout = 300 * time.Millisecond
	sidecarAbsentAfter  = 3 * time.Second
)

const defaultSidecarWait = 10 * time.Second

// sidecarOutcome 探测结论
type sidecarOutcome string

const (
	sidecarReady    sidecarOutcome = "ready"    // envoy 已就绪
	sidecarAbsent   sidecarOutcome = "absent"   // 持续连接被拒，判无 sidecar
	sidecarTimeout  sidecarOutcome = "timeout"  // 到上限仍未就绪
	sidecarDisabled sidecarOutcome = "disabled" // 上限为 0，未探测
)

// sidecarWaitFromEnv 读取等待上限；未设置或非法值回落默认，负数按 0
func sidecarWaitFromEnv() time.Duration {
	v := os.Getenv(cst.ENV_STARTUP_PROXY_WAIT_SECONDS)
	if v == "" {
		return defaultSidecarWait
	}
	sec, err := strconv.Atoi(v)
	if err != nil {
		return defaultSidecarWait
	}
	if sec < 0 {
		sec = 0
	}
	return time.Duration(sec) * time.Second
}

// waitSidecarReady 轮询就绪端点直到就绪 / 判无 sidecar / 到上限 / ctx 取消，返回结论与实际等待时长
func waitSidecarReady(ctx context.Context, limit time.Duration) (sidecarOutcome, time.Duration) {
	if limit <= 0 {
		return sidecarDisabled, 0
	}
	start := time.Now()
	deadline := start.Add(limit)
	client := &http.Client{Timeout: sidecarProbeTimeout}
	var refusedSince time.Time
	for {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sidecarReadyURL, nil)
		resp, err := client.Do(req)
		now := time.Now()
		switch {
		case err != nil:
			// 连不上：可能是没注入 sidecar，也可能 pilot-agent 还没起；持续一段时间仍连不上才判无 sidecar
			if refusedSince.IsZero() {
				refusedSince = now
			}
			if now.Sub(refusedSince) >= sidecarAbsentAfter {
				return sidecarAbsent, now.Sub(start)
			}
		case resp.StatusCode == http.StatusOK:
			resp.Body.Close()
			return sidecarReady, now.Sub(start)
		default:
			// 503：sidecar 在，envoy 未就绪
			resp.Body.Close()
			refusedSince = time.Time{}
		}
		if now.Add(sidecarPollInterval).After(deadline) {
			return sidecarTimeout, now.Sub(start)
		}
		select {
		case <-ctx.Done():
			return sidecarTimeout, time.Since(start)
		case <-time.After(sidecarPollInterval):
		}
	}
}

// logSidecarWait 把探测结论落一条日志：就绪/无 sidecar/关闭为 INFO，到上限为 WARN
func logSidecarWait(outcome sidecarOutcome, waited time.Duration) {
	fields := []zap.Field{zap.String("outcome", string(outcome)), zap.Duration("waited", waited)}
	if outcome == sidecarTimeout {
		log.Warn("[Init] 等待 sidecar 就绪超时，继续启动", fields...)
		return
	}
	log.Info("[Init] sidecar 就绪探测", fields...)
}
