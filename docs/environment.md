# 环境变量与相关公共方法

> 返回 [README](../README.md)。YAML 配置见 [configuration.md](configuration.md)，租户相关行为见 [multi-tenant.md](multi-tenant.md)。

适用对象：使用 gomicro-util / pmicro 启动服务的研发与部署人员。业务服务自身、第三方 SDK 或部署平台还可能定义额外变量，不属于本清单。

**必填性口径**：必填＝所有 pmicro 启动场景都要求；条件必填＝仅在指定场景下要求；选填＝可省略并沿用默认行为；无需配置＝当前公共库未读取。

**生效时机**：多数环境变量在 Go 包 `init` 时读取并缓存，应在进程启动前注入，修改后需重启服务。`env.SyncEnv()` 只做校验，不会重新读取。

## 启动与配置来源

MySQL、Redis、Kafka 连接信息及 HTTP/gRPC 端口从服务 YAML 加载，不是环境变量。

| 环境变量 | 是否必填 | 含义、默认值与生效条件 |
|-|-|-|
| `ENVIRON` | 必填 | 运行环境，无默认值。取值：`server-local`、`server-dev`、`server-test`、`server-pre`、`server-prod`、`server-roadshow`、`server-test-auto`、`server-pre-auto`。缺失时启动失败；非法值在 pmicro 加载配置时报「ENVIRON类型错误」（`env.SyncEnv` 本身不校验取值） |
| `NACOS_URLS` | 条件必填：使用 Nacos | Nacos 地址列表，英文逗号分隔，`host:port`，不带 `http://` |
| `NACOS_NAMESPACE_ID` | 条件必填：使用 Nacos | 服务配置所在的 Nacos Namespace **ID**（不是显示名称）。本地连接 Nacos 同样需要。会**无条件**覆盖 YAML 中的 `registry.name_id`（本地 YAML 模式下未设置时会把它清空） |
| `NACOS_USERNAME` / `NACOS_PASSWORD` | 条件必填：Nacos 开启鉴权 | 配置中心及租户映射客户端使用的账号密码；默认空（匿名登录）。值可以写成[凭据引用](components/secret.md#nacos-账密)（如 `csms://<凭据名>/<键名>`），也可以写明文；密码为明文时打 WARN，`NACOS_AUTH_ALLOW_PLAINTEXT=false` 时禁止明文 |
| `ETCD_URLS` | 条件必填：使用 etcd（当前有缺陷） | etcd 地址列表，英文逗号分隔。当前单独配置该变量时无法正常启动（非本地直接失败，本地会用空地址连接 etcd），见[已知问题](#已知问题) |
| `SERVER_VERSION` | 选填 | 服务版本，用于服务注册和日志；默认 `latest` |
| `ID` | 选填 | 实例 ID，用于注册、一致性哈希分片、Redis 同步任务等；默认生成 UUID 并写回进程环境。手工指定时各实例必须唯一 |

本地且未配置配置中心地址时，读取 `option.ConfigFile(...)` 指定的 YAML；配置 Nacos 后读取 `NACOS_NAMESPACE_ID / DEFAULT_GROUP / <服务名>.yaml`。服务名来自 `option.ServiceName(...)`，`SERVER_NAME` 环境变量不生效。

## 租户与部署模式

| 环境变量 | 是否必填 | 含义、默认值与生效条件 |
|-|-|-|
| `NAMESPACE` | 条件必填：按 K8S 命名空间筛选租户 | 默认空。须与租户映射记录的 Namespace 匹配。仅在使用 Nacos 配置源时生效：启动时无 active 匹配项且未设置 `TENANT_FALLBACK` 会启动失败；运行期变更后无匹配项只记错误日志并保留旧列表。它不是 `NACOS_NAMESPACE_ID` |
| `ENABLE_LISTEN_TENANT` | 选填 | 默认 true。只有精确值 `false` 才关闭租户配置加载和监听（`0`、`False` 不会关闭）。关闭监听不等于切换为单租户模式 |
| `TENANT_CFG_NACOS_NAMESPACE` | 选填 | 租户映射所在的 Nacos Namespace ID，默认 `tenant` |
| `TENANT_CFG_NACOS_DATAID` | 选填 | 租户映射的 Data ID，默认 `saas-tenant-meta-mapping`。⚠️ 只对启动时首次加载生效，运行期监听写死为默认值，修改后热更新会监听另一份配置 |
| `TENANT_CFG_NACOS_GROUP` | 选填 | 租户映射的 Group，默认 `DEFAULT_GROUP`。⚠️ 与 Data ID 相同，运行期监听写死为默认值 |
| `TENANT_FALLBACK` | 条件必填：独享部署/兜底模式 | 默认空。非空表示该 pod 数据层保留原始资源命名、不加租户前缀。共享多租户部署通常留空，不能仅为消除启动错误而填写 |
| `TENANT_NAME_CASE` | 选填 | 资源名中租户段的大小写：`lower`（默认）、`upper`、`original`。非法值启动报错；修改会指向不同的资源名 |
| `NAMESPACE_TENANT` | 选填，仅本地调试 | 仅 `server-local` 生效，英文逗号分隔（如 `tenant_a,tenant_b`），用于初始化本地租户列表；未设置时为一个空名租户。本地同时连接 Nacos 且开启监听时会被 Nacos 映射覆盖 |
| `CLOUD_TYPE` | 选填 | 显式指定云厂商：`huawei`（或 `hw`）、`aws`、`azure`、`google`（或 `gcp`）、`tencent`、`aliyun`，不区分大小写。非空但无法识别的值直接返回 `other`，不再探测。不填则按 DMI → IMDS 探测（本地跳过 IMDS），未识别返回 `other` |

## 可选功能

| 环境变量 | 是否必填 | 含义、默认值与生效条件 |
|-|-|-|
| `DISABLE_EONID_HASHTAG` | 选填（**NS 必须填 `1`**） | 雪花 ID Redis key 前缀：空或 `0` 使用 `{eonId}`；`1` 使用旧版 `eonId`；其他值启动报错。与旧服务共享注册池时，Go/PHP 服务须使用一致的前缀、Redis、DB、DataCenter |
| `TRACE_TYPE` | 选填 | 链路追踪模式：`default`、`skywalking`、`orchestrion`；空或未知值回退 `default`。目前只有 `orchestrion` 会改变行为，`skywalking` 与 `default` 相同 |
| `local_ip` | 选填 | 覆盖 gRPC 服务注册 IP（变量名为小写） |
| `kafka_local_dev` | 选填，仅本地调试 | 仅 `ENVIRON=server-local` 且值为 `true` 时，生产消息的 topic 追加 `_dev`；不影响消费端 |

## 凭据与启动

| 环境变量 | 是否必填 | 含义、默认值与生效条件 |
|-|-|-|
| `SECRET_RETRY_DEADLINE_SECONDS` | 选填 | [凭据引用](components/secret.md)解析的重试整体 deadline（秒），按凭据名计算，默认 20；`0` 为只尝试一次不重试，负数按 0，非法值回落默认。调大时须配置 startupProbe |
| `STARTUP_PROXY_WAIT_SECONDS` | 选填 | 非本地环境启动时，加载配置前等待 istio sidecar 就绪的上限（秒），默认 10；`0` 为不等待。连续约 3s 连接被拒即判定无 sidecar 并放行，到上限仍未就绪只告警不阻塞。**无 istio 注入的 Pod（如 CronJob）建议设为 `0`**，否则每次启动多等约 3s |
| `NACOS_AUTH_ALLOW_PLAINTEXT` | 选填 | 是否允许 `NACOS_PASSWORD` 为明文，默认允许；`false` / `0` / `no` / `off`（不区分大小写）时，密码为明文直接启动失败 |
| `HUAWEICLOUD_SDK_REGION` | 选填 | `csms://` 凭据所在的华为云 region，默认 `ap-southeast-3` |
| `AWS_REGION` / `AWS_DEFAULT_REGION` | 选填 | `awssm://` 凭据所在的 AWS region，默认 `ap-southeast-1` |

云凭证走各 SDK 的默认链（华为云 CCE Pod Identity、AWS IRSA 等），容器内无需配置 AK/SK，见 [secret · Provider 与云凭证](components/secret.md#provider-与云凭证)。

## 已声明但无需配置

`NACOS_NAME_ID`、`POD_NAME`、`SERVER_NAME`（`model/cst`）和 `ENABLE_MUL_TENANT`（`internal/mul-tenant`）有常量或注释，但公共库当前不读取。

## 配置示例

本地 YAML 模式（进程未继承 `NACOS_URLS` / `ETCD_URLS`）：

```bash
export ENVIRON=server-local
```

共享多租户 Nacos 模式（尖括号为占位符）：

```bash
export ENVIRON=server-test
export NACOS_URLS='nacos.example.internal:8848'
export NACOS_NAMESPACE_ID='<服务配置 Namespace ID>'
export NACOS_USERNAME='<账号>'
export NACOS_PASSWORD='csms://<凭据名>/<键名>'   # 推荐凭据引用，也可写明文
export NAMESPACE='<租户映射中的 Namespace>'
export SERVER_VERSION='1.0.0'
```

## 相关公共方法

### `library/env`

| 方法 | 对应变量 | 行为 |
|-|-|-|
| `GetEnv() string` | `ENVIRON` | 原始值 |
| `IsLocal()` / `IsDev()` / `IsTest()` / `IsPre()` / `IsRelease()` | `ENVIRON` | 分别对应 `server-local` / `-dev` / `-test` / `-pre` / `-prod`，精确匹配 |
| `IsTestAuto()` / `IsPreAuto()` | `ENVIRON` | `server-test-auto` / `server-pre-auto` |
| `IsFormal() bool` | `ENVIRON` | `IsPre() \|\| IsRelease()` |
| `GetNacosNamespaceId() string` | `NACOS_NAMESPACE_ID` | |
| `GetVersion() string` | `SERVER_VERSION` | 默认 `latest` |
| `GetId() string` | `ID` | 未配置时为启动时生成的 UUID |
| `GetNamespace() string` | `NAMESPACE` | |
| `GetEnableListenTenant() bool` | `ENABLE_LISTEN_TENANT` | |
| `HasFallbackTenant() bool` | `TENANT_FALLBACK` | 非空即 true |
| `GetTenantNameCase() string` | `TENANT_NAME_CASE` | `lower` / `upper` / `original` |
| `GetDisableEonidHashTag() (bool, error)` | `DISABLE_EONID_HASHTAG` | 非法值返回 error |
| `GetCloudType() tenant.CloudType` | `CLOUD_TYPE` | 首次调用后缓存 |
| `GetOsEnv(key, defaultValue...) string` | 任意 | 实时读取，空值时返回默认值 |
| `SyncEnv() error` | — | 启动校验，pmicro 启动时自动调用 |

> ⚠️ 判断环境时注意：`IsTest()`、`IsPre()` 不包含 `*-auto` 环境，`IsFormal()` 不包含 `server-roadshow`。

`SetEnv`、`SetFallbackTenantName`、`SetTenantNameCase` 仅用于测试注入，线上运行中不要调用。

### 间接读取环境变量的方法

| 方法 | 依赖变量 |
|-|-|
| `namespace.NormalizeTenantName` | `TENANT_NAME_CASE` |
| `namespace.FmtByType` / `FmtByTypeOnCtx` | `TENANT_FALLBACK`、`TENANT_NAME_CASE` |
| `namespace.GetTenantsInNamespace` / `GetAllTenantsInfo` / `OnTenantsChange` | `NAMESPACE`、`ENABLE_LISTEN_TENANT`、`TENANT_CFG_NACOS_*`、`NAMESPACE_TENANT` |
| `trace.GetTraceCtx` 等 | `TRACE_TYPE`（无公开 getter） |

变量名常量在 `model/cst/env.go`（`cst.ENV_*`），运行环境枚举为 `cst.RUNTIME_ENV_*`。

## 已知问题

- **etcd 地址被覆盖**：`library/pmicro/config/server_config.go` 的 `InitCfg` 读取 `ETCD_URLS` 后又无条件用 `NACOS_URLS` 覆盖同一变量。仅配置 etcd 时非本地启动会失败，错误文案还会误报缺少 `ENVIRON`；本地则会用空地址连接 etcd；两者都有值时选择 Nacos。
- **租户映射热更新位置写死**：`internal/mul-tenant` 的 `listen()` 固定监听 `saas-tenant-meta-mapping / DEFAULT_GROUP`，不读取 `TENANT_CFG_NACOS_DATAID` / `TENANT_CFG_NACOS_GROUP`。
- **租户映射只在 Nacos 配置源下加载**：本地 YAML 模式与 etcd 模式都不会加载租户映射。

## 代码依据

| 文件 | 内容 |
|-|-|
| `model/cst/env.go` | 环境变量名与运行环境枚举 |
| `library/env/env.go` | `init`、默认值、`checkEnv` 校验 |
| `library/env/tenant_name.go`、`eonid.go`、`cloud.go` | 租户命名、雪花 ID 前缀、云厂商识别 |
| `library/pmicro/config/server_config.go` | 配置源选择、Nacos 账号 |
| `internal/mul-tenant/namespace_tenant.go` | 租户映射位置、本地租户列表 |
| `library/component/secret/retry.go`、`provider/*`、`library/pmicro/config/nacos_credential.go`、`library/pmicro/startup_wait.go` | 凭据重试 deadline、region、Nacos 账密来源、sidecar 等待 |
| `library/trace/trace_pro.go`、`library/pmicro/plugin/server_grpc/grpc.go`、`library/mq/kafka/producer_confluent.go` | `TRACE_TYPE`、`local_ip`、`kafka_local_dev` |

**新增或修改环境变量时，请同步更新本文。**
