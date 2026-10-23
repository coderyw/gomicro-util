// Package ratelimit_test 对 TokenBucketLimiter 的行为进行白盒验证。
// 使用项目内置的 mock 框架替换 Redis，所有测试均不依赖真实 Redis 实例。
package ratelimit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	cachedRedis "github.com/coderyw/gomicro-util/library/db/cache/redis"
	util_mock "github.com/coderyw/gomicro-util/library/mock"
	redisMock "github.com/coderyw/gomicro-util/mocks/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/util/ratelimit"
)

// setupMockRedis 初始化带 mock Redis 的 context，返回 (ctx, mock 对象)。
// 测试结束后 mock 的 AssertExpectations 由各 case 自行调用。
func setupMockRedis(t *testing.T) (context.Context, *redisMock.MockRedisInterface) {
	t.Helper()
	ctx := util_mock.MockSetUp(context.Background(), t)
	m := cachedRedis.GetRedis(ctx).(*redisMock.MockRedisInterface)
	return ctx, m
}

// evalReturn 为 mock Eval 方法注册一次返回值。
// retVal 传 int64(1) 表示令牌充足，int64(0) 表示令牌不足，nil+err 表示 Redis 错误。
func evalReturn(m *redisMock.MockRedisInterface, retVal interface{}, err error) *mock.Call {
	return m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(retVal, err).Once()
}

// ── 基础行为 ──────────────────────────────────────────────────────────────────

// TestWaitN_Zero n=0 时立即返回，不访问 Redis
func TestWaitN_Zero(t *testing.T) {
	limiter := ratelimit.NewTokenBucketLimiter("test:rl:zero", 10)
	// 不使用 mock ctx：若触碰 Redis 则会 panic，验证确实不访问
	err := limiter.WaitN(context.Background(), 0)
	assert.NoError(t, err)
}

// TestWaitN_TokenAvailable 令牌充足时一次成功，立即返回
func TestWaitN_TokenAvailable(t *testing.T) {
	ctx, m := setupMockRedis(t)
	evalReturn(m, int64(1), nil)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:available", 10)
	err := limiter.WaitN(ctx, 1)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// TestWaitN_RetryUntilAvailable 令牌先不足后充足，正确重试
func TestWaitN_RetryUntilAvailable(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 第一次不足，第二次充足
	evalReturn(m, int64(0), nil)
	evalReturn(m, int64(1), nil)

	// ratePerSec=10000 使 waitDur ≈ 0.1ms，测试不会因等待超时
	limiter := ratelimit.NewTokenBucketLimiter("test:rl:retry", 10000)
	err := limiter.WaitN(ctx, 1)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// ── 分段消耗（n > ratePerSec） ────────────────────────────────────────────────

// TestWaitN_Chunking_Exact n 是 ratePerSec 整数倍：n=4, rate=2 → 2 次 Eval
func TestWaitN_Chunking_Exact(t *testing.T) {
	ctx, m := setupMockRedis(t)
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(1), nil).Times(2)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:chunk_exact", 2)
	err := limiter.WaitN(ctx, 4)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// TestWaitN_Chunking_Remainder n 非整数倍：n=5, rate=2 → chunk 序列 2,2,1 → 3 次 Eval
func TestWaitN_Chunking_Remainder(t *testing.T) {
	ctx, m := setupMockRedis(t)
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(1), nil).Times(3)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:chunk_rem", 2)
	err := limiter.WaitN(ctx, 5)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// TestWaitN_Chunking_Rate150_N2000 rate=150，n=2000 的真实业务场景。
// 分段序列：13 段 × 150 + 1 段 × 50 = 14 次 Eval，总消耗令牌数 = 2000。
func TestWaitN_Chunking_Rate150_N2000(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 14 次 Eval 均立即返回令牌充足（bucket 已有足够令牌）
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(1), nil).Times(14)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:rate150_n2000", 150)
	err := limiter.WaitN(ctx, 2000)

	assert.NoError(t, err)
	m.AssertExpectations(t) // 精确断言调用次数为 14
}

// TestWaitN_Chunking_Rate150_N2000_WithRetry rate=150，n=2000，部分段令牌不足需重试。
// 验证分段与重试可以组合正确工作：第 1 段首次不足，重试后成功，其余段一次成功。
func TestWaitN_Chunking_Rate150_N2000_WithRetry(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 第 1 段（chunk=150）：先不足后成功 → 2 次 Eval
	evalReturn(m, int64(0), nil)
	evalReturn(m, int64(1), nil)
	// 剩余 13 段（12 段×150 + 1 段×50）：一次成功 → 13 次 Eval
	// 合计：2 + 13 = 15 次 Eval，消耗令牌：150×13 + 50 = 2000
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(1), nil).Times(13)

	// ratePerSec=150，waitDur for chunk=150 → 约 1s；为让测试快速完成，
	// 这里仅验证逻辑正确性（mock 立即返回，不实际等待）
	limiter := ratelimit.NewTokenBucketLimiter("test:rl:rate150_n2000_retry", 150)
	err := limiter.WaitN(ctx, 2000)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// TestWaitN_Chunking_Equal n == ratePerSec：不分段，1 次 Eval
func TestWaitN_Chunking_Equal(t *testing.T) {
	ctx, m := setupMockRedis(t)
	evalReturn(m, int64(1), nil)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:chunk_eq", 5)
	err := limiter.WaitN(ctx, 5)

	assert.NoError(t, err)
	m.AssertExpectations(t)
}

// ── ctx 取消 ──────────────────────────────────────────────────────────────────

// TestWaitN_CtxCancelledDuringWait 令牌不足进入 select 时，ctx 取消立即返回 Canceled
func TestWaitN_CtxCancelledDuringWait(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 令牌持续不足 → WaitN 进入 select；ctx 已取消 → Done() 立即触发
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), nil)

	cancelCtx, cancel := context.WithCancel(ctx)
	cancel() // 提前取消

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:ctx_wait", 10000)
	err := limiter.WaitN(cancelCtx, 1)

	assert.ErrorIs(t, err, context.Canceled)
}

// TestWaitN_CtxDeadlineExceeded ctx 超时导致 tryAcquire 直接返回错误时，WaitN 透传该错误
func TestWaitN_CtxDeadlineExceeded(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 模拟 go-redis 因 ctx 超时返回 DeadlineExceeded
	evalReturn(m, nil, context.DeadlineExceeded)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:ctx_deadline", 10)
	err := limiter.WaitN(ctx, 1)

	assert.ErrorIs(t, err, context.DeadlineExceeded)
	m.AssertExpectations(t)
}

// TestWaitN_CtxCanceledFromRedis tryAcquire 直接返回 context.Canceled 时，WaitN 透传而非降级
func TestWaitN_CtxCanceledFromRedis(t *testing.T) {
	ctx, m := setupMockRedis(t)
	evalReturn(m, nil, context.Canceled)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:ctx_cancel_redis", 10)
	err := limiter.WaitN(ctx, 1)

	assert.ErrorIs(t, err, context.Canceled)
	m.AssertExpectations(t)
}

// TestWaitN_CtxTimeout context.WithTimeout 到期后返回 DeadlineExceeded
func TestWaitN_CtxTimeout(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 令牌持续不足，等待时间 ≈ waitDur；ctx 在 5ms 后超时
	m.On("Eval", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(int64(0), nil)

	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancel()

	// ratePerSec=10000 → waitDur≈0.1ms，循环多次后 ctx 超时
	limiter := ratelimit.NewTokenBucketLimiter("test:rl:ctx_timeout", 10000)
	err := limiter.WaitN(timeoutCtx, 1)

	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"expected ctx deadline/cancel, got: %v", err)
}

// ── 降级（Redis 不可用） ──────────────────────────────────────────────────────

// TestWaitN_RedisConnError Redis 连接错误（非 ctx 错误）时降级放行，返回 nil
func TestWaitN_RedisConnError(t *testing.T) {
	ctx, m := setupMockRedis(t)
	evalReturn(m, nil, errors.New("redis: connection refused"))

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:conn_err", 10)
	err := limiter.WaitN(ctx, 1)

	assert.NoError(t, err) // 降级放行
	m.AssertExpectations(t)
}

// TestWaitN_UnexpectedEvalType Eval 返回非 int64 类型，视为 Redis 异常降级放行
func TestWaitN_UnexpectedEvalType(t *testing.T) {
	ctx, m := setupMockRedis(t)
	// 返回字符串而非整数，触发类型断言失败路径
	evalReturn(m, "unexpected", nil)

	limiter := ratelimit.NewTokenBucketLimiter("test:rl:type_err", 10)
	err := limiter.WaitN(ctx, 1)

	assert.NoError(t, err) // 非 ctx 错误 → 降级放行
	m.AssertExpectations(t)
}
