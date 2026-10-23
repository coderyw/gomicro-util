package pmicro

import (
	"runtime/debug"
	"testing"

	"go.uber.org/zap"
)

func fieldMap(fields []zap.Field) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		m[f.Key] = f.String
	}
	return m
}

func TestBuildInfoFields(t *testing.T) {
	t.Run("挑出 go 版本、vcs 字段与 common-util 依赖版本", func(t *testing.T) {
		bi := &debug.BuildInfo{
			GoVersion: "go1.25.0",
			Settings: []debug.BuildSetting{
				{Key: "-ldflags", Value: "-X main.token=should-not-log"},
				{Key: "GOFLAGS", Value: "-mod=vendor"},
				{Key: "vcs.revision", Value: "976aceadf2c14b7e"},
				{Key: "vcs.time", Value: "2026-09-22T02:00:00Z"},
				{Key: "vcs.modified", Value: "false"},
			},
			Deps: []*debug.Module{
				{Path: "go.uber.org/zap", Version: "v1.27.0"},
				{Path: commonUtilModulePath, Version: "v1.5.0-rc.3"},
			},
		}
		got := fieldMap(buildInfoFields(bi))
		want := map[string]string{
			"go": "go1.25.0", "vcs.revision": "976aceadf2c14b7e", "vcs.time": "2026-09-22T02:00:00Z",
			"vcs.modified": "false", "common_util": "v1.5.0-rc.3",
		}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("字段 %s 期望 %q 实际 %q", k, v, got[k])
			}
		}
		for _, k := range []string{"-ldflags", "GOFLAGS", "go.uber.org/zap"} {
			if _, ok := got[k]; ok {
				t.Errorf("非白名单字段 %s 不应出现", k)
			}
		}
		if len(got) != len(want) {
			t.Errorf("期望 %d 个字段，实际 %d：%v", len(want), len(got), got)
		}
	})
	t.Run("replace 到本地目录时另打 replace 目标", func(t *testing.T) {
		bi := &debug.BuildInfo{
			GoVersion: "go1.25.0",
			Deps: []*debug.Module{{
				Path: commonUtilModulePath, Version: "v1.5.0-rc.2",
				Replace: &debug.Module{Path: "../ca-common-util", Version: "(devel)"},
			}},
		}
		got := fieldMap(buildInfoFields(bi))
		if got["common_util"] != "v1.5.0-rc.2" || got["common_util_replace"] != "../ca-common-util@(devel)" {
			t.Errorf("replace 信息不符：%v", got)
		}
	})
	t.Run("无 vcs 信息与无依赖时只有 go 版本", func(t *testing.T) {
		got := fieldMap(buildInfoFields(&debug.BuildInfo{GoVersion: "go1.23.3"}))
		if len(got) != 1 || got["go"] != "go1.23.3" {
			t.Errorf("期望只有 go 字段，实际 %v", got)
		}
	})
}
