package cron

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/namespace"
	"go.uber.org/zap"
)

const (
	defaultMaxConcurrency = 10
	defaultTimeout        = 6 * time.Hour
)

// LockFunc 分布式锁的抽象签名，方便测试时替换。
// serviceName 对应 Redis key 的前缀枚举（如 "bmp-activity-redpacket"），
// lockKey 为实际的锁名（如 "ok:warmup_ticket"）。
type LockFunc func(ctx context.Context, serviceName, lockKey string, fn func() error) error

// Option 用于配置 TenantCron 的可选项。
type Option func(*TenantCron)

// TenantCron 封装了租户感知的定时任务执行逻辑。
// 它的职责是：
//  1. 获取当前 namespace 下所有租户的 context
//  2. 使用带租户前缀的 key 执行分布式锁
//  3. 在锁保护下为每个租户并发执行业务逻辑
type TenantCron struct {
	serviceName    string
	lockFunc       LockFunc
	tenantsFunc    func() []context.Context
	maxConcurrency int
	timeout        time.Duration
}

// NewTenantCron 创建一个新的 TenantCron 实例。
// serviceName 用作分布式锁的 Redis key 前缀。
// 默认通过 namespace.GetTenantsInNamespace 获取租户列表，独享和共享模式均适用。
// 注意：必须通过 WithLockFunc 传入分布式锁实现（如 Redis 锁），否则默认无锁直接执行。
func NewTenantCron(serviceName string, opts ...Option) *TenantCron {
	tc := &TenantCron{
		serviceName:    serviceName,
		lockFunc:       defaultNoOpLock,
		tenantsFunc:    namespace.GetTenantsInNamespace,
		maxConcurrency: defaultMaxConcurrency,
		timeout:        defaultTimeout,
	}
	for _, opt := range opts {
		opt(tc)
	}
	return tc
}

func WithLockFunc(fn LockFunc) Option {
	return func(tc *TenantCron) {
		tc.lockFunc = fn
	}
}

// WithTenantsFunc 自定义租户 context 来源，主要用于测试。
func WithTenantsFunc(fn func() []context.Context) Option {
	return func(tc *TenantCron) {
		tc.tenantsFunc = fn
	}
}

// WithMaxConcurrency 设置最大并发执行的租户数量，默认 10。
func WithMaxConcurrency(n int) Option {
	return func(tc *TenantCron) {
		if n > 0 {
			tc.maxConcurrency = n
		}
	}
}

// WithTimeout 设置每个租户任务的超时时间，默认 6 小时。
func WithTimeout(d time.Duration) Option {
	return func(tc *TenantCron) {
		if d > 0 {
			tc.timeout = d
		}
	}
}

// Wrap 将一个 func(ctx) 包装成兼容 CronJob.DoJob 的 func()。
// 自动完成：遍历租户 → 构建租户级锁 key → 加锁执行。
// 多个租户并发执行（受 maxConcurrency 限制），单个租户失败不影响其他租户。
func (tc *TenantCron) Wrap(lockName string, job func(ctx context.Context)) func() {
	return func() {
		// 兜底租户模式：数据层保持原始命名，不区分租户，用无租户 ctx 执行一次即可。
		// 不能复用租户列表里的 ctx，否则锁 key 会被拼上租户前缀
		if env.HasFallbackTenant() {
			tc.execForTenant(context.Background(), lockName, job)
			return
		}
		tenants := tc.tenantsFunc()
		if len(tenants) == 0 {
			return
		}
		// 单租户直接同步执行，无需额外并发开销
		if len(tenants) == 1 {
			tc.execForTenant(tenants[0], lockName, job)
			return
		}
		// 多租户并发执行，受信号量限制
		sem := make(chan struct{}, tc.maxConcurrency)
		var wg sync.WaitGroup
		for _, ctx := range tenants {
			sem <- struct{}{} // 在主 goroutine 阻塞，控制同时存在的 goroutine 数量
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				defer func() { <-sem }()
				tc.execForTenant(ctx, lockName, job)
			}(ctx)
		}
		wg.Wait()
	}
}

func (tc *TenantCron) execForTenant(ctx context.Context, lockName string, job func(ctx context.Context)) {
	if tc.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, tc.timeout)
		defer cancel()
	}
	tc.execWithLock(ctx, lockName, job)
}

func (tc *TenantCron) execWithLock(ctx context.Context, lockName string, job func(ctx context.Context)) {
	ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	lockKey := buildLockKeyFromNS(ns, lockName)

	err := tc.lockFunc(ctx, tc.serviceName, lockKey, func() error {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorCtx(ctx, "TenantCron panic recovered",
					zap.String("lockName", lockName),
					zap.String("namespace", ns),
					zap.Any("panic", r))
			}
		}()
		job(ctx)
		return nil
	})
	if err != nil {
		log.ErrorCtx(ctx, "TenantCron lock failed",
			zap.String("lockKey", lockKey),
			zap.String("namespace", ns),
			zap.Error(err))
	}
}

func buildLockKeyFromNS(ns, lockName string) string {
	if ns != "" {
		return fmt.Sprintf("%s:%s", namespace.NormalizeTenantName(ns), lockName)
	}
	return lockName
}

// defaultNoOpLock 不加锁，直接执行 fn。
// 生产环境应通过 WithLockFunc 注入分布式锁实现（如 Redis LockWithRenewal）。
func defaultNoOpLock(_ context.Context, _, _ string, fn func() error) error {
	return fn()
}
