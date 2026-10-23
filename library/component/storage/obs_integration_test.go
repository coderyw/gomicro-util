package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/storage"
)

// TestIntegration_HuaweiObs 打真实华为云 OBS，验证 ns 线的完整上传链路。
//
// 默认跳过；需要显式提供环境变量才会执行，因此不影响 CI：
//
//	OBS_IT_HW_AK=<ak> \
//	OBS_IT_HW_SK=<sk> \
//	OBS_IT_HW_ENDPOINT=obs.ap-southeast-3.myhuaweicloud.com \
//	OBS_IT_HW_BUCKET=<真实桶名> \
//	go test ./library/component/storage/ -run TestIntegration_HuaweiObs -v
//
// 可选：
//   - OBS_IT_TENANT —— 租户名。**ns 线不需要**（ns 没有租户这一说），
//     只有想顺带验 saas 多租户前缀时才设，设了才会多跑一个子用例
//   - OBS_IT_HW_PUBLIC_BASE_URL —— CDN 域名。不配则用 SDK 拼的地址，
//     这也是华为与 S3 的关键差异（S3 不配就返回空串）
//   - OBS_IT_HW_ACL —— 桶的 default_acl，默认 "public-read"。
//     置为 "-" 表示不设 ACL（可见性完全交给桶策略），此时跳过匿名下载断言
//
// 子用例对应三种 pod 形态下 key 的差异：
//
//	| 形态                        | ctx 租户 | TENANT_FALLBACK | key           |
//	|-----------------------------|---------|-----------------|---------------|
//	| ns 独享（生产形态）           | 无      | 有              | 原样，不加前缀  |
//	| saas 多租户 + 丢了租户上下文   | 无      | 无              | tenantns/ ⚠️  |
//	| saas 多租户（可选）           | 有      | 无              | tenant{租户}/  |
//
// 第一种是 ns 生产的真实形态：FmtByType 第一行见 HasFallbackTenant() 为真就原样返回，
// 优先级高于一切租户逻辑，所以 ns 从 other.obs_cfg 直连迁到本组件时 key 形态不变。
//
// 三种状态都由用例自己用 env.SetFallbackTenantName 强制切换（同一进程里要跑全三种，
// 靠 shell 里的 TENANT_FALLBACK 驱动不了），跑完按 os.Getenv 还原。
//
// ⚠️ 第二种是已知缺陷，且**只影响 saas**：getPrefix 在无租户上下文时兜底成 "ns"
// （见 registry.go 注释），后台任务 / 异步 goroutine 这类丢了 ctx 租户的路径，
// 会把 saas 的对象写进 tenantns/ 目录，混进 ns 的命名空间。
func TestIntegration_HuaweiObs(t *testing.T) {

	ak := os.Getenv("OBS_IT_HW_AK")
	sk := os.Getenv("OBS_IT_HW_SK")
	endpoint := os.Getenv("OBS_IT_HW_ENDPOINT")
	bucket := os.Getenv("OBS_IT_HW_BUCKET")
	// 租户是可选的：ns 线没有租户概念，不该为了跑通测试硬造一个
	tenant := os.Getenv("OBS_IT_TENANT")

	if ak == "" || sk == "" || endpoint == "" || bucket == "" {
		t.Skip("跳过：需设置 OBS_IT_HW_AK / OBS_IT_HW_SK / OBS_IT_HW_ENDPOINT / OBS_IT_HW_BUCKET")
	}

	// 独享子用例要往 fallbackTenantName 里塞一个非空值。取本机真实的 TENANT_FALLBACK
	// （没设就用 "ns" 占位）：值本身不影响 key —— FmtByType 只看它非不非空就短路了，
	// 所以这里不额外发明一个测试专用的环境变量。
	fallback := os.Getenv(cst.ENV_TENANT_FALLBACK)
	if fallback == "" {
		fallback = "ns"
	}

	acl := os.Getenv("OBS_IT_HW_ACL")
	if acl == "" {
		acl = "public-read"
	}
	if acl == "-" {
		acl = ""
	}

	swapRegistry(t, registryPtr.Load())

	closeFn, err := InitObjectStorage(&config.ObjectStorage{
		DefaultInstance: "oss_core",
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{
			"oss_core": {
				Ak:       ak,
				Sk:       sk,
				Endpoint: endpoint,
				Buckets: []config.HuaweiObsBucketConfig{
					{
						IsDefault:     true,
						Alias:         "default",
						Name:          bucket,
						PublicBaseURL: os.Getenv("OBS_IT_HW_PUBLIC_BASE_URL"),
						DefaultACL:    acl,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("初始化对象存储失败: %v", err)
	}
	if closeFn != nil {
		t.Cleanup(closeFn)
	}

	// 顺带验证默认实例：下面所有上传都不传 Instance
	if got := DefaultInstance(); got != "oss_core" {
		t.Fatalf("默认实例应为 oss_core, got %q", got)
	}

	cases := []struct {
		name string
		// fallbackTenant 模拟 pod 的 TENANT_FALLBACK；非空即走独享逻辑
		fallbackTenant string
		// ctxTenant 模拟网关下发的租户上下文；ns 线为空
		ctxTenant  string
		wantPrefix string
	}{
		{
			name:           "ns独享_设了TENANT_FALLBACK_key原样",
			fallbackTenant: fallback,
			ctxTenant:      "",
			wantPrefix:     "",
		},
		{
			// 已知缺陷的当场取证，模拟的是 **saas 多租户 pod 丢了租户上下文**
			// （后台任务 / 异步 goroutine / 无状态回调），不是 ns：
			// ns 设了 TENANT_FALLBACK，上一条就短路了，走不到这里。
			// 此时 getPrefix 的默认命名空间 "ns" 生效，saas 的对象被写进 tenantns/ 目录。
			// 将来把默认命名空间改成配置项、默认留空后，本用例要跟着改。
			name:           "saas丢租户上下文_key被兜底成tenantns",
			fallbackTenant: "",
			ctxTenant:      "",
			wantPrefix:     "tenantns/",
		},
	}
	if tenant != "" {
		cases = append(cases, struct {
			name           string
			fallbackTenant string
			ctxTenant      string
			wantPrefix     string
		}{
			name:           "saas多租户_key带租户前缀",
			fallbackTenant: "",
			ctxTenant:      tenant,
			wantPrefix:     "tenant" + tenant + "/",
		})
	} else {
		t.Log("未设 OBS_IT_TENANT，跳过 saas 多租户子用例（ns 线本就没有租户概念）")
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// fallbackTenantName 是包级变量，用例之间必须还原。
			// env 包没有 getter，按它自己的初始化方式（读 TENANT_FALLBACK）还原
			env.SetFallbackTenantName(c.fallbackTenant)
			t.Cleanup(func() { env.SetFallbackTenantName(os.Getenv(cst.ENV_TENANT_FALLBACK)) })

			// ctxTenant 为空时不写 CONTEXT_NAME_SPACE，如实模拟 ns 的无租户上下文
			ctx := context.Background()
			if c.ctxTenant != "" {
				ctx = metadata.SetCtxData(ctx, metadata.CONTEXT_NAME_SPACE, c.ctxTenant)
			}

			content := []byte(fmt.Sprintf("huawei obs upload check %d\n", time.Now().UnixMilli()))
			objectKey := fmt.Sprintf("temp/excel/import_batch_send_sms_%d.txt", time.Now().UnixMilli())

			res, err := Upload(ctx, &storage.UploadRequest{
				// 不传 Instance —— 走 default_instance
				Key:            objectKey,
				Body:           bytes.NewReader(content),
				ContentLength:  int64(len(content)),
				AllowOverwrite: true,
				// Timeout 对华为 provider 不生效（SDK 只有客户端级超时），这里传了也只是无害
			})
			if err != nil {
				t.Fatalf("上传失败: %+v", err)
			}
			t.Logf("上传成功: instance=%s bucket=%s key=%s etag=%s url=%s",
				res.Instance, res.Bucket, res.Key, res.ETag, res.ObjectURL)

			if res.Instance != "oss_core" {
				t.Errorf("应由默认实例 oss_core 承接, got %q", res.Instance)
			}
			if want := c.wantPrefix + objectKey; res.Key != want {
				t.Errorf("对象 key 应为 %q，实际为 %q", want, res.Key)
			}

			// 华为侧未配 public_base_url 时会回落到 SDK 拼的地址，任何情况下都不该为空
			if res.ObjectURL == "" {
				t.Fatal("ObjectURL 为空 —— 华为 provider 应回落到 SDK 地址，这里为空说明 provider 有问题")
			}

			if acl == "" {
				t.Log("未设 default_acl，跳过匿名下载断言（可见性由桶策略决定）")
				return
			}

			// 匿名可读 —— 迁移后最容易踩的坑：桶没配 default_acl，下载链接变 403
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, res.ObjectURL, nil)
			if err != nil {
				t.Fatalf("构造下载请求失败: %v", err)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("匿名下载失败: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("匿名下载返回 %s —— 对象非公开可读，"+
					"检查桶的 default_acl（当前 %q）与桶策略", resp.Status, acl)
			}

			got, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取下载内容失败: %v", err)
			}
			if !bytes.Equal(got, content) {
				t.Errorf("下载内容与上传内容不一致\n上传: %q\n下载: %q", content, got)
			}
		})
	}
}

// TestIntegration_HuaweiObs_BucketAlias 验证多桶实例按 alias 选桶。
//
// 需要两个真实桶：
//
//	OBS_IT_HW_AK / OBS_IT_HW_SK / OBS_IT_HW_ENDPOINT 同上，外加
//	OBS_IT_HW_BUCKET=<默认桶> OBS_IT_HW_BUCKET2=<第二个桶>
//
// 选桶与租户无关（租户隔离体现在 key 前缀上），所以这里按 ns 形态跑：
// 设上兜底租户、ctx 不带租户，key 保持原样，断言只看落到了哪个桶。
func TestIntegration_HuaweiObs_BucketAlias(t *testing.T) {
	ak := os.Getenv("OBS_IT_HW_AK")
	sk := os.Getenv("OBS_IT_HW_SK")
	endpoint := os.Getenv("OBS_IT_HW_ENDPOINT")
	bucket := os.Getenv("OBS_IT_HW_BUCKET")
	bucket2 := os.Getenv("OBS_IT_HW_BUCKET2")

	if ak == "" || sk == "" || endpoint == "" || bucket == "" || bucket2 == "" {
		t.Skip("跳过：需在基础环境变量之外再设置 OBS_IT_HW_BUCKET2")
	}

	swapRegistry(t, registryPtr.Load())
	env.SetFallbackTenantName("ns")
	t.Cleanup(func() { env.SetFallbackTenantName(os.Getenv(cst.ENV_TENANT_FALLBACK)) })

	closeFn, err := InitObjectStorage(&config.ObjectStorage{
		DefaultInstance: "oss_core",
		HuaweiObsConfig: map[string]*config.HuaweiObsConfig{
			"oss_core": {
				Ak:       ak,
				Sk:       sk,
				Endpoint: endpoint,
				Buckets: []config.HuaweiObsBucketConfig{
					{IsDefault: true, Alias: "core", Name: bucket, DefaultACL: "public-read"},
					{Alias: "archive", Name: bucket2, DefaultACL: "public-read"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("初始化对象存储失败: %v", err)
	}
	if closeFn != nil {
		t.Cleanup(closeFn)
	}

	ctx := context.Background()
	content := []byte("bucket alias check\n")

	for _, tc := range []struct {
		bucketAlias string
		wantBucket  string
	}{
		{bucketAlias: "", wantBucket: "core"}, // 留空 → is_default 的那个
		{bucketAlias: "archive", wantBucket: "archive"},
	} {
		res, err := Upload(ctx, &storage.UploadRequest{
			Bucket:         tc.bucketAlias,
			Key:            fmt.Sprintf("demo/alias_%d.txt", time.Now().UnixNano()),
			Body:           bytes.NewReader(content),
			ContentLength:  int64(len(content)),
			AllowOverwrite: true,
		})
		if err != nil {
			t.Fatalf("bucket=%q 上传失败: %+v", tc.bucketAlias, err)
		}
		if res.Bucket != tc.wantBucket {
			t.Errorf("bucket=%q 应落到 %q，实际 %q", tc.bucketAlias, tc.wantBucket, res.Bucket)
		}
	}

	// 别名不存在应报 ErrBucketNotFound，而不是静默落到默认桶
	_, err = Upload(ctx, &storage.UploadRequest{
		Bucket:        "no-such-alias",
		Key:           "demo/x.txt",
		Body:          bytes.NewReader(content),
		ContentLength: int64(len(content)),
	})
	if err == nil || !strings.Contains(err.Error(), "bucket not found") {
		t.Errorf("别名不存在应报 bucket not found, got %v", err)
	}
}
