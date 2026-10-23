package csms

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/provider"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdkerr"
	csmsv1 "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/csms/v1"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/csms/v1/model"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/csms/v1/region"

	"github.com/coderyw/gomicro-util/library/component/secret"
)

const (
	Scheme        = "csms"
	envRegion     = "HUAWEICLOUD_SDK_REGION"
	defaultRegion = "ap-southeast-3"
	latestVersion = "latest"
	// 单次 HTTP 请求超时：SDK 默认 120s，远超启动期重试 deadline，黑洞型故障下会让 deadline 失效
	requestTimeout = 5 * time.Second
)

type Client interface {
	ShowSecretVersion(req *model.ShowSecretVersionRequest) (*model.ShowSecretVersionResponse, error)
}

type Provider struct {
	mu        sync.Mutex
	client    Client
	newClient func() (Client, error)
}

func New() *Provider {
	return &Provider{newClient: buildClient}
}

func (p *Provider) Scheme() string { return Scheme }

// Fetch 拉取凭据。华为 SDK 的调用不接 ctx，这里在 ctx 结束（单次超时 / 调用方取消）时立即返回 ctx 错误，
// 在途请求由客户端的 HTTP 超时兜底收尾；客户端初始化（凭据链取临时凭证同样是出站请求）也在同一约束内
func (p *Provider) Fetch(ctx context.Context, name string) (string, error) {
	type result struct {
		val string
		err error
	}
	ch := make(chan result, 1) // 带缓冲：ctx 先结束时在途 goroutine 写入后自行退出，不泄漏阻塞
	go func() {
		v, err := p.fetch(name)
		ch <- result{val: v, err: err}
	}()
	select {
	case r := <-ch:
		return r.val, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (p *Provider) fetch(name string) (string, error) {
	client, err := p.getClient()
	if err != nil {
		return "", err
	}
	resp, err := client.ShowSecretVersion(&model.ShowSecretVersionRequest{
		SecretName: name,
		VersionId:  latestVersion,
	})
	if err != nil {
		if se, ok := err.(*sdkerr.ServiceResponseError); ok && (se.StatusCode == 404 || se.StatusCode == 403 || se.StatusCode == 400) {
			return "", secret.NonRetryable(err)
		}
		return "", err
	}
	if resp == nil || resp.Version == nil || resp.Version.SecretString == nil {
		return "", secret.NonRetryable(fmt.Errorf("csms secret %s: empty secret_string", name))
	}
	return *resp.Version.SecretString, nil
}

// getClient 懒初始化并复用客户端。初始化失败不缓存错误：本次解析内按可重试处理（凭据链取临时凭证是出站请求，
// 会被尚未就绪的 sidecar 拦住而超时，退避后即可恢复），下一次 Fetch 也重新初始化，热更新阶段可自愈而不必重启；
// region 非法、客户端构建参数错误这类确定性失败由 buildClient 标为不可重试
func (p *Provider) getClient() (Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		return p.client, nil
	}
	c, err := p.newClient()
	if err != nil {
		return nil, fmt.Errorf("csms client init: %w", err)
	}
	p.client = c
	return c, nil
}

func buildClient() (Client, error) {
	regionID := os.Getenv(envRegion)
	if regionID == "" {
		regionID = defaultRegion
	}
	reg, err := region.SafeValueOf(regionID)
	if err != nil {
		return nil, secret.NonRetryable(err)
	}
	cred, err := provider.BasicCredentialProviderChain().GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("huaweicloud credentials: %w", err)
	}
	hc, err := csmsv1.CsmsClientBuilder().WithRegion(reg).WithCredential(cred).
		WithHttpConfig(config.DefaultHttpConfig().WithTimeout(requestTimeout)).SafeBuild()
	if err != nil {
		return nil, secret.NonRetryable(err)
	}
	return csmsv1.NewCsmsClient(hc), nil
}

func init() { secret.Register(New()) }
