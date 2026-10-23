package huawei

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/coderyw/gomicro-util/internal/storage"
	"github.com/coderyw/gomicro-util/model/config"
	storagemodel "github.com/coderyw/gomicro-util/model/storage"
)

func baseCfg() config.HuaweiObsConfig {
	return config.HuaweiObsConfig{
		Ak:       "ak-value",
		Sk:       "sk-value",
		Endpoint: "obs.example.com",
		Buckets: []config.HuaweiObsBucketConfig{
			{Alias: "first", Name: "bucket-first"},
			{Alias: "second", Name: "bucket-second"},
		},
	}
}

// New 的必填校验：实例名 / endpoint / buckets 缺一不可
func TestNew_Validation(t *testing.T) {
	if _, err := New("", baseCfg()); err == nil {
		t.Error("实例名为空时应报错")
	}

	cfg := baseCfg()
	cfg.Endpoint = ""
	if _, err := New("core", cfg); err == nil {
		t.Error("endpoint 为空时应报错")
	}

	cfg = baseCfg()
	cfg.Buckets = nil
	if _, err := New("core", cfg); err == nil {
		t.Error("buckets 为空时应报错")
	}
}

// 桶别名重复应被拒绝
func TestNew_DuplicateBucketAlias(t *testing.T) {
	cfg := baseCfg()
	cfg.Buckets = []config.HuaweiObsBucketConfig{
		{Alias: "same", Name: "bucket-a"},
		{Alias: "same", Name: "bucket-b"},
	}
	if _, err := New("core", cfg); err == nil {
		t.Error("桶别名重复时应报错")
	}
}

// 默认桶解析：未标 IsDefault 时取列表第一个，与 s3 provider 行为一致
func TestNew_DefaultBucketFallsBackToFirst(t *testing.T) {
	p, err := New("core", baseCfg())
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	if p.defaultBucketAlias != "first" {
		t.Errorf("未标默认桶时应取第一个 first, got %q", p.defaultBucketAlias)
	}
}

// 默认桶解析：显式 IsDefault 优先
func TestNew_DefaultBucketHonoursIsDefault(t *testing.T) {
	cfg := baseCfg()
	cfg.Buckets[1].IsDefault = true
	p, err := New("core", cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	if p.defaultBucketAlias != "second" {
		t.Errorf("应取标了 IsDefault 的 second, got %q", p.defaultBucketAlias)
	}
}

func TestProvider_Kind(t *testing.T) {
	p, err := New("core", baseCfg())
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	if p.Provider() != storage.ProviderHuaweiObs {
		t.Errorf("provider 类型应为 %q, got %q", storage.ProviderHuaweiObs, p.Provider())
	}
}

// Close 必须是空实现：client 为全进程共享，SDK 的 Close 会把 conf 置 nil，
// 关停期若仍有在途上传就会 nil 解引用 panic。
//
// 断言方式：Close 之后仍能走完一次 Upload。若把 Close 改成调 obsClient.Close()，
// 这里会 panic 从而变红——不能只断言 p.client 非 nil，SDK 改的是所指结构体的字段，
// 指针本身永远不会被置空，那样的断言恒真。
func TestClose_IsNoop(t *testing.T) {
	cfg := baseCfg()
	cfg.Endpoint = "http://127.0.0.1:1"
	cfg.MaxRetries = 1

	p, err := New("core", cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}

	p.Close()
	p.Close() // 可重复调用

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Close 之后 Upload 不应 panic: %v", r)
		}
	}()
	// 预期是连接失败，不是 panic
	if _, err := p.Upload(context.Background(), &storagemodel.UploadRequest{
		Key:            "a.txt",
		Body:           strings.NewReader("x"),
		AllowOverwrite: true,
	}); err == nil {
		t.Error("endpoint 不可达时应返回错误")
	}
}

// 参数校验走 OpError 包装，且复用公共哨兵错误；这几条都在发请求之前返回，不触网
func TestUpload_ArgumentValidation(t *testing.T) {
	p, err := New("core", baseCfg())
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}
	ctx := context.Background()

	cases := []struct {
		name   string
		req    *storagemodel.UploadRequest
		expect error
	}{
		{"req 为 nil", nil, storage.ErrInvalidArgument},
		{"body 为 nil", &storagemodel.UploadRequest{Key: "a.txt"}, storage.ErrInvalidArgument},
		{"key 为空", &storagemodel.UploadRequest{Body: strings.NewReader("x")}, storage.ErrInvalidArgument},
		{
			"桶别名不存在",
			&storagemodel.UploadRequest{Key: "a.txt", Body: strings.NewReader("x"), Bucket: "nope", AllowOverwrite: true},
			storage.ErrBucketNotFound,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := p.Upload(ctx, c.req)
			if err == nil {
				t.Fatal("应返回错误")
			}
			if !errors.Is(err, c.expect) {
				t.Errorf("应包装 %v, got %v", c.expect, err)
			}
			var opErr *storage.OpError
			if !errors.As(err, &opErr) {
				t.Errorf("应为 *storage.OpError, got %T", err)
			}
		})
	}
}

// Bucket 留空时应回填默认桶别名，而不是报 BucketNotFound。
//
// 桶解析发生在发请求之前，这里把 endpoint 指向本地保留端口让连接立即被拒，
// 避免单测触网或等待超时。
func TestUpload_EmptyBucketResolvesToDefault(t *testing.T) {
	cfg := baseCfg()
	cfg.Endpoint = "http://127.0.0.1:1"
	cfg.MaxRetries = 1

	p, err := New("core", cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}

	req := &storagemodel.UploadRequest{Key: "a.txt", Body: strings.NewReader("x"), AllowOverwrite: true}
	_, err = p.Upload(context.Background(), req)

	if errors.Is(err, storage.ErrBucketNotFound) {
		t.Error("Bucket 留空应解析到默认桶，不应报 BucketNotFound")
	}
	// endpoint 不可达，上传必然失败，但错误里带的 bucket 就是解析后的别名，
	// 可借此确认默认桶解析生效（不必真的连上 OBS）
	var opErr *storage.OpError
	if errors.As(err, &opErr) && opErr.Bucket != "first" {
		t.Errorf("Bucket 留空应解析到默认桶别名 first, got %q", opErr.Bucket)
	}
	// 入参属于调用方：解析默认桶只能用局部变量，不能就地回填。
	// 否则调用方复用同一个 req 时会被我们悄悄改掉。
	if req.Bucket != "" {
		t.Errorf("不应改写调用方的 req.Bucket, got %q", req.Bucket)
	}
}

// 共享一个长生命周期 client 是本 provider 与旧实现最大的差异，
// 这里在 -race 下并发上传，确认 provider 自身与 SDK 请求路径没有数据竞争。
// 请求会因 endpoint 不可达而失败，这不影响竞态检测。
func TestUpload_ConcurrentIsRaceFree(t *testing.T) {
	cfg := baseCfg()
	cfg.Endpoint = "http://127.0.0.1:1"
	cfg.MaxRetries = 1

	p, err := New("core", cfg)
	if err != nil {
		t.Fatalf("New 失败: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = p.Upload(context.Background(), &storagemodel.UploadRequest{
				Key:            "demo/a.txt",
				Body:           strings.NewReader("x"),
				AllowOverwrite: true,
			})
		}()
	}
	wg.Wait()
}

// ACL 优先级：req.ACL > 桶的 default_acl > 不设。
//
// 这条决定 ns 从「业务侧写死 AclPublicRead」迁移过来后对象还公不公开，
// 用一个本地 httptest server 接住请求，直接检查发出去的 x-obs-acl 头。
func TestUpload_ACLPrecedence(t *testing.T) {
	cases := []struct {
		name       string
		reqACL     string
		defaultACL string
		expect     string
	}{
		{"都不设则不带 ACL 头", "", "", ""},
		{"仅桶配置 default_acl", "", "public-read", "public-read"},
		{"调用方显式指定优先", "private", "public-read", "private"},
		{"调用方指定而桶未配", "public-read", "", "public-read"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var gotACL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// SDK 按签名模式在 x-amz-acl / x-obs-acl 之间切换，两个都收
				gotACL = r.Header.Get("x-obs-acl")
				if gotACL == "" {
					gotACL = r.Header.Get("x-amz-acl")
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			cfg := baseCfg()
			cfg.Endpoint = srv.URL
			cfg.Buckets[0].DefaultACL = c.defaultACL

			p, err := New("core", cfg)
			if err != nil {
				t.Fatalf("New 失败: %v", err)
			}

			if _, err := p.Upload(context.Background(), &storagemodel.UploadRequest{
				Key:            "a.txt",
				Body:           strings.NewReader("x"),
				ACL:            c.reqACL,
				AllowOverwrite: true,
			}); err != nil {
				t.Fatalf("上传失败: %v", err)
			}

			if gotACL != c.expect {
				t.Errorf("x-obs-acl 应为 %q, got %q", c.expect, gotACL)
			}
		})
	}
}

// MaxRetries 未配置时必须回落到 SDK 默认的 3 次重试，而不是零重试。
//
// 这条是本 provider 最容易写错的地方：SDK 只在 maxRetryCount < 0 时才回填默认值
// （obs.New 先置 -1，prepareConfig 再判 < 0），所以把配置里的 0 原样传下去
// 会变成真的一次都不重试。New 里用 <= 0 回填就是为了避开这个坑。
//
// 注意 SDK 只对 repeatable 的 body 重试：PutObject 仅当 body 是 *strings.Reader
// 时才置 repeatable=true（client_object.go 的 PutObject）。这里用 strings.Reader
// 正是为了让重试路径可观测；生产代码传的是 bytes.Reader，PutObject 本就不会重试。
func TestNew_MaxRetriesFallsBackToSDKDefault(t *testing.T) {
	cases := []struct {
		name        string
		maxRetries  int
		wantAttempt int32 // 首次请求 + 重试次数
	}{
		// SDK 重试之间有退避等待，用例数量按需克制
		{"未配置 → SDK 默认 3 次重试", 0, 1 + obs.DEFAULT_MAX_RETRY_COUNT},
		{"显式配 1 → 只重试 1 次", 1, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var attempts int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&attempts, 1)
				// 5xx 才会触发重试（4xx 走 canNotRetry 直接放弃）
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer srv.Close()

			cfg := baseCfg()
			cfg.Endpoint = srv.URL
			cfg.MaxRetries = c.maxRetries

			p, err := New("core", cfg)
			if err != nil {
				t.Fatalf("New 失败: %v", err)
			}

			// strings.Reader 才是 repeatable 的，能触发重试
			if _, err := p.Upload(context.Background(), &storagemodel.UploadRequest{
				Key:            "a.txt",
				Body:           strings.NewReader("x"),
				AllowOverwrite: true,
			}); err == nil {
				t.Fatal("服务端持续返回 500，上传应失败")
			}

			if got := atomic.LoadInt32(&attempts); got != c.wantAttempt {
				t.Errorf("服务端应收到 %d 次请求（首次+重试），实际 %d 次", c.wantAttempt, got)
			}
		})
	}
}

func TestBuildObjectURL(t *testing.T) {
	cases := []struct {
		name   string
		base   string
		key    string
		expect string
	}{
		{"未配置返回空串", "", "a/b.txt", ""},
		{"正常拼接", "https://cdn.example.com", "a/b.txt", "https://cdn.example.com/a/b.txt"},
		{"去掉多余斜杠", "https://cdn.example.com/", "/a/b.txt", "https://cdn.example.com/a/b.txt"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := buildObjectURL(config.HuaweiObsBucketConfig{PublicBaseURL: c.base}, c.key)
			if got != c.expect {
				t.Errorf("expect %q, got %q", c.expect, got)
			}
		})
	}
}

// isNotFound 必须认华为云的 ObsError（值类型），不能照搬 s3 的 smithy 判定
func TestIsNotFound(t *testing.T) {
	if isNotFound(nil) {
		t.Error("nil 不应判为 NotFound")
	}

	var byStatus obs.ObsError
	byStatus.StatusCode = 404
	if !isNotFound(byStatus) {
		t.Error("StatusCode=404 应判为 NotFound")
	}

	var byCode obs.ObsError
	byCode.StatusCode = 400
	byCode.Code = "NoSuchKey"
	if !isNotFound(byCode) {
		t.Error("Code=NoSuchKey 应判为 NotFound")
	}

	var other obs.ObsError
	other.StatusCode = 403
	other.Code = "AccessDenied"
	if isNotFound(other) {
		t.Error("AccessDenied 不应判为 NotFound")
	}
}
