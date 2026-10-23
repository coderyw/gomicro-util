// Package config
// @Author: yinwei
// @File: metrics
// @Version: 1.0.0
// @Date: 2026/8/12 10:00

package config

import "go.uber.org/zap/zapcore"

// MetricsCfg 监控指标上报配置。
// 采集侧复用 prometheus 全局 registry（go 运行时、grpc、http 指标都在其中），
// 开启后由框架定时把指标快照推送到 Truewatch DataWay；/metrics 拉取端点不受影响。
type MetricsCfg struct {
	// 总开关，默认 false 不上报
	Enable bool `json:"enable" yaml:"enable"`
	// 关闭业务自定义打点（metrics.Client 的 Inc/Latency）。默认 false 即开启采集；
	// 作为业务侧高基数误用等场景的运维紧急开关，关闭后打点直接空操作
	DisableBusiness bool `json:"disable_business" yaml:"disable_business"`
	// 上报间隔，单位 s，默认 15
	IntervalS int `json:"interval_s" yaml:"interval_s"`
	// Truewatch 上报配置
	Truewatch TruewatchCfg `json:"truewatch" yaml:"truewatch"`
}

// TruewatchCfg Truewatch DataWay HTTP 上报所需的连接信息
type TruewatchCfg struct {
	// DataWay 地址，如 https://xxx.dataway.truewatch.com
	DatawayUrl string `json:"dataway_url" yaml:"dataway_url"`
	// 工作空间 token
	WorkspaceToken string `json:"workspace_token" yaml:"workspace_token"`
}

func (s MetricsCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("enable", s.Enable)
	enc.AddBool("disable_business", s.DisableBusiness)
	enc.AddInt("interval_s", s.IntervalS)
	// token 不打日志
	enc.AddString("dataway_url", s.Truewatch.DatawayUrl)
	return nil
}
