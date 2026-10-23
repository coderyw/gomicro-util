package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

// HuaweiObsBucketConfig 单个桶配置
type HuaweiObsBucketConfig struct {
	// 是否是默认桶，默认桶只能有一个
	IsDefault bool `json:"is_default" yaml:"is_default"`
	// Alias 桶别名（业务使用）
	// 必填
	// 用于业务侧调用 Upload 时指定 Bucket
	// 示例： "public-images" / "private-assets"
	Alias string `json:"alias" yaml:"alias"`

	// Name 实际 OBS Bucket 名称
	// 必填
	// 示例： "my-prod-public-images"
	Name string `json:"name" yaml:"name"`

	// PublicBaseURL 公共访问域名（可选）
	// 默认值：""（空，表示使用 SDK 返回的 ObjectUrl）
	// 用于拼接返回的访问 URL（通常是 CDN 域名）
	// 示例： "https://cdn.example.com"
	PublicBaseURL string `json:"public_base_url" yaml:"public_base_url"`

	// DefaultACL 上传时的默认对象 ACL（可选）
	// 默认值：""（不设置对象 ACL，可见性完全由桶策略决定）
	// 仅在调用方未通过 UploadRequest.ACL 显式指定时生效
	// 取值为 canned ACL，与 S3 同名，如 "public-read" / "private"
	//
	// 何时需要填：桶策略不开放匿名读、而业务依赖对象级公共读时。
	// 从「业务侧自行调用 SDK 并写死 AclPublicRead」迁移过来的场景，
	// 填 "public-read" 可保持行为完全不变。
	DefaultACL string `json:"default_acl" yaml:"default_acl"`
}

func (s HuaweiObsBucketConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("is_default", s.IsDefault)
	enc.AddString("alias", s.Alias)
	enc.AddString("name", s.Name)
	enc.AddString("public_base_url", s.PublicBaseURL)
	enc.AddString("default_acl", s.DefaultACL)
	return nil
}

// HuaweiObsConfig 华为云 OBS 实例配置（支持多实例）
type HuaweiObsConfig struct {
	// Ak 华为云 Access Key
	// 必填
	Ak string `json:"ak" yaml:"ak"`

	// Sk 华为云 Secret Key
	// 必填
	Sk string `json:"sk" yaml:"sk"`

	// Endpoint OBS 服务地址
	// 必填
	// 示例： "obs.ap-southeast-3.myhuaweicloud.com"
	// ⚠️ SDK 不会补全 scheme，返回的 ObjectUrl 直接由 endpoint 拼接而来，
	// 迁移既有配置时请逐字照抄，避免新旧对象 URL 形态不一致
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// ---------------- 客户端参数 ----------------
	//
	// ⚠️ 下面三项均为“秒”，且 0 表示不向 SDK 传该选项、沿用 SDK 自身默认值。
	// 不要改成 time.Duration：SDK 的 obs.WithConnectTimeout 等选项收的就是 int 秒；
	// 且 yaml 里裸写整数 5 会被解析成 5ns，是个隐蔽的坑。

	// MaxRetries 最大重试次数
	// 默认值：0（不传，SDK 默认 3）
	// ⚠️ 不要“为了显式”而填 0 以外的兜底值：SDK 只在 maxRetryCount < 0 时才回填默认，
	// 传 0 会变成真的零重试
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// ConnectTimeout TCP 建连超时（秒）
	// 默认值：0（不传，SDK 默认 60）
	ConnectTimeout int `json:"connect_timeout" yaml:"connect_timeout"`

	// SocketTimeout 读写超时（秒）
	// 默认值：0（不传，SDK 默认 60）
	SocketTimeout int `json:"socket_timeout" yaml:"socket_timeout"`

	// ---------------- Bucket 配置 ----------------

	// Buckets 桶配置列表
	// 必填（至少一个）
	Buckets []HuaweiObsBucketConfig `json:"buckets" yaml:"buckets"`
}

func (s HuaweiObsConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	// ak 保留少量字符便于比对配置，sk 全量脱敏
	enc.AddString("ak", util.Mask(s.Ak, 2, 3))
	enc.AddString("sk", util.Mask(s.Sk, 0, 0))
	enc.AddString("endpoint", s.Endpoint)
	enc.AddInt("max_retries", s.MaxRetries)
	enc.AddInt("connect_timeout", s.ConnectTimeout)
	enc.AddInt("socket_timeout", s.SocketTimeout)
	return enc.AddArray("buckets", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Buckets {
			if err := encoder.AppendObject(v); err != nil {
				return err
			}
		}
		return nil
	}))
}
