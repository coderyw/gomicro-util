package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	srv_config "github.com/coderyw/gomicro-util/model/config"
)

// TestInitReportE2E 用假 DataWay 验证 Init → 后台定时上报 → 行协议落地整条链路，
// 并覆盖停机时的最后一轮上报与 token 传递。
func TestInitReportE2E(t *testing.T) {
	var (
		mu      sync.Mutex
		bodies  []string
		gotPath string
		gotTok  string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		gotPath = r.URL.Path
		gotTok = r.URL.Query().Get("token")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := Init(srv_config.MetricsCfg{
		Enable:    true,
		IntervalS: 1,
		Truewatch: srv_config.TruewatchCfg{
			DatawayUrl:     srv.URL,
			WorkspaceToken: "tkn_test",
		},
	}, "unit-test-svc")
	if c.CloseFunc == nil {
		t.Fatal("expected CloseFunc when enabled")
	}

	time.Sleep(1500 * time.Millisecond) // 等一轮定时上报
	c.CloseFunc()                       // 停机触发最后一轮

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) == 0 {
		t.Fatal("no report received")
	}
	if gotPath != "/v1/write/metric" {
		t.Errorf("unexpected path %q", gotPath)
	}
	if gotTok != "tkn_test" {
		t.Errorf("unexpected token %q", gotTok)
	}
	payload := bodies[0]
	// 全局 registry 默认自带 go 运行时指标
	if !strings.Contains(payload, "go_goroutines") {
		t.Errorf("payload missing go runtime metrics:\n%.500s", payload)
	}
	// 公共 tag
	if !strings.Contains(payload, "service=unit-test-svc") {
		t.Errorf("payload missing service tag:\n%.500s", payload)
	}
	// measurement 为 serviceName
	if !strings.HasPrefix(payload, "unit-test-svc,") {
		t.Errorf("payload measurement unexpected:\n%.200s", payload)
	}
}

// TestInitDisabledOrBadConfig 未开启/配置不全时必须安全降级
func TestInitDisabledOrBadConfig(t *testing.T) {
	if c := Init(srv_config.MetricsCfg{Enable: false}, "svc"); c.CloseFunc != nil {
		t.Error("disabled should return empty CloseS")
	}
	// 开启但缺 token：降级不上报，且不 panic
	if c := Init(srv_config.MetricsCfg{
		Enable:    true,
		Truewatch: srv_config.TruewatchCfg{DatawayUrl: "https://x"},
	}, "svc"); c.CloseFunc != nil {
		t.Error("bad config should degrade to no-op")
	}
}
