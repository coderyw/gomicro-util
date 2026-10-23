// Package etcd_config
// @Author: yinwei
// @File: rabbitmq
// @Version: 1.0.0
// @Date: 2023/11/2 13:33

package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

// mq配置
type RabbitMqCfgs map[string]RabbitMqCfg

type RabbitMqCfg struct {
	// 是否开启
	Enable      bool   `json:"enable" yaml:"enable"`
	Host        string `json:"host" yaml:"host"`
	Username    string `json:"username" yaml:"username"`
	Pwd         string `json:"pwd" yaml:"pwd"`
	VirtualHost string `json:"virtual_host" yaml:"virtual_host"`
	// 自动重连间隔（单位秒），默认 3s。
	ReconnectInterval int `json:"reconnect_interval"`
	// 最大重连等待，默认 30s
	MaxReconnectWait int `json:"max_reconnect_wait"`
	// Channel 池 / Channel 管理
	// 池大小，<=0 表示默认 10
	ChannelPoolSize int `json:"channel_pool_size"`
	// 空闲 channel 回收时间，0 表示不回收 单位 s,不填写会使用默认值 5*60 s
	ChannelIdleTTL *int `json:"channel_idle_ttl"`

	// 优雅关闭：Shutdown 时等待正在进行的操作的最长时间
	// 默认 30s，单位 s， 0 表示不等待仅标记关闭
	ShutdownTimeout *int `json:"shutdown_timeout"`

	// 心跳（传给 amqp） 单位 s
	// 0 使用库默认 10s
	Heartbeat int `json:"heartbeat"`
}

func (s RabbitMqCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("enable", s.Enable)
	enc.AddString("host", s.Host)
	enc.AddString("username", s.Username)
	enc.AddString("pwd", util.Mask(s.Pwd, 2, 3))
	enc.AddString("virtual_host", s.VirtualHost)
	enc.AddInt("reconnect_interval", s.ReconnectInterval)
	enc.AddInt("max_reconnect_wait", s.MaxReconnectWait)
	enc.AddInt("channel_pool_size", s.ChannelPoolSize)
	if s.ChannelIdleTTL != nil {
		enc.AddInt("channel_idle_ttl", *s.ChannelIdleTTL)
	} else {
		enc.AddString("channel_idle_ttl", "nil")
	}
	if s.ShutdownTimeout != nil {
		enc.AddInt("shutdown_timeout", *s.ShutdownTimeout)
	} else {
		enc.AddString("shutdown_timeout", "nil")
	}
	enc.AddInt("heartbeat", s.Heartbeat)
	return nil
}
