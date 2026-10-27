# Casino Golang common util

Golang公共包，以及框架库。包含微服务gomicro、数据库mysql，redis、链路追踪jaeger，skywalking、gin、轻量级延迟队列beanstalk、

## 版本变动

### BREAKING CHANGE
 - config(nacos): Nacos NamespaceId 必须通过 NACOS_NAMESPACE_ID 环境变量显式指定，
   不再从 ENVIRON 推导。所有依赖本框架的服务升级时必须同步在部署中注入该变量，
   否则启动失败。
 - 建议同步在本次版本号上 bump major 号（如 v2.0.0），让下游能通过 go.mod 显式控制升级时点。

## v1.1.0

> 正式版，代码与 `v1.1.0-rc.16` 一致（`saas/feat-sync-ns-v2.5.6.0` @ `ed68bfc`）。
> 以下为相对上一正式版 `v1.0.16-saas-20260721-1` 的全部变更，汇总自 v1.1.0-rc.1 ~ rc.16 及 CHANGELOG-UNSTABLE.md。

### BREAKING CHANGE（升级前必读）

**部署 / 配置**

- **Nacos NamespaceId 改由环境变量 `NACOS_NAMESPACE_ID` 指定**，不再从 `ENVIRON` 推导；非 local 环境走 nacos 时未注入直接启动失败。yaml 中的 `registry.name_id` 会被该变量覆盖。须先在部署侧注入该变量再升级
- **探活重构**：删除 `HttpCfg.EnableLiveApi`，yaml 中 `enable_live_api` 静默失效；`http.host` 为空时不再兜底以 `:8080` 启动主 HTTP 服务。`http.host` 为空的服务必须新增 `http.liveness.host`（或显式配置 `http.host`），否则探针端口无监听，pod 会被 livenessProbe 反复重启
- **RPC 客户端重试**：原固定重试 3 次，改为 `rpc_server.client.attempts`（总尝试次数）控制——不配置默认 3 次（即重试 2 次），`-1` 关闭重试；重试间隔 `attempt_interval_ms` 默认 500ms；go-micro 自身错误不再重试（无法确认服务端是否已收到）
- **Kafka librdkafka C 侧队列默认值收紧**：消费端 `groups[].queued_min_messages` 默认 2000、`groups[].queued_max_messages_kbytes` 默认 8192（单分区约 8MB）；生产端 `producer[].queue_buffering_max_kbytes` 默认 65536（约 64MB，原约 1GB）；每分区 Go channel 容量 1024 → 256（`groups[].msg_chan_cap`）。高吞吐服务升级后如出现消费延迟，按需调大
- **Kafka 消费组订阅方式变更**：废弃正则匹配 topic，改为按租户列表精确生成订阅 topic，租户列表变更时动态重订阅

**代码 / API**

- `redis.InitRedis` 返回值新增 `*cron.CronJob`：`(micro.CloseS, *cron.CronJob, error)`
- `model.RedisSyncFunc` 增加 `error` 返回值：`func(context.Context, string) error`
- `sync.NewRedisSyncJob` 新增 `*ants.Pool` 参数；移除 `RedisSyncMgr.SyncTaskRun`
- 接口新增方法，自行实现接口（而非使用本库实现 / mock）的调用方需补齐：`RedisInterface.PutWaitSyncKeys`、`LocalCache.ItemsOnCtx` / `FlushOnCtx`
- `Redis.Scan` 契约变更：`match` 自动加租户前缀；返回的 key 剥回原始 key；`match` 为空时收敛为本租户 `*`（原为遍历全库）。无租户 / `TENANT_FALLBACK` 下行为不变，`.Client().Scan` 旁路不受影响
- 移除导出 API：`redis.PrefixHookV2` / `NewPrefixHookV2` / `ExtractPrefixableArgs`（key_prefix_hook_v2）、`mysql.FormatSchema`、`http.SetReady`

### 新增功能

**feat(tenant/namespace): 多租户能力增强**

- 新增 `TENANT_FALLBACK` 兜底租户：pod 设置后数据层（MySQL 库名改写、Redis / Kafka / OSS / common 前缀）一律不加租户前缀，用于 NS 等独享型租户；租户定时任务在兜底模式下使用无租户 ctx 执行
- 新增 `model/tenant` 包（`Tenant`、`CloudType` / `ParseCloudType`）；mul-tenant 支持活跃状态租户过滤及兜底租户配置
- 新增 `namespace.GetAllTenantsInfo()`、`namespace.OnTenantsChange(fn)`：仅在租户列表内容变化时通知监听器，监听器 panic 隔离
- 新增 `env.GetCloudType()`：优先读环境变量 `CLOUD_TYPE`，否则依次探测 DMI 与 IMDS 元数据，支持华为云、AWS、Azure、Google、腾讯云、阿里云
- `LocalCache` 新增 `ItemsOnCtx` / `FlushOnCtx`，按租户遍历 / 清空本地缓存，原 `Items()` / `Flush()` 不变

**feat(pmicro): 独立探活服务与 readiness 探针**

- 配置 `http.liveness.host` 后启动独立探活服务：`GET /api/liveness`、`GET /api/readiness`、`POST /api/set/unreadiness`、`POST /api/set/readiness`
- 配置 `http.liveness.secret` 后，两个 set 接口校验请求头 `Authorization`，不匹配返回 401；不配置则不校验
- readiness 语义：`Start()` 完成后置为就绪，`Stop()` 开始时立即置为未就绪，供滚动发布提前摘流；新增 `single.SetReadinessProbe` / `GetReadinessProbe`
- 主 HTTP 服务无条件注册 `/api/liveness`；HTTP / 探活服务监听失败改为同步返回错误，不再在 goroutine 内 `log.Fatal`
- RPC 客户端新增 `rpc_server.client.pool_ttl_s` 连接池连接存活时间，默认 3600s，`-1` 不过期

**feat(pmicro/http): 请求日志支持按配置打印请求 header**

- `http.log_headers`：不配置不打印；`*` 打印全部；否则为逗号分隔的 header 名（大小写不敏感），输出到 `HTTP Handle start` 日志的 `headers` 字段，支持同名多值
- `middleware.CustomLogger` 改为 `CustomLogger(logHeaders ...string)`，原无参调用兼容

**feat(metrics): 框架与业务指标采集及 Truewatch 上报**

- 新增顶层 `metrics` 配置：`enable`（默认 false）、`disable_business`、`interval_s`（默认 15）、`truewatch.dataway_url` / `truewatch.workspace_token`
- gin 中间件 `middleware.Metrics()`：按路由模板统计 HTTP 请求量与耗时
- 业务打点 `metrics.NewClient(service)` 的 `Inc` / `Latency`：惰性注册、标签一致性校验，打点异常只告警不影响业务；`Latency` 约定传**毫秒**，默认分桶 1ms ~ 300000ms
- 新增 `metrics.SetGauge` 瞬时值上报
- MySQL / Redis 连接池统计：`pool_stats_mode`（-1 关闭，0 默认同 1，1 仅日志，2 仅上报，3 两者）、`pool_stats_interval`（mysql 默认 60s，redis 默认 15s）
- Kafka librdkafka 统计：`stats_mode`（取值同上）、`statistics_interval_ms`（默认 60000，负数关闭），输出 fetchq / msg_cnt 等 C 侧队列内存观测

**feat(storage): 对象存储**

- 新增华为云 OBS provider（`storage.object_storage.huawei_obs_config`）：支持多桶、桶级 `default_acl`、按 alias 选桶
- 新增 `object_storage.default_instance`：`UploadRequest.Instance` 留空时回落到该实例；配置了但找不到对应实例则启动失败
- 新增 `storage.DefaultInstance()` / `HasInstance()` / `Instances()`
- 新增 `library/component/object-storage` 通用对象存储接口（S3 实现 Put / Get 及重试）

**feat(mq): RabbitMQ 与 Kafka**

- 新增 `library/mq/rabbitmq`：连接池 / channel 池、自动重连消费、发布确认、拓扑声明，入口 `mq.InitRabbitMq`
- Kafka 生产者消息头注入 `trace_id`，消费端读取；新增 `mq.GetCtxFromKafkaHeader`、`trace.NewOrAddTraceIdCtx`
- Kafka 新增 `groups[].partition_assignment_strategy`（默认空沿用 eager，可切 `cooperative-sticky`，需两步升级保证全组协议兼容）
- 订阅注册 rebalance 回调：分区被调走时关闭对应 channel、回收 handler goroutine 并丢弃残留待提交 offset；subscribe 轮询由 `ReadMessage` 改为 `Poll`

**feat(redis): 同步任务重构**

- 同步流程改为 Lua 脚本（sync_add / sync_range / sync_rem）+ 版本号，ZSET 与版本操作原子化，避免取 / 删窗口内的新写入被误删；重试写入使用 GT 推迟生效
- 分区同步走 ants 协程池并发执行，池满或 pool 为 nil 时降级为当前 goroutine；cron 周期加 ±clamp(Ts/5, 1, 30)s 抖动；`Ts` / `QueueNum` 零值兜底（5s / 10）
- 新增 `Redis.PutWaitSyncKeys` / `RedisSyncMgr.PutSyncKeys` 批量投递待同步数据

**feat(util): 分布式令牌桶限流**

- 新增 `util/ratelimit`：基于 Redis Lua 的分布式令牌桶（`NewTokenBucketLimiter`），接口化并提供 `NoopLimiter`，便于调用方 mock

### 问题修复

- **fix(tenant)**：SQL 改写后同步重排参数，修复 `LIMIT ? OFFSET ?` 被还原为 `LIMIT ?,?` 导致 limit / offset 值互换（列表第 3 页起重复展示第 2 页）；一致性校验不通过时降级为只改写库名、不重排参数
- **fix(redis/sync)**：多租户前缀注入后同步任务 key 与 versionKey 的 hash tag 不一致，集群下报 CROSSSLOT；versionKey 改为把加前缀后的 key 放入 `{}`，key 本身命名不变
- **fix(redis-sync/consistent_hash)**：修复同步分区静默无主——哈希环只取 healthy && enable 实例，约 1 分钟强制重下发分区分配、约 5 分钟打印分配快照；分区每轮写心跳 key，缺失即抢占代扫并打 ERROR `partition orphan detected`
- **fix(kafka,mysql)**：停机时 rebalance 回调二次关闭 channel 导致 panic；mysql 连接池统计开关改为局部变量，避免二次 `InitConnect` 状态互相覆盖
- **fix(kafka)**：租户变更重订阅投递加 nil 守卫
- **fix(aes)**：AES 解密补齐边界检查与 PKCS7 填充校验，非法输入返回错误而非 panic
- **fix(rsa)**：RSA 默认密钥位数 512 → 2048
- **fix(log)**：移除开发输出，`password` / `secret` 字段脱敏为 `****`
- **fix(config)**：清理配置示例中的敏感信息；`updateFromJSON` 避免操作 nil 配置

### 依赖

- `github.com/redis/go-redis/v9` v9.14.0 → v9.20.1
- 新增 `github.com/rabbitmq/amqp091-go` v1.13.0、`github.com/huaweicloud/huaweicloud-sdk-go-obs` v3.25.4+incompatible、`github.com/prometheus/client_model` v0.6.2、`github.com/alicebob/miniredis/v2` v2.38.0

## v1.0.14-saas-20260425-3

**refactor(pmicro): 重构微服务代码结构并优化连接重试机制**

- 引入strings包用于错误信息处理
- 修复import语句格式问题，移除多余空行
- 添加SetService函数定义，完善API接口
- 将var声明改为短变量声明语法，提升代码简洁性
- 统一注释格式，确保注释前后有适当空格
- 增强gRPC调用重试逻辑，添加invalid connection错误的重试处理
- 优化变量声明方式，统一使用短变量声明语法
- 清理多余空行，改善代码可读性

### v1.0.14-saas-20260425-2

**feat(http): 重构HTTP客户端实现并优化连接池配置**

- 将默认HTTP客户端配置为使用连接池，设置最大空闲连接数1000，
  每主机最大空闲连接数200
- 提取公共doRequest函数，统一处理POST和GET请求逻辑
- 优化代码结构，减少重复代码
- 更新测试用例以匹配新的泛型类型参数

chore: 在CHANGELOG中添加新版本条目v1.0.14-saas-20260425-2

### v1.0.14-saas-20260425-1

**refactor(kafka): 共享调优参数从 KafkaProducer 上提到 KafkaCfg，生产者与消费者同时下发**

- KafkaCfg 新增 5 个集群级共享参数：socket_keepalive_enable、connections_max_idle_ms、metadata_max_age_ms、socket_timeout_ms、debug，并从 KafkaProducer 移除（不兼容变更，但生产环境暂未使用）
- 新增 KafkaCfg.Init() 集中管理共享默认值，在 InitConfluentKafkaManager 入口统一调用
- 消费者首次下发 socket.keepalive.enable / connections.max.idle.ms / metadata.max.age.ms / socket.timeout.ms / debug，与生产者一致，提升云上长连接抗抖动与拓扑感知能力
- 修复消费者硬编码的 auto.commit.interval.ms 默认值（5000）与 Group.Init() 默认值（3000）不一致问题
- 扩展 KafkaCfg 与 KafkaProducer 的 MarshalLogObject，输出更多生效字段便于排障

### v1.0.14-saas-20260425

**refactor(kafka): 优化 Confluent 生产者配置参数**

- KafkaProducer 新增可调配置：linger_ms、compression_type、max_in_flight_requests_per_connection、socket_keepalive_enable、connections_max_idle_ms、metadata_max_age_ms、socket_timeout_ms
- 新增长连接与拓扑刷新默认值：connections.max.idle.ms=540000、metadata.max.age.ms=180000、socket.timeout.ms=30000

### v1.0.14-saas-20260403

**refactor(registry): 重构nacos watcher订阅逻辑**

- 简化了nacos watcher中的订阅流程，移除复杂的条件判断
- 直接使用传入参数进行订阅，减少无效请求和资源消耗
- 提升订阅代码可读性和维护性

### v1.0.14-saas-20260402

**refactor(registry): 优化nacos注册中心并集成健康检查任务**

- 新增nacos注册中心不健康计数，并实现周期性健康检查任务
- 优化注册与注销逻辑，确保线程安全及防止重复注销
- nacos注册函数支持返回cron定时任务，自动执行健康检测
- 调整watcher逻辑，修复不必要的return影响实例更新处理
- 将ServiceAliveHandler迁移到独立router包，支持服务就绪状态判断
- 微服务启动时集成定时任务，自动添加nacos健康检查
- 将注册中心状态ready管理集成到http组件，实现探活接口状态控制
- 优化consistent_hash和nacos selector轮询相关代码，提升稳定性
- 清理和调整微服务和注册中心相关模块的依赖与初始化顺序

**docs(namespace): 补充命名空间组件说明及示例**

- 为 namespace 包增加命名格式类型注释，明确各类型对应的分隔符和前缀含义
- 在 README.md 中新增 namespace 包文档，详细介绍其主要能力和典型用途
- 对 README 中目录结构进行补充，新增 customer-config 和 namespace 模块说明
- 更新 CHANGELOG 版本号，准备发布新版本，体现对象存储 OSS 类型支持
- 优化 internal/apm/orchestrion 代码注释，提升可读性

### v1.0.14-saas-20260401

**feat(storage): 增加 AWS S3 对象存储支持及命名空间 OSS 类型支持**

- 新增 internal/storage/s3 目录，实现 AWS S3 Provider，支持基本的上传操作
- 实现 storage 的 Registry 管理，支持多实例上传器的注册与调用
- 新增错误定义及 OpError 封装，方便上传操作错误处理和追踪
- 在 library/component/storage 中初始化 ObjectStorage 并封装统一上传接口，支持命名空间前缀自动添加
- 新增大量测试用例，覆盖 S3 上传及组件集成验证
- 添加 etc/cfg.yaml 默认配置，实现对 AWS S3 及 Redis、MySQL 等服务的基础配置
- namespace 包增加 NAMESPACE_TYPE_OSS 类型支持，支持 OSS 类型命名空间格式化及解析
- 优化 ns_provider 对不同类型命名空间处理逻辑，提升代码可维护性和扩展性

### v1.0.14-saas-20260305

fix(redis): 修复类型断言和返回类型不匹配问题

- 修改 redisSyncMap.Load 方法返回类型为 RedisSyncJob
- 修复 Load 方法中类型断言错误导致的潜在 panic
- 调整调用 putWaitSync 时去除指针调用，匹配方法接收者
- 确保键值使用命名空间格式统一处理 context 情况


### v1.0.14-saas-20260304

feat(redis): 支持命令执行时可选择不带租户前缀

- 新增 contextOption 结构体及相关辅助方法管理上下文选项
- 实现 WithoutTPrefix 函数，允许在上下文中标记跳过租户前缀
- 修改 PrefixHookV3，依据上下文选项决定是否重写命令前缀
- 在单元测试中增加不带租户前缀命令执行的用例验证


### v1.0.14-saas-20260303

**refactor(kafka): 优化消费者消息通道结构与并发模型**

- 将 msgChans 从单层 map[int32]chan 扩展为 map[string]map[int32]chan，支持多 topic 消费
- 按 topic 分层管理分区通道，实现单个 ConsumerGroup 支持多个主题的消费隔离
- 改进关闭逻辑，确保所有 topic 和分区的通道均被正确关闭并记录日志
- 增加 topic_partition_channel_size 配置，控制每个分区通道的缓冲区大小，默认值为 1024
- 增加消息拉取超时和错误的日志打印，提升异常可观察性
- 优化 handleMessage 函数逻辑，避免 topic 为空导致消息处理异常
- 消息推送至分区通道时，增加通道满时的预警并阻塞，避免消息丢失
- 新增 Kafka 架构文档，详细描述生产者和消费者工作流及消费者并发模型
- 规范消费者停机收敛流程，确保消费通道关闭顺序与 offset 提交一致

**feat(kafka): 添加日志并优化消费者组消费逻辑**

- consumer_batch 增加批量消费启动和停止时的日志记录
- consumer_confluent 修复 GetCronJob 返回类型为切片，支持多个定时任务
- consumer_confluent 初始化消费者时添加日志输出
- consumer_group 新增 key 字段用于标识实例，消费循环中增加日志及超时处理调整
- consumer_group 单条和批量消费时分别打印详细日志，区分消费分区信息
- consumer_group 优化提交偏移量的处理，使用复合 key 管理等待提交偏移
- consumer_single 修改 singleHandler 接收分区参数，单条消费结束时打印日志
- kafka_test 更新测试配置，调整组ID、主题及地址，符合当前集群环境验证需求

### v1.0.14-saas-20260302

fix(namespace): 修正parseOriginKey去除前缀逻辑错误并添加测试

- 在parseOriginKey中对返回结果的第一个元素使用TrimPrefix去除前缀
- 统一mysql、kafka、redis、common等命名空间类型的处理逻辑
- 防止当resetKey不带前缀时错误返回原字符串
- 新增单元测试覆盖不同命名空间及多样化输入情况
- 确保返回的拆分结果符合预期前缀格式处理

### v1.0.14-saas-20260301

**refactor(redis): 优化Redis同步任务与命名空间处理逻辑**

- 重构RedisSyncMgr，改用带上下文命名空间格式化的同步任务映射
- RedisSyncJob新增ctx字段，支持多租户上下文信息
- 同步任务添加、执行及变更监听均适配命名空间上下文
- consistent_hash.OnChange改签名，增加是否首次监听标识
- 删除Redis中FmtNamespaceKey旧格式化函数，改用namespace库实现
- 新增Redis XAdd接口简化流数据写入操作
- 优化mysql雪花ID生成器初始化与停止时机，避免未初始化即停止
- namespace包新增获取原始键及上下文的解析函数，支持多种类型键格式
- 调整测试代码，添加上下文命名空间支持，确保一致性
- 修正部分调用签名，清理未使用import，提高代码规范和健壮性

### v1.0.13

**feat(mysql/callback): 增强数据库测试与 SQL 解析工具**

- 引入用于测试数据库交互的 mock 连接池
- 为多种 SQL 解析函数补充单元测试，涵盖规范化、拆分及库表名重写等能力
- 增强 schema 前缀访问器，在租户命名空间重写时同时处理表名与列名
- 提升复杂 SQL 与多租户数据库相关场景的测试覆盖率

### v1.0.12

**feat(cache): 新增本地缓存实现及命名空间支持**

- 新增基于 go-cache 的本地缓存实现 LocalGoCache
- 定义了 LocalCache 接口及相关操作方法
- 提供本地缓存初始化方法 NewLocalCache 支持自定义过期时间及清理间隔
- 增加命名空间格式化支持 NAMESPACE_TYPE_COMMON，用于缓存键的统一管理
- 命名空间工具中新增 GetOriginKey 方法，支持获取除商户信息外的原始键
- 新增缓存条目结构体 Item，包含对象和过期时间
- 添加本地缓存单元测试覆盖基本缓存操作的行为
- 完善命名空间相关的单元测试，覆盖新加逻辑
- 新增本地缓存使用示例文档说明文件 doc.go


### v1.0.11

**feat(mysql): 优化多库表名重写及SQL解析支持**

- 新增 TableDBResolver 用于关联 JOIN 时裸表名反查数据库名
- 增强 Query 拦截器，使用 TiDB SQL parser 实现复杂语句重写
- 重构库名重写逻辑，防止重复前缀并支持排除系统库
- 支持结构化Clause及RAW SQL统一改写，提升改写准确性
- 增加子查询、JOIN、WHERE等多场景库名重写递归处理
- 新增单元测试覆盖多表联查、子查询、更新、删除、执行等场景
- 修复并优化schemaPrefixVisitor，确保AST正确重写库名前缀
- 调整mysql包初始化参数，默认开启原生SQL解析兜底模式
- 添加关联查询测试示例，验证JOIN时的库名重写正确性

### v1.0.10

**feat(db): 实现多租户数据库名自动重写插件**

- 新增租户数据库操作插件，支持多种 GORM 回调事件的数据库名重写
- 通过配置 NamespaceResolver 获取租户命名空间，实现多租户数据隔离
- 优先改写结构化查询的数据库表名，兼容复杂 SQL 语句的重写
- 支持原生 SQL 的解析和重写，开启 EnableRawSQL 选项启用
- 子查询及子表达式中的库表名自动重写
- 旧的租户回调替换为插件化实现，增强灵活性和扩展性
- 完善单元测试，覆盖多种查询和插入场景的新租户插件
- 日志系统获取上下文附加信息改为统一调用 getZap 函数，简化代码
- 修复日志打印时局部变量未使用问题，日志打印更统一规范
- 更新 Kafka 测试配置，改用新的私网地址及调整偏移参数
- 添加脚本 remote sync 配置文件，支持特定项目的远程同步管理

### v1.0.9

feat(redis): 添加雪花ID Redis实例支持

- 新增常量 REDIS_CLIENT_NAME_SNOWFLAKE 表示雪花ID的Redis客户端名称
- 实现 SnowflakeOrDefault 函数，获取雪花ID专用Redis实例，若未配置则使用默认实例
- 修改雪花ID生成器初始化，改为使用 SnowflakeOrDefault 提供的Redis客户端
- 在MySQL的测试配置中添加雪花IDRedis集群配置信息
- 修改MySQL测试用例中获取数据库连接时指定使用“game”库
- 注释MySQL测试配置中关于雪花ID生成器的部分代码避免自动启用

### v1.0.8

feat(mysql): 优化雪花ID初始化和分配逻辑

- 修复了重复数据中心检测的错误提示信息
- 添加了默认雪花ID初始化标志，避免重复初始化默认数据中心
- 改善了雪花ID初始化函数的逻辑结构
- 修改GORM回调插件注册逻辑，确保无论是否有注册都执行回调
- 优化雪花ID生成时根据配置选择数据中心的逻辑
- 重构了根据实体类型分配雪花ID的处理流程，支持多种类型映射
- 在测试配置中添加了game库的雪花ID相关配置示例
- 注释掉了部分无效或未使用的代码，提升代码可维护性

### v1.0.7

**fix(redis): 统一测试用例中 Redis key 前缀为 tenantok**

- 将所有 Redis key 和相关命令的测试期望值前缀从 "ok:" 改为 "tenantok:"
- 修改 PrefixHookV3 相关逻辑，调用 namespace.FmtByType 方法格式化 key 前缀
- 优化 rewriteCmdWithPrefix 函数，调整前缀拼接逻辑
- 将 kafka consumer 的最大处理时间由 3000ms 调整为 30000ms，增加处理容忍度

**feat(redis): 增加模式订阅相关功能接口及模拟实现**

- 新增 PSubscribe、PSubscribeC、SSubscribe、SSubscribeC 订阅接口支持
- 在 redis.go 文件添加对应方法实现，支持模式和通配符订阅
- 在 MockRedisInterface 中添加上述新方法的 mock 实现，便于单元测试使用
- 优化 redis 连接获取失败情况下日志记录，避免 panic
- 修改 kafka 消费者相关代码，优化 namespace 处理逻辑
- namespace 模块增加从 key 中解析和设置 namespace 的辅助方法，统一 namespace 格式化规则
- 多租户模块优化配置解析，增加空配置警告和默认 tenant 上下文处理
- 对本地调试环境中 tenants 初始化逻辑进行优化，避免 tenants 为空问题


### v1.0.6

**fix(redis): 支持更多Redis命令的key前缀重写规则**

- 新增LMOVE、BLMOVE、LMPOP、BLMPOP等列表命令的key重写支持
- 增加SUNION、SINTER、SDIFF、SUNIONSTORE等集合命令的key重写
- 支持ZINCRBY、ZCOUNT、SSCAN、ZLEXCOUNT、ZSCAN等有序集合和位图命令的key处理
- 新增对Redis Stream相关命令（XLEN、XRANGE、XREVRANGE、XDEL、XTRIM、XGROUP、XINFO）的前缀重写
- redis连接权重修改为-1，确保优先级最低，最后关闭

**fix(mysql): 优化mysql连接获取失败的日志记录方式**

- mysql连接获取失败时使用日志记录替代panic，避免服务崩溃
- 保持返回nil以防止因错误导致后续调用异常
- 将数据库连接池状态检查定时任务独立为函数，提升代码结构清晰度
- 定时任务每10秒执行一次，检查所有多数据库连接池状态，并打印详细统计信息

**feat(redis): 添加支持多租户的Redis命令前缀自动重写功能**

- 新增PrefixHookV2结构体，实现redis v9客户端命令参数自动加前缀
- 支持对普通key命令、发布订阅channel及pattern命令加前缀处理
- 根据不同命令类型自动识别需要加前缀的参数位置
- 提供ProcessHook和ProcessPipelineHook接口支持单条及批量命令的处理
- 实现applyPrefix函数，避免重复添加前缀
- 添加对EVAL、XREAD、XREADGROUP等复杂命令参数的解析支持
- 修改测试用例支持新版UniversalClient接口
- 编写完整单元测试覆盖各种redis命令及特殊场景，确保前缀正确注入

### v1.0.5

**feat(http): 添加租户命名空间中间件支持**

- 在 gin 中新增 ParseNamespace 中间件解析请求头中的租户命名空间
- 通过 X-Tenant-Name 请求头设置上下文中的命名空间数据
- 在 gin_http 中集成 ParseNamespace 中间件以自动处理租户命名空间
- 修改 Kafka 消费者主题匹配规则，支持使用下划线分隔的主题格式

### v1.0.4 

**chore(mocks): 删除自动生成的测试mock文件**

- 移除library/component/lark-message包下的mock实现文件
- 移除library/component/throttle包下的完整mock实现文件
- 移除library/consistent_hash/hash_circle包下的mock实现文件
- 清理不再使用的自动生成测试代码，减小代码体积
- 减少维护成本，避免mock代码与实际接口不一致问题

### v1.0.3

**feat(mul-tenant): 添加本地调试的租户配置初始化**

- 在本地运行环境通过环境变量加载租户配置
- 使用逗号分割环境变量字符串得到多个租户信息
- 将租户信息设置到上下文中便于后续使用
- 初始化时自动完成租户上下文的准备工作

**refactor(namespace): 统一实现命名空间格式化逻辑及应用**

- 新增 namespace 包，包含针对 MySQL、Redis、Kafka 的命名空间格式化函数
- 使用 namespace.FmtByType 替代原先的手写字符串拼接，实现统一格式化
- Kafka 相关代码移除冗余的字符串处理函数
- MySQL 回调和 Redis 缓存模块中应用统一命名空间格式化逻辑，提升代码一致性
- Kafka 消费和推送代码集成命名空间处理，简化命名空间解析流程
- 测试代码更新，改用统一的 Redis 连接获取方式，调整 Redis 集群配置参数

### v1.0.1

**fix(snowflake): 修复雪花ID初始化逻辑及错误日志处理**

- 增加获取自定义配置时的类型错误日志输出
- 优化雪花ID初始化流程，增加已初始化标志变量 snowflakeFlag
- 改进默认数据中心初始化逻辑，确保中心过滤器正确更新
- 移除无效的配置判断，提升代码健壮性
- 注册GORM回调时添加错误堆栈信息反馈

### v1.0.0


**feat(mul_tenant): 实现多租户命名空间配置监听功能**

- 新增mul_tenant包，实现多租户命名空间相关操作
- 支持通过Nacos加载和监听租户配置信息，实现热更新
- 通过环境变量控制是否开启监听商户命名空间配置
- 在env包中新增获取命名空间及监听开关的环境变量支持
- 集成mul_tenant的监听启动至pmicro服务配置初始化
- 新增namespace包，提供获取当前服务命名空间下租户的接口
- 添加相关单元测试和示例代码验证多租户配置监听功能
- 优化nacos相关代码的错误处理和测试用例稳定性

**feat(trace): 集成 DataDog APM 和 Orchestrion 分布式追踪**

- 引入 DataDog dd-trace-go 库及其相关插件
- 新增 orchestrion 包，实现基于 orchestrion 的 traceId 和 spanId 获取功能
- 统一 trace 包接口，支持多种 trace 类型（default、skywalking、orchestrion）
- 更新 context traceId 处理，支持携带 spanId 信息
- 在 Redis 客户端增加 DataDog trace 包装，支持链路追踪
- Gorm 日志、gin http、grpc 服务等日志输出新增 spanId 字段
- kafka producer header 中 traceId 改为 mq_trace_id 兼容 DataDog
- 优化 ctx 数据写入，支持 gin 上下文与标准 context 兼容
- 切换 eonid 依赖至 v2 版本，更新相关代码引用
- 移除旧的 skywalking 依赖与代码，替换为 DataDog 及 orchestrion 方案
- mocha、mysql 配置调整和注释更正，提升整体代码规范和稳定性


**refactor(db): 重构数据库SSL配置实现**

- 移除PostgreSQL相关依赖包导入
- 修改MySQL SSL配置逻辑，从内联证书内容改为读取证书文件路径
- 更新PostgreSQL连接配置为使用标准连接字符串格式
- 调整SSL配置结构体，将ca、cert、key字段合并为path字段
- 更新测试配置中的数据库连接参数和SSL配置方式
- 在README文档中更新SSL配置示例和Kubernetes部署说明

**feat(kafka): 添加Kafka消费者SSL安全认证配置支持**

- 在confluentGroupConfig中增加SaslMechanisms和SecurityProtocol字段
- 更新测试配置使用真实的Kafka集群连接参数
- 配置用户名密码及SCRAM-SHA-512安全协议
- 修改默认生产者名称为ca-msk-demo
- 更新主题名称为topic-test_dev
- 添加定时任务用于测试消息推送功能
- 实现MySQL SSL配置的日志对象序列化


**feat(db): 添加数据库SSL配置支持**

- 引入crypto/tls和crypto/x509包支持SSL连接
- 添加pgx/v5依赖库用于PostgreSQL SSL连接
- 修改initConnect函数增加instance参数
- 实现getDialector函数返回错误类型
- 添加initMysqlDialector函数支持MySQL SSL连接
- 添加initPostgresDialector函数支持PostgreSQL SSL连接
- 在DbConfig和DbConfigOther结构体中添加SSLConfig字段
- 实现SSL配置的CA证书、客户端证书和密钥配置
- 支持MySQL和PostgreSQL的TLS连接配置

**refactor(redis): 移除已弃用的EvalShaOrScript方法**

- 从Redis接口中删除EvalShaOrScript方法定义
- 移除Redis实现中的EvalShaOrScript方法及其相关逻辑
- 删除SHA1哈希计算和脚本缓存相关代码
- 清理mocks中对应的模拟方法实现
- 简化Redis facade接口定义

**feat(library/gin): 添加自定义参数校验证器接口**

- 引入 CValidator 接口用于自定义参数校验
- 在 Validate 函数中集成 CValidator 验证逻辑
- 实现 DoValidate 方法支持自定义校验规则
- 保持原有 Validator 接口兼容性
- 提供 INVALID_PARAM 错误码统一处理机制

**feat(redis): 重构Redis日志钩子实现并改进日志控制机制**

- 添加debug字段到logHook结构体以支持调试模式控制
- 修改logHook方法接收器为指针类型以支持内部状态访问
- 将WithoutRedisLog函数重命名为WithRedisLog并修改逻辑实现
- 使用结构体类型作为context键值替代字符串类型
- 更新isLogEnabled方法为logHook的方法并优化日志启用判断逻辑
- 调整Redis客户端初始化中的日志钩子配置方式
- 修改测试配置中的Debug默认值为false
- 添加TestRedis_withLog测试函数验证日志功能

**refactor(metadata): 重构元数据管理功能**

- 移除 CONTEXT_PHONE 相关字段和处理逻辑
- 为常量添加 string 类型声明
- 添加 Other 字段支持自定义元数据存储
- 实现 ToMicroMetadata 和 FromMicroMetadata 方法支持微服务元数据转换
- 添加自定义元数据前缀 CONTEXT_CUSTOM_PREFIX 支持
- 优化 SetCtxData 函数避免不必要的 context 重建
- 更新文档注释和完善函数说明
- 移除 parsePhone 相关逻辑和依赖

**feat(http): 添加HTTP服务器配置和优化中间件**

- 添加HttpServerCfg配置结构体，支持读写超时、空闲超时和请求头大小限制
- 在生产环境自动设置Gin为Release模式
- 将IP请求头配置提取为全局变量，统一管理CloudFlare相关头信息
- 移除日志中间件中的响应体捕获功能以提升性能
- 注释掉IP获取失败时的日志警告，避免不必要的日志输出


**feat(kafka): 扩展生产者消息类型兼容性**

- 支持字符串、字节数组和 JSON 原始消息类型的直接传递
- 使用类型断言替代简单的字符串检查
- 添加了对 json.RawMessage 类型的原生支持
- 保持了对其他类型的 JSON 序列化回退机制
- 优化了消息转换的性能和类型安全性

**feat(tracer): 支持多种trace id头部字段**

- 添加了对 X-Trace-Id、Trace-Id 和 Traceid 头部字段的支持
- 实现了从请求头部获取现有trace id的逻辑
- 当请求中没有trace id时生成新的uuid作为trace id
- 将获取到的trace id设置到上下文中供后续使用
- 统一了trace id的管理和传递机制


**feat(auth): 添加 JWT 认证支持**

- 新增 grpc_jwt.go 文件实现 JWT token 解析功能
- 添加 AuthGrpcClaims 结构体用于存储认证信息
- 实现 ParseGrpcJwtToken 函数解析和验证 JWT token
- 在 auth_server.go 中添加 JWT 认证类型处理逻辑
- 实现 jwtCheck 函数支持多密钥轮询验证
- 修改 client wrapper 支持 JWT 认证类型
- 添加单元测试验证 JWT 解析功能


**refactor(kafka): 优化Confluent Kafka消息推送逻辑与测试代码**

- 替换旧InitKafka调用为InitConfluentKafkaManager以统一管理
- pushMessageWithProducer方法新增参数校验和简化逻辑
- 抽离消息构造到getKafkaMsg方法，支持string类型特殊处理
- 处理本地环境topic自动添加后缀避免误消费
- 新增msgMarshal函数及单元测试覆盖多种数据类型序列化
- 测试代码中替换Push调用为mgr.Push以使用管理器实例
- 日志统计改为调用mgr获取生产统计数据接口

**refactor(gorm): 替换旧版gorm依赖为官方gorm模块**

- 统一替换所有代码中的 github.com/coderyw/gorm 为 gorm.io/gorm
- 替换相关驱动和插件依赖地址为 gorm.io/driver/mysql、gorm.io/driver/postgres 及 gorm.io/plugin/dbresolver
- 更新 go.mod 和 go.sum，移除旧版gorm相关依赖，添加官方gorm最新版本依赖
- 修正相关包引用导入路径，确保代码兼容官方gorm库
- 注释和删除部分旧版gorm相关代码实现及调用
- 新增和完善部分测试代码，使用官方gorm接口进行数据库操作验证
- 修正部分日志和错误处理代码中gorm包的导入和使用

**perf(log): 移除HTTP中间件中的请求响应体日志记录**

- 注释掉最大body长度限制配置
- 移除请求body读取和打印功能
- 移除响应body读取和打印功能
- 减少日志输出大小提升性能
- 避免敏感数据在日志中暴露

**feat(log): 添加不记录响应日志的功能并优化日志中间件**

- 新增 SetResponseWithNoResponseLog 函数用于设置响应值且不打印响应日志
- 优化 Tiny 结构体的 MarshalLogObject 方法，正确输出字段值
- 修改日志中间件移除原始查询参数记录
- 为 NsHttpResp 结构体实现 MarshalLogObject 接口支持结构化日志输出


refactor(log): 统一日志字段类型提升日志性能

- 将所有zap.Any字段替换为具体类型如zap.String、zap.Int、zap.Int64等
- 修复consistent_hash模块中nodeIdx和current字段的日志类型
- 修复cron模块中name字段的日志类型
- 修复redis模块中多个字段的日志类型包括syncKey、key、values等
- 修复mysql模块中连接池统计信息的日志类型
- 修复http模块中code、body、Duration等字段的日志类型
- 修复gorm日志中Duration和SQL相关字段的日志类型
- 优化日志序列化性能，避免不必要的类型转换

refactor(redis): convert Redis methods to pointer receivers

- 将 Redis 结构体的方法接收者统一改为指针类型以统一操作和避免拷贝
- 修正cache测试中GetRedis的类型断言为指针类型
- 优化锁相关方法，改为指针接收者，提升性能与一致性
- 统一redis.go中所有Redis方法的接收者为指针类型，改进资源管理与调用效率
- 保持原有功能不变，提升代码可维护性和扩展性

### v0.5.2

fix(cache): 修复Redis缓存加载逻辑中的错误处理

- 调整包导入顺序，将标准库包移到第三方包之前
- 修复loadCache调用后的错误检查逻辑
- 当loadCache返回redis.Nil错误时，继续执行从数据库加载逻辑
- 避免因target为空但无错误时的提前返回

chore(docs): 更新版本变动记录

- 在README中添加v0.5.2版本变更记录
- 为新版本预留文档空间

修改文件
- library/db/cache/redis/cache.go GetFromCacheOrDb
```go
// 319行
target, err = loadCache(ctx)
if err != nil {
if !errors.Is(err, redis.Nil) {
return err
}
// 错误是redis.Nil，则从数据库加载
} else if !IsEmpty(target) { //target非空，则直接返回
return nil
}
```
主要增加在获取到分布式锁后，重试从 Redis 获取数据时，对redis.Nil错误进行特殊处理

### v0.5.1

feat(auth): 实现基于endpoint的认证机制

- 新增服务端和客户端认证配置结构体AuthServer和AuthClient
- 支持基于endpoint的精确匹配和前缀匹配认证规则
- 实现认证密钥的版本管理和新旧密钥切换机制
- 添加强制认证和非强制认证两种模式
- 在服务端handler中集成认证逻辑，支持basic认证方式
- 在客户端wrapper中实现认证信息注入
- 更新配置文件结构以支持多认证规则配置
- 升级ca-common-model依赖版本至v0.2.37
- 新增自动化测试环境和预发环境

### v0.4.61

feat(metadata): 添加设备ID字段支持

- 在上下文常量中新增 CONTEXT_DEVICE_ID 字段
- 扩展 Metadata 结构体以支持 DeviceId 属性
- 更新 ToMicroMetadata 方法以包含设备 ID 映射
- 在日志元数据过滤器中加入设备 ID 白名单
- 实现设备 ID 的解析与获取逻辑

### v0.4.60

fix(metadata):修复上下文元数据解析错误返回空context导致panic问题

- 修改 FromMicroMetadata 错误处理逻辑，返回原始 context 防止 panic
- 添加 TestMapStruct 测试用例验证 mapstructure 解码功能
- 确保元数据解析失败时不会丢失原有的 context信息

### v0.4.58

feat(mysql): 增加数据库连接空闲时间配置

- 新增 MaxIdleTime 配置项，用于设置连接最大空闲时间
- 默认设置 MaxLifeTime 为1 小时
- 默认设置 MaxIdleTime 为 5 分钟
- 在数据库配置结构体中添加相应字段及注释
- 实现 SetConnMaxIdleTime 方法以应用空闲时间设置

### v0.4.57

- mysql的事务增加 mock 方法
- 元数据新增 version 和 AppVersion

### v0.4.56

- 增加菲律宾手机号验证方法

### v0.4.55

fix(http): 更新中间件中的上下文数据设置方法
- 将 metadata.SetCtxData 替换为 metadata.SetGinCtxData
- 调整导入包顺序并优化日志记录字段
- 使用 zap.Any 记录完整请求头代替 skywalking trace id

### v0.4.53

fix(http):修正客户端IP获取的请求头名称

- 将 "True-UniversalClient-IP" 更正为 "True-Client-IP"
- 确保从 CloudFlare 正确获取客户端 IP 地址
- 保持与其他标准 IP 请求头的一致性

### v0.4.52

- 修复获取 ip 错误情况

- ```
  优化注册中心健康检查逻辑
  
  将原有的简单服务端健康状态判断，改为通过实际调用 SelectInstances 接口进行服务实例查询来验证注册中心状态。  
  增加重试次数限制，避免无限 panic 导致程序异常退出。调整检查间隔时间为10 秒，并引入计数器控制连续失败次数，提升系统稳定性。
  ```

### v0.4.51

fix(hash):修复哈希环节点删除逻辑与索引计算
- 修复节点删除时的索引判断条件
- 移除未使用的 hashCircleStart 字段
- 调整 startIndexCul 方法中的索引计算逻辑
- 更新 Gets 方法中的节点索引计算方式
- 修改 GetAll 方法中的节点分配逻辑
- 简化测试用例中的数据规模和循环次数
- 添加调试日志以便追踪节点分配过程

### v0.4.50

```
feat(redis): 新增 LPopCount 和 RPopCount 接口及实现

新增 RedisInterface 接口中的 LPopCount 和 RPopCount 方法，用于支持批量弹出操作。同时更新了 mock 文件以支持新接口的测试，并升级了 go-redis 版本以兼容新功能。
此外，调整了 import 包顺序并修复了 mock 类型断言逻辑。
```

### v0.4.49

feat(library/mq/kafka): 修改消费者初始偏移量设置

- 将消费者初始偏移量从 OffsetOldest 修改为 OffsetNewest
- 这一变更可以避免消费者在启动时读取到过期的消息

### v0.4.48

refactor(library/cron): 优化 cron 任务调度器的初始化逻辑

- 在包级别初始化 cron 实例，避免多次初始化
- 检查 cron 实例是否已存在，防止重复创建
- 更新 README.md，添加新的版本号和变更说明


### v0.4.47

refactor(library): 重构部分组件并优化错误处理

- 重命名 mock 文件夹为 facade，更新相关的 import 路径
- 优化 Redis 测试用例中的 mock 对象使用
- 改进 Kafka 消费者中的消息处理逻辑，增加停止消费的处理
- 更新 gRPC 客户端中的错误处理，增加日志记录和重试逻辑优化
- 修复 micro 服务框架中的错误处理，增加对 unavailable 错误的处理

### v0.4.46

perf(kafka): 修改消息发送确认模式

- 将 RequiredAcks 从 WaitForAll 改为 WaitForLocal
- 这一变更可以减少等待时间，提高消息发送效率
- 适用于对消息可靠性要求较高但对实时性有一定容忍度的场景

### v0.4.45

perf(kafka): 修改消息发送确认模式

- 将 RequiredAcks 从 WaitForAll 改为 WaitForLocal
- 这一变更可以减少等待时间，提高消息发送效率
- 适用于对消息可靠性要求较高但对实时性有一定容忍度的场景


### v0.4.44

feat(http.middleware): 增加接口日志过滤功能

- 在日志中间件中添加了对特定接口不打印日志的功能- 通过调用 canLog函数来判断是否需要打印日志
- 如果不需要打印日志，则直接调用 c.Next() 继续执行后续中间件和处理器

### v0.4.43

refactor(library/db/mysql): 重构 Mock 数据库连接逻辑

- 移除了对 env 模块的依赖
- 重构了 DB函数中的 Mock 逻辑，使用新的 CheckAndGetMockClient 方法
- 添加了 testing 包的导入

### v0.4.42

- 增加 Mock

### v0.4.41

refactor(http): 优化 HTTP客户端和服务器的功能

- 修改 HTTP 客户端请求体处理逻辑，支持字符串和 JSON 数据
- 优化 HTTP 服务器跨域请求处理，提高性能和可读性
- 添加 singleflight测试用例，验证并发控制功能
- 更新 go.mod 文件，直接依赖 golang.org/x/sync

### v0.4.40

feat(component/limiter): 优化令牌桶限流器并添加 IP 限流支持

- 重构了 TokenBucketLimiter 的 Consume 方法，提高了性能和可读性
- 添加了对 IP 限流的支持，可以通过传入 true来启用 IP 限流
- 优化了 Lua 脚本的加载和执行逻辑，提高了 Redis 操作的效率
- 修改了返回结果的处理方式，更加准确地反映了允许或拒绝请求的状态

### v0.4.39

feat(util/excel): 优化 Excel 流式写入功能

- 新增 StreamStartIndex 字段，用于设置流式写入的起始索引
- 修改 WriteArray 和 BatchWriteArray 方法，使用内部索引进行写入
- 增加内部索引的初始化和自增逻辑
- 更新测试用例，验证新功能的正确性


### v0.4.38

fix(registry/nacos): 修复服务注册元数据传递问题

- 之前直接将服务节点的元数据赋值给注册参数，可能导致潜在的引用问题
- 现在通过复制元数据的方式，确保注册参数的独立性和安全性

### v0.4.37

- 移除了消费者通道满时的警告日志
- 添加了 Kafka 消费者功能测试
- 更新了 README.md 版本变动说明

### v0.4.36

- 生产 jwt Token 时候使用外部传递过期时间参数

### v0.4.35

- 调整了 ExcelFile 结构初始化的逻辑，确保 sort 字段正确设置
- 修改了 BatchWriteArray 的调用参数，从 1 修改为 2，可能影响批量写入的性能

### v0.4.34

- 在多个文件中添加了 Deprecated 注释，提示使用 common-model 替代
- 保留了原有函数的实现，以确保向后兼容性
- 此修改旨在引导开发者迁移到新的 common-model 库，而不是继续使用 e-error

### v0.4.33

- 在 Kafka 生产者配置中添加了 SASL 认证支持
- 当配置了用户名时，启用 SASL 认证并设置相关参数
- 使用明文机制进行 SASL 认证


### v0.4.31

- 在日志中间件中添加 skywalking traceId 字段
- 修改 tracer 中 traceId 的创建和传递逻辑
- 更新 wrapper 中 traceId 的处理方式
- 优化 TracePro 中 traceId 的获取逻辑

### v0.4.30

- 在日志中间件中添加 skywalking traceId 字段
- 修改 tracer 中 traceId 的创建和传递逻辑
- 更新 wrapper 中 traceId 的处理方式
- 调整 trace_pro 中 traceId 的获取逻辑

### v0.4.29

- 在响应中添加 MSG_CODE_MSG 字段，用于区分错误消息和业务消息
- 重构错误处理逻辑，使用 GrpcCode2Error 方法统一处理 gRPC 错误
- 优化日志记录，增加 gRPC 响应头信息的记录
- 调整 Tracer 中的 TraceId 记录级别为 Debug

### v0.4.28

- 在 tracer 中添加了生成唯一请求 ID 的功能
- 使用 uuid.NewString()生成唯一 ID
- 在 trace2.CreateEntrySpan 中添加了返回唯一请求 ID 的逻辑
- 此修改有助于在微服务架构中跟踪和监控请求

### v0.4.27

- 使用 uuid 生成唯一的请求 ID
- 在 trace2.CreateEntrySpan 中添加请求 ID 作为返回值
- 此修改有助于在微服务架构中跟踪和监控请求

### v0.4.26

- Excel 文件写入优化
    - 移除了泛型参数，简化了 Excel 结构体定义
    - 添加了 WriteArray、BatchWriteArray 和 WriteMap 方法，提高了灵活性和效率
    - 优化了内部实现，使用 StreamWrite 增加写入性能
    - 更新了单元测试，覆盖了新的功能

### v0.4.25

- 删除了 library/mq/rabbitmq 目录下的所有文件
- 移除了与 RabbitMQ 相关的初始化和消费逻辑
- 优化了 library/mq/mq.go 中的接口定义
- 在 library/mq/kafka/consumer.go 中添加了 SASL 认证的相关配置
- 在获取 trace ID 时，如果为空则生成新的 UUID 作为 trace ID
- 使用 google/uuid 包生成唯一 UUID
- 优化了中间件逻辑，确保每个请求都有唯一的追踪标识

### v0.4.15

- 修改处理耗时日志标识为兼容方式
- 统一全局耗时日志标识key名

### v0.4.14

- http请求解析错误时候打印日志
- grpc处理错误时候，不打印error堆栈

### v0.4.13

- 修复 非 skywalking 的trace 传递失败

### v0.4.12

- 元数据转换通过 mapstruct
- metadata 修复 grpc 传递失败

### v0.4.11

### v0.4.10

- 修改元数据调用方法

### v0.4.9

- jwt 校验增加不校验合法性参数

### v0.4.8

- jwt 解析时候，userId 使用 json.Number 兼容

### v0.4.6

- jwt增加过期时间配置
- 元数据增加 terminal

### v0.4.4

- 增加 JWT 组件
- 打通 全链路上下文参数传递

### v0.4.3

- 增加根据 cgroup 限制的 cpu 核心数配置最大的 GOMAXPROCS
- 升级 nacos 客户端版本

### v0.4.1

- 优化 redis 脚本执行
- 优化 kafka 消费逻辑，增强打印
- 增加统一的停服监听通知

### v0.4.0

- trace 优化
- 增加lark 机器人通知

### v0.3.7

- trace 链路增加子链路
- Http handler 增加忽略默认Response middleware

### v0.3.6

- 添加【MySQL 唯一约束冲突错误判断方法】

### v0.3.5

- 增加系统耗时等级字段
- 增加 gin http handler 处理耗时级别日志

### v0.3.4

- 修复分分布式锁枷锁错误

### v0.3.3

- 增加平台配置项,token salt配置项

### v0.3.2

- 兼容 解锁 lua 脚本为字符串

### v0.3.0-0427

- 修改 mysql ping

### v0.3.0

- redis 不在向前兼容
- http 日志增加不需要打印的接口过滤

### v0.2.33

- Gin打印优化，增加执行时间打印，以及执行时间超过阈值打印
- Gorm 日志增加当前执行 sql 数据打印：连接数，空闲连接数，等待连接数量。
- 优化 skywalking 链路
- kafka 增加 sky 链路

### v0.2.32

- 增加分布式锁打印
- redis 获取 hm 缓存去掉 easymap 方式

### v0.2.31

- 增加对华为云 DDM 连接兼容

### v0.2.30

- 去掉mysql后缀

### v0.2.29

- 增加mysql连接配置databaseSuffix，用于配置mysql数据库的后缀

### v0.2.28-1

- 修复skywalking无法通过gin传递到gomicro

### v0.2.28

- 修复多个数据库连接时候雪花id配置有问题

### v0.2.27-2

- Sql日志优化
    - 每个类型日志上增加时间
    - 去掉warn日志上的err打印

### v0.2.27-1

- redis从db加载函数移除json接口
- kafka停服不使用close接口

### v0.2.27

- 修改redis方法，移除掉json接口
- 修改kafka停服时候顺序

### v0.2.26

- http请求增加Get请求
- 停服时候增加停止顺序
- gin HTTP监听增加错误分类处理

### v0.2.25

- 邮件增加一次性发送多个收件人

### v0.2.24

- http请求增加对空的返回处理

### v0.2.23

- 修改停服时候组建停止顺序
- http请求增加对空的返回处理
- gin 增加请求返回值打印

### v0.2.22

- 缓存增加通用方法：缓存加载失败时候从数据库加载

### v0.2.21

- 兼容数据库多个实例时候公用同一个名称的雪花id数据中心

### v0.2.20

- 增加excel接口注释
- 增加跳表算法实现`util/sort_queue/skiplist`
- 跨域头增加请求头`grayscale`
- 增加基于跳表的有序集合实现`util/sort_queue/zset`
- 增加`options`新的参数：`OtherUnmarshal`；用于自定义反序列化配置
- 修复 grpc 中间件 panic 处理时候没法抛出 error

### v0.2.19

- 增加excel写入包装方法

### v0.2.18

- grpc请求和消息回复中间件移除对老错误码兼容，默认使用新错误码

### v0.2.17

- 数据库实例获取方法增加新的方法。可以直接传入上下文，使用传入的上下文作为新的实例

### v0.2.16

- 延迟队列发布消息移出lua脚本外执行

### v0.2.15

- 延迟队列消息增加延迟时间返回

## 一. library组件包

library包含了各种组件包，以及微服务gomicro包

- **beanstalk**

  极其轻量的延迟队列组件，server使用c写，客户端可通过go连接处理。

    - 优点：轻量，便携。支持持久化binlog
    - 缺点：server端单机，无法保证高可用。

  [了解详情](https://zhangquan.me/2021/06/29/beanstalkd-xue-xi-yan-jiu/)

- **component/ali-sms**

  阿里云短信发送组件

  使用方式：

  ```go
  // 直接调用方法
  ali_sms.SendAliVerCodeSms("13778238223", "123456")
  ```



- **db/cache/redis**

  redis缓存组件

  开启方式：

  ```yaml
  storage:
    redis:
      enable: true
      host:
        - 'redis.example.com:6379'
      password: ''
      db: 0,
      # 如果redis server是集群模式，且没有配置proxy，则需要开启is_cluster=true
      is_cluster: false
  ```

  使用方式：

  ```
  // 直接引用redis包
  redis.GetRedis().GetValue("prefix", "key")
  ```

- **db/mysql**

  mysql/postgresql组件

  开启方式：

  ```yaml
  storage:
    mysql:
      enable: true
      host: 'mysql.example.com'
      port: '3306'
      user: datamanager
      password: 'password'
      database: 'db_game_color'
      debug: true
      open: -1
      # 如果需要配置分表，则需要下面的配置
      sharding:
        - table_name: 'b_color_order'
          sharding_key: 'sys_user_id'
          sharding_num: 100
  ```

  使用方式

  ```go
  // 直接引用mysql包
  u := new(User)
  mysql.Conn().Model(User{}).Select("id").Where("id=?", 123).First(u)
  
  //开启大事物
  mysql.Conn().Transaction(func(tx *gorm.DB) (err error) {
    tx.Model(User{}).Select("id").Where("id=?", 123).First(u)
  }
  ```

- **e-error**

  通用错误码与error解析组件，支持通用错误码解析到error，error解析到通用错误码

- **env**

  环境变量包

- **jaeger**

  jaeger链路追踪组件

- **log**

  日志组件

  ```
  // 直接引用
  log.Info("哈哈哈", zap.Any("key", "value"))
  ```

- **mq/kafka**

  kafka组件

  开启：

  ```yaml
  kafka:
  	# 如果需要生成者，需要开启这个配置
  	# 消费者根据代码是否传入handler判断是否开启
    enable_producer: true
    host:
      - kafka.example.com:29092
  ```

  使用：

  ```go
  // 需要在服务main配置kafka消费handler
  option.MQHandler(cst.TOPIC_COLOR_ORDER, internal.InitColorGameLogic().HandleColorOrderMq))
  // 具体示例如下pmicro
  ```



- **mq/rabbitmq**

  rabbitmq组件

- **pmicro**

  系统组件，包含微服务，http。

  服务启动条件

    - 环境变量 `ENVIRON=server-local`

    - 环境变量(可选) `NACOS_URLS=127.0.0.1:8848`,如果配置了这个地址，则使用nacos作为配置中心，否则使用本地的配置文件

    - 配置文件

      ```yaml
      # 服务名
      server_name: ''
      # 运行环境 server-local|server-dev|server-test|server-pre|server-prod
      environ: ''
      # 程序版本
      version: ''
      
      # 注册中心
      registry:
        # 注册中心类型: etcd|nacos|k8s
        type: ''
        hosts: null
        account: ''
        pwd: ''
        name_id: ''
        # 是否开启istio服务调用
        # 注册中心type和istio可以同时开启，同时开启时候注册中心只会去对应type注册服务，本服务调用则会通过istio服务网格方式调用
        # 如果只配置了注册中心type，则注册和发现都会通过注册中心实现
        # 如果只配置了istio，则注册（这里的注册只是名词，而不是动作，即不会去做任何注册的动作）和发现都会通过istio服务网格方式调用
        enable_istio: false
      
      # grpc服务相关配置
      rpc_server:
        listen_on: ''
      
      # log 日志配置
      # level 日志登记，只会打印大于等于level级别的日志
      # -1=debug 0=info 1=warn 2=error 3=dpanic 4=panic 5=fatal
      log:
        level: 0
        stacktrace_level: null
        #  开发模式
        development: false
        #  DisableStacktrace完全禁用自动堆栈跟踪捕获。默认情况下，在开发中为WarnLevel及以上日志捕获堆栈跟踪，在生产中为ErrorLevel及以上级别捕获堆栈跟踪。
        disableStacktrace: false
      
      # 存储相关
      storage:
        #  mysql
        mysql:
          # 是否开启
          enable: false
          #   地址，如果port不填写，请在host携带端口，或设置为域名
          host: ''
          port: ''
          user: ''
          password: ''
          database: ''
          debug: false
          #    最大连接数，设置0表示使用默认值200，,-1表示不限制
          open: 0
          #    最大空闲连接数 设置为0表示使用默认值2，-1表示不限制
          idle: 0
          #    连接最大生存时间
          max_life_time: 0
          #    数据库分表配置
          sharding:
            #    分表表名
            - table_name: ''
              #        分表key
              sharding_key: ''
              #        分表数量
              sharding_num: 0
        #  redis配置
        redis:
          enable: false
          host:
            - '127.0.0.1:6379'
          password: ''
          db: 0
          #    redis server是否是集群模式，默认false
          is_cluster: false
      
      
      # rabbitmq配置
      rabbitmq:
        enable: false
        host: ''
        username: ''
        pwd: ''
        virtual_host: ''
      # kafka配置
      # 如果host不配置，则不启动kafka。
      # enable_producer 是否开启生产者，如果没有生产者，不用配置，不会启动生产者；
      # 消费者则是代码配置了handler才会去启动
      kafka:
        enable_producer: false
        consumer_log: false
        host:
          - ''
        username: ''
        pwd: ''
      
      
      # http配置,不配置的话默认端口8080
      http:
        host: ''
        # 是否开启探活接口，不配置默认开启
        enable_live_api: false
        # 是否开启Prometheus的metrics接口
        enable_prometheus: false
      
      
      # 自定义配置，需要配合代码
      other: null
      
      ```

      对于一个本地运行的需要mysql、redis、kafka的配置文件如下：

      ```yaml
      server_name: ca-game-hilo
      version: v0.0.1
      
      registry:
        type: 'nacos'
        hosts:
          -  127.0.0.1:8848
        enable_istio: true
      
      
      rpc_server:
        listen_on: 0.0.0.0:9901
      
      http:
        host: ':9091'
      
      
      log:
        level: -1
        development: true
        disableStacktrace: false
        stacktrace_level: 3
      
      storage:
        mysql:
          enable: true
          host: '127.0.0.1'
          port: '3306'
          user: 'root'
          password: '123456'
          database: 'db_game_hilo'
          sharding:
            - table_name: 'b_ho_order'
              sharding_key: 'sys_user_id'
              sharding_num: 3
        redis:
          enable: true
          host:
            - '127.0.0.1:6490'
          password: '123456'
          db: 0
      
      kafka:
        enable_producer: true
        host:
          - 127.0.0.1:29092
      ```

- **site-var**

  配置中心组件

- **skywalking**

  skywalking链路追踪组件。

  对于微服务已经做了链路追踪中间件，对于其他函数内链路需要使用如下方式

  ```
  // 函数开头
  func A() {
  	skywalking.Start()
  	defer skywalking.StopSpan()
  	
  	// your code
  }
  
  ```

  

