// Package metrics
// @Author: yinwei
// @File: gauge
// @Version: 1.0.0
// @Date: 2026/8/20 10:00

package metrics

import "sort"

// SetGauge 框架内部组件（mysql/redis 连接池统计等）上报瞬时值（Gauge 语义，Set 覆盖）。
// 仅当 metrics 上报已开启（Init 成功启动上报循环）时才写入；未开启时直接丢弃，
// 不注册指标、不进 /metrics 端点，调用方无需自行判断开关。
// 与业务打点（Client.Inc/Latency）不同：不附加 service label——上报器推送时
// 已通过公共 tag 附加 service/env/host。
func SetGauge(name string, tags Tags, value float64) {
	defer recoverAndLog("SetGauge", name)
	if !Enabled() {
		return
	}
	keys, values := sortedKV(tags)
	vec := gaugeVecFor(sanitizeName(name), keys)
	if vec == nil {
		return
	}
	vec.WithLabelValues(values...).Set(value)
}

// sortedKV 把 tags 的 key 清洗后按字典序拆成两个平行数组，
// 供惰性注册（keys 决定 label 维度）和打点（values 按同一顺序传入）复用。
func sortedKV(tags Tags) (keys, values []string) {
	cleaned := make(Tags, len(tags))
	for k, v := range tags {
		cleaned[sanitizeName(k)] = v
	}
	keys = make([]string, 0, len(cleaned))
	for k := range cleaned {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	values = make([]string, len(keys))
	for i, k := range keys {
		values[i] = cleaned[k]
	}
	return keys, values
}
