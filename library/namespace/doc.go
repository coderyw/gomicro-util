// Package namespace 多租户：租户列表与资源命名。
//
//   - GetTenantsInNamespace / GetAllTenantsInfo / OnTenantsChange：当前部署（环境变量 NAMESPACE）下的租户
//   - FmtByType / FmtByTypeOnCtx：按资源类型（MySQL / Redis / Kafka / OSS / 通用）拼接租户前缀
//   - GetOriginKey / GetCtxAndOriginKey 等：从带前缀的资源名反解租户与原始 key
//   - NormalizeTenantName：按 TENANT_NAME_CASE 转换租户名大小写
//
// 设置了 TENANT_FALLBACK 的 pod 不加租户前缀，保持原始命名。
//
// 文档：docs/multi-tenant.md
package namespace
