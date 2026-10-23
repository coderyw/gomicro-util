package storage

import (
	"io"
	"time"
)

type UploadRequest struct {
	// 目标实例，例如 "aws-apse1-prod"
	Instance string

	// 目标桶别名，例如 "private-assets"、"public-images"
	//
	// 对应实例配置里某个 bucket 的 alias。留空则使用该实例的默认桶：
	// 标了 is_default 的那个，都没标则取 buckets 列表的第一个。
	// 与 tenant / 上下文无关——租户隔离体现在 Key 前缀上，不体现在选桶上。
	Bucket string

	// 对象 key，例如 "user/1001/avatar.png"
	Key string

	// 数据流
	Body io.Reader

	// 可选：已知长度时传入，便于 SDK 优化
	ContentLength int64

	// 可选：MIME
	ContentType string

	// 可选：文件名，下载时可见
	ContentDisposition string

	// 可选：对象元数据
	Metadata map[string]string

	// 可选：标签
	Tags map[string]string

	// 可选：缓存控制
	CacheControl string

	// 可选：对象 canned ACL，如 "public-read"。
	//
	// ⚠️ 只对华为云 OBS 实例生效，S3 provider 会忽略本字段——S3 桶普遍已禁用 ACL，
	// 可见性由桶策略决定。业务代码要保持云无关，正常不该设置本字段：
	// 华为侧的可见性用实例配置里桶的 default_acl 声明即可。
	ACL string

	// 可选：是否覆盖
	AllowOverwrite bool

	// 可选：请求级超时
	Timeout time.Duration
}

type UploadResult struct {
	Provider  string
	Instance  string
	Bucket    string
	Key       string
	ETag      string
	VersionID string
	Location  string
	RequestID string
	ObjectURL string
}
