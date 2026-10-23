package awssm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"

	"github.com/coderyw/gomicro-util/library/component/secret"
)

const (
	Scheme        = "awssm"
	defaultRegion = "ap-southeast-1"
)

type Client interface {
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type Provider struct {
	mu        sync.Mutex
	client    Client
	newClient func(ctx context.Context) (Client, error)
}

// New 返回懒初始化的 AWS Secrets Manager Provider
func New() *Provider {
	return &Provider{newClient: buildClient}
}

func (p *Provider) Scheme() string { return Scheme }

func (p *Provider) Fetch(ctx context.Context, name string) (string, error) {
	client, err := p.getClient(ctx)
	if err != nil {
		return "", err
	}
	out, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(name)})
	if err != nil {
		if isNonRetryable(err) {
			return "", secret.NonRetryable(err)
		}
		return "", err
	}
	if out.SecretString == nil {
		return "", secret.NonRetryable(fmt.Errorf("awssm secret %s: empty SecretString", name))
	}
	return *out.SecretString, nil
}

// getClient 懒初始化并复用客户端。初始化失败不缓存错误：本次解析内按可重试处理（与 csms 一致，
// 凭证链可能经出站请求取临时凭证，启动早期会被未就绪的 sidecar 拦住），下一次 Fetch 也重新初始化，热更新阶段可自愈而不必重启
func (p *Provider) getClient(ctx context.Context) (Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.client != nil {
		return p.client, nil
	}
	c, err := p.newClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("awssm client init: %w", err)
	}
	p.client = c
	return c, nil
}

func buildClient(ctx context.Context) (Client, error) {
	regionID := os.Getenv("AWS_REGION")
	if regionID == "" {
		regionID = os.Getenv("AWS_DEFAULT_REGION")
	}
	if regionID == "" {
		regionID = defaultRegion
	}
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(regionID))
	if err != nil {
		return nil, err
	}
	// 关掉 SDK 自带重试（默认 3 次）：重试统一由 secret 组件按 deadline 收口，两层叠加会让单次尝试远超单次超时
	return secretsmanager.NewFromConfig(awsCfg, func(o *secretsmanager.Options) { o.RetryMaxAttempts = 1 }), nil
}

// 权限拒绝没有对应的建模错误类型，只能按错误码识别
const errCodeAccessDenied = "AccessDeniedException"

// isNonRetryable 确定性失败：凭据不存在、请求/参数非法、KMS 解密失败、权限拒绝——重试也不会成功，立即失败
func isNonRetryable(err error) bool {
	var nf *types.ResourceNotFoundException
	var ir *types.InvalidRequestException
	var ip *types.InvalidParameterException
	var df *types.DecryptionFailure
	if errors.As(err, &nf) || errors.As(err, &ir) || errors.As(err, &ip) || errors.As(err, &df) {
		return true
	}
	var ae smithy.APIError
	return errors.As(err, &ae) && ae.ErrorCode() == errCodeAccessDenied
}

func init() { secret.Register(New()) }
