// Package testenv 为依赖外部服务的测试提供连接信息。
//
// 连接地址和账号密码一律从环境变量读取，不写进代码；
// 必需的环境变量未设置时跳过当前测试，避免在没有对应环境时报错。
// 环境变量统一以 GOMICRO_TEST_ 开头，清单见 README「测试」一节。
package testenv

import (
	"os"
	"testing"
)

// Get 读取环境变量，未设置时返回 def
func Get(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Require 按顺序返回环境变量的值；任一未设置时跳过当前测试
func Require(t testing.TB, keys ...string) []string {
	t.Helper()
	vals := make([]string, len(keys))
	for i, k := range keys {
		v := os.Getenv(k)
		if v == "" {
			t.Skipf("跳过：未设置环境变量 %s", k)
		}
		vals[i] = v
	}
	return vals
}

// DB 测试数据库的连接信息
type DB struct {
	Host     string
	User     string
	Password string
}

// DBConn 读取 GOMICRO_TEST_<name>_HOST / _USER / _PASSWORD；任一未设置时跳过当前测试
func DBConn(t testing.TB, name string) DB {
	t.Helper()
	v := Require(t, "GOMICRO_TEST_"+name+"_HOST", "GOMICRO_TEST_"+name+"_USER", "GOMICRO_TEST_"+name+"_PASSWORD")
	return DB{Host: v[0], User: v[1], Password: v[2]}
}
