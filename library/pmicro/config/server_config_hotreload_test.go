package config

import (
	"testing"

	srv_config "github.com/coderyw/gomicro-util/model/config"
)

// 未注册自定义配置（多为无 other 段）的服务收到配置变更时，热更回调不得 panic
func TestReloadCfg_未注册自定义配置不panic(t *testing.T) {
	saved := cc
	cc = nil
	defer func() { cc = saved }()

	s := &ServerCfg{cfg: &srv_config.ServerConfig{}}
	newCfg := &srv_config.ServerConfig{}
	newCfg.Other = nil

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("reloadCfg 不应 panic: %v", r)
		}
	}()
	s.reloadCfg(newCfg)
}
