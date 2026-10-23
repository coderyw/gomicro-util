/**
 * @Author: yinwei
 * @Description: redis配置相关
 * @File: redis
 * @Version: 1.0.0
 * @Date: 2023/10/24 16:49
 */
package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

type RedisConfig struct {
	// 是否开启全部，默认 false
	Enable bool `json:"enable" yaml:"enable"`
	// 默认链接
	Default string `json:"default" yaml:"default"`
	// 连接池统计（打印/上报）间隔，单位 s。默认 15s 一次，-1 表示不开启
	PoolStatsInterval int `json:"pool_stats_interval" yaml:"pool_stats_interval"`
	// 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报，3 日志和上报都开启。默认 1
	PoolStatsMode int `json:"pool_stats_mode" yaml:"pool_stats_mode"`
	// 其他redis
	Instance map[string]RedisInstanceConfig `json:"instance" yaml:"instance"`
}

func (s RedisConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("enable", s.Enable)
	enc.AddString("default", s.Default)
	enc.AddInt("pool_stats_interval", s.PoolStatsInterval)
	enc.AddInt("pool_stats_mode", s.PoolStatsMode)
	if s.Instance != nil {
		enc.AddObject("instance", zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
			for k, v := range s.Instance {
				v := v
				encoder.AddObject(k, v)
			}
			return nil
		}))
	}
	return nil
}

type RedisInstanceConfig struct {
	// 是否开启，默认 false
	Enable   bool     `json:"enable" yaml:"enable"`
	Host     []string `json:"host"`
	Password string   `json:"password"`
	Db       int      `json:"db"`
	// 是否是集群模式
	IsCluster bool `json:"is_cluster" yaml:"is_cluster"`
	// 连接池数量大小 -1 不限制，0默认
	PoolSize int `json:"pool_size" yaml:"pool_size"`
	// 最大空闲数量
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`
	//是否开启debug模式
	Debug bool `json:"debug" yaml:"debug"`
}

func (s RedisInstanceConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("enable", s.Enable)
	enc.AddArray("host", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Host {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("password", util.Mask(s.Password, 2, 3))
	enc.AddInt("db", s.Db)
	enc.AddBool("is_cluster", s.IsCluster)
	enc.AddInt("pool_size", s.PoolSize)
	enc.AddInt("max_idle_conns", s.MaxIdleConns)
	enc.AddBool("debug", s.Debug)
	return nil
}
