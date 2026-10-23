package config

import (
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap/zapcore"
)

// 启动日志会 dump 整份配置，所有密码/密钥字段必须已脱敏（规范：日志禁止打印敏感凭证）
func TestServerConfig_MarshalLogObject_敏感字段脱敏(t *testing.T) {
	secrets := map[string]string{
		"registry.pwd":     "Playmate@123456",
		"mysql.password":   "mysql-secret-pw-1",
		"mysql.replica.pw": "replica-db-secret-9",
		"redis.password":   "redis-secret-pw-2",
		"http.jwt_secret":  "jwt-secret-value-3",
		"token_salt(不应输出)": "token-salt-plain-4",
	}
	cfg := ServerConfig{
		TokenSalt: secrets["token_salt(不应输出)"],
		Http:      HttpCfg{Host: ":8080", JwtSecret: secrets["http.jwt_secret"]},
		Registry:  RegistryCfg{Type: "nacos", Hosts: []string{"h:8848"}, Account: "nacos-ns-test", Pwd: secrets["registry.pwd"]},
	}
	primary := DbInstanceConfig{
		DbBaseInstanceConfig: DbBaseInstanceConfig{User: "u", Password: secrets["mysql.password"]},
		Replicas: []DbReplicas{{
			DbBaseInstanceConfig: DbBaseInstanceConfig{User: "u", Password: secrets["mysql.replica.pw"]},
		}},
	}
	cfg.Storage.Mysql.Default = "core"
	cfg.Storage.Mysql.Instance = map[string]DbInstanceConfig{"core": primary}
	cfg.Storage.Redis.Instance = map[string]RedisInstanceConfig{"core": {Host: []string{"r:6379"}, Password: secrets["redis.password"]}}

	enc := zapcore.NewMapObjectEncoder()
	if err := cfg.MarshalLogObject(enc); err != nil {
		t.Fatal(err)
	}
	dump := fmt.Sprintf("%v", enc.Fields)
	for field, plain := range secrets {
		if strings.Contains(dump, plain) {
			t.Errorf("%s 的明文出现在启动日志中: %s", field, plain)
		}
	}
	if !strings.Contains(dump, "nacos-ns-test") {
		t.Error("账号名（非敏感）应保留可见以便排障")
	}
	if !strings.Contains(dump, "***") {
		t.Error("期望出现脱敏占位 ***")
	}
}
