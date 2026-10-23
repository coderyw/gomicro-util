package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"

	"github.com/coderyw/gomicro-util/library/component/secret"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cst"
)

// Nacos 账密在 env 里的三种形态
const (
	credentialSourceUnset     = "未设置"
	credentialSourceRef       = "引用"
	credentialSourcePlaintext = "明文"
)

// nacosCredential 登录 Nacos 用的账密及其来源（来源只进日志，不进配置）
type nacosCredential struct {
	account        string
	password       string
	accountSource  string
	passwordSource string
}

// loadNacosCredential 读取 NACOS_USERNAME / NACOS_PASSWORD：
// 值是已注册 scheme 的凭据引用 → 经 secret 组件从凭据后端取明文（失败按组件的分类重试后返回错误，调用方 fail-fast）；
// 空 → 匿名登录（未开鉴权的集群不需要账密）；其余按明文原样使用。
// 明文是迁移期与应急回滚的兜底，只对密码设限：密码为明文时打 WARN 留痕（观测云可按文案审计），
// 且 NACOS_AUTH_ALLOW_PLAINTEXT=false 时直接失败；账号名不是秘密，只记来源。任何路径不打印值
func loadNacosCredential(ctx context.Context) (nacosCredential, error) {
	var c nacosCredential
	var err error
	if c.account, c.accountSource, err = credentialFromEnv(ctx, cst.NACOS_USERNAME); err != nil {
		return nacosCredential{}, err
	}
	if c.password, c.passwordSource, err = credentialFromEnv(ctx, cst.NACOS_PASSWORD); err != nil {
		return nacosCredential{}, err
	}
	if c.password != "" {
		log.RegisterRedactValue(c.password)
	}
	fields := []zap.Field{zap.String("account_source", c.accountSource), zap.String("password_source", c.passwordSource)}
	if c.passwordSource == credentialSourcePlaintext {
		if !nacosAllowPlaintext() {
			return nacosCredential{}, fmt.Errorf("Nacos 密码为明文而 %s=false，请改为凭据引用（如 csms://<凭据名>/<键名>）", cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT)
		}
		log.Warn("[Init] Nacos 账密来源=明文", fields...)
		return c, nil
	}
	log.Info("[Init] Nacos 账密来源", fields...)
	return c, nil
}

// credentialFromEnv 按引用 / 空 / 明文三分支取单个 env 的值；引用解析失败的错误只带引用本身（非秘密），不带值
func credentialFromEnv(ctx context.Context, key string) (value, source string, err error) {
	raw := strings.TrimSpace(os.Getenv(key))
	switch {
	case raw == "":
		return "", credentialSourceUnset, nil
	case secret.IsRef(raw):
		v, e := secret.Resolve(ctx, raw)
		if e != nil {
			return "", credentialSourceRef, fmt.Errorf("解析 %s 的凭据引用 %s 失败: %w", key, raw, e)
		}
		return v, credentialSourceRef, nil
	default:
		return raw, credentialSourcePlaintext, nil
	}
}

// nacosAllowPlaintext 是否允许密码明文：默认允许；false/0/no/off 视为禁止
func nacosAllowPlaintext() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT))) {
	case "false", "0", "no", "off":
		return false
	}
	return true
}
