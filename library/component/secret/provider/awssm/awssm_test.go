package awssm

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/smithy-go"

	"github.com/coderyw/gomicro-util/library/component/secret"
)

type fakeClient struct {
	out *secretsmanager.GetSecretValueOutput
	err error
	got *secretsmanager.GetSecretValueInput
}

func (f *fakeClient) GetSecretValue(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	f.got = in
	return f.out, f.err
}

func newWith(c Client) *Provider {
	return &Provider{newClient: func(context.Context) (Client, error) { return c, nil }}
}

func TestProvider_Fetch(t *testing.T) {
	t.Run("按SecretId取值返回SecretString", func(t *testing.T) {
		fc := &fakeClient{out: &secretsmanager.GetSecretValueOutput{SecretString: aws.String(`{"a":"1"}`)}}
		got, err := newWith(fc).Fetch(context.Background(), "ok-svc")
		if err != nil || got != `{"a":"1"}` || aws.ToString(fc.got.SecretId) != "ok-svc" {
			t.Fatalf("got=%q err=%v req=%+v", got, err, fc.got)
		}
	})
	t.Run("凭据不存在为不可重试错误", func(t *testing.T) {
		_, err := newWith(&fakeClient{err: &types.ResourceNotFoundException{}}).Fetch(context.Background(), "x")
		if err == nil || secret.IsRetryable(err) {
			t.Fatalf("期望被 NonRetryable 包装，实际 %v", err)
		}
	})
	t.Run("网络类错误保持可重试", func(t *testing.T) {
		_, err := newWith(&fakeClient{err: errors.New("dial timeout")}).Fetch(context.Background(), "x")
		if err == nil || !secret.IsRetryable(err) {
			t.Fatalf("期望原样返回可重试错误，实际 %v", err)
		}
	})
	t.Run("空SecretString视为不可重试错误", func(t *testing.T) {
		_, err := newWith(&fakeClient{out: &secretsmanager.GetSecretValueOutput{}}).Fetch(context.Background(), "x")
		if err == nil || secret.IsRetryable(err) {
			t.Fatalf("期望不可重试错误，实际 %v", err)
		}
	})
	t.Run("客户端初始化失败为可重试错误且不缓存，下次重新初始化", func(t *testing.T) {
		calls := 0
		p := &Provider{newClient: func(context.Context) (Client, error) { calls++; return nil, errors.New("no credentials") }}
		for i := 0; i < 2; i++ {
			if _, err := p.Fetch(context.Background(), "x"); err == nil || !secret.IsRetryable(err) {
				t.Fatalf("期望可重试错误，实际 %v", err)
			}
		}
		if calls != 2 {
			t.Fatalf("期望每次 Fetch 重新初始化共 2 次，实际 %d", calls)
		}
	})
	t.Run("客户端初始化成功后复用", func(t *testing.T) {
		calls := 0
		fc := &fakeClient{out: &secretsmanager.GetSecretValueOutput{SecretString: aws.String("v")}}
		p := &Provider{newClient: func(context.Context) (Client, error) { calls++; return fc, nil }}
		for i := 0; i < 2; i++ {
			if _, err := p.Fetch(context.Background(), "x"); err != nil {
				t.Fatal(err)
			}
		}
		if calls != 1 {
			t.Fatalf("期望只初始化 1 次，实际 %d", calls)
		}
	})
}

func TestProvider_Fetch_确定性失败不重试(t *testing.T) {
	cases := map[string]error{
		"KMS 解密失败": &types.DecryptionFailure{},
		"权限拒绝":     &smithy.GenericAPIError{Code: errCodeAccessDenied, Message: "not authorized"},
	}
	for name, e := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := newWith(&fakeClient{err: e}).Fetch(context.Background(), "x")
			if err == nil || secret.IsRetryable(err) {
				t.Fatalf("期望不可重试错误，实际 %v", err)
			}
		})
	}
	t.Run("其他 API 错误码保持可重试", func(t *testing.T) {
		_, err := newWith(&fakeClient{err: &smithy.GenericAPIError{Code: "ThrottlingException"}}).Fetch(context.Background(), "x")
		if err == nil || !secret.IsRetryable(err) {
			t.Fatalf("期望可重试错误，实际 %v", err)
		}
	})
}

func TestBuildClient_关闭SDK自带重试(t *testing.T) {
	t.Setenv("AWS_REGION", "ap-southeast-1")
	c, err := buildClient(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := c.(*secretsmanager.Client).Options().RetryMaxAttempts; got != 1 {
		t.Fatalf("期望 RetryMaxAttempts=1，实际 %d", got)
	}
}
