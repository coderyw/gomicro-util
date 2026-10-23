// Package metrics
// @Author: yinwei
// @File: registry
// @Version: 1.0.0
// @Date: 2026/8/12 14:00

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// 业务指标惰性注册：指标按名字首次打点时才声明并注册到 prometheus 全局 registry，
// 同时记录当时的 label key 集合（已排序），用于校验后续同名打点的一致性。
// key 集合不一致视为业务打点用法错误：只记 warning 并丢弃这次打点，不 panic、不影响业务。

// defaultLatencyBuckets Client.Latency 惰性声明 Histogram 的默认桶，单位毫秒。
//
// 不能直接复用 prometheus.DefBuckets——那组默认桶（{.005,.01,...,10}）是按"值以秒为单位"
// 设计的，最大边界只有 10；Client.Latency 的调用约定是传毫秒（见调用方 bmp-game-vendor-api/
// internal/metrics/metrics.go 顶部注释），如果沿用 DefBuckets，任何超过 10（这里的 10 会被
// 当成 10 毫秒，而不是 10 秒）的观测值都会被无差别地扔进 +Inf 桶——15ms 和 5 分钟的请求在
// histogram 里长得一模一样，P90/P99 插值出来的结果被死死封顶在个位数毫秒，慢请求完全看不出来。
// 这里改成毫秒量级、覆盖到 5 分钟（300000ms）的桶：小粒度覆盖常见的毫秒级请求，指数级放大
// 覆盖到分钟级的慢请求/超时场景（如 EVO LiveCard 轮询封顶 1 分钟）。
var defaultLatencyBuckets = []float64{
	1, 2, 5, 10, 25, 50, 100, 250, 500, // 1ms ~ 500ms：常见请求耗时
	1000, 2500, 5000, 10000, // 1s ~ 10s：偏慢请求/下游超时边界
	30000, 60000, 120000, 300000, // 30s ~ 5min：轮询/重试/极端慢请求
}

type counterEntry struct {
	keys []string
	vec  *prometheus.CounterVec
}

type histogramEntry struct {
	keys []string
	vec  *prometheus.HistogramVec
}

type gaugeEntry struct {
	keys []string
	vec  *prometheus.GaugeVec
}

var (
	vecCacheMu     sync.Mutex
	counterCache   = map[string]*counterEntry{}
	histogramCache = map[string]*histogramEntry{}
	gaugeCache     = map[string]*gaugeEntry{}
)

func counterVecFor(name string, keys []string) *prometheus.CounterVec {
	vecCacheMu.Lock()
	defer vecCacheMu.Unlock()

	if entry, ok := counterCache[name]; ok {
		if !equalKeys(entry.keys, keys) {
			log.Warn("metrics: counter tag key 集合与首次打点不一致，丢弃本次打点",
				zap.String("name", name), zap.Strings("registered_keys", entry.keys), zap.Strings("got_keys", keys))
			return nil
		}
		return entry.vec
	}

	vec := prometheus.NewCounterVec(prometheus.CounterOpts{Name: name, Help: "auto-registered by metrics.Client"}, keys)
	if !registerCollector(name, vec) {
		return nil
	}
	counterCache[name] = &counterEntry{keys: keys, vec: vec}
	return vec
}

func histogramVecFor(name string, keys []string) *prometheus.HistogramVec {
	vecCacheMu.Lock()
	defer vecCacheMu.Unlock()

	if entry, ok := histogramCache[name]; ok {
		if !equalKeys(entry.keys, keys) {
			log.Warn("metrics: histogram tag key 集合与首次打点不一致，丢弃本次打点",
				zap.String("name", name), zap.Strings("registered_keys", entry.keys), zap.Strings("got_keys", keys))
			return nil
		}
		return entry.vec
	}

	vec := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name, Help: "auto-registered by metrics.Client", Buckets: defaultLatencyBuckets}, keys)
	if !registerCollector(name, vec) {
		return nil
	}
	histogramCache[name] = &histogramEntry{keys: keys, vec: vec}
	return vec
}

func gaugeVecFor(name string, keys []string) *prometheus.GaugeVec {
	vecCacheMu.Lock()
	defer vecCacheMu.Unlock()

	if entry, ok := gaugeCache[name]; ok {
		if !equalKeys(entry.keys, keys) {
			log.Warn("metrics: gauge tag key 集合与首次打点不一致，丢弃本次打点",
				zap.String("name", name), zap.Strings("registered_keys", entry.keys), zap.Strings("got_keys", keys))
			return nil
		}
		return entry.vec
	}

	vec := prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name, Help: "auto-registered by metrics.SetGauge"}, keys)
	if !registerCollector(name, vec) {
		return nil
	}
	gaugeCache[name] = &gaugeEntry{keys: keys, vec: vec}
	return vec
}

// registerCollector 注册到全局 registry；失败（与框架指标重名、label 名非法等）只记
// warning 并丢点，不让业务路径受影响。
func registerCollector(name string, c prometheus.Collector) bool {
	if err := prometheus.DefaultRegisterer.Register(c); err != nil {
		log.Warn("metrics: 业务指标注册失败，丢弃该指标",
			zap.String("name", name), zap.Error(err))
		return false
	}
	return true
}

func equalKeys(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
