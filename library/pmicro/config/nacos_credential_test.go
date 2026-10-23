package config

import (
	"context"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/component/secret"
	"github.com/coderyw/gomicro-util/model/cst"
)

func TestLoadNacosCredential(t *testing.T) {
	secret.Register(stubProvider{data: map[string]string{"ns-uat": `{"nacos_username":"nacos-ns-uat","nacos_password":"real-pwd"}`}})
	secret.ResetCache()
	ctx := context.Background()

	t.Run("账号密码都是引用：从凭据后端取明文", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "stubsm://ns-uat/nacos_username")
		t.Setenv(cst.NACOS_PASSWORD, "stubsm://ns-uat/nacos_password")
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, "")
		c, err := loadNacosCredential(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if c.account != "nacos-ns-uat" || c.password != "real-pwd" || c.accountSource != credentialSourceRef || c.passwordSource != credentialSourceRef {
			t.Fatalf("解析结果不符: %+v", c)
		}
	})
	t.Run("都为空：匿名登录", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "")
		t.Setenv(cst.NACOS_PASSWORD, "")
		c, err := loadNacosCredential(ctx)
		if err != nil || c.account != "" || c.password != "" || c.passwordSource != credentialSourceUnset {
			t.Fatalf("期望匿名，实际 %+v err=%v", c, err)
		}
	})
	t.Run("明文且默认允许：原样使用", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "nacos-ns-test")
		t.Setenv(cst.NACOS_PASSWORD, "plain-pwd")
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, "")
		c, err := loadNacosCredential(ctx)
		if err != nil || c.password != "plain-pwd" || c.passwordSource != credentialSourcePlaintext {
			t.Fatalf("期望明文透传，实际 %+v err=%v", c, err)
		}
	})
	t.Run("明文且开关禁止：启动失败且错误不含密码", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "nacos-ns-test")
		t.Setenv(cst.NACOS_PASSWORD, "plain-pwd")
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, "false")
		_, err := loadNacosCredential(ctx)
		if err == nil {
			t.Fatal("期望报错")
		}
		if strings.Contains(err.Error(), "plain-pwd") {
			t.Fatalf("错误信息泄露密码: %v", err)
		}
	})
	t.Run("账号明文密码引用：开关禁止也放行（只约束密码）", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "nacos-ns-uat")
		t.Setenv(cst.NACOS_PASSWORD, "stubsm://ns-uat/nacos_password")
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, "false")
		c, err := loadNacosCredential(ctx)
		if err != nil || c.password != "real-pwd" || c.accountSource != credentialSourcePlaintext || c.passwordSource != credentialSourceRef {
			t.Fatalf("期望放行，实际 %+v err=%v", c, err)
		}
	})
	t.Run("引用的键不存在：报错且带引用名不带值", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "")
		t.Setenv(cst.NACOS_PASSWORD, "stubsm://ns-uat/no_such_key")
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, "")
		_, err := loadNacosCredential(ctx)
		if err == nil || !strings.Contains(err.Error(), "stubsm://ns-uat/no_such_key") || strings.Contains(err.Error(), "real-pwd") {
			t.Fatalf("期望含引用名不含值的错误，实际 %v", err)
		}
	})
	t.Run("引用的凭据不存在：不可重试错误直接返回", func(t *testing.T) {
		t.Setenv(cst.NACOS_USERNAME, "")
		t.Setenv(cst.NACOS_PASSWORD, "stubsm://ns-missing/nacos_password")
		if _, err := loadNacosCredential(ctx); err == nil {
			t.Fatal("期望报错")
		}
	})
}

func TestNacosAllowPlaintext(t *testing.T) {
	for val, want := range map[string]bool{"": true, "true": true, "1": true, "FALSE": false, "0": false, "no": false, "off": false} {
		t.Setenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT, val)
		if got := nacosAllowPlaintext(); got != want {
			t.Fatalf("值 %q 期望 %v，实际 %v", val, want, got)
		}
	}
}
