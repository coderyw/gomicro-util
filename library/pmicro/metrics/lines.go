// Package metrics
// @Author: yinwei
// @File: lines
// @Version: 1.0.0
// @Date: 2026/8/12 10:00

package metrics

import (
	"math"
	"sort"
	"strconv"
	"strings"

	dto "github.com/prometheus/client_model/go"
)

// toLineProtocol 把一批 prometheus MetricFamily 快照翻译成 DataWay 行协议文本，
// 一个 metric 的一个 field 一行：`measurement,tag1=v1,tag2=v2 field=value timestamp`。
// 覆盖 COUNTER/GAUGE/UNTYPED/HISTOGRAM/SUMMARY 全部类型（现有 grpc wrapper 用到了 SUMMARY）。
func toLineProtocol(measurement string, commonTags map[string]string, families []*dto.MetricFamily, timestampNS int64) string {
	var b strings.Builder
	for _, mf := range families {
		name := mf.GetName()
		for _, m := range mf.GetMetric() {
			tags := mergeTags(commonTags, labelTags(m))
			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				writeLine(&b, measurement, tags, name, formatFloat(m.GetCounter().GetValue()), timestampNS)
			case dto.MetricType_GAUGE:
				writeLine(&b, measurement, tags, name, formatFloat(m.GetGauge().GetValue()), timestampNS)
			case dto.MetricType_UNTYPED:
				writeLine(&b, measurement, tags, name, formatFloat(m.GetUntyped().GetValue()), timestampNS)
			case dto.MetricType_HISTOGRAM:
				writeHistogram(&b, measurement, name, tags, m.GetHistogram(), timestampNS)
			case dto.MetricType_SUMMARY:
				writeSummary(&b, measurement, name, tags, m.GetSummary(), timestampNS)
			}
		}
	}
	return b.String()
}

func writeHistogram(b *strings.Builder, measurement, name string, tags map[string]string, h *dto.Histogram, timestampNS int64) {
	for _, bucket := range h.GetBucket() {
		le := strconv.FormatFloat(bucket.GetUpperBound(), 'f', -1, 64)
		if math.IsInf(bucket.GetUpperBound(), 1) {
			le = "+Inf"
		}
		bucketTags := mergeTags(tags, map[string]string{"le": le})
		writeLine(b, measurement, bucketTags, name+"_bucket", formatUint(bucket.GetCumulativeCount()), timestampNS)
	}
	writeLine(b, measurement, tags, name+"_sum", formatFloat(h.GetSampleSum()), timestampNS)
	writeLine(b, measurement, tags, name+"_count", formatUint(h.GetSampleCount()), timestampNS)
}

func writeSummary(b *strings.Builder, measurement, name string, tags map[string]string, s *dto.Summary, timestampNS int64) {
	for _, q := range s.GetQuantile() {
		v := q.GetValue()
		// summary 无样本时分位数为 NaN，行协议不接受 NaN，跳过
		if math.IsNaN(v) {
			continue
		}
		quantileTags := mergeTags(tags, map[string]string{
			"quantile": strconv.FormatFloat(q.GetQuantile(), 'f', -1, 64),
		})
		writeLine(b, measurement, quantileTags, name, formatFloat(v), timestampNS)
	}
	writeLine(b, measurement, tags, name+"_sum", formatFloat(s.GetSampleSum()), timestampNS)
	writeLine(b, measurement, tags, name+"_count", formatUint(s.GetSampleCount()), timestampNS)
}

func writeLine(b *strings.Builder, measurement string, tags map[string]string, field, value string, timestampNS int64) {
	b.WriteString(escapeMeasurement(measurement))
	for _, k := range sortedKeys(tags) {
		// 行协议不允许空的 tag key/value（如 micro_id=""），写出去会导致 DataWay
		// 拒收整个批次；prometheus 语义里空 label 值等价于无此维度，直接跳过
		if k == "" || tags[k] == "" {
			continue
		}
		b.WriteByte(',')
		b.WriteString(escapeTag(k))
		b.WriteByte('=')
		b.WriteString(escapeTag(tags[k]))
	}
	b.WriteByte(' ')
	b.WriteString(escapeTag(field))
	b.WriteByte('=')
	b.WriteString(value)
	b.WriteByte(' ')
	b.WriteString(strconv.FormatInt(timestampNS, 10))
	b.WriteByte('\n')
}

func labelTags(m *dto.Metric) map[string]string {
	if len(m.GetLabel()) == 0 {
		return nil
	}
	tags := make(map[string]string, len(m.GetLabel()))
	for _, lp := range m.GetLabel() {
		tags[lp.GetName()] = lp.GetValue()
	}
	return tags
}

func mergeTags(base, extra map[string]string) map[string]string {
	if len(extra) == 0 {
		return base
	}
	merged := make(map[string]string, len(base)+len(extra))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range extra {
		merged[k] = v
	}
	return merged
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// escapeTag 转义行协议 tag key/value 和 field key 中的特殊字符（逗号、等号、空格）
func escapeTag(s string) string {
	if !strings.ContainsAny(s, ", =") {
		return s
	}
	r := strings.NewReplacer(",", `\,`, "=", `\=`, " ", `\ `)
	return r.Replace(s)
}

// escapeMeasurement 转义 measurement 中的特殊字符（逗号、空格；measurement 不转义等号）
func escapeMeasurement(s string) string {
	if !strings.ContainsAny(s, ", ") {
		return s
	}
	r := strings.NewReplacer(",", `\,`, " ", `\ `)
	return r.Replace(s)
}

func formatFloat(v float64) string {
	// NaN/Inf 行协议不接受，统一写 0，避免整批数据被 DataWay 拒收
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func formatUint(v uint64) string {
	return strconv.FormatUint(v, 10) + "i"
}
