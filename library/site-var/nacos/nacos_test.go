package nacos

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/component/secret"
	"github.com/coderyw/gomicro-util/model/config"
)

func TestSyncNacos(t *testing.T) {
	type args struct {
		cfg *config.RegistryCfg
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "测试链接", args: args{cfg: &config.RegistryCfg{
			Type: "nacos",
			Hosts: []string{
				"dev-nacos-internal-NLB-664ef66d616ee60a.elb.ap-southeast-1.amazonaws.com:8848",
			},
			Account:     "nacos",
			Pwd:         "hU6csGhudotd0HBCqzrW",
			NameId:      "",
			EnableIstio: false,
		}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SyncNacos(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Fatalf("SyncNacos() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := GetNacosCfgData("test-config", "DEFAULT_GROUP", nil); err != nil {
				t.Fatalf("GetNacosCfgData() error = %v", err)
			}
		})
	}
}

type stubProvider struct{ data map[string]string }

func (s stubProvider) Scheme() string { return "stubsm" }
func (s stubProvider) Fetch(_ context.Context, name string) (string, error) {
	v, ok := s.data[name]
	if !ok {
		return "", secret.NonRetryable(errors.New("not found"))
	}
	return v, nil
}

type demoCfg struct {
	Storage struct {
		Mysql struct {
			Host     string `yaml:"host" json:"host"`
			Password string `yaml:"password" json:"password"`
		} `yaml:"mysql" json:"mysql"`
	} `yaml:"storage" json:"storage"`
}

func TestDecodeConfig(t *testing.T) {
	secret.Register(stubProvider{data: map[string]string{"ns-demo": `{"mysql_password":"real-pw"}`}})
	secret.ResetCache()

	t.Run("yaml配置中的凭据引用在反序列化前被替换", func(t *testing.T) {
		var c demoCfg
		yamlText := "storage:\n  mysql:\n    host: '1.2.3.4'\n    password: 'stubsm://ns-demo/mysql_password'\n"
		if err := decodeConfig("svc.yaml", yamlText, &c); err != nil {
			t.Fatal(err)
		}
		if c.Storage.Mysql.Password != "real-pw" || c.Storage.Mysql.Host != "1.2.3.4" {
			t.Fatalf("解析结果不符: %+v", c)
		}
	})
	t.Run("json配置同样支持", func(t *testing.T) {
		var c demoCfg
		if err := decodeConfig("svc.json", `{"storage":{"mysql":{"host":"h","password":"stubsm://ns-demo/mysql_password"}}}`, &c); err != nil {
			t.Fatal(err)
		}
		if c.Storage.Mysql.Password != "real-pw" {
			t.Fatalf("解析结果不符: %+v", c)
		}
	})
	t.Run("明文配置原样透传", func(t *testing.T) {
		var c demoCfg
		if err := decodeConfig("svc.yaml", "storage:\n  mysql:\n    password: 'plain'\n", &c); err != nil || c.Storage.Mysql.Password != "plain" {
			t.Fatalf("透传失败: %+v %v", c, err)
		}
	})
	t.Run("引用解析失败返回错误且不反序列化", func(t *testing.T) {
		var c demoCfg
		err := decodeConfig("svc.yaml", "storage:\n  mysql:\n    password: 'stubsm://ns-demo/no_such_key'\n", &c)
		if err == nil || c.Storage.Mysql.Password != "" {
			t.Fatalf("期望报错且不写入: %+v %v", c, err)
		}
	})
}

func TestDecodeConfig_类型错误不泄露凭据明文(t *testing.T) {
	secret.Register(stubProvider{data: map[string]string{"ns-leak": `{"password":"real-pw-value"}`}})
	secret.ResetCache()
	var c struct {
		Port int `yaml:"port" json:"port"`
	}
	t.Run("yaml：引用替换成明文后落到 int 字段的类型错误不含明文", func(t *testing.T) {
		err := decodeConfig("svc.yaml", "port: stubsm://ns-leak/password\n", &c)
		if err == nil || strings.Contains(err.Error(), "real-pw-value") {
			t.Fatalf("错误应存在且不含凭据明文，实际 %v", err)
		}
	})
	t.Run("json：同样不含明文", func(t *testing.T) {
		err := decodeConfig("svc.json", `{"port": "stubsm://ns-leak/password"}`, &c)
		if err == nil || strings.Contains(err.Error(), "real-pw-value") {
			t.Fatalf("错误应存在且不含凭据明文，实际 %v", err)
		}
	})
}
