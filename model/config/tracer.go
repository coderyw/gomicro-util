// Package config
// @Author: yinwei
// @File: tracer
// @Version: 1.0.0
// @Date: 2024/7/1 13:55

package config

// 链路追踪配置
type TracerCfg struct {
	Host string `yaml:"host", json:"host"`
	Port string `yaml:"port", json:"port"`
}
