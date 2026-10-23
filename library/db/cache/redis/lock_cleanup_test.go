package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/env"
)

func TestRenewalCleanupAfterCancellation(t *testing.T) {
	for _, tc := range []struct {
		name                                 string
		deadline, panics, replaced, fallback bool
	}{
		{name: "cancel"}, {name: "deadline", deadline: true}, {name: "panic", panics: true},
		{name: "different_owner", replaced: true}, {name: "fallback", fallback: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env.SetFallbackTenantName("")
			if tc.fallback {
				env.SetFallbackTenantName("ns")
			}
			t.Cleanup(func() { env.SetFallbackTenantName("") })
			r, mr := newScanTestRedis(t)
			base := tenantTestCtx("OK")
			ctx, cancel := context.WithCancel(base)
			if tc.deadline {
				cancel()
				ctx, cancel = context.WithTimeout(base, 30*time.Millisecond)
			}
			defer cancel()
			key := "tenantok:Login:UserID"
			if tc.fallback {
				key = "Login:UserID"
			}
			sentinel := errors.New("business error")
			var recovered any
			var gotErr error
			func() {
				defer func() { recovered = recover() }()
				_, gotErr = r.LockWithRenewal(ctx, "", "Login:UserID", 5*time.Second, time.Hour, func() error {
					if tc.deadline {
						<-ctx.Done()
					} else {
						cancel()
					}
					if !mr.Exists(key) {
						t.Fatal("lock released before business callback finished")
					}
					if tc.replaced {
						if err := mr.Set(key, "new-owner"); err != nil {
							t.Fatal(err)
						}
					}
					if tc.panics {
						panic("business panic")
					}
					return sentinel
				})
			}()
			if tc.panics {
				if recovered != "business panic" {
					t.Fatalf("panic changed: %v", recovered)
				}
			} else if !errors.Is(gotErr, sentinel) {
				t.Fatalf("business error changed: %v", gotErr)
			}
			if tc.replaced {
				if v, err := mr.Get(key); err != nil || v != "new-owner" {
					t.Fatalf("other owner's lock deleted: %q, %v", v, err)
				}
			} else if mr.Exists(key) {
				t.Fatal("lock remained after callback exited")
			}
		})
	}
}

// Observe Done being selected even when the renewal tick is an hour away.
// This catches implementations that notice cancellation only on the next Eval.
type renewalCancelContext struct {
	context.Context
	observed chan struct{}
}

func (c renewalCancelContext) Done() <-chan struct{} {
	select {
	case c.observed <- struct{}{}:
	default:
	}
	return c.Context.Done()
}

func TestRenewalObservesCancellationWithoutWaitingForTick(t *testing.T) {
	for _, max := range [][]time.Duration{nil, {time.Hour}} {
		r, _ := newScanTestRedis(t)
		base, cancel := context.WithCancel(context.Background())
		ctx := renewalCancelContext{Context: base, observed: make(chan struct{}, 1)}
		stop := r.RenewalLock(ctx, "", "lock", "owner", time.Hour, 5*time.Second, max...)
		cancel()
		select {
		case <-ctx.observed:
		case <-time.After(time.Second):
			t.Error("renewal does not observe ctx cancellation promptly")
		}
		stop()
	}
}

func TestRenewalSurvivesRequestCancellationUntilBusinessExits(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)
	ctx, cancel := context.WithCancel(tenantTestCtx("ok"))
	defer cancel()
	acquired, err := r.LockWithRenewal(ctx, "", "BusinessLock", 5*time.Second, 10*time.Millisecond, func() error {
		cancel()
		// Advance the server clock beyond the original lease cumulatively. The
		// callback stays active throughout, so every cycle must renew the same lock.
		for i := 0; i < 3; i++ {
			mr.FastForward(4 * time.Second)
			deadline := time.Now().Add(2 * time.Second)
			for mr.TTL("tenantok:BusinessLock") < 5*time.Second {
				if !mr.Exists("tenantok:BusinessLock") || time.Now().After(deadline) {
					t.Fatal("request cancellation stopped renewal while business was running")
				}
				time.Sleep(time.Millisecond)
			}
		}
		return nil
	})
	if !acquired || err != nil {
		t.Fatalf("lock result: %v %v", acquired, err)
	}
	if mr.Exists("tenantok:BusinessLock") {
		t.Fatal("business exit did not release lock")
	}
}
