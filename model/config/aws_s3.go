package config

import (
	"time"

	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

// AwsS3BucketConfig 单个桶配置
type AwsS3BucketConfig struct {
	// 是否是默认桶，默认桶只能有一个
	IsDefault bool `json:"is_default" yaml:"is_default"`
	// Alias 桶别名（业务使用）
	// 必填
	// 用于业务侧调用 Upload 时指定 Bucket
	// 示例： "public-images" / "private-assets"
	Alias string `json:"alias" yaml:"alias"`

	// Name 实际 S3 Bucket 名称
	// 必填
	// 示例： "my-prod-public-images"
	Name string `json:"name" yaml:"name"`

	// PublicBaseURL 公共访问域名（可选）
	// 默认值：""（空，表示不生成 ObjectURL）
	// 用于拼接返回的访问 URL（通常是 CDN 域名）
	// 示例： "https://cdn.example.com"
	//
	// S3 侧没有对象 ACL 配置，可见性完全由桶策略决定，公共访问地址就靠这一项拼出，
	// 需要业务拿到可访问 URL 时必须配置。原因见 internal/storage/s3 的包注释。
	PublicBaseURL string `json:"public_base_url" yaml:"public_base_url"`
}

func (s AwsS3BucketConfig) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("is_default", s.IsDefault)
	enc.AddString("alias", s.Alias)
	enc.AddString("name", s.Name)
	enc.AddString("public_base_url", s.PublicBaseURL)
	return nil
}

// AwsS3Config S3 实例配置（支持多实例）
type AwsS3Config struct {
	// Region AWS 区域
	// 必填
	// 示例： "ap-southeast-1"
	Region string `json:"region" yaml:"region"`

	// ---------------- 凭证配置 ----------------

	// UseDefaultCredentialChain 是否使用默认凭证链
	// 默认值：false（代码中不会自动设 true，需要你显式指定）
	// 推荐：生产环境必须为 true（IRSA / ECS Task Role / EC2 IAM Role）
	UseDefaultCredentialChain bool `json:"use_default_credential_chain" yaml:"use_default_credential_chain"`

	// AccessKeyID AWS Access Key（静态凭证）
	// 默认值：""
	// 当 UseDefaultCredentialChain=false 时必须填写
	AccessKeyID string `json:"access_key_id" yaml:"access_key_id"`

	// SecretAccessKey AWS Secret Key
	// 默认值：""
	SecretAccessKey string `json:"secret_access_key" yaml:"secret_access_key"`

	// SessionToken 临时凭证 Token（STS）
	// 默认值：""
	SessionToken string `json:"session_token" yaml:"session_token"`

	// ---------------- Endpoint 配置 ----------------

	// Endpoint 自定义 S3 Endpoint
	// 默认值：""（使用 AWS 官方 endpoint）
	// 用于 MinIO / Ceph / 私有 S3 网关
	// 示例： "http://127.0.0.1:9000"
	Endpoint string `json:"endpoint" yaml:"endpoint"`

	// UsePathStyle 是否使用 path-style 访问
	// 默认值：false
	// true:  http://endpoint/bucket/key
	// false: http://bucket.endpoint/key（AWS 推荐）
	// ⚠️ MinIO 必须为 true
	UsePathStyle bool `json:"use_path_style" yaml:"use_path_style"`

	// ---------------- 重试策略 ----------------

	// MaxRetries 最大重试次数
	// 默认值：3（代码中 fallback）
	// 建议：
	//   普通业务：3
	//   弱网络 / 跨区域：5
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// ---------------- HTTP 连接池配置 ----------------

	// DialTimeout TCP 建连超时
	// 默认值：5s
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout"`

	// TLSHandshakeTimeout TLS 握手超时
	// 默认值：5s
	TLSHandshakeTimeout time.Duration `json:"tls_handshake_timeout" yaml:"tls_handshake_timeout"`

	// ResponseHeaderTimeout 等待响应头超时
	// 默认值：30s
	// ⚠️ 上传大文件时建议调大
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout" yaml:"response_header_timeout"`

	// ExpectContinueTimeout 100-continue 超时
	// 默认值：1s
	ExpectContinueTimeout time.Duration `json:"expect_continue_timeout" yaml:"expect_continue_timeout"`

	// IdleConnTimeout 空闲连接保持时间
	// 默认值：90s
	IdleConnTimeout time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`

	// MaxIdleConns 最大空闲连接数（全局）
	// 默认值：200
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// MaxIdleConnsPerHost 单 host 最大空闲连接
	// 默认值：100
	MaxIdleConnsPerHost int `json:"max_idle_conns_per_host" yaml:"max_idle_conns_per_host"`

	// ---------------- Bucket 配置 ----------------

	// Buckets 桶配置列表
	// 必填（至少一个）
	Buckets []AwsS3BucketConfig `json:"buckets" yaml:"buckets"`
}

func (s AwsS3Config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("region", s.Region)
	enc.AddBool("use_default_credential_chain", s.UseDefaultCredentialChain)
	enc.AddBool("use_path_style", s.UsePathStyle)
	enc.AddString("access_key_id", util.Mask(s.AccessKeyID, 2, 3))
	enc.AddString("secret_access_key", util.Mask(s.SecretAccessKey, 2, 3))
	enc.AddString("session_token", util.Mask(s.SessionToken, 2, 3))
	enc.AddString("endpoint", s.Endpoint)
	enc.AddString("dial_timeout", s.DialTimeout.String())
	enc.AddString("tls_handshake_timeout", s.TLSHandshakeTimeout.String())
	enc.AddString("response_header_timeout", s.ResponseHeaderTimeout.String())
	enc.AddString("expect_continue_timeout", s.ExpectContinueTimeout.String())
	enc.AddString("idle_conn_timeout", s.IdleConnTimeout.String())
	enc.AddInt("max_idle_conns", s.MaxIdleConns)
	enc.AddInt("max_idle_conns_per_host", s.MaxIdleConnsPerHost)
	enc.AddInt("max_retries", s.MaxRetries)
	return enc.AddArray("buckets", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Buckets {
			if err := encoder.AppendObject(v); err != nil {
				return err
			}
		}
		return nil
	}))
}
