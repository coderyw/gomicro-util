# 多租户

> 返回 [README](../README.md)。相关环境变量见 [environment.md · 租户与部署模式](environment.md#租户与部署模式)。

## 概念

| 概念 | 来源 | 作用 |
|-|-|-|
| 租户映射 | Nacos `TENANT_CFG_NACOS_NAMESPACE / TENANT_CFG_NACOS_GROUP / TENANT_CFG_NACOS_DATAID`（默认 `tenant / DEFAULT_GROUP / saas-tenant-meta-mapping`） | 所有租户的元数据：`tenant_name`、`namespace`、`status` 等 |
| 部署命名空间 | 环境变量 `NAMESPACE` | 当前 pod 只加载 `Namespace == NAMESPACE` 且 `Status == active` 的租户 |
| 请求租户 | ctx 中的 `metadata.CONTEXT_NAME_SPACE` | 决定本次请求访问哪个租户的资源 |
| 兜底模式 | 环境变量 `TENANT_FALLBACK` | 非空时数据层一律保持原始命名，不加租户前缀（独享部署，如 NS） |

`NAMESPACE`、`NACOS_NAMESPACE_ID`、`TENANT_CFG_NACOS_NAMESPACE` 是三个不同的东西：分别决定「哪些租户归当前部署」「服务 YAML 在哪」「租户映射文档在哪」。

使用 Nacos 配置源时，启动会加载并监听租户映射（`ENABLE_LISTEN_TENANT=false` 可关闭监听）；启动时没有匹配租户且未设置 `TENANT_FALLBACK` 会启动失败，运行期变更后没有匹配项则只记错误日志、保留旧列表。本地 YAML 模式和 etcd 模式不加载租户映射。本地（`server-local`）可用 `NAMESPACE_TENANT=tenant_a,tenant_b` 模拟租户列表。

> ⚠️ 运行期监听固定使用 `saas-tenant-meta-mapping / DEFAULT_GROUP`，自定义 `TENANT_CFG_NACOS_DATAID` / `GROUP` 只影响启动时的首次加载。

## 租户如何进入 ctx

- **HTTP**：中间件 `ParseNamespace` 读取请求头 `X-Tenant-Name` 写入 ctx
- **gRPC**：metadata wrapper 在服务间透传
- **Kafka**：生产时按 ctx 租户给 topic 加前缀；消费时从 topic 名解析租户写入 ctx（批量消费见每条消息的 `msg.Ctx`）
- **定时任务 / 后台任务**：没有请求 ctx，需要遍历租户：`namespace.GetTenantsInNamespace()`，或直接用 [TenantCron](components/scheduling.md#多租户定时任务tenantcron)

> 资源隔离完全依赖 ctx。**不要在业务链路中用 `context.Background()` 替代请求 ctx**，否则会访问到无租户前缀的资源。

## 资源命名规则

各组件拿到 ctx 中的租户后自动拼接前缀（MySQL 由 GORM 回调单独实现，系统库和已带前缀的库名会跳过；其余组件走 `namespace.FmtByType`）：

| 资源 | 格式 | 示例（租户 `ok`） |
|-|-|-|
| MySQL 库名 | `{tenant}_{db}` | `ok_ca_user` |
| Redis key | `tenant{tenant}:{key}` | `tenantok:ca-pay:xxx` |
| Kafka topic | `tenant{tenant}_{topic}` | `tenantok_topic.order` |
| 通用（本地缓存等） | `tenant{tenant}:{key}` | |
| 对象存储路径 | `tenant{tenant}/{path}` | |

- 租户为空，或设置了 `TENANT_FALLBACK` 时，返回原始名称。**例外**：`component/storage` 在 ctx 没有租户时使用默认租户 `ns`，key 会变成 `tenantns/{key}`
- Kafka 消费端从 topic 前缀解析租户；兜底模式下 topic 没有前缀，消费 ctx 中也就没有租户
- 租户段大小写由 `TENANT_NAME_CASE` 决定（默认 `lower`），见 [upgrade/v3.0.0.0.md](upgrade/v3.0.0.0.md)

**自动隔离的组件**：MySQL、Redis、Kafka、`component/storage`（S3/OBS）、本地缓存、Redis 同步任务；定时任务需使用 `TenantCron` 且传入 `WithLockFunc` 才会按租户加锁（锁 key 为 `{tenant}:{lockName}`）。

**不自动隔离的组件**：RabbitMQ、旧 `component/object-storage` 接口，使用方需通过命名、连接等方式自行保证隔离。

## 公共方法（`library/namespace`）

| 方法 | 说明 |
|-|-|
| `GetTenantsInNamespace() []context.Context` | 当前部署下各租户的 ctx；租户未加载时兜底返回一个 `context.Background()` |
| `GetAllTenantsInfo() []tenant.Tenant` | 租户信息浅拷贝；未加载时返回空切片 |
| `OnTenantsChange(fn)` | 运行期租户列表变化时回调（启动首次加载不触发，注册后不可注销，回调需快速返回） |
| `FmtByType(t, nsKey, key)` / `FmtByTypeOnCtx(ctx, t, key, defaultNs...)` | 按资源类型拼接租户前缀 |
| `GetNameSpaceFromKey` / `GetOriginKey` / `GetCtxAndOriginKey` / `GetNameSpaceCtxFromKey` | 从带前缀的资源名反解租户与原始 key |
| `NormalizeTenantName(name)` | 按 `TENANT_NAME_CASE` 转换租户名 |

判断部署模式用 `env.HasFallbackTenant()`、`env.GetNamespace()`，见 [environment.md · 相关公共方法](environment.md#相关公共方法)。
