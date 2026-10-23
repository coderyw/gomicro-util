package namespace

import (
	"context"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
)

func TestGetOriginKey(t *testing.T) {
	type args struct {
		t        NamespaceFmt
		resetKey string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ok_ca_user",
		}, want: "ca_user"},
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ca-user",
		}, want: "ca-user"},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "tenantok:ca_user",
		}, want: "ca_user"},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "ok:ca_user",
		}, want: "ok:ca_user"},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "tenantok_ca_user",
		}, want: "ca_user"},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "ok_ca_user",
		}, want: "ok_ca_user"},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "tenantok:ca_user",
		}, want: "ca_user"},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "ok:ca_user",
		}, want: "ok:ca_user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOriginKey(tt.args.t, tt.args.resetKey); got != tt.want {
				t.Errorf("GetOriginKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseOriginKey(t *testing.T) {
	type args struct {
		t        NamespaceFmt
		resetKey string
	}
	tests := []struct {
		name  string
		args  args
		want1 string
		want2 string
	}{
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ok_ca_user",
		}, want1: "ok", want2: "ca_user"},
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ca-user",
		}, want1: "", want2: "ca-user"},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "tenantok:ca_user",
		}, want1: "ok", want2: "ca_user"},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "ok:ca_user",
		}, want1: "", want2: "ok:ca_user"},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "tenantok_ca_user",
		}, want1: "ok", want2: "ca_user"},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "ok_ca_user",
		}, want1: "", want2: "ok_ca_user"},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "tenantok:ca_user",
		}, want1: "ok", want2: "ca_user"},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "ok:ca_user",
		}, want1: "", want2: "ok:ca_user"},
		{name: "oss", args: args{
			t:        NAMESPACE_TYPE_OSS,
			resetKey: "tenantok/ca_user",
		}, want1: "ok", want2: "ca_user"},
		{name: "oss", args: args{
			t:        NAMESPACE_TYPE_OSS,
			resetKey: "ca_user",
		}, want1: "", want2: "ca_user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got1, got2 := parseOriginKey(tt.args.t, tt.args.resetKey); got1 != tt.want1 || got2 != tt.want2 {
				t.Errorf("GetOriginKey() = %v,%v, want %v,%v", got1, got2, tt.want1, tt.want2)
			}
		})
	}
}

func TestGetNameSpaceFromKey(t *testing.T) {
	type args struct {
		t        NamespaceFmt
		resetKey string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ok_ca_user",
		}, want: "ok"},
		{name: "mysql", args: args{
			t:        NAMESPACE_TYPE_MYSQL,
			resetKey: "ca-user",
		}, want: ""},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "tenantok:ca_user",
		}, want: "ok"},
		{name: "redis", args: args{
			t:        NAMESPACE_TYPE_REDIS,
			resetKey: "user:ca_user",
		}, want: ""},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "tenantok_ca_user",
		}, want: "ok"},
		{name: "kafka", args: args{
			t:        NAMESPACE_TYPE_KAFKA,
			resetKey: "topic_ca_user",
		}, want: ""},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "tenantok:ca_user",
		}, want: "ok"},
		{name: "common", args: args{
			t:        NAMESPACE_TYPE_COMMON,
			resetKey: "ca_user",
		}, want: ""},
		{name: "oss1", args: args{
			t:        NAMESPACE_TYPE_OSS,
			resetKey: "tenantok/ca_user",
		}, want: "ok"},
		{name: "oss2", args: args{
			t:        NAMESPACE_TYPE_OSS,
			resetKey: "ca_user",
		}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNameSpaceFromKey(tt.args.t, tt.args.resetKey); got != tt.want {
				t.Errorf("GetNameSpaceFromKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFmtByType(t *testing.T) {
	type args struct {
		t      NamespaceFmt
		nsKey  string
		fmtKey string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "mysql", args: args{
			t:      NAMESPACE_TYPE_MYSQL,
			nsKey:  "ok",
			fmtKey: "ca_user",
		}, want: "ok_ca_user"},
		{name: "mysql no nsKey", args: args{
			t:      NAMESPACE_TYPE_MYSQL,
			nsKey:  "",
			fmtKey: "ca_user",
		}, want: "ca_user"},
		{name: "redis", args: args{
			t:      NAMESPACE_TYPE_REDIS,
			nsKey:  "ok",
			fmtKey: "ca_user",
		}, want: "tenantok:ca_user"},
		{name: "redis  no nsKey", args: args{
			t:      NAMESPACE_TYPE_REDIS,
			nsKey:  "",
			fmtKey: "ca_user",
		}, want: "ca_user"},
		{name: "kafka", args: args{
			t:      NAMESPACE_TYPE_KAFKA,
			nsKey:  "ok",
			fmtKey: "ca_user",
		}, want: "tenantok_ca_user"},
		{name: "kafka  no nsKey", args: args{
			t:      NAMESPACE_TYPE_KAFKA,
			nsKey:  "",
			fmtKey: "ca_user",
		}, want: "ca_user"},
		{name: "common", args: args{
			t:      NAMESPACE_TYPE_COMMON,
			nsKey:  "ok",
			fmtKey: "ca_user",
		}, want: "tenantok:ca_user"},
		{name: "common  no nsKey", args: args{
			t:      NAMESPACE_TYPE_COMMON,
			nsKey:  "",
			fmtKey: "ca_user",
		}, want: "ca_user"},
		{name: "oss", args: args{
			t:      NAMESPACE_TYPE_OSS,
			nsKey:  "ok",
			fmtKey: "ca_user",
		}, want: "tenantok/ca_user"},
		{name: "oss  no nsKey", args: args{
			t:      NAMESPACE_TYPE_OSS,
			nsKey:  "",
			fmtKey: "ca_user",
		}, want: "ca_user"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FmtByType(tt.args.t, tt.args.nsKey, tt.args.fmtKey); got != tt.want {
				t.Errorf("FmtByType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test_FmtByType_FallbackTenant 验证：配置了兜底租户（TENANT_FALLBACK）后，
// FmtByType 对所有类型一律返回原始 key（不加租户前缀）
func Test_FmtByType_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	types := []struct {
		name string
		t    NamespaceFmt
	}{
		{"redis", NAMESPACE_TYPE_REDIS},
		{"mysql", NAMESPACE_TYPE_MYSQL},
		{"kafka", NAMESPACE_TYPE_KAFKA},
		{"common", NAMESPACE_TYPE_COMMON},
		{"oss", NAMESPACE_TYPE_OSS},
	}
	for _, tt := range types {
		if got := FmtByType(tt.t, "ok", "mykey"); got != "mykey" {
			t.Errorf("FmtByType(%s) 配置兜底租户后应返回原始 key mykey, got %q", tt.name, got)
		}
	}
}

// Test_FmtByTypeOnCtx_FallbackTenant 验证 FmtByTypeOnCtx 在兜底租户下的行为：
// 即使带了 defaultNs（s3 实际调用形态 FmtByTypeOnCtx(ctx, OSS, key, "ns")），配置兜底后也返回原始 key。
func Test_FmtByTypeOnCtx_FallbackTenant(t *testing.T) {
	defer env.SetFallbackTenantName("")

	// 未配置兜底：空 ctx 下 defaultNs="ns" 会被用于加前缀（s3 默认行为）
	env.SetFallbackTenantName("")
	if got := FmtByTypeOnCtx(context.Background(), NAMESPACE_TYPE_OSS, "mykey", "ns"); got != "tenantns/mykey" {
		t.Errorf("未配置兜底时 FmtByTypeOnCtx(OSS, defaultNs=ns) 应加前缀 tenantns/mykey, got %q", got)
	}

	// 配置兜底：即使带 defaultNs="ns" 也返回原始 key（不加前缀）
	env.SetFallbackTenantName("ns")
	if got := FmtByTypeOnCtx(context.Background(), NAMESPACE_TYPE_OSS, "mykey", "ns"); got != "mykey" {
		t.Errorf("配置兜底后 FmtByTypeOnCtx(OSS, defaultNs=ns) 应返回原始 key mykey, got %q", got)
	}
}
