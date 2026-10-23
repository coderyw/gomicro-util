// Package rabbitmq 基于 amqp091-go 的 RabbitMQ 包装，支持自动重连、Channel 池、Confirm 监听、消费者自动恢复。
package rabbitmq

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/coderyw/gomicro-util/model/config"
)

// Config 包装层配置，在 RabbitMqCfg 之上增加池、重连、优雅关闭等参数。
type Config struct {
	// 连接配置（来自 RabbitMqCfg 或等价）
	Host        string
	Username    string
	Password    string
	VirtualHost string

	// 自动重连
	ReconnectInterval time.Duration // 重连间隔，默认 3s
	MaxReconnectWait  time.Duration // 最大重连等待，默认 30s

	// Channel 池 / Channel 管理
	ChannelPoolSize int           // 池大小，<=0 表示默认 10
	ChannelIdleTTL  time.Duration // 空闲 channel 回收时间，0 表示不回收

	// 优雅关闭：Shutdown 时等待正在进行的操作的最长时间
	// 默认 30s，0 表示不等待仅标记关闭
	ShutdownTimeout time.Duration

	// 心跳（传给 amqp）
	Heartbeat time.Duration // 0 使用库默认 10s
}

// DefaultConfig 返回默认配置。
func DefaultConfig() Config {
	return Config{
		ReconnectInterval: 3 * time.Second,
		MaxReconnectWait:  30 * time.Second,
		ChannelPoolSize:   10,
		ChannelIdleTTL:    5 * time.Minute,
		ShutdownTimeout:   30 * time.Second,
		Heartbeat:         10 * time.Second,
	}
}

// FromRabbitMqCfg 从 model/config.RabbitMqCfg 生成 Config。
func FromRabbitMqCfg(c config.RabbitMqCfg) Config {
	cfg := DefaultConfig()
	cfg.Host = c.Host
	cfg.Username = c.Username
	cfg.Password = c.Pwd
	if c.VirtualHost != "" {
		cfg.VirtualHost = c.VirtualHost
	} else {
		cfg.VirtualHost = "/"
	}
	if c.ReconnectInterval != 0 {
		cfg.ReconnectInterval = time.Duration(c.ReconnectInterval) * time.Second
	}
	if c.MaxReconnectWait != 0 {
		cfg.MaxReconnectWait = time.Duration(c.MaxReconnectWait) * time.Second
	}
	if c.ChannelPoolSize != 0 {
		cfg.ChannelPoolSize = c.ChannelPoolSize
	}
	if c.ChannelIdleTTL != nil {
		cfg.ChannelIdleTTL = time.Duration(*c.ChannelIdleTTL) * time.Second
	}
	if c.ShutdownTimeout != nil {
		cfg.ShutdownTimeout = time.Duration(*c.ShutdownTimeout) * time.Second
	}
	if c.Heartbeat != 0 {
		cfg.Heartbeat = time.Duration(c.Heartbeat) * time.Second
	}
	return cfg
}

// AMQPURL 生成 amqp:// 连接 URL。若 Host 未包含端口则使用默认 5672。
func (c Config) AMQPURL() string {
	vhost := c.VirtualHost
	if vhost == "" {
		vhost = "/"
	}
	host := c.Host
	if host != "" && !strings.Contains(host, ":") {
		host = host + ":5672"
	}
	u := url.URL{
		Scheme: "amqp",
		User:   url.UserPassword(c.Username, c.Password),
		Host:   host,
		Path:   vhost,
	}
	return u.String()
}

// DialConfig 用于 amqp091 的拨号配置（仅暴露必要字段）。
func (c Config) DialConfig() (url string, heartbeat time.Duration) {
	return c.AMQPURL(), c.Heartbeat
}

// Validate 校验配置。
func (c Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("rabbitmq: host is required")
	}
	if c.Username == "" {
		return fmt.Errorf("rabbitmq: username is required")
	}
	return nil
}
