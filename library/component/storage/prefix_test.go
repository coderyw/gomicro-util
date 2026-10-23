package storage

import (
	"context"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

// ==========================================================
// 对象存储 key 前缀单元测试（不依赖真实 S3）
//
// 覆盖三类场景：
//  1. ctx 携带租户 → key 加租户前缀
//  2. 配置兜底租户（TENANT_FALLBACK）→ 一律不加前缀
//  3. 未配置兜底租户 → 按 ctx 决定（无租户时走 s3 默认 ns）
// ==========================================================

// 正向：ctx 携带租户，key 应加租户前缀
func TestGetPrefix_WithTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")

	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")
	if got := getPrefix(ctx, "demo/a.txt"); got != "tenantok/demo/a.txt" {
		t.Errorf("ctx 携带租户时 key 应为 tenantok/demo/a.txt, got %q", got)
	}
}

// 反向：未配置兜底租户且 ctx 不携带租户，走 s3 默认命名空间 ns
func TestGetPrefix_NoTenantCtx_DefaultNs(t *testing.T) {
	env.SetFallbackTenantName("")

	if got := getPrefix(context.Background(), "demo/a.txt"); got != "tenantns/demo/a.txt" {
		t.Errorf("ctx 无租户时应使用默认 ns 前缀 tenantns/demo/a.txt, got %q", got)
	}
}

// 兜底：配置了兜底租户后，无论 ctx 是否携带租户都不加前缀
func TestGetPrefix_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")
	if got := getPrefix(ctx, "demo/a.txt"); got != "demo/a.txt" {
		t.Errorf("兜底租户模式下 key 不应加前缀, got %q", got)
	}
	if got := getPrefix(context.Background(), "demo/a.txt"); got != "demo/a.txt" {
		t.Errorf("兜底租户模式下（无租户 ctx）key 不应加前缀, got %q", got)
	}
}

func TestGetPrefix_TenantCaseCompatibility(t *testing.T) {
	env.SetFallbackTenantName("")
	for _, ns := range []string{"ok", "OK", "Ok"} {
		ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
		if got := getPrefix(ctx, "Images/Logo.PNG"); got != "tenantok/Images/Logo.PNG" {
			t.Fatal(got)
		}
	}
}
