package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/component/secret"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
)

type stubProvider struct{ data map[string]string }

func (s stubProvider) Scheme() string { return "stubsm" }
func (s stubProvider) Fetch(_ context.Context, name string) (string, error) {
	v, ok := s.data[name]
	if !ok {
		return "", secret.NonRetryable(errors.New("not found"))
	}
	return v, nil
}

type otherCfg struct {
	ApiKey string `json:"api_key"`
}

func writeCfg(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.yaml")
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// 本地 cfg.yaml 路径：ReadFile → ResolveText → yaml 反序列化 三段都要覆盖
func TestInitFromCfgFile(t *testing.T) {
	secret.Register(stubProvider{data: map[string]string{"ns-demo": `{"salt":"real-salt","api_key":"real-key"}`}})
	secret.ResetCache()
	saved := cc
	defer func() { cc = saved }()

	t.Run("本地配置中的凭据引用被替换，other 段映射到自定义配置", func(t *testing.T) {
		if err := initCustomerConfig(otherCfg{}, nil); err != nil {
			t.Fatal(err)
		}
		s := &ServerCfg{defaultCustomerCfg: otherCfg{}}
		opt := &option.Options{ConfigFile: writeCfg(t, "platform: ns\ntoken_salt: 'stubsm://ns-demo/salt'\nother:\n  api_key: 'stubsm://ns-demo/api_key'\n")}
		if err := s.initFromCfgFile(opt); err != nil {
			t.Fatal(err)
		}
		oc, _ := GetCustomerConfig().(otherCfg)
		if s.cfg.TokenSalt != "real-salt" || s.cfg.Platform != "ns" || oc.ApiKey != "real-key" {
			t.Fatalf("解析结果不符: salt=%q platform=%q api_key=%q", s.cfg.TokenSalt, s.cfg.Platform, oc.ApiKey)
		}
	})
	t.Run("明文配置原样透传", func(t *testing.T) {
		s := &ServerCfg{}
		if err := s.initFromCfgFile(&option.Options{ConfigFile: writeCfg(t, "token_salt: 'plain'\n")}); err != nil || s.cfg.TokenSalt != "plain" {
			t.Fatalf("透传失败: %q %v", s.cfg.TokenSalt, err)
		}
	})
	t.Run("引用解析失败返回错误", func(t *testing.T) {
		s := &ServerCfg{}
		if err := s.initFromCfgFile(&option.Options{ConfigFile: writeCfg(t, "token_salt: 'stubsm://ns-demo/no_such'\n")}); err == nil {
			t.Fatal("期望报错")
		}
	})
	t.Run("文件不存在返回错误", func(t *testing.T) {
		s := &ServerCfg{}
		if err := s.initFromCfgFile(&option.Options{ConfigFile: "/nonexistent/cfg.yaml"}); err == nil {
			t.Fatal("期望报错")
		}
	})
}

func TestInitFromCfgFile_类型错误不泄露凭据明文(t *testing.T) {
	secret.Register(stubProvider{data: map[string]string{"ns-leak": `{"salt":"real-salt-value"}`}})
	secret.ResetCache()
	s := &ServerCfg{}
	// 引用替换成明文后落到 int 字段，yaml 会报类型错误；错误信息与日志都不能带出凭据值
	err := s.initFromCfgFile(&option.Options{ConfigFile: writeCfg(t, "metrics:\n  interval_s: 'stubsm://ns-leak/salt'\n")})
	if err == nil || strings.Contains(err.Error(), "real-salt-value") {
		t.Fatalf("错误应存在且不含凭据明文，实际 %v", err)
	}
}
