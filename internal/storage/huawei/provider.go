// Package huawei 华为云 OBS 对象存储 provider。
//
// 与 internal/storage/s3 保持同构，两者共用 internal/storage 的 Uploader 契约与错误类型，
// 由 library/component/storage 的 registry 按实例名统一分派。
//
// 与 S3 provider 的两点能力差异（调用方需知晓）：
//  1. 不支持 UploadRequest.Timeout —— 华为云 SDK 的 ctx 是客户端级选项
//     （obs.WithRequestContext 传给 obs.New），PutObject 的 extensionOptions 只支持
//     自定义 header 与进度回调，无法做请求级超时/取消。请改用实例配置里的
//     connect_timeout / socket_timeout。
//  2. 桶未配置 public_base_url 时，返回 SDK 拼出的 ObjectUrl（S3 provider 在这种情况下返回空串）。
//  3. 不支持 UploadRequest.Tags —— OBS 的对象标签要走独立的 SetObjectTagging 接口，
//     PutObject 入参里没有对应字段。该字段会被忽略，需要标签请另行调用 SDK。
package huawei

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/coderyw/gomicro-util/internal/storage"
	"github.com/coderyw/gomicro-util/model/config"
	storage2 "github.com/coderyw/gomicro-util/model/storage"
)

type Provider struct {
	instance           string
	client             *obs.ObsClient
	defaultBucketAlias string
	buckets            map[string]config.HuaweiObsBucketConfig
}

func New(instance string, cfg config.HuaweiObsConfig) (*Provider, error) {
	if instance == "" {
		return nil, fmt.Errorf("instance is empty")
	}
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is empty")
	}
	// 与 s3 provider 的静态凭证校验对齐：ak/sk 为空时 obs.New 依然会成功返回，
	// 直到真正上传才报 403，不如启动就失败
	if cfg.Ak == "" || cfg.Sk == "" {
		return nil, fmt.Errorf("credentials are empty")
	}
	if len(cfg.Buckets) == 0 {
		return nil, fmt.Errorf("buckets is empty")
	}

	// 未配置的项回填 SDK 自身的默认值，保证与「不传任何选项」时行为一致。
	//
	// 这里直接引用 SDK 导出的常量而不是写死数字，避免 SDK 改默认值后本处悄悄漂移。
	// 特别注意 MaxRetries：SDK 只在 maxRetryCount < 0 时才回填默认值（conf.go 的
	// prepareConfig），若把配置里的 0 原样传下去会变成真的零重试。
	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = obs.DEFAULT_MAX_RETRY_COUNT
	}
	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = obs.DEFAULT_CONNECT_TIMEOUT
	}
	socketTimeout := cfg.SocketTimeout
	if socketTimeout <= 0 {
		socketTimeout = obs.DEFAULT_SOCKET_TIMEOUT
	}

	client, err := obs.New(cfg.Ak, cfg.Sk, cfg.Endpoint,
		obs.WithMaxRetryCount(maxRetries),
		obs.WithConnectTimeout(connectTimeout),
		obs.WithSocketTimeout(socketTimeout),
	)
	if err != nil {
		return nil, fmt.Errorf("create obs client failed: %w", err)
	}

	bucketMap := make(map[string]config.HuaweiObsBucketConfig, len(cfg.Buckets))
	defaultBucket := ""
	for _, b := range cfg.Buckets {
		if b.Name == "" {
			return nil, fmt.Errorf("bucket alias/name is empty")
		}
		if _, ok := bucketMap[b.Alias]; ok {
			return nil, fmt.Errorf("duplicate bucket alias: %s", b.Alias)
		}
		if b.IsDefault {
			defaultBucket = b.Alias
		}
		bucketMap[b.Alias] = b
	}
	// 如果没有配置，则使用第一个
	if defaultBucket == "" {
		defaultBucket = cfg.Buckets[0].Alias
	}

	return &Provider{
		instance:           instance,
		client:             client,
		buckets:            bucketMap,
		defaultBucketAlias: defaultBucket,
	}, nil
}

func (p *Provider) Provider() storage.Provider {
	return storage.ProviderHuaweiObs
}

// Close 有意为空。
//
// obs.ObsClient.Close() 会把 httpClient 与 conf 置为 nil 且没有任何同步，
// 而本 provider 的 client 是启动时创建、全进程共享的：关停阶段若仍有在途上传，
// Close 与 PutObject 会构成数据竞争，之后进来的请求还会因 conf == nil 直接 panic。
// 进程即将退出时释放空闲连接没有收益，故与 s3 provider 一样不做任何事。
func (p *Provider) Close() {
}

func (p *Provider) Upload(ctx context.Context, req *storage2.UploadRequest) (*storage2.UploadResult, error) {
	if req == nil || req.Body == nil {
		bucket := ""
		key := ""
		if req != nil {
			bucket = req.Bucket
			key = req.Key
		}
		return nil, &storage.OpError{
			Op:       "upload",
			Provider: p.Provider(),
			Instance: p.instance,
			Bucket:   bucket,
			Key:      key,
			Err:      storage.ErrInvalidArgument,
		}
	}
	if req.Key == "" {
		return nil, &storage.OpError{
			Op:       "upload",
			Provider: p.Provider(),
			Instance: p.instance,
			Bucket:   req.Bucket,
			Key:      req.Key,
			Err:      storage.ErrInvalidArgument,
		}
	}
	// 用局部变量而不是就地改写 req.Bucket：入参属于调用方，复用同一个 req 时
	// 不该被我们悄悄填上默认桶
	bucketAlias := req.Bucket
	if bucketAlias == "" {
		bucketAlias = p.defaultBucketAlias
	}

	bucketCfg, ok := p.buckets[bucketAlias]
	if !ok {
		return nil, &storage.OpError{
			Op:       "upload",
			Provider: p.Provider(),
			Instance: p.instance,
			Bucket:   bucketAlias,
			Key:      req.Key,
			Err:      storage.ErrBucketNotFound,
		}
	}

	// 不允许覆盖时，先探测
	if !req.AllowOverwrite {
		_, err := p.client.HeadObject(&obs.HeadObjectInput{
			Bucket: bucketCfg.Name,
			Key:    req.Key,
		})
		if err == nil {
			return nil, &storage.OpError{
				Op:       "upload",
				Provider: p.Provider(),
				Instance: p.instance,
				Bucket:   bucketAlias,
				Key:      req.Key,
				Err:      storage.ErrObjectExists,
			}
		}
		if !isNotFound(err) {
			return nil, wrapErr("head_object", p.Provider(), p.instance, bucketAlias, req.Key, err)
		}
	}

	input := &obs.PutObjectInput{
		PutObjectBasicInput: obs.PutObjectBasicInput{
			ObjectOperationInput: obs.ObjectOperationInput{
				Bucket:   bucketCfg.Name,
				Key:      req.Key,
				Metadata: req.Metadata,
			},
		},
		Body: req.Body,
	}

	if req.ContentLength > 0 {
		input.ContentLength = req.ContentLength
	}
	if req.ContentType != "" {
		input.ContentType = req.ContentType
	}
	if req.ContentDisposition != "" {
		input.ContentDisposition = req.ContentDisposition
	}
	if req.CacheControl != "" {
		input.CacheControl = req.CacheControl
	}
	// ACL 优先级：调用方显式指定 > 桶配置的 default_acl > 不设置（由桶策略决定可见性）。
	// 华为云的 canned ACL 取值与 S3 同名（如 "public-read"）。
	if acl := req.ACL; acl != "" {
		input.ACL = obs.AclType(acl)
	} else if bucketCfg.DefaultACL != "" {
		input.ACL = obs.AclType(bucketCfg.DefaultACL)
	}

	out, err := p.client.PutObject(input)
	if err != nil {
		return nil, wrapErr("put_object", p.Provider(), p.instance, bucketAlias, req.Key, err)
	}

	objectURL := buildObjectURL(bucketCfg, req.Key)
	if objectURL == "" {
		// 桶没配 public_base_url 时回落到 SDK 拼出的地址（endpoint/bucket/key）
		objectURL = out.ObjectUrl
	}

	result := &storage2.UploadResult{
		Provider:  string(p.Provider()),
		Instance:  p.instance,
		Bucket:    bucketAlias,
		Key:       req.Key,
		ETag:      out.ETag,
		VersionID: out.VersionId,
		ObjectURL: objectURL,
		Location:  objectURL,
		RequestID: out.RequestId,
	}

	return result, nil
}

func wrapErr(op string, provider storage.Provider, instance, bucket, key string, err error) error {
	return &storage.OpError{
		Op:       op,
		Provider: provider,
		Instance: instance,
		Bucket:   bucket,
		Key:      key,
		Err:      err,
	}
}

// isNotFound 判断是否为「对象不存在」。
//
// 华为云 SDK 用 obs.ObsError 承载服务端错误，携带 HTTP 状态码与错误码，
// 与 S3 的 smithy.APIError 不是一套，不能照搬 s3 provider 的实现。
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	// SDK 以值类型返回 ObsError（Error() 为值接收者），故此处按值取
	var obsErr obs.ObsError
	if errors.As(err, &obsErr) {
		if obsErr.StatusCode == 404 {
			return true
		}
		switch obsErr.Code {
		case "NoSuchKey", "NotFound", "404":
			return true
		}
	}
	return false
}

func buildObjectURL(bucket config.HuaweiObsBucketConfig, key string) string {
	if bucket.PublicBaseURL == "" {
		return ""
	}
	base := strings.TrimRight(bucket.PublicBaseURL, "/")
	k := strings.TrimLeft(key, "/")
	return base + "/" + k
}
