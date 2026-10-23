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

	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/storage"
)

// TestIntegration_S3_SaasLine 打真实 S3，验证 saas 线的完整上传链路。
//
// 默认跳过；需要显式提供环境变量才会执行，因此不影响 CI：
//
//	OBS_IT_REGION=ap-southeast-1 \
//	OBS_IT_BUCKET=<真实桶名> \
//	OBS_IT_PUBLIC_BASE_URL=https://<cdn 或 桶域名> \
//	OBS_IT_TENANT=ok \
//	go test ./library/component/storage/ -run TestIntegration_S3_SaasLine -v
//
// 凭证默认走 AWS 默认凭证链（本机 aws cli / 环境变量 / IRSA）。
// 若要用静态 AK/SK，额外提供 OBS_IT_AK 与 OBS_IT_SK。
//
// 断言四件事，对应 saas 上线前最需要确认的风险点：
//  1. 上传成功
//  2. 对象 key 带上了 tenant{租户}/ 前缀（多租户隔离）
//  3. 返回的 ObjectURL 非空（桶配了 public_base_url）
//  4. 该 URL 能被匿名 GET 到 —— 下游 ca-schedule-go 就是这么下载的
func TestIntegration_S3_SaasLine(t *testing.T) {
	region := os.Getenv("OBS_IT_REGION")
	bucket := os.Getenv("OBS_IT_BUCKET")
	publicBaseURL := os.Getenv("OBS_IT_PUBLIC_BASE_URL")
	tenant := os.Getenv("OBS_IT_TENANT")

	if region == "" || bucket == "" || publicBaseURL == "" || tenant == "" {
		t.Skip("跳过：需设置 OBS_IT_REGION / OBS_IT_BUCKET / OBS_IT_PUBLIC_BASE_URL / OBS_IT_TENANT")
	}

	swapRegistry(t, registryPtr.Load())

	instanceCfg := &config.AwsS3Config{
		Region:                    region,
		UseDefaultCredentialChain: true,
		Buckets: []config.AwsS3BucketConfig{
			{
				IsDefault:     true,
				Alias:         "default",
				Name:          bucket,
				PublicBaseURL: publicBaseURL,
				// saas 线不设对象 ACL，可见性靠桶策略 —— 这正是第 4 条断言要验的
			},
		},
	}
	if ak, sk := os.Getenv("OBS_IT_AK"), os.Getenv("OBS_IT_SK"); ak != "" && sk != "" {
		instanceCfg.UseDefaultCredentialChain = false
		instanceCfg.AccessKeyID = ak
		instanceCfg.SecretAccessKey = sk
	}

	// 实例名与 saas 线部署配置里的实例名保持一致
	closeFn, err := InitObjectStorage(&config.ObjectStorage{
		AwsS3Config: map[string]*config.AwsS3Config{"s3_core": instanceCfg},
	})
	if err != nil {
		t.Fatalf("初始化对象存储失败: %v", err)
	}
	if closeFn != nil {
		t.Cleanup(closeFn)
	}

	// 模拟网关下发的租户上下文（生产上由 X-Tenant-Name 头经中间件写入）
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, tenant)

	content := []byte(fmt.Sprintf("saas line upload check %d\n", time.Now().UnixMilli()))
	objectKey := fmt.Sprintf("excel/import_batch_send_sms_%d.txt", time.Now().UnixMilli())

	res, err := Upload(ctx, &storage.UploadRequest{
		Instance:       "s3_core",
		Key:            objectKey,
		Body:           bytes.NewReader(content),
		ContentLength:  int64(len(content)),
		AllowOverwrite: true,
		Timeout:        30 * time.Second,
	})
	if err != nil {
		t.Fatalf("上传失败: %+v", err)
	}
	t.Logf("上传成功: instance=%s bucket=%s key=%s url=%s", res.Instance, res.Bucket, res.Key, res.ObjectURL)

	// 2. 租户前缀
	wantPrefix := "tenant" + tenant + "/"
	if !strings.HasPrefix(res.Key, wantPrefix) {
		t.Errorf("对象 key 应带租户前缀 %q，实际为 %q —— 多租户隔离未生效", wantPrefix, res.Key)
	}
	if res.Key != wantPrefix+objectKey {
		t.Errorf("对象 key 应为 %q，实际为 %q", wantPrefix+objectKey, res.Key)
	}

	// 3. ObjectURL 非空
	if res.ObjectURL == "" {
		t.Fatal("ObjectURL 为空：桶未配 public_base_url，会导致导入任务记录里存空路径")
	}

	// 4. 匿名可读 —— 与 ca-schedule-go/internal/util/file.go 的下载方式一致
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
		t.Fatalf("匿名下载返回 %s —— 下游 ca-schedule-go 将无法读取该对象，"+
			"需放开桶策略（S3 侧没有对象 ACL 可用）", resp.Status)
	}

	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取下载内容失败: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("下载内容与上传内容不一致\n上传: %q\n下载: %q", content, got)
	}
}
