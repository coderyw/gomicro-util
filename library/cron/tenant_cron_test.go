package cron

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

func stubLock(ctx context.Context, svc, key string, fn func() error) error {
	return fn()
}

func makeTenantCtxs(namespaces ...string) func() []context.Context {
	return func() []context.Context {
		ctxs := make([]context.Context, len(namespaces))
		for i, ns := range namespaces {
			ctxs[i] = metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
		}
		return ctxs
	}
}

// ==================== 单租户测试 ====================

func TestWrap_InjectsNamespaceIntoContext(t *testing.T) {
	var capturedNS string

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
	)

	job := tc.Wrap("test_job", func(ctx context.Context) {
		capturedNS = metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	})

	job()

	if capturedNS != "ok" {
		t.Errorf("expected namespace 'ok', got '%s'", capturedNS)
	}
}

func TestWrap_LockKeyPrefixedWithNamespace(t *testing.T) {
	var capturedKey string

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(func(_ context.Context, _, key string, fn func() error) error {
			capturedKey = key
			return fn()
		}),
	)

	tc.Wrap("warmup_ticket", func(ctx context.Context) {})()

	if capturedKey != "ok:warmup_ticket" {
		t.Errorf("expected lock key 'ok:warmup_ticket', got '%s'", capturedKey)
	}
}

func TestWrap_LockKeyUnchangedWithoutNamespace(t *testing.T) {
	var capturedKey string

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(func() []context.Context {
			return []context.Context{context.Background()}
		}),
		WithLockFunc(func(_ context.Context, _, key string, fn func() error) error {
			capturedKey = key
			return fn()
		}),
	)

	tc.Wrap("warmup_ticket", func(ctx context.Context) {})()

	if capturedKey != "warmup_ticket" {
		t.Errorf("expected lock key 'warmup_ticket', got '%s'", capturedKey)
	}
}

func TestWrap_ServiceNamePassedToLockFunc(t *testing.T) {
	var capturedSvc string

	tc := NewTenantCron("bmp-activity-redpacket",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(func(_ context.Context, svc, _ string, fn func() error) error {
			capturedSvc = svc
			return fn()
		}),
	)

	tc.Wrap("test_job", func(ctx context.Context) {})()

	if capturedSvc != "bmp-activity-redpacket" {
		t.Errorf("expected 'bmp-activity-redpacket', got '%s'", capturedSvc)
	}
}

func TestWrap_JobExecutedExactlyOnce(t *testing.T) {
	var count int32

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		atomic.AddInt32(&count, 1)
	})()

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected job executed once, got %d", count)
	}
}

func TestWrap_LockFailureSkipsJob(t *testing.T) {
	var executed bool

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(func(_ context.Context, _, _ string, _ func() error) error {
			return fmt.Errorf("lock acquisition failed")
		}),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		executed = true
	})()

	if executed {
		t.Error("expected job not to execute when lock fails")
	}
}

func TestWrap_RecoversPanic(t *testing.T) {
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
	)

	job := tc.Wrap("test_job", func(ctx context.Context) {
		panic("boom")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected panic to be recovered inside wrapper, got: %v", r)
		}
	}()
	job()
}

// ==================== 多租户测试 ====================

func TestWrap_ExecutesForEachTenant(t *testing.T) {
	var mu sync.Mutex
	executedNS := make(map[string]bool)

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "yy")),
		WithLockFunc(stubLock),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
		mu.Lock()
		executedNS[ns] = true
		mu.Unlock()
	})()

	if len(executedNS) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executedNS))
	}
	if !executedNS["ok"] || !executedNS["yy"] {
		t.Errorf("expected namespaces {ok, yy}, got %v", executedNS)
	}
}

func TestWrap_LockKeyPerTenant(t *testing.T) {
	var mu sync.Mutex
	capturedKeys := make(map[string]bool)

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "yy")),
		WithLockFunc(func(_ context.Context, _, key string, fn func() error) error {
			mu.Lock()
			capturedKeys[key] = true
			mu.Unlock()
			return fn()
		}),
	)

	tc.Wrap("warmup", func(ctx context.Context) {})()

	if len(capturedKeys) != 2 {
		t.Fatalf("expected 2 lock keys, got %d", len(capturedKeys))
	}
	if !capturedKeys["ok:warmup"] || !capturedKeys["yy:warmup"] {
		t.Errorf("expected lock keys {ok:warmup, yy:warmup}, got %v", capturedKeys)
	}
}

func TestWrap_ContinuesOnLockFailure(t *testing.T) {
	var executedCount int32
	// 按 namespace 决定是否失败，避免依赖并发顺序
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "yy")),
		WithLockFunc(func(ctx context.Context, _, _ string, fn func() error) error {
			ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
			if ns == "ok" {
				return fmt.Errorf("lock failed for tenant ok")
			}
			return fn()
		}),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		atomic.AddInt32(&executedCount, 1)
	})()

	if atomic.LoadInt32(&executedCount) != 1 {
		t.Fatalf("expected 1 execution (yy only), got %d", executedCount)
	}
}

func TestWrap_RecoversPanicAndContinues(t *testing.T) {
	var mu sync.Mutex
	var executedNS []string

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "yy")),
		WithLockFunc(stubLock),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
		mu.Lock()
		executedNS = append(executedNS, ns)
		mu.Unlock()
		if ns == "ok" {
			panic("boom from ok tenant")
		}
	})()

	if len(executedNS) != 2 {
		t.Errorf("expected 2 executions (panic recovered, continued to next), got %d", len(executedNS))
	}
}

// ==================== 并发控制测试 ====================

func TestWithMaxConcurrency(t *testing.T) {
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
		WithMaxConcurrency(5),
	)

	if tc.maxConcurrency != 5 {
		t.Errorf("expected maxConcurrency=5, got %d", tc.maxConcurrency)
	}
}

func TestWithTimeout(t *testing.T) {
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
		WithTimeout(30*time.Second),
	)

	if tc.timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", tc.timeout)
	}
}

func TestNewTenantCron_DefaultValues(t *testing.T) {
	tc := NewTenantCron("test-svc")

	if tc.maxConcurrency != defaultMaxConcurrency {
		t.Errorf("expected default maxConcurrency=%d, got %d", defaultMaxConcurrency, tc.maxConcurrency)
	}
	if tc.timeout != defaultTimeout {
		t.Errorf("expected default timeout=%v, got %v", defaultTimeout, tc.timeout)
	}
}

func TestWrap_ConcurrencyLimit(t *testing.T) {
	const maxConc = 3
	namespaces := make([]string, 10)
	for i := range namespaces {
		namespaces[i] = fmt.Sprintf("tenant-%d", i)
	}

	var peak int32
	var current int32

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs(namespaces...)),
		WithLockFunc(stubLock),
		WithMaxConcurrency(maxConc),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		cur := atomic.AddInt32(&current, 1)
		for {
			old := atomic.LoadInt32(&peak)
			if cur <= old || atomic.CompareAndSwapInt32(&peak, old, cur) {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&current, -1)
	})()

	observedPeak := atomic.LoadInt32(&peak)
	if observedPeak > int32(maxConc) {
		t.Errorf("peak concurrency %d exceeded maxConcurrency %d", observedPeak, maxConc)
	}
	if observedPeak == 0 {
		t.Error("expected at least one execution")
	}
}

func TestWrap_SingleTenantNoGoroutine(t *testing.T) {
	var executed bool
	var executedGoroutineID sync.Mutex // just to verify completion

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("solo")),
		WithLockFunc(stubLock),
		WithMaxConcurrency(3),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		executedGoroutineID.Lock()
		executed = true
		executedGoroutineID.Unlock()
	})()

	if !executed {
		t.Error("expected single tenant job to be executed")
	}
}

func TestWrap_TimeoutCancelsJob(t *testing.T) {
	var ctxCancelled int32

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok")),
		WithLockFunc(stubLock),
		WithTimeout(100*time.Millisecond),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		select {
		case <-ctx.Done():
			atomic.StoreInt32(&ctxCancelled, 1)
		case <-time.After(500 * time.Millisecond):
		}
	})()

	if atomic.LoadInt32(&ctxCancelled) != 1 {
		t.Error("expected context to be cancelled after timeout")
	}
}

func TestWrap_TimeoutDoesNotBlockOtherTenants(t *testing.T) {
	var completedCount int32

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("slow", "fast")),
		WithLockFunc(stubLock),
		WithTimeout(100*time.Millisecond),
		WithMaxConcurrency(10),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		ns := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
		if ns == "slow" {
			select {
			case <-ctx.Done():
			case <-time.After(500 * time.Millisecond):
			}
		}
		atomic.AddInt32(&completedCount, 1)
	})()

	if atomic.LoadInt32(&completedCount) != 2 {
		t.Errorf("expected 2 tenants to complete, got %d", completedCount)
	}
}

func TestWrap_EmptyTenantsDoesNothing(t *testing.T) {
	var executed bool

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(func() []context.Context { return nil }),
		WithLockFunc(stubLock),
	)

	tc.Wrap("test_job", func(ctx context.Context) {
		executed = true
	})()

	if executed {
		t.Error("expected no execution when tenants list is empty")
	}
}

// ==================== 兜底租户模式 ====================

// 兜底：配置兜底租户后，即使有多个租户也只执行一次，且 ctx 不携带租户、锁 key 不加前缀
func TestWrap_FallbackTenant_ExecutesOnceWithoutNamespace(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	var execCount atomic.Int32
	var capturedNS, capturedKey string

	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "foo", "bar")),
		WithLockFunc(func(_ context.Context, _, key string, fn func() error) error {
			capturedKey = key
			return fn()
		}),
	)

	tc.Wrap("warmup_ticket", func(ctx context.Context) {
		execCount.Add(1)
		capturedNS = metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
	})()

	if execCount.Load() != 1 {
		t.Errorf("兜底租户模式应只执行一次, got %d", execCount.Load())
	}
	if capturedNS != "" {
		t.Errorf("兜底租户模式下 ctx 不应携带租户, got %q", capturedNS)
	}
	if capturedKey != "warmup_ticket" {
		t.Errorf("兜底租户模式下锁 key 不应加租户前缀, got %q", capturedKey)
	}
}

// 兜底：即使租户列表为空也应执行（不依赖租户配置）
func TestWrap_FallbackTenant_ExecutesEvenWithoutTenants(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	executed := false
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs()),
		WithLockFunc(stubLock),
	)
	tc.Wrap("job", func(ctx context.Context) { executed = true })()

	if !executed {
		t.Error("兜底租户模式下即使租户列表为空也应执行")
	}
}

// 反向：未配置兜底租户，多租户仍按租户各执行一次（原有行为不受影响）
func TestWrap_NoFallback_MultiTenantFansOut(t *testing.T) {
	env.SetFallbackTenantName("")

	var execCount atomic.Int32
	tc := NewTenantCron("test-svc",
		WithTenantsFunc(makeTenantCtxs("ok", "foo")),
		WithLockFunc(stubLock),
	)
	tc.Wrap("job", func(ctx context.Context) { execCount.Add(1) })()

	if execCount.Load() != 2 {
		t.Errorf("未配置兜底租户时应按租户各执行一次, got %d", execCount.Load())
	}
}

func TestTenantCaseLockNameCompatibility(t *testing.T) {
	for _, ns := range []string{"ok", "OK", "Ok"} {
		if got := buildLockKeyFromNS(ns, "Warmup:ID"); got != "ok:Warmup:ID" {
			t.Fatal(got)
		}
	}
	if got := buildLockKeyFromNS("", "Warmup:ID"); got != "Warmup:ID" {
		t.Fatal(got)
	}
}
