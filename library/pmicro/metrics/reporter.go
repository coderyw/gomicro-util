// Package metrics 监控指标采集与上报：
//   - 框架指标：go 运行时、grpc 服务端/客户端、http 请求，框架自动采集；
//   - 业务指标：业务代码通过 NewClient 的 Inc/Latency 自定义打点（见 client.go），
//     可用配置 metrics.disable_business 一键关闭。
//
// 两类指标统一注册在 prometheus 全局 registry，由本包定时推送到 Truewatch DataWay，
// /metrics 拉取端点同样可见。
//
// 指标是旁路能力，全链路弱依赖：配置错误降级为不上报，不影响服务启动；
// 上报在独立后台协程执行，失败/panic 只记日志；/metrics 拉取端点不受影响。
//
// @Author: yinwei
// @File: reporter
// @Version: 1.0.0
// @Date: 2026/8/12 10:00
package metrics

import (
	"context"
	"os"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	micro "github.com/coderyw/gomicro-util/model/micro"
	"go.uber.org/zap"
)

const (
	defaultInterval = 15 * time.Second
	// reportTimeout 单次上报的超时预算，与上报间隔独立：间隔配得再短，
	// 一次真实的 HTTP 往返仍需要正常的时间预算
	reportTimeout = 10 * time.Second
)

// reportingEnabled 标记上报循环是否已成功启动。
// 框架内部组件（mysql/redis 连接池统计等）通过 Enabled/SetGauge 感知：未开启时数据直接丢弃。
var reportingEnabled atomic.Bool

// Enabled 返回 metrics 上报是否已开启（Init 成功启动了后台上报循环）
func Enabled() bool {
	return reportingEnabled.Load()
}

// Init 按配置启动后台上报循环，返回挂到 Program 优雅停机的关闭结构。
// 弱依赖：未开启或配置不全时只记日志并返回空 CloseS，绝不让服务启动失败。
// 由 program.go 在配置加载完成后调用一次，业务服务零代码接入。
func Init(cfg srv_config.MetricsCfg, serviceName string) micro.CloseS {
	var c micro.CloseS
	c.Weight = 1 // 比 redis(0) 先关，业务组件之后关
	// 业务打点开关独立于上报开关：即使不推 Truewatch，/metrics 拉取端点仍可消费业务指标
	businessDisabled.Store(cfg.DisableBusiness)
	if cfg.DisableBusiness {
		log.Info("metrics: 业务自定义打点已按配置关闭")
	}
	if !cfg.Enable {
		return c
	}
	reporter, err := newTruewatchReporter(cfg.Truewatch, serviceName, commonTags(serviceName))
	if err != nil {
		log.Warn("metrics: truewatch 配置不全，降级为不上报", zap.Error(err))
		return c
	}
	interval := time.Duration(cfg.IntervalS) * time.Second
	if interval <= 0 {
		interval = defaultInterval
	}
	m := startManager(reporter, interval)
	reportingEnabled.Store(true)
	c.CloseFunc = func() {
		reportingEnabled.Store(false)
		m.stop()
	}
	log.Info("metrics: 指标上报已启动",
		zap.String("dataway", cfg.Truewatch.DatawayUrl),
		zap.Duration("interval", interval))
	return c
}

// commonTags 每条指标附带的公共 tag：服务名、运行环境、主机名（k8s 下即 pod 名）
func commonTags(serviceName string) map[string]string {
	tags := map[string]string{
		"service": serviceName,
		"env":     env.GetEnv(),
	}
	if host, err := os.Hostname(); err == nil && host != "" {
		tags["host"] = host
	}
	return tags
}

// manager 在后台按固定间隔把全局 registry 快照喂给上报器
type manager struct {
	reporter *truewatchReporter
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
}

func startManager(reporter *truewatchReporter, interval time.Duration) *manager {
	m := &manager{
		reporter: reporter,
		interval: interval,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	go m.run()
	return m
}

func (m *manager) run() {
	defer close(m.doneCh)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.reportOnce()
		case <-m.stopCh:
			log.Info("metrics: 停止上报. 准备开始最后一轮上报")
			// 停机前最后上报一轮，尽量不丢最后一段窗口的数据
			m.reportOnce()
			return
		}
	}
}

// reportOnce 弱依赖原则的落地点：任何 panic 或 error 都只记日志，绝不向上传播。
// Counter/Histogram 是累计值，丢一轮快照不丢数据，失败只做一次轻量重试。
func (m *manager) reportOnce() {
	defer func() {
		if r := recover(); r != nil {
			log.Error("metrics: 上报 panic 已恢复", zap.Any("panic", r))
		}
	}()

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		log.Warn("metrics: gather 失败", zap.Error(err))
		if len(families) == 0 {
			return
		}
		// Gather 部分失败仍会返回成功部分，继续上报
	}
	//log.Info("metrics: 上报开始", zap.Int("metrics_count", len(families)), zap.Any("metrics_families", families))
	if len(families) == 0 {
		return
	}

	if err := m.reportWithTimeout(families); err != nil {
		// 重试用全新的超时预算，不复用第一次已耗尽的 ctx
		if err = m.reportWithTimeout(families); err != nil {
			log.Warn("metrics: 上报失败", zap.Error(err))
		}
	}
}

func (m *manager) reportWithTimeout(families []*dto.MetricFamily) error {
	ctx, cancel := context.WithTimeout(context.Background(), reportTimeout)
	defer cancel()
	return m.reporter.report(ctx, families)
}

// stop 停止后台上报循环，等待最后一轮上报结束
func (m *manager) stop() {
	close(m.stopCh)
	<-m.doneCh
}
