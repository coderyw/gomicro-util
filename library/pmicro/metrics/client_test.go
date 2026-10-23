package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// gatherFamily 从全局 registry 取指定指标，找不到返回 nil
func gatherFamily(t *testing.T, name string) *dto.MetricFamily {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

func TestClientInc(t *testing.T) {
	c := NewClient("svc-a")
	c.Inc("test_client_inc_total", Tags{"endpoint": "place_bet", "ret": "0000"}, 1)
	c.Inc("test_client_inc_total", Tags{"endpoint": "place_bet", "ret": "0000"}, 2)

	mf := gatherFamily(t, "test_client_inc_total")
	if mf == nil {
		t.Fatal("metric not registered")
	}
	m := mf.GetMetric()[0]
	if got := m.GetCounter().GetValue(); got != 3 {
		t.Errorf("counter = %v, want 3", got)
	}
	// service tag 自动附加
	var hasService bool
	for _, lp := range m.GetLabel() {
		if lp.GetName() == "service" && lp.GetValue() == "svc-a" {
			hasService = true
		}
	}
	if !hasService {
		t.Error("missing auto-attached service label")
	}
}

func TestClientLatency(t *testing.T) {
	c := NewClient("svc-a")
	c.Latency("test_client_latency_seconds", Tags{"dep": "redis"}, 0.05)
	c.Latency("test_client_latency_seconds", Tags{"dep": "redis"}, 0.2)

	mf := gatherFamily(t, "test_client_latency_seconds")
	if mf == nil {
		t.Fatal("metric not registered")
	}
	h := mf.GetMetric()[0].GetHistogram()
	if h.GetSampleCount() != 2 {
		t.Errorf("sample count = %d, want 2", h.GetSampleCount())
	}
}

func TestClientInconsistentKeysDropped(t *testing.T) {
	c := NewClient("svc-a")
	c.Inc("test_client_keys_total", Tags{"a": "1"}, 1)
	c.Inc("test_client_keys_total", Tags{"b": "1"}, 1) // key 集合不一致，应丢弃

	mf := gatherFamily(t, "test_client_keys_total")
	if mf == nil {
		t.Fatal("metric not registered")
	}
	if got := mf.GetMetric()[0].GetCounter().GetValue(); got != 1 {
		t.Errorf("counter = %v, want 1 (second point dropped)", got)
	}
}

func TestClientNegativeValueDropped(t *testing.T) {
	c := NewClient("svc-a")
	c.Inc("test_client_negative_total", Tags{}, -5)
	if mf := gatherFamily(t, "test_client_negative_total"); mf != nil {
		t.Error("negative Inc should not register/record")
	}
}

func TestClientNameSanitized(t *testing.T) {
	c := NewClient("svc-a")
	c.Inc("test-client-dash.total", Tags{}, 1)
	if mf := gatherFamily(t, "test_client_dash_total"); mf == nil {
		t.Error("dashed name should be sanitized and registered")
	}
}

func TestClientDisabledByConfig(t *testing.T) {
	businessDisabled.Store(true)
	defer businessDisabled.Store(false)

	c := NewClient("svc-a")
	c.Inc("test_client_disabled_total", Tags{}, 1)
	if mf := gatherFamily(t, "test_client_disabled_total"); mf != nil {
		t.Error("disabled business metrics should be noop")
	}
}

func TestClientNoPanicOnBadLabel(t *testing.T) {
	c := NewClient("svc-a")
	// label 名清洗后合法；即便内部注册失败也不允许 panic 到业务路径
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic leaked to caller: %v", r)
		}
	}()
	c.Inc("test_client_badlabel_total", Tags{"用户": "x"}, 1)
}
