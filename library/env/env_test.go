package env

import "testing"

// TestHasFallbackTenant 验证兜底租户开关（环境变量 TENANT_FALLBACK）的读取/设置
func TestHasFallbackTenant(t *testing.T) {
	defer SetFallbackTenantName("")

	// 未配置时为 false
	SetFallbackTenantName("")
	if HasFallbackTenant() {
		t.Errorf("未配置兜底租户时 HasFallbackTenant() 应为 false")
	}

	// 配置后为 true
	SetFallbackTenantName("ns")
	if !HasFallbackTenant() {
		t.Errorf("配置兜底租户 ns 后 HasFallbackTenant() 应为 true")
	}
}
