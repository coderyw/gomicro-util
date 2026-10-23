package secret

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/model/cst"
)

// 固定重试参数并记录每次等待，返回等待序列
func captureRetry(t *testing.T, base, cap, deadline time.Duration, rnd float64) *[]time.Duration {
	t.Helper()
	savedBase, savedCap, savedDeadline, savedRand, savedSleep := retryBase, retryCap, retryDeadline, retryRand, retrySleep
	t.Cleanup(func() {
		retryBase, retryCap, retryDeadline, retryRand, retrySleep = savedBase, savedCap, savedDeadline, savedRand, savedSleep
	})
	retryBase, retryCap, retryDeadline = base, cap, deadline
	retryRand = func() float64 { return rnd }
	waits := &[]time.Duration{}
	retrySleep = func(_ context.Context, d time.Duration) error {
		*waits = append(*waits, d)
		return nil
	}
	return waits
}

func TestRetry(t *testing.T) {
	retryable := errors.New("network timeout")

	t.Run("可重试错误按指数退避重试直到成功", func(t *testing.T) {
		waits := captureRetry(t, time.Second, 5*time.Second, time.Hour, 1)
		calls := 0
		err := retry(context.Background(), "fakesm://n", func(context.Context) error {
			calls++
			if calls <= 3 {
				return retryable
			}
			return nil
		})
		if err != nil || calls != 4 {
			t.Fatalf("期望第 4 次成功，实际 err=%v calls=%d", err, calls)
		}
		want := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
		if len(*waits) != 3 || (*waits)[0] != want[0] || (*waits)[1] != want[1] || (*waits)[2] != want[2] {
			t.Fatalf("退避序列不符: %v", *waits)
		}
	})
	t.Run("等待上界封顶在 cap", func(t *testing.T) {
		waits := captureRetry(t, time.Second, 3*time.Second, time.Hour, 1)
		calls := 0
		_ = retry(context.Background(), "fakesm://n", func(context.Context) error {
			calls++
			if calls <= 5 {
				return retryable
			}
			return nil
		})
		for i, w := range *waits {
			if w > 3*time.Second {
				t.Fatalf("第 %d 次等待 %v 超过 cap", i+1, w)
			}
		}
		if (*waits)[4] != 3*time.Second {
			t.Fatalf("第 5 次等待应钉在 cap，实际 %v", (*waits)[4])
		}
	})
	t.Run("全抖动：随机因子缩放等待且不超过上界", func(t *testing.T) {
		waits := captureRetry(t, time.Second, 5*time.Second, time.Hour, 0.5)
		calls := 0
		_ = retry(context.Background(), "fakesm://n", func(context.Context) error {
			calls++
			if calls <= 2 {
				return retryable
			}
			return nil
		})
		if (*waits)[0] != 500*time.Millisecond || (*waits)[1] != time.Second {
			t.Fatalf("抖动缩放不符: %v", *waits)
		}
	})
	t.Run("deadline 不够下一次等待时放弃并返回最后错误", func(t *testing.T) {
		// base 1s、deadline 1.5s：第一次等 1s 可以，第二次需等 2s 超预算 → 放弃，共调用 2 次
		waits := captureRetry(t, time.Second, 5*time.Second, 1500*time.Millisecond, 1)
		calls := 0
		err := retry(context.Background(), "fakesm://n", func(context.Context) error {
			calls++
			return retryable
		})
		if !errors.Is(err, retryable) || len(*waits) != 1 {
			t.Fatalf("期望等待 1 次后放弃，实际 err=%v waits=%v calls=%d", err, *waits, calls)
		}
	})
	t.Run("deadline 为 0 不重试", func(t *testing.T) {
		waits := captureRetry(t, time.Second, 5*time.Second, 0, 1)
		calls := 0
		err := retry(context.Background(), "fakesm://n", func(context.Context) error { calls++; return retryable })
		if !errors.Is(err, retryable) || calls != 1 || len(*waits) != 0 {
			t.Fatalf("期望只调用 1 次且不等待，实际 calls=%d waits=%v err=%v", calls, *waits, err)
		}
	})
	t.Run("不可重试错误立即返回", func(t *testing.T) {
		waits := captureRetry(t, time.Second, 5*time.Second, time.Hour, 1)
		calls := 0
		err := retry(context.Background(), "fakesm://n", func(context.Context) error { calls++; return NonRetryable(errors.New("403")) })
		if err == nil || calls != 1 || len(*waits) != 0 {
			t.Fatalf("期望不重试，实际 calls=%d waits=%v", calls, *waits)
		}
	})
	t.Run("ctx 取消在等待期间返回 ctx 错误", func(t *testing.T) {
		captureRetry(t, time.Second, 5*time.Second, time.Hour, 1)
		retrySleep = sleepCtx
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := retry(ctx, "fakesm://n", func(context.Context) error { return retryable })
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("期望 context.Canceled，实际 %v", err)
		}
	})
}

func TestRetryDeadlineFromEnv(t *testing.T) {
	cases := []struct {
		name string
		val  string
		want time.Duration
	}{
		{"未设置用默认", "", defaultRetryDeadline},
		{"合法秒数", "45", 45 * time.Second},
		{"0 表示不重试", "0", 0},
		{"负数按 0", "-3", 0},
		{"非法值回落默认", "abc", defaultRetryDeadline},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.val == "" {
				t.Setenv(cst.ENV_SECRET_RETRY_DEADLINE_SECONDS, "")
			} else {
				t.Setenv(cst.ENV_SECRET_RETRY_DEADLINE_SECONDS, c.val)
			}
			if got := retryDeadlineFromEnv(); got != c.want {
				t.Fatalf("期望 %v，实际 %v", c.want, got)
			}
		})
	}
}

func TestIsRef(t *testing.T) {
	setup(t, &fakeProvider{scheme: "fakesm", data: map[string]string{}})
	cases := map[string]bool{
		"fakesm://ns-uat/nacos_username": true,
		" fakesm://ns-uat ":              true,
		"fakesm://ns-uat/x/y":            false, // 多级路径不是合法引用
		"plain-password":                 false,
		"":                               false,
		"https://example.com/a":          false, // 忽略名单
		"nosuch://ns-uat/k":              false, // 未注册 scheme
		"prefixfakesm://ns-uat/k":        false, // 拼接在其他内容里
	}
	for in, want := range cases {
		if got := IsRef(in); got != want {
			t.Fatalf("IsRef(%q) 期望 %v，实际 %v", in, want, got)
		}
	}
}

// blockUntilDone 模拟连接建立后响应停滞：一直阻塞到 ctx 结束
func blockUntilDone(calls *int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		*calls++
		<-ctx.Done()
		return ctx.Err()
	}
}

func TestRetry_单次超时(t *testing.T) {
	t.Run("单次请求停滞按单次超时失败并重试，总耗时不超过 deadline", func(t *testing.T) {
		captureRetry(t, time.Millisecond, time.Millisecond, 100*time.Millisecond, 1)
		retryAttemptTimeout = 20 * time.Millisecond
		calls := 0
		start := time.Now()
		err := retry(context.Background(), "fakesm://n", blockUntilDone(&calls))
		elapsed := time.Since(start)
		if !errors.Is(err, ErrAttemptTimeout) {
			t.Fatalf("期望 ErrAttemptTimeout，实际 %v", err)
		}
		if calls < 2 {
			t.Fatalf("单次超时应可重试，期望至少 2 次尝试，实际 %d", calls)
		}
		if elapsed > 150*time.Millisecond {
			t.Fatalf("总耗时 %v 超出 deadline 100ms 过多", elapsed)
		}
	})
	t.Run("单次预算不超过 deadline 剩余时间", func(t *testing.T) {
		captureRetry(t, time.Millisecond, time.Millisecond, 50*time.Millisecond, 1)
		retryAttemptTimeout = time.Hour
		calls := 0
		start := time.Now()
		err := retry(context.Background(), "fakesm://n", blockUntilDone(&calls))
		if elapsed := time.Since(start); elapsed > 100*time.Millisecond || !errors.Is(err, ErrAttemptTimeout) {
			t.Fatalf("期望约 50ms 内以超时放弃，实际 elapsed=%v err=%v", elapsed, err)
		}
	})
	t.Run("deadline 为 0 时只尝试一次，仍受单次超时约束", func(t *testing.T) {
		captureRetry(t, time.Millisecond, time.Millisecond, 0, 1)
		retryAttemptTimeout = 20 * time.Millisecond
		calls := 0
		start := time.Now()
		err := retry(context.Background(), "fakesm://n", blockUntilDone(&calls))
		if calls != 1 || !errors.Is(err, ErrAttemptTimeout) || time.Since(start) > 100*time.Millisecond {
			t.Fatalf("期望 1 次尝试后超时返回，实际 calls=%d err=%v", calls, err)
		}
	})
	t.Run("调用方 ctx 取消返回 ctx 错误而不是单次超时", func(t *testing.T) {
		captureRetry(t, time.Millisecond, time.Millisecond, time.Hour, 1)
		retryAttemptTimeout = time.Hour
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		calls := 0
		err := retry(ctx, "fakesm://n", blockUntilDone(&calls))
		if !errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ErrAttemptTimeout) || calls != 1 {
			t.Fatalf("期望 1 次尝试后返回调用方 ctx 错误，实际 calls=%d err=%v", calls, err)
		}
	})
}

func TestAttemptWithTimeout_不可重试错误不被改写为超时(t *testing.T) {
	notFound := NonRetryable(errors.New("secret not found"))
	err := attemptWithTimeout(context.Background(), 10*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return notFound
	})
	if !errors.Is(err, notFound) || IsRetryable(err) {
		t.Fatalf("期望保留不可重试错误，实际 %v", err)
	}
}
