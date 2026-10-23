# pmicro：微服务应用框架

> 返回 [README](../../README.md)。快速上手见 [getting-started.md](../getting-started.md)。

`library/pmicro` 是基于 go-micro v4 的应用脚手架，包含 gRPC、HTTP 协议。业务服务通过 `pmicro.NewProgram(option...)` + `prog.Run()` 启动，框架完成配置加载、组件初始化与优雅停机。全部 Option 定义在 `library/pmicro/option`。

启动条件：

- 环境变量 `ENVIRON` 必填；本地为 `server-local`
- 配置了 `NACOS_URLS` 时从 Nacos 读取配置；本地且未配置配置中心时读取 `option.ConfigFile(...)` 指定的文件
- 代码中还有 etcd 配置源（`ETCD_URLS`），但目前地址会被 `NACOS_URLS` 覆盖，单独使用 etcd 无法启动，见 [environment.md · 已知问题](../environment.md#已知问题)
- 完整环境变量见 [environment.md](../environment.md)

## 启动与停机生命周期

`Run()` 按序执行（见 `library/pmicro/program.go` 的 `Init` / `Start`）：

1. **Init**：（非本地环境）等待 istio sidecar 就绪（`STARTUP_PROXY_WAIT_SECONDS`，默认最多 10s）→ 加载配置（本地 yaml / nacos，同时校验环境变量、解析[凭据引用](secret.md)）→ 按配置重建日志并打印一条构建信息（Go 版本、服务 git 提交、gomicro-util 版本）→ Redis → MySQL → 对象存储 → `BeforeStartFunc`
2. **Start**：启动 gRPC 微服务（挂载默认 wrapper，见下文）→ Kafka → 指标上报 → `AfterFunc` → 业务 HTTP server → 探活 HTTP server → 定时任务 → 就绪探针置为 true
3. 阻塞等待停服信号

注意 `AfterFunc` 执行时 HTTP 服务和定时任务**尚未启动**。

收到 SIGINT/SIGTERM 后 `Stop()` 按序执行：就绪探针置为 false（k8s 摘流）→ 停 gRPC → 停 HTTP（5s 超时排空连接）→ 停定时任务 → `BeforeStopFunc` → 按权重降序执行各组件关闭函数（Kafka=10 → 指标=1 → MySQL / 对象存储=0 → Redis=-1）→ 关闭 tracer。

## 子包一览

| 子包 | 职责 |
|---|---|
| `plugin/client_grpc` `plugin/server_grpc` | 自研 gRPC 客户端/服务端插件。客户端带连接池（每地址池内保留 50 个连接，池满或都在忙时会临时新建、用完即关，并非硬上限；单连接 20 并发流、寿命 `pool_ttl_s` 可配；**业务错误不关闭连接**，仅传输层 Unavailable/DataLoss 丢弃）|
| `registry/` | 注册中心适配：etcd / nacos / k8s，可叠加 istio（`registry/redis` 目录存在但未接入）。istio 模式不做真实服务发现，固定拨号 `<服务名>-svc:9804` 交由 Envoy 负载均衡 |
| `selector/` | 与 registry 配套的节点选择器（etcd 轮询 / nacos 轮询 / istio 固定节点） |
| `wrapper/` | gRPC 中间件链：`auth`（服务端认证）、`validator`（参数校验）、`response`（业务错误码经 gRPC trailer 传递与还原）、`recovery`（panic 恢复）、`metadata`（元数据透传）、`monitoring/prometheus`（请求量/耗时指标）、`tracer`（链路）。**默认挂载**：服务端 recovery / tracer / response / auth / validator / prometheus / metadata；客户端 client / recovery / prometheus / response / validator / metadata。`skywalking`、`limit`、`seata`、`running` 包存在但**默认不挂载**，option 也没有追加 wrapper 的入口 |
| `http/` | 业务 HTTP server（gin，含指标/日志/跨域/恢复中间件）与运维探活 server |
| `config/` `option/` | 配置加载（本地/nacos，含变更监听）与 `NewProgram` 的全部 Option 定义 |
| `session/` | `UserId(ctx)` / `MustUserId(ctx)` / `ClientType(ctx)` / `MustClientType(ctx)`。⚠️ 读取的是 `session.UserIdKey{}` 等独立 key，本库内**没有任何代码写入**，与 `metadata.CONTEXT_USER_ID` 不是一回事；读取请求用户请用 `metadata` |
| `wait_stop/` | 优雅停服的在途请求计数器：入口 `IsStop()` 拒新请求 + `Add()`/`Done()` 计数，停服流程 `Stop()` + `Wait()` 排空。⚠️ 负责计数的 `wrapper/running` 默认没有挂载，业务不自行调用 `Add`/`Done` 时 `Wait()` 不会等待任何请求 |
| `metrics/` | 监控指标采集与上报，见 [observability.md](observability.md#metrics监控指标采集与上报) |

## RPC 客户端行为

- **重试**：仅对 `codes.Unavailable`（连接不通，如上游发布窗口）及错误信息含 `invalid connection` 的错误重试，业务错误与其余错误一律不重试（无法确认服务端是否已收到请求，避免非幂等接口重复执行）。次数/间隔由 `rpc_server.client.attempts` / `attempt_interval_ms` 配置。
- **错误传递**：服务端 handler 返回的业务错误由 `wrapper/response` 解析错误码写入 gRPC trailer，客户端侧还原为 `error-code` 业务错误，业务代码两端无感知。
- **连接池**：详见上表 `plugin/client_grpc`；Istio 环境下建议保持默认长连接配置，由 Envoy 做流级负载均衡。

## 探活与就绪（liveness server）

由 `http.liveness` 配置启动的独立运维端口（⚠️禁止暴露公网），供 k8s 探针与运维脚本使用：

| 接口 | 用途 |
|---|---|
| `GET /api/liveness` | 存活探测 |
| `GET /api/readiness` | 就绪探测（`Stop()` 第一步会将其置为未就绪实现摘流） |
| `POST /api/set/unreadiness` | 手动摘流：下线/维护前取消就绪 |
| `POST /api/set/readiness` | 手动恢复就绪 |

两个 POST 接口可配置 `secret` 鉴权（请求带 `Authorization: <secret>` 头）；GET 探测接口无鉴权，兼容 k8s 探针。

## 相关基础包

### svc

服务生命周期编排 + 优雅停服：`svc.Run(service, sig...)` 执行 `Init()` → `Start()` → 阻塞等待停服信号（默认 SIGINT/SIGTERM）→ `Stop()`。`service` 实现 `model/facade` 的 `Service` 接口即可。

> 注：`Run` 启动前会固定 sleep 5s，规避华为云环境 nacos 配置未就绪导致的启动重启。pmicro 的 `Program.Run()` 内部即基于此模式；独立进程（非 pmicro）可直接使用，并配合 `pmicro/wait_stop` 的 `Add/Done/Wait/Stop/IsStop` 实现在途请求排空。

### single

监听停服信号通知包。

```go
stopChan := single.OsKillSingle()
// 等待停服消息
<-stopChan
```

### automaxprocs

自动配置cpu核数，组件通过感应系统自动配置cpu核数，要求设置核心数不超过系统可用核心数（对于容器内应用，通过 cgroup去获取核心数），防止出现上下文切换频繁导致延迟问题。

[了解详情](https://mp.weixin.qq.com/s/9Ra4hmf327f4ddE88gHClA)

## 子包补充说明

- `library/pmicro/config`：程序启动配置读取与装配入口，承接配置中心或本地配置文件；内部调用 `env.SyncEnv()` 校验环境变量。
- `library/pmicro/http/middleware`：内置中间件集合，覆盖 header、日志、租户、恢复、链路等场景。
- gRPC 插件细节见 [client_grpc/README.md](../../library/pmicro/plugin/client_grpc/README.md)、[server_grpc/README.md](../../library/pmicro/plugin/server_grpc/README.md)。
- 指标采集与上报见 [observability.md · metrics](observability.md#metrics监控指标采集与上报)。
