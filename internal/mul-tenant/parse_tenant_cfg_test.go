package mul_tenant

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/model/tenant"
)

// snapshotGlobals 保存全局租户列表与兜底租户配置，用例结束后恢复
func snapshotGlobals(t *testing.T) {
	t.Helper()
	lock.RLock()
	oldTenants := tenants
	lock.RUnlock()
	oldFallback := os.Getenv(cst.ENV_TENANT_FALLBACK)
	t.Cleanup(func() {
		lock.Lock()
		tenants = oldTenants
		lock.Unlock()
		env.SetFallbackTenantName(oldFallback)
	})
}

// 有 active 租户：仅保留命名空间匹配且 active 的租户，ctx 携带租户名
func TestParseTenantCfg_ActiveTenant(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")
	ns := env.GetNamespace()
	data := fmt.Sprintf(`{
		"t1": {"tenant_id":"1","tenant_name":"tenant-a","namespace":"%s","type":"exclusive","status":"active"},
		"t2": {"tenant_id":"2","tenant_name":"tenant-b","namespace":"%s","type":"exclusive","status":"disabled"},
		"t3": {"tenant_id":"3","tenant_name":"tenant-c","namespace":"__other_ns__","type":"exclusive","status":"active"}
	}`, ns, ns)

	if err := parseTenantCfg(data); err != nil {
		t.Fatalf("parseTenantCfg() error = %v, want nil", err)
	}
	ctxs := GetTenants()
	if len(ctxs) != 1 {
		t.Fatalf("GetTenants() len = %d, want 1", len(ctxs))
	}
	if got := metadata.Get[string](ctxs[0].Ctx, metadata.CONTEXT_NAME_SPACE); got != "tenant-a" {
		t.Errorf("ctx 租户名 = %q, want %q", got, "tenant-a")
	}
}

// 租户存在但全部非 active 且未配置兜底租户：返回错误，且保留旧租户列表
func TestParseTenantCfg_AllInactiveNoFallback(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")
	lock.Lock()
	tenants = []tenant.Tenant{{TenantId: "old", Ctx: context.TODO()}}
	lock.Unlock()

	data := fmt.Sprintf(`{"t1":{"tenant_id":"1","tenant_name":"tenant-a","namespace":"%s","status":"disabled"}}`, env.GetNamespace())
	if err := parseTenantCfg(data); err == nil {
		t.Fatal("parseTenantCfg() error = nil, want error")
	}
	lock.RLock()
	defer lock.RUnlock()
	if len(tenants) != 1 || tenants[0].TenantId != "old" {
		t.Errorf("解析失败时应保留旧租户列表, got %+v", tenants)
	}
}

// 空配置 + 已配置兜底租户：使用 defaultTenant
func TestParseTenantCfg_EmptyWithFallback(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("ns")

	if err := parseTenantCfg(""); err != nil {
		t.Fatalf("parseTenantCfg() error = %v, want nil", err)
	}
	lock.RLock()
	defer lock.RUnlock()
	if len(tenants) != 1 {
		t.Fatalf("tenants len = %d, want 1", len(tenants))
	}
	got := tenants[0]
	if got.TenantId != defaultTenant.TenantId || got.Status != TenantStatusActive || got.Type != TenantTypeExclusive {
		t.Errorf("兜底租户 = %+v, want defaultTenant", got)
	}
	if got.TenantName != "" {
		t.Errorf("兜底租户 TenantName = %q, want 空（数据层走原始命名）", got.TenantName)
	}
}

// 空配置 + 未配置兜底租户：返回错误
func TestParseTenantCfg_EmptyNoFallback(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")

	if err := parseTenantCfg(""); err == nil {
		t.Fatal("parseTenantCfg() error = nil, want error")
	}
}

// 租户列表变更监听：内容变化才触发，相同内容重放不触发
func TestParseTenantCfg_ChangeListener(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")
	lock.Lock()
	tenants = nil
	lock.Unlock()
	// 保存并恢复监听器列表，避免污染其他用例
	listenerLock.Lock()
	oldListeners := changeListeners
	changeListeners = nil
	listenerLock.Unlock()
	t.Cleanup(func() {
		listenerLock.Lock()
		changeListeners = oldListeners
		listenerLock.Unlock()
	})

	var notified [][]tenant.Tenant
	RegisterChangeListener(func(ts []tenant.Tenant) { notified = append(notified, ts) })

	ns := env.GetNamespace()
	dataA := fmt.Sprintf(`{"t1":{"tenant_id":"1","tenant_name":"tenant-a","namespace":"%s","status":"active"}}`, ns)
	dataB := fmt.Sprintf(`{"t1":{"tenant_id":"1","tenant_name":"tenant-a","namespace":"%s","status":"active"},"t2":{"tenant_id":"2","tenant_name":"tenant-b","namespace":"%s","status":"active"}}`, ns, ns)

	// 首次加载（nil → A）：触发
	if err := parseTenantCfg(dataA); err != nil {
		t.Fatalf("parseTenantCfg(dataA) error = %v", err)
	}
	if len(notified) != 1 || len(notified[0]) != 1 || notified[0][0].TenantName != "tenant-a" {
		t.Fatalf("首次变更应触发一次监听, notified = %+v", notified)
	}
	// 相同内容重放：不触发
	if err := parseTenantCfg(dataA); err != nil {
		t.Fatalf("parseTenantCfg(dataA 重放) error = %v", err)
	}
	if len(notified) != 1 {
		t.Fatalf("相同内容不应重复触发, 触发次数 = %d", len(notified))
	}
	// 内容变化（A → A+B）：再次触发
	if err := parseTenantCfg(dataB); err != nil {
		t.Fatalf("parseTenantCfg(dataB) error = %v", err)
	}
	if len(notified) != 2 || len(notified[1]) != 2 {
		t.Fatalf("内容变化应再次触发, notified = %+v", notified)
	}
}

// 监听器 panic 被隔离：不打断通知流程，后续监听器正常触发
func TestParseTenantCfg_ListenerPanicIsolated(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")
	lock.Lock()
	tenants = nil
	lock.Unlock()
	listenerLock.Lock()
	oldListeners := changeListeners
	changeListeners = nil
	listenerLock.Unlock()
	t.Cleanup(func() {
		listenerLock.Lock()
		changeListeners = oldListeners
		listenerLock.Unlock()
	})

	var secondCalled bool
	RegisterChangeListener(func([]tenant.Tenant) { panic("boom") })
	RegisterChangeListener(func([]tenant.Tenant) { secondCalled = true })

	data := fmt.Sprintf(`{"t1":{"tenant_id":"1","tenant_name":"tenant-a","namespace":"%s","status":"active"}}`, env.GetNamespace())
	if err := parseTenantCfg(data); err != nil {
		t.Fatalf("parseTenantCfg() error = %v, 监听器 panic 不应影响加载", err)
	}
	if !secondCalled {
		t.Error("首个监听器 panic 后，后续监听器仍应被触发")
	}
}

// 非法 JSON：返回错误，且保留旧租户列表
func TestParseTenantCfg_InvalidJSON(t *testing.T) {
	snapshotGlobals(t)
	env.SetFallbackTenantName("")
	lock.Lock()
	tenants = []tenant.Tenant{{TenantId: "old", Ctx: context.TODO()}}
	lock.Unlock()

	if err := parseTenantCfg("{not-json"); err == nil {
		t.Fatal("parseTenantCfg() error = nil, want error")
	}
	lock.RLock()
	defer lock.RUnlock()
	if len(tenants) != 1 || tenants[0].TenantId != "old" {
		t.Errorf("解析失败时应保留旧租户列表, got %+v", tenants)
	}
}
