package local_cache

import (
	"context"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

// ==========================================================
// 本地缓存多租户前缀测试（兜底租户场景）
//
// 常规多租户下不同租户 ctx 的 key 相互隔离（已有用例覆盖）；
// 配置兜底租户（TENANT_FALLBACK）后 key 保持原始命名，不同租户 ctx 访问同一份数据。
// ==========================================================

// 兜底：配置兜底租户后不加前缀，任意租户 ctx 与无租户 ctx 读写的是同一个 key
func TestLocalCache_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	c := NewLocalCache()
	ctxOk := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")

	c.Set(ctxOk, "mykey", "v1", NoExpiration)

	// 无租户 ctx 也能读到（key 未加前缀）
	if v, ok := c.Get(context.Background(), "mykey"); !ok || v != "v1" {
		t.Errorf("兜底租户模式下 key 不应加前缀，无租户 ctx 应能读到, got %v ok=%v", v, ok)
	}
	// 其他租户 ctx 同样命中同一个 key
	ctxOther := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "foo")
	if v, ok := c.Get(ctxOther, "mykey"); !ok || v != "v1" {
		t.Errorf("兜底租户模式下不同租户 ctx 应命中同一 key, got %v ok=%v", v, ok)
	}
	// 底层存储的 key 应为原始 key
	if _, ok := c.Items(false)["mykey"]; !ok {
		t.Errorf("兜底租户模式下底层 key 应为原始 mykey, items=%v", c.Items(false))
	}
}

// 反向：未配置兜底租户，不同租户 ctx 的 key 相互隔离
func TestLocalCache_NoFallback_TenantIsolation(t *testing.T) {
	env.SetFallbackTenantName("")

	c := NewLocalCache()
	ctxOk := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok")
	ctxFoo := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "foo")

	c.Set(ctxOk, "mykey", "v1", NoExpiration)

	if _, ok := c.Get(ctxFoo, "mykey"); ok {
		t.Error("未配置兜底租户时不同租户的 key 应相互隔离")
	}
	if v, ok := c.Get(ctxOk, "mykey"); !ok || v != "v1" {
		t.Errorf("同租户 ctx 应能读到自己的 key, got %v ok=%v", v, ok)
	}
}
