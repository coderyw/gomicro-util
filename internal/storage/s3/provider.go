// Package s3 AWS S3 对象存储 provider。
//
// 与 internal/storage/huawei 保持同构，两者共用 internal/storage 的 Uploader 契约与错误类型，
// 由 library/component/storage 的 registry 按实例名统一分派。
//
// 与华为 OBS provider 的能力差异（调用方需知晓）：
//  1. 不支持对象 ACL —— UploadRequest.ACL 会被忽略，也没有桶级 default_acl 配置。
//     2023-04 之后新建的 S3 桶默认 Object Ownership = Bucket owner enforced
//     且 Block Public Access 全开，此时 ACL 已被禁用，带 ACL 的 PutObject 会直接
//     返回 AccessControlListNotSupported (400)。对象可见性一律由桶策略决定，
//     公共访问地址由桶配置的 public_base_url（通常是 CDN 域名）拼出。
//     ⚠️ 不要照抄华为实例的 default_acl: public-read —— 那是 OBS 侧的要求，
//     搬到 S3 上会让上传全线失败。
//  2. 桶未配置 public_base_url 时返回空 ObjectURL（华为 provider 会回落 SDK 拼的地址）。
package s3

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/coderyw/gomicro-util/internal/storage"
	"github.com/coderyw/gomicro-util/model/config"
	storage2 "github.com/coderyw/gomicro-util/model/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsretry "github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithy "github.com/aws/smithy-go"
)

type Provider struct {
	instance           string
	client             *s3.Client
	defaultBucketAlias string
	buckets            map[string]config.AwsS3BucketConfig
}

func New(instance string, cfg config.AwsS3Config) (*Provider, error) {
	if instance == "" {
		return nil, fmt.Errorf("instance is empty")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("region is empty")
	}
	if len(cfg.Buckets) == 0 {
		return nil, fmt.Errorf("buckets is empty")
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	if cfg.DialTimeout <= 0 {
		cfg.DialTimeout = 5 * time.Second
	}
	if cfg.TLSHandshakeTimeout <= 0 {
		cfg.TLSHandshakeTimeout = 5 * time.Second
	}
	if cfg.ResponseHeaderTimeout <= 0 {
		cfg.ResponseHeaderTimeout = 30 * time.Second
	}
	if cfg.ExpectContinueTimeout <= 0 {
		cfg.ExpectContinueTimeout = 1 * time.Second
	}
	if cfg.IdleConnTimeout <= 0 {
		cfg.IdleConnTimeout = 90 * time.Second
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 200
	}
	if cfg.MaxIdleConnsPerHost <= 0 {
		cfg.MaxIdleConnsPerHost = 100
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   cfg.DialTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          cfg.MaxIdleConns,
			MaxIdleConnsPerHost:   cfg.MaxIdleConnsPerHost,
			IdleConnTimeout:       cfg.IdleConnTimeout,
			TLSHandshakeTimeout:   cfg.TLSHandshakeTimeout,
			ExpectContinueTimeout: cfg.ExpectContinueTimeout,
			ResponseHeaderTimeout: cfg.ResponseHeaderTimeout,
		},
	}

	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithHTTPClient(httpClient),
		awsconfig.WithRetryer(func() aws.Retryer {
			return awsretry.NewStandard(func(o *awsretry.StandardOptions) {
				o.MaxAttempts = cfg.MaxRetries
			})
		}),
	}

	if !cfg.UseDefaultCredentialChain {
		if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
			return nil, fmt.Errorf("static credentials are empty")
		}
		loadOpts = append(loadOpts,
			awsconfig.WithCredentialsProvider(
				aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
					cfg.AccessKeyID,
					cfg.SecretAccessKey,
					cfg.SessionToken,
				)),
			),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config failed: %w", err)
	}

	var s3Opts []func(*s3.Options)
	s3Opts = append(s3Opts, func(o *s3.Options) {
		o.UsePathStyle = cfg.UsePathStyle
	})

	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	bucketMap := make(map[string]config.AwsS3BucketConfig, len(cfg.Buckets))
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
	return storage.ProviderS3
}

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

	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}

	// 不允许覆盖时，先探测
	if !req.AllowOverwrite {
		_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(bucketCfg.Name),
			Key:    aws.String(req.Key),
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

	input := &s3.PutObjectInput{
		Bucket:   aws.String(bucketCfg.Name),
		Key:      aws.String(req.Key),
		Body:     req.Body,
		Metadata: req.Metadata,
	}

	if req.ContentLength > 0 {
		input.ContentLength = aws.Int64(req.ContentLength)
	}
	if req.ContentType != "" {
		input.ContentType = aws.String(req.ContentType)
	}
	if req.ContentDisposition != "" {
		input.ContentDisposition = aws.String(req.ContentDisposition)
	}
	if req.CacheControl != "" {
		input.CacheControl = aws.String(req.CacheControl)
	}
	// 有意不设置对象 ACL，UploadRequest.ACL 在本 provider 被忽略。见包注释。
	if len(req.Tags) > 0 {
		input.Tagging = aws.String(encodeTags(req.Tags))
	}

	out, err := p.client.PutObject(ctx, input)
	if err != nil {
		return nil, wrapErr("put_object", p.Provider(), p.instance, bucketAlias, req.Key, err)
	}

	result := &storage2.UploadResult{
		Provider:  string(p.Provider()),
		Instance:  p.instance,
		Bucket:    bucketAlias,
		Key:       req.Key,
		VersionID: aws.ToString(out.VersionId),
		ObjectURL: buildObjectURL(bucketCfg, req.Key),
		Location:  buildObjectURL(bucketCfg, req.Key),
	}

	if reqID, ok := awsmiddleware.GetRequestIDMetadata(out.ResultMetadata); ok {
		result.RequestID = reqID
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

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "NotFound" || code == "NoSuchKey" || code == "404"
	}
	return false
}

func encodeTags(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	vals := url.Values{}
	for k, v := range tags {
		vals.Set(k, v)
	}
	return vals.Encode()
}

func buildObjectURL(bucket config.AwsS3BucketConfig, key string) string {
	if bucket.PublicBaseURL == "" {
		return ""
	}
	base := strings.TrimRight(bucket.PublicBaseURL, "/")
	k := strings.TrimLeft(key, "/")
	return base + "/" + k
}
