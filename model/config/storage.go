package config

import "go.uber.org/zap/zapcore"

// 存储相关配置
type StorageCfg struct {
	Mysql         DbConfig       `yaml:"mysql" json:"mysql"`
	Redis         RedisConfig    `yaml:"redis" json:"redis"`
	ObjectStorage *ObjectStorage `yaml:"object_storage" json:"object_storage"`
}

func (s StorageCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if err := enc.AddObject("mysql", s.Mysql); err != nil {
		return err
	}
	if err := enc.AddObject("redis", s.Redis); err != nil {
		return err
	}
	// ObjectStorage 是可选配置，未配时为 nil，跳过即可（AddObject 传 nil 会 panic）
	if s.ObjectStorage != nil {
		if err := enc.AddObject("object_storage", s.ObjectStorage); err != nil {
			return err
		}
	}
	return nil
}

// ObjectStorage 对象存储配置
//
// 每个 provider 一张 map，key 为实例名。实例名是业务侧使用的逻辑名称，
// 它挂在哪张 map 下就决定了该实例走哪个云；同一实例名不允许在多张 map 中重复出现。
type ObjectStorage struct {
	// DefaultInstance 默认实例名，业务侧 UploadRequest.Instance 留空时使用。
	//
	// 可选。配了就必须能在下面两张 map 里找到，否则启动直接失败——
	// 一个指向不存在实例的默认值，比没有默认值更危险。
	//
	// 用途：绝大多数服务只用一个实例，配了它之后业务侧就不必再有自己的实例名配置项，
	// 少维护一处「跨配置段的字符串引用」。真要在一个服务里用多个实例，
	// 在调用处显式传 UploadRequest.Instance，不要退回去加一个只能指一个实例的通用配置字段。
	//
	// ⚠️ 只支持显式指定，不做「只有一个实例时自动成为默认」的隐式推导：
	// 那会让「部署方新增第二个实例」这个纯配置动作静默改变既有业务的落桶行为。
	DefaultInstance string `json:"default_instance" yaml:"default_instance"`

	// AwsS3Config AWS S3 实例
	AwsS3Config map[string]*AwsS3Config `json:"aws_s3_config" yaml:"aws_s3_config"`
	// HuaweiObsConfig 华为云 OBS 实例
	HuaweiObsConfig map[string]*HuaweiObsConfig `json:"huawei_obs_config" yaml:"huawei_obs_config"`
}

func (s ObjectStorage) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("default_instance", s.DefaultInstance)

	for name, cfg := range s.AwsS3Config {
		if cfg == nil {
			continue
		}
		if err := enc.AddObject(name, cfg); err != nil {
			return err
		}
	}
	for name, cfg := range s.HuaweiObsConfig {
		if cfg == nil {
			continue
		}
		if err := enc.AddObject(name, cfg); err != nil {
			return err
		}
	}
	return nil
}
