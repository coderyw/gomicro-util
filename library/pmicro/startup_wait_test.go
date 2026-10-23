package pmicro

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/model/cst"
)

// 把探测参数缩到毫秒级并指向测试服务器
func useProbe(t *testing.T, url string) {
	t.Helper()
	savedURL, savedPoll, savedAbsent := sidecarReadyURL, sidecarPollInterval, sidecarAbsentAfter
	t.Cleanup(func() { sidecarReadyURL, sidecarPollInterval, sidecarAbsentAfter = savedURL, savedPoll, savedAbsent })
	sidecarReadyURL, sidecarPollInterval, sidecarAbsentAfter = url, 5*time.Millisecond, 30*time.Millisecond
}

func TestWaitSidecarReady(t *testing.T) {
	t.Run("就绪端点返回 200 立即放行", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
		defer srv.Close()
		useProbe(t, srv.URL)
		outcome, waited := waitSidecarReady(context.Background(), time.Second)
		if outcome != sidecarReady || waited > 500*time.Millisecond {
			t.Fatalf("期望 ready，实际 %s waited=%v", outcome, waited)
		}
	})
	t.Run("先 503 后 200：等到就绪为止", func(t *testing.T) {
		var n int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if atomic.AddInt32(&n, 1) <= 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()
		useProbe(t, srv.URL)
		outcome, _ := waitSidecarReady(context.Background(), time.Second)
		if outcome != sidecarReady || atomic.LoadInt32(&n) != 4 {
			t.Fatalf("期望第 4 次探测就绪，实际 %s calls=%d", outcome, n)
		}
	})
	t.Run("一直 503：到上限返回 timeout", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
		defer srv.Close()
		useProbe(t, srv.URL)
		outcome, waited := waitSidecarReady(context.Background(), 40*time.Millisecond)
		if outcome != sidecarTimeout || waited > 200*time.Millisecond {
			t.Fatalf("期望 timeout，实际 %s waited=%v", outcome, waited)
		}
	})
	t.Run("端口无人监听：持续被拒后判无 sidecar 放行", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
		url := srv.URL
		srv.Close()
		useProbe(t, url)
		outcome, waited := waitSidecarReady(context.Background(), time.Second)
		if outcome != sidecarAbsent || waited < sidecarAbsentAfter || waited > 500*time.Millisecond {
			t.Fatalf("期望 absent 且约等 absentAfter，实际 %s waited=%v", outcome, waited)
		}
	})
	t.Run("上限为 0 不探测", func(t *testing.T) {
		useProbe(t, "http://127.0.0.1:1")
		if outcome, _ := waitSidecarReady(context.Background(), 0); outcome != sidecarDisabled {
			t.Fatalf("期望 disabled，实际 %s", outcome)
		}
	})
	t.Run("ctx 取消提前返回", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
		defer srv.Close()
		useProbe(t, srv.URL)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if outcome, waited := waitSidecarReady(ctx, time.Second); outcome != sidecarTimeout || waited > 200*time.Millisecond {
			t.Fatalf("期望快速返回 timeout，实际 %s waited=%v", outcome, waited)
		}
	})
}

func TestSidecarWaitFromEnv(t *testing.T) {
	for val, want := range map[string]time.Duration{"": defaultSidecarWait, "15": 15 * time.Second, "0": 0, "-1": 0, "x": defaultSidecarWait} {
		t.Setenv(cst.ENV_STARTUP_PROXY_WAIT_SECONDS, val)
		if got := sidecarWaitFromEnv(); got != want {
			t.Fatalf("值 %q 期望 %v，实际 %v", val, want, got)
		}
	}
}
