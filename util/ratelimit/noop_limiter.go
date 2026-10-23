package ratelimit

import "context"

// NoopLimiter 是用于测试的空实现限流器，WaitN 始终立即返回 nil。
// 在单元测试中使用 NoopLimiter 替换 TokenBucketLimiter，避免依赖真实 Redis。
type NoopLimiter struct{}

// WaitN 立即返回 nil，不执行任何限流逻辑。
func (NoopLimiter) WaitN(ctx context.Context, n int) error {
	return nil
}

// 编译期检查：确保 NoopLimiter 实现 RateLimiter 接口
var _ RateLimiter = NoopLimiter{}
