package csms

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdkerr"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/services/csms/v1/model"

	"github.com/coderyw/gomicro-util/library/component/secret"
)

type fakeClient struct {
	resp *model.ShowSecretVersionResponse
	err  error
	got  *model.ShowSecretVersionRequest
}

func (f *fakeClient) ShowSecretVersion(req *model.ShowSecretVersionRequest) (*model.ShowSecretVersionResponse, error) {
	f.got = req
	return f.resp, f.err
}

func newWith(c Client, initErr error) *Provider {
	return &Provider{newClient: func() (Client, error) { return c, initErr }}
}

func strp(s string) *string { return &s }

func TestProvider_Fetch(t *testing.T) {
	t.Run("按凭据名取最新版本并返回SecretString", func(t *testing.T) {
		fc := &fakeClient{resp: &model.ShowSecretVersionResponse{Version: &model.Version{SecretString: strp(`{"a":"1"}`)}}}
		p := newWith(fc, nil)
		got, err := p.Fetch(context.Background(), "ns-svc")
		if err != nil || got != `{"a":"1"}` {
			t.Fatalf("got=%q err=%v", got, err)
		}
		if fc.got.SecretName != "ns-svc" || fc.got.VersionId != latestVersion {
			t.Fatalf("请求参数不符: %+v", fc.got)
		}
	})
	t.Run("凭据不存在(404)为不可重试错误", func(t *testing.T) {
		p := newWith(&fakeClient{err: &sdkerr.ServiceResponseError{StatusCode: 404, ErrorMessage: "not found"}}, nil)
		_, err := p.Fetch(context.Background(), "x")
		if err == nil || !isNonRetryable(err) {
			t.Fatalf("期望不可重试错误，实际 %v", err)
		}
	})
	t.Run("网络类错误保持可重试", func(t *testing.T) {
		p := newWith(&fakeClient{err: errors.New("dial timeout")}, nil)
		_, err := p.Fetch(context.Background(), "x")
		if err == nil || isNonRetryable(err) {
			t.Fatalf("期望可重试错误，实际 %v", err)
		}
	})
	t.Run("空SecretString视为不可重试错误", func(t *testing.T) {
		p := newWith(&fakeClient{resp: &model.ShowSecretVersionResponse{Version: &model.Version{}}}, nil)
		if _, err := p.Fetch(context.Background(), "x"); err == nil || !isNonRetryable(err) {
			t.Fatalf("期望不可重试错误，实际 %v", err)
		}
	})
	t.Run("客户端初始化失败为可重试错误且不缓存，下次重新初始化", func(t *testing.T) {
		calls := 0
		p := &Provider{newClient: func() (Client, error) { calls++; return nil, errors.New("no credentials") }}
		for i := 0; i < 2; i++ {
			if _, err := p.Fetch(context.Background(), "x"); err == nil || isNonRetryable(err) {
				t.Fatalf("期望可重试错误（sidecar 未就绪导致的凭据链超时须退避重试），实际 %v", err)
			}
		}
		if calls != 2 {
			t.Fatalf("期望每次 Fetch 重新初始化共 2 次，实际 %d", calls)
		}
	})
	t.Run("region 非法为不可重试错误", func(t *testing.T) {
		t.Setenv(envRegion, "no-such-region")
		if _, err := buildClient(); err == nil || !isNonRetryable(err) {
			t.Fatalf("期望不可重试错误，实际 %v", err)
		}
	})
	t.Run("客户端初始化成功后复用", func(t *testing.T) {
		calls := 0
		fc := &fakeClient{resp: &model.ShowSecretVersionResponse{Version: &model.Version{SecretString: strp("v")}}}
		p := &Provider{newClient: func() (Client, error) { calls++; return fc, nil }}
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

func isNonRetryable(err error) bool { return !secret.IsRetryable(err) }

// stallClient 模拟连接建立后响应停滞：直到 release 关闭才返回
type stallClient struct{ release chan struct{} }

func (s *stallClient) ShowSecretVersion(*model.ShowSecretVersionRequest) (*model.ShowSecretVersionResponse, error) {
	<-s.release
	return nil, errors.New("released")
}

func TestProvider_Fetch_ctx结束立即返回(t *testing.T) {
	sc := &stallClient{release: make(chan struct{})}
	defer close(sc.release)
	p := newWith(sc, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := p.Fetch(ctx, "x")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("期望返回 ctx 超时错误，实际 %v", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("SDK 调用停滞时 Fetch 应随 ctx 返回，实际耗时 %v", elapsed)
	}
	if !secret.IsRetryable(err) {
		t.Fatal("单次超时应为可重试错误")
	}
}
