package local_cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

func ctxOfTenant(ns string) context.Context {
	return metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
}

// 租户 A 的条目不应出现在租户 B 的视图里
func TestItemsOnCtx_isolatesTenants(t *testing.T) {
	c := NewLocalCache()
	ctxA, ctxB := ctxOfTenant("ta"), ctxOfTenant("tb")

	c.Set(ctxA, "1", "a1", NoExpiration)
	c.Set(ctxA, "2", "a2", NoExpiration)
	c.Set(ctxB, "1", "b1", NoExpiration)

	itemsA := c.ItemsOnCtx(ctxA)
	assert.Len(t, itemsA, 2)
	// key 已剥掉租户前缀, 且拿到的是本租户的值
	assert.Equal(t, "a1", itemsA["1"].Object)
	assert.Equal(t, "a2", itemsA["2"].Object)

	itemsB := c.ItemsOnCtx(ctxB)
	assert.Len(t, itemsB, 1)
	assert.Equal(t, "b1", itemsB["1"].Object)
}

// ItemsOnCtx 返回的 key 可以直接回传给 Set/Delete
func TestItemsOnCtx_keyRoundTrip(t *testing.T) {
	c := NewLocalCache()
	ctxA := ctxOfTenant("ta")
	c.Set(ctxA, "1", "old", NoExpiration)

	for k := range c.ItemsOnCtx(ctxA) {
		c.Set(ctxA, k, "new", NoExpiration)
	}

	v, ok := c.Get(ctxA, "1")
	assert.True(t, ok)
	assert.Equal(t, "new", v)
}

// FlushOnCtx 只清当前租户, 不能波及其他租户
func TestFlushOnCtx_onlyCurrentTenant(t *testing.T) {
	c := NewLocalCache()
	ctxA, ctxB := ctxOfTenant("ta"), ctxOfTenant("tb")

	c.Set(ctxA, "1", "a1", NoExpiration)
	c.Set(ctxB, "1", "b1", NoExpiration)
	c.Set(ctxB, "2", "b2", NoExpiration)

	c.FlushOnCtx(ctxA)

	_, ok := c.Get(ctxA, "1")
	assert.False(t, ok, "租户 A 的条目应已清空")
	assert.Len(t, c.ItemsOnCtx(ctxB), 2, "租户 B 的条目不应被波及")
}

// 前缀相似的租户名不能互相误伤 (t / t2)
func TestItemsOnCtx_similarTenantNames(t *testing.T) {
	c := NewLocalCache()
	ctx1, ctx2 := ctxOfTenant("t"), ctxOfTenant("t2")

	c.Set(ctx1, "k", "v1", NoExpiration)
	c.Set(ctx2, "k", "v2", NoExpiration)

	// 前缀分别是 "tenantt:" 和 "tenantt2:", 分隔符保证不会互相命中
	items1 := c.ItemsOnCtx(ctx1)
	assert.Len(t, items1, 1)
	assert.Equal(t, "v1", items1["k"].Object)

	c.FlushOnCtx(ctx1)
	assert.Len(t, c.ItemsOnCtx(ctx2), 1, "t2 不应被 t 的 Flush 清掉")
}

// 无 namespace 的 ctx 不能碰到任何带租户前缀的条目
func TestItemsOnCtx_missingNamespaceSkipsTenantKeys(t *testing.T) {
	c := NewLocalCache()
	ctxA := ctxOfTenant("ta")
	c.Set(ctxA, "1", "a1", NoExpiration)

	bare := context.Background()
	assert.Empty(t, c.ItemsOnCtx(bare), "带租户前缀的条目不应出现在无租户作用域里")

	c.FlushOnCtx(bare)
	assert.Len(t, c.ItemsOnCtx(ctxA), 1, "无 namespace 的 Flush 不能清掉任何租户")
}

// 未配置多租户的部署(所有 key 都是裸 key)必须仍能正常批量操作,
// 否则会比多租户改造前更差
func TestItemsOnCtx_bareKeysStillWorkWithoutNamespace(t *testing.T) {
	c := NewLocalCache()
	bare := context.Background()

	c.Set(bare, "1", "v1", NoExpiration)
	c.Set(bare, "2", "v2", NoExpiration)

	items := c.ItemsOnCtx(bare)
	assert.Len(t, items, 2, "裸 key 必须能被无 namespace 的 ctx 遍历到")
	assert.Equal(t, "v1", items["1"].Object)

	c.FlushOnCtx(bare)
	assert.Empty(t, c.ItemsOnCtx(bare), "裸 key 必须能被清空")
}

// 裸 key 与租户 key 共存时, 两个作用域互不干扰
func TestItemsOnCtx_bareAndTenantKeysCoexist(t *testing.T) {
	c := NewLocalCache()
	bare := context.Background()
	ctxA := ctxOfTenant("ta")

	c.Set(bare, "1", "bare", NoExpiration)
	c.Set(ctxA, "1", "tenant", NoExpiration)

	assert.Len(t, c.ItemsOnCtx(bare), 1)
	assert.Equal(t, "bare", c.ItemsOnCtx(bare)["1"].Object)
	assert.Equal(t, "tenant", c.ItemsOnCtx(ctxA)["1"].Object)

	// 清裸 key 作用域, 租户数据必须原样保留
	c.FlushOnCtx(bare)
	assert.Empty(t, c.ItemsOnCtx(bare))
	assert.Len(t, c.ItemsOnCtx(ctxA), 1, "清裸 key 不应波及租户数据")
}

// 兜底租户(NS 单租户部署)下不加前缀, 行为必须与改造前一致
func TestItemsOnCtx_fallbackTenantBehavesAsSingleTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	c := NewLocalCache()
	ctx := context.Background()

	c.Set(ctx, "1", "v1", NoExpiration)
	c.Set(ctx, "2", "v2", NoExpiration)

	items := c.ItemsOnCtx(ctx)
	assert.Len(t, items, 2)
	assert.Equal(t, "v1", items["1"].Object)

	c.FlushOnCtx(ctx)
	assert.Empty(t, c.ItemsOnCtx(ctx))
}

// 并发读写下不应出现 data race (配合 -race)
func TestItemsOnCtx_concurrent(t *testing.T) {
	c := NewLocalCache()
	ctxA, ctxB := ctxOfTenant("ta"), ctxOfTenant("tb")
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			c.Set(ctxB, "k", i, time.Minute)
		}
	}()
	for i := 0; i < 200; i++ {
		_ = c.ItemsOnCtx(ctxA)
	}
	<-done
}

func TestTenantCaseCacheCompatibility(t *testing.T) {
	env.SetFallbackTenantName("")
	c := NewLocalCache()
	c.Set(ctxOfTenant("ok"), "UserID", "existing", NoExpiration)
	v, ok := c.Get(ctxOfTenant("OK"), "UserID")
	if !ok || v != "existing" {
		t.Fatalf("existing cache entry inaccessible: %v %v", v, ok)
	}
	if _, ok := c.Get(ctxOfTenant("OK"), "userid"); ok {
		t.Fatal("business key case changed")
	}
	if _, ok := c.Get(ctxOfTenant("other"), "UserID"); ok {
		t.Fatal("tenant isolation lost")
	}
	c.FlushOnCtx(ctxOfTenant("OK"))
	if _, ok := c.Get(ctxOfTenant("ok"), "UserID"); ok {
		t.Fatal("flush used a different prefix")
	}
}
