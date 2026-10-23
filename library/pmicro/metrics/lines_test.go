package metrics

import (
	"math"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	dto "github.com/prometheus/client_model/go"
)

func mf(name string, t dto.MetricType, metrics ...*dto.Metric) *dto.MetricFamily {
	return &dto.MetricFamily{Name: proto.String(name), Type: t.Enum(), Metric: metrics}
}

func label(k, v string) *dto.LabelPair {
	return &dto.LabelPair{Name: proto.String(k), Value: proto.String(v)}
}

func TestToLineProtocolCounterGauge(t *testing.T) {
	families := []*dto.MetricFamily{
		mf("requests_total", dto.MetricType_COUNTER, &dto.Metric{
			Label:   []*dto.LabelPair{label("endpoint", "place_bet")},
			Counter: &dto.Counter{Value: proto.Float64(42)},
		}),
		mf("go_goroutines", dto.MetricType_GAUGE, &dto.Metric{
			Gauge: &dto.Gauge{Value: proto.Float64(88)},
		}),
	}
	got := toLineProtocol("my-svc", map[string]string{"service": "my-svc"}, families, 1000)
	want := "my-svc,endpoint=place_bet,service=my-svc requests_total=42 1000\n" +
		"my-svc,service=my-svc go_goroutines=88 1000\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestToLineProtocolHistogram(t *testing.T) {
	families := []*dto.MetricFamily{
		mf("latency_seconds", dto.MetricType_HISTOGRAM, &dto.Metric{
			Histogram: &dto.Histogram{
				SampleCount: proto.Uint64(10),
				SampleSum:   proto.Float64(1.5),
				Bucket: []*dto.Bucket{
					{UpperBound: proto.Float64(0.1), CumulativeCount: proto.Uint64(7)},
					{UpperBound: proto.Float64(math.Inf(1)), CumulativeCount: proto.Uint64(10)},
				},
			},
		}),
	}
	got := toLineProtocol("svc", nil, families, 1000)
	for _, want := range []string{
		"svc,le=0.1 latency_seconds_bucket=7i 1000",
		"svc,le=+Inf latency_seconds_bucket=10i 1000",
		"svc latency_seconds_sum=1.5 1000",
		"svc latency_seconds_count=10i 1000",
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing line %q in:\n%s", want, got)
		}
	}
}

func TestToLineProtocolSummary(t *testing.T) {
	families := []*dto.MetricFamily{
		mf("latency_us", dto.MetricType_SUMMARY, &dto.Metric{
			Summary: &dto.Summary{
				SampleCount: proto.Uint64(3),
				SampleSum:   proto.Float64(300),
				Quantile: []*dto.Quantile{
					{Quantile: proto.Float64(0.99), Value: proto.Float64(150)},
					{Quantile: proto.Float64(0.5), Value: proto.Float64(math.NaN())}, // NaN 分位数应跳过
				},
			},
		}),
	}
	got := toLineProtocol("svc", nil, families, 1000)
	for _, want := range []string{
		"svc,quantile=0.99 latency_us=150 1000",
		"svc latency_us_sum=300 1000",
		"svc latency_us_count=3i 1000",
	} {
		if !strings.Contains(got, want+"\n") {
			t.Errorf("missing line %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "NaN") || strings.Contains(got, "quantile=0.5") {
		t.Errorf("NaN quantile should be skipped:\n%s", got)
	}
}

func TestToLineProtocolEscaping(t *testing.T) {
	families := []*dto.MetricFamily{
		mf("m", dto.MetricType_GAUGE, &dto.Metric{
			Label: []*dto.LabelPair{label("path", "/api/user, list")},
			Gauge: &dto.Gauge{Value: proto.Float64(1)},
		}),
	}
	got := toLineProtocol("my svc", nil, families, 1000)
	want := `my\ svc,path=/api/user\,\ list m=1 1000` + "\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestToLineProtocolEmptyTagValueSkipped(t *testing.T) {
	// micro_* 指标带空值 label（micro_id=""、micro_version=""），行协议不允许空 tag value，
	// 必须跳过而不是写出 `micro_id=`，否则 DataWay 拒收整批数据
	families := []*dto.MetricFamily{
		mf("micro_request_total", dto.MetricType_COUNTER, &dto.Metric{
			Label: []*dto.LabelPair{
				label("micro_endpoint", "Svc.Pay"),
				label("micro_id", ""),
				label("micro_version", ""),
			},
			Counter: &dto.Counter{Value: proto.Float64(1)},
		}),
	}
	got := toLineProtocol("svc", map[string]string{"service": "svc"}, families, 1000)
	want := "svc,micro_endpoint=Svc.Pay,service=svc micro_request_total=1 1000\n"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestToLineProtocolNaNInfValue(t *testing.T) {
	families := []*dto.MetricFamily{
		mf("bad", dto.MetricType_GAUGE, &dto.Metric{Gauge: &dto.Gauge{Value: proto.Float64(math.NaN())}}),
	}
	got := toLineProtocol("svc", nil, families, 1000)
	if strings.Contains(got, "NaN") {
		t.Errorf("NaN should be replaced: %s", got)
	}
}
