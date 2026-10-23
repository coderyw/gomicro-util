package metrics

import (
	"testing"
)

func TestSetGaugeDroppedWhenDisabled(t *testing.T) {
	reportingEnabled.Store(false)
	SetGauge("test_gauge_disabled", Tags{"db": "default"}, 1)

	if mf := gatherFamily(t, "test_gauge_disabled"); mf != nil {
		t.Fatal("metrics 未开启时 SetGauge 不应注册指标")
	}
}

func TestSetGauge(t *testing.T) {
	reportingEnabled.Store(true)
	defer reportingEnabled.Store(false)

	SetGauge("test_gauge_pool", Tags{"db": "default"}, 5)
	// Gauge 语义：Set 覆盖，不累加
	SetGauge("test_gauge_pool", Tags{"db": "default"}, 3)

	mf := gatherFamily(t, "test_gauge_pool")
	if mf == nil {
		t.Fatal("metric not registered")
	}
	m := mf.GetMetric()[0]
	if got := m.GetGauge().GetValue(); got != 3 {
		t.Errorf("gauge = %v, want 3", got)
	}
	// SetGauge 不附加 service label（公共 tag 由上报器统一附加）
	for _, lp := range m.GetLabel() {
		if lp.GetName() == "service" {
			t.Error("SetGauge 不应附加 service label")
		}
	}
}

func TestSetGaugeKeyMismatchDropped(t *testing.T) {
	reportingEnabled.Store(true)
	defer reportingEnabled.Store(false)

	SetGauge("test_gauge_mismatch", Tags{"db": "default"}, 1)
	// tag key 集合与首次不一致：丢弃，不 panic
	SetGauge("test_gauge_mismatch", Tags{"instance": "core"}, 2)

	mf := gatherFamily(t, "test_gauge_mismatch")
	if mf == nil {
		t.Fatal("metric not registered")
	}
	if got := mf.GetMetric()[0].GetGauge().GetValue(); got != 1 {
		t.Errorf("gauge = %v, want 1（key 不一致的打点应被丢弃）", got)
	}
}
