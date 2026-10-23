package kafka

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/namespace"
)

// ==========================================================
// 多租户 topic 前缀单元测试（生产端 formatTopic / 消费端 makeTopics）
//
// 覆盖三类场景：
//  1. ctx 携带租户 → topic 加租户前缀
//  2. 配置兜底租户（TENANT_FALLBACK）→ 一律不加前缀 / 精确订阅
//  3. 未配置兜底租户 → 按 ctx 是否携带租户决定
// ==========================================================

func tenantCtx(ns string) context.Context {
	return metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
}

// 正向：ctx 携带租户，生产端 topic 应加租户前缀
func TestFormatTopic_WithTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")
	if got := formatTopic(tenantCtx("ok"), "order_created"); got != "tenantok_order_created" {
		t.Errorf("ctx 携带租户时 topic 应加前缀 tenantok_order_created, got %q", got)
	}
}

// 反向：未配置兜底租户且 ctx 不携带租户，topic 保持原样
func TestFormatTopic_NoTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")
	if got := formatTopic(context.Background(), "order_created"); got != "order_created" {
		t.Errorf("ctx 无租户时 topic 应保持原样 order_created, got %q", got)
	}
}

// 兜底：配置了兜底租户后，即使 ctx 携带租户也不加前缀
func TestFormatTopic_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	if got := formatTopic(tenantCtx("ok"), "order_created"); got != "order_created" {
		t.Errorf("兜底租户模式下 topic 不应加前缀, got %q", got)
	}
	// ctx 无租户同样保持原样
	if got := formatTopic(context.Background(), "order_created"); got != "order_created" {
		t.Errorf("兜底租户模式下（无租户 ctx）topic 不应加前缀, got %q", got)
	}
}

// 正向：常规多租户模式，消费端按已加载租户列表精确订阅（不再使用正则后缀匹配，
// 避免误订同集群上其他命名空间租户的 topic）；租户明细矩阵见 TestMakeTopicsForTenants
func TestMakeTopics_Normal(t *testing.T) {
	env.SetFallbackTenantName("")
	cm := &ConfluentConsumerManager{}
	got := cm.makeTopics("order_created")
	// 与当前租户列表推导的精确列表一致，验证 makeTopics 的租户数据源接线
	want := makeTopicsForTenants("order_created", namespace.GetAllTenantsInfo())
	if !slices.Equal(got, want) {
		t.Errorf("makeTopics() = %v, want %v", got, want)
	}
	for _, s := range got {
		if strings.HasPrefix(s, "^") {
			t.Errorf("不应再返回正则订阅项: %v", s)
		}
	}
}

// 兜底：配置兜底租户后，消费端只精确订阅原始 topic，避免消费到其他租户的前缀 topic
func TestMakeTopics_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	cm := &ConfluentConsumerManager{}
	got := cm.makeTopics("order_created")
	if len(got) != 1 || got[0] != "order_created" {
		t.Errorf("兜底租户模式应精确订阅 order_created, got %v", got)
	}
}
