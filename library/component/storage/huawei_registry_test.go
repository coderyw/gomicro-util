package storage

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	storage2 "github.com/coderyw/gomicro-util/internal/storage"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/storage"
)

// registryPtr 是包级变量，用例之间必须还原，避免影响同包其它测试
func swapRegistry(t *testing.T, r *storage2.Registry) {
	t.Helper()
	old := registryPtr.Load()
	registryPtr.Store(r)
	t.Cleanup(func() { registryPtr.Store(old) })
}

func huaweiCfg() *config.HuaweiObsConfig {
	return &config.HuaweiObsConfig{
		Ak:       "ak-value",
		Sk:       "sk-value",
		Endpoint: "http://127.0.0.1:1",
		Buckets: []config.HuaweiObsBucketConfig{
			{IsDefault: true, Alias: "default", Name: "bucket-name"},
		},
	}
}

// 同一实例名分属两个云会导致分派歧义，必须在初始化阶段就拒绝。
//
// 两份配置都必须是合法的：否则 provider 构造本身就会失败，
// 测试即使在查重逻辑被删掉的情况下也会「通过」，成为假阳性。
func TestInitObjectStorage_DuplicateInstanceAcrossProviders(t *testing.T) {
	swapRegistry(t, registryPtr.Load())

	_, err := InitObjectStorage(&config.ObjectStorage{
		AwsS3Config: map[string]*config.AwsS3Config{
			"core": {
				Region:                    "ap-southeast-1",
				UseDefaultCredentialChain: true,
				Buckets: []config.AwsS3BucketConfig{
					{IsDefault: true, Alias: "default", Name: "bucket-name"},
				},
			},
		},
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{"core": huaweiCfg()},
	})
	if err == nil {
		t.Fatal("跨 provider 的同名实例应报错")
	}
	if !strings.Contains(err.Error(), "duplicated object storage instance") {
		t.Errorf("应因实例重名而失败，而不是 provider 构造失败: %v", err)
	}
}

// 华为云实例应被注册进同一个 registry，与 s3 实例共存
func TestInitObjectStorage_RegistersHuaweiInstance(t *testing.T) {
	swapRegistry(t, registryPtr.Load())

	closeFn, err := InitObjectStorage(&config.ObjectStorage{
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{"obs_core": huaweiCfg()},
	})
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	if closeFn == nil {
		t.Fatal("应返回关闭函数")
	}

	up, ok := registryPtr.Load().Get("obs_core")
	if !ok {
		t.Fatal("华为云实例 obs_core 未注册")
	}
	if up.Provider() != storage2.ProviderHuaweiObs {
		t.Errorf("provider 类型应为 %q, got %q", storage2.ProviderHuaweiObs, up.Provider())
	}

	closeFn() // 不应 panic
}

// default_instance 指向不存在的实例时必须启动失败。
// 放行的话业务侧留空 Instance 会以为「跟随部署方默认」，实际每次上传都 instance not found。
func TestInitObjectStorage_DefaultInstanceNotFound(t *testing.T) {
	swapRegistry(t, registryPtr.Load())

	_, err := InitObjectStorage(&config.ObjectStorage{
		DefaultInstance: "typo_core",
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{"oss_core": huaweiCfg()},
	})
	if err == nil {
		t.Fatal("默认实例不存在时应报错")
	}
	if !strings.Contains(err.Error(), "default_instance") {
		t.Errorf("错误信息应点明是 default_instance 配错: %v", err)
	}
}

// 配了 default_instance 时应落到 registry，业务侧可据此省掉自己的实例名配置
func TestInitObjectStorage_SetsDefaultInstance(t *testing.T) {
	swapRegistry(t, registryPtr.Load())

	if _, err := InitObjectStorage(&config.ObjectStorage{
		DefaultInstance: "oss_core",
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{"oss_core": huaweiCfg()},
	}); err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	if got := DefaultInstance(); got != "oss_core" {
		t.Errorf("默认实例应为 oss_core, got %q", got)
	}
}

// 未配 default_instance 时不做任何隐式推导——哪怕只有一个实例。
// 隐式推导会让「部署方新增第二个实例」这个纯配置动作静默改变既有业务的落桶行为。
func TestInitObjectStorage_NoImplicitDefault(t *testing.T) {
	swapRegistry(t, registryPtr.Load())

	if _, err := InitObjectStorage(&config.ObjectStorage{
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{"oss_core": huaweiCfg()},
	}); err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	if got := DefaultInstance(); got != "" {
		t.Errorf("未配 default_instance 时不应有默认实例, got %q", got)
	}
}

// 组件未初始化时 DefaultInstance 不应 panic
func TestDefaultInstance_RegistryNotInitialized(t *testing.T) {
	swapRegistry(t, nil)

	if got := DefaultInstance(); got != "" {
		t.Errorf("registry 未初始化时应返回空串, got %q", got)
	}
}

// 配置为 nil 时直接返回，registry 保持未初始化
func TestInitObjectStorage_NilConfig(t *testing.T) {
	swapRegistry(t, nil)

	closeFn, err := InitObjectStorage(nil)
	if err != nil {
		t.Fatalf("配置为 nil 不应报错: %v", err)
	}
	if closeFn != nil {
		t.Error("配置为 nil 时不应返回关闭函数")
	}
}

// 未配置 storage.object_storage 时 registry 为 nil，
// Upload 必须返回明确错误而不是 nil 解引用 panic
func TestUpload_RegistryNotInitialized(t *testing.T) {
	swapRegistry(t, nil)

	_, err := Upload(context.Background(), &storage.UploadRequest{
		Instance: "core",
		Key:      "demo/a.txt",
		Body:     bytes.NewReader([]byte("x")),
	})
	if err == nil {
		t.Fatal("registry 未初始化时应报错")
	}
	if !errors.Is(err, storage2.ErrInstanceNotFound) {
		t.Errorf("应包装 ErrInstanceNotFound, got %v", err)
	}
}

func TestUpload_NilRequest(t *testing.T) {
	swapRegistry(t, nil)

	if _, err := Upload(context.Background(), nil); !errors.Is(err, storage2.ErrInvalidArgument) {
		t.Errorf("req 为 nil 应返回 ErrInvalidArgument, got %v", err)
	}
}

// registry 未初始化时 close 不应 panic
func TestClose_NilRegistry(t *testing.T) {
	swapRegistry(t, nil)
	close()
}

// Upload 给 key 加租户前缀时必须作用在副本上，不能改写调用方的 req。
//
// 就地改写会让「复用同一个 UploadRequest 重试」变成二次加前缀：
// 第一次 demo/a.txt → tenantok/demo/a.txt，第二次再加一层 → tenantok/tenantok/demo/a.txt。
func TestUpload_DoesNotMutateRequest(t *testing.T) {
	swapRegistry(t, storage2.NewRegistry()) // 空 registry：实例查不到即返回，不会真的发请求
	env.SetFallbackTenantName("")
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")

	req := &storage.UploadRequest{
		Instance: "core",
		Key:      "demo/a.txt",
		Body:     bytes.NewReader([]byte("x")),
	}
	if _, err := Upload(ctx, req); err == nil {
		t.Fatal("实例未注册时应报错")
	}
	if req.Key != "demo/a.txt" {
		t.Errorf("不应改写调用方的 req.Key, got %q", req.Key)
	}
}
