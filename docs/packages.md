# 包索引

> 返回 [README](../README.md)。按目录列出各包职责；用法见各组件文档，API 细节用 `go doc` 查看（例如 `go doc ./library/env`）。

## library

| 包 | 职责 | 文档 |
|-|-|-|
| `library/pmicro` | 微服务应用框架：配置加载、gRPC/HTTP、Kafka 消费、Cron、优雅停机 | [pmicro](components/pmicro.md) |
| `library/pmicro/config` | 启动配置读取与装配（本地 / Nacos） | [configuration](configuration.md) |
| `library/pmicro/option` | `NewProgram` 的全部 Option | [getting-started](getting-started.md) |
| `library/pmicro/http`、`http/middleware` | 业务 HTTP server 与内置中间件（header、日志、租户、恢复、链路） | [pmicro](components/pmicro.md) |
| `library/pmicro/plugin/client_grpc`、`server_grpc` | 自研 gRPC 客户端/服务端插件 | [pmicro](components/pmicro.md) |
| `library/pmicro/registry/*`、`selector/*` | 注册中心（etcd/nacos/istio/redis）与节点选择 | [pmicro](components/pmicro.md) |
| `library/pmicro/wrapper/*` | gRPC 中间件链：认证、校验、错误码、恢复、元数据、监控、链路、限流、seata | [pmicro](components/pmicro.md) |
| `library/pmicro/session` | 从 ctx 读取请求身份 | [http](components/http.md#metadata上下文元数据) |
| `library/pmicro/wait_stop` | 停服时在途请求计数与排空 | [pmicro](components/pmicro.md) |
| `library/pmicro/metrics` | 监控指标采集与 TrueWatch 上报 | [observability](components/observability.md#metrics监控指标采集与上报) |
| `library/env` | 环境变量读取与校验 | [environment](environment.md) |
| `library/namespace` | 租户列表与资源命名 | [multi-tenant](multi-tenant.md) |
| `library/metadata` | ctx 元数据设置与透传 | [http](components/http.md#metadata上下文元数据) |
| `library/db/mysql` | GORM 多实例、分表、雪花主键 | [mysql](components/mysql.md) |
| `library/db/cache/redis` | Redis 多实例、锁、缓存读穿、定时同步 | [redis](components/redis.md) |
| `library/db/cache/local-cache` | 进程内缓存（按租户隔离） | [redis](components/redis.md#本地缓存local-cache) |
| `library/mq` | Kafka / RabbitMQ（`mq/rocketmq` 为空目录） | [mq](components/mq.md) |
| `library/beanstalk` | 轻量延迟队列客户端 | [mq](components/mq.md#beanstalk) |
| `library/cron` | 定时任务、多租户定时任务 | [scheduling](components/scheduling.md) |
| `library/consistent_hash` | 基于注册中心的一致性哈希任务分片 | [scheduling](components/scheduling.md#consistent_hash) |
| `library/gin` | 参数校验与统一响应 | [http](components/http.md#gin) |
| `library/http` | HTTP 客户端 | [http](components/http.md#http-客户端) |
| `library/log` | zap 日志门面与三方库日志适配 | [observability](components/observability.md#log) |
| `library/trace`、`trace/jaeger` | traceId 透传与链路追踪 | [observability](components/observability.md#trace) |
| `library/skywalking` | skywalking span API | [observability](components/observability.md#skywalking) |
| `library/site-var` | etcd / nacos 读写监听 | [config-center](components/config-center.md) |
| `library/customer-config` | 业务自定义配置泛型读取 | [config-center](components/config-center.md#customer-config) |
| `library/component/secret` | 凭据引用解析（`csms://` / `awssm://`），含 `provider/csms`、`provider/awssm` | [secret](components/secret.md) |
| `library/component/*` | jwt、lark-message、ali-sms、limiter、throttle、storage、object-storage | [component](components/component.md) |
| `library/mock` | 基于 ctx 的 mock 开关 | [testing](components/testing.md) |
| `library/svc`、`library/single`、`library/automaxprocs` | 服务生命周期、停服信号、CPU 核数自适应 | [pmicro](components/pmicro.md#相关基础包) |
| `library/e-error` | ⚠️ 已废弃，请改用本库 `model/error-code` 包 | — |

## internal

仅供本仓库内部复用，业务仓库不能直接依赖。

- `internal/apm/orchestrion`
  - APM span 生命周期管理封装，负责开启、结束 span，并提取 trace/span 标识。

- `internal/cache`
  - 内部缓存实现集合。

- `internal/cache/go-cache`
  - 基于 `github.com/patrickmn/go-cache` 的本地缓存实现，自动结合命名空间处理 key。

- `internal/mul-tenant`
  - 多租户上下文与租户枚举能力，当前主要结合 nacos 命名空间工作。

- `internal/storage`
  - 文件上传 provider 注册表与统一上传入口。

- `internal/storage/s3`
  - S3 兼容对象存储实现。

- `internal/storage/huawei`
  - 华为云 OBS 实现。

## model

共享数据结构、配置模型、常量与 facade 接口。

| 包 | 职责 |
|---|---|
| `model/config` | 各类服务配置结构体（MySQL / Redis / Kafka / RabbitMQ / 注册中心 / Metrics / Lark 等）及日志脱敏序列化 |
| `model/cron` | 定时任务模型 `CronJob` |
| `model/cst` | 全局常量（环境、context key、etcd key、雪花 key 等）；`cst/ctx` 子包为 gin 中转 key 常量（供 library/gin 与 pmicro 中间件配套使用） |
| `model/facade` | 通用接口定义（配置校验、Service 生命周期、proto 响应等） |
| `model/error-code` | 错误码类型 `StatusCode`：内置框架错误码，`NewErrorCode` 供业务项目定义并注册自己的错误码（数值重复时 panic） |
| `model/common-result` | gRPC 公共结果错误码枚举 `ERROR_CODE` 与 `CommonResult`（手写类型，不注册 protobuf，可与 ca-common-model 共存） |
| `model/http` | 默认 HTTP 响应体结构 `NsHttpResp` |
| `model/lark` | 飞书消息卡片模型 |
| `model/constrict` | 泛型类型约束（整型/浮点/数值/字符串） |
| `model/consistent_hash` `model/micro` | 一致性哈希监听模型、微服务优雅关闭钩子 `CloseS` |
| `model/commonproto` `model/init-md` | 前者为空目录、后者仅剩 easyjson 桩代码（proto 已迁出本仓库），勿使用 |
| `model/storage` | 文件上传请求和结果模型 |
| `model/apm` | APM tag 等追踪模型 |
| `model/tenant` | 租户模型 `Tenant` 与云厂商枚举 `CloudType` |

## util

`util/` 下是一组**不依赖框架**的纯工具包，可独立引用。

| 包 | 职责 | 关键 API |
|---|---|---|
| `util/snowflake` | 雪花 ID 位运算（42bit 时间戳 + 可变 node/sequence 位），无状态 | `NextId` / `GetNodeIdById` |
| `util/ratelimit` | 分布式令牌桶阻塞限流（见[限流](components/component.md#限流)） | `NewTokenBucketLimiter` / `WaitN` |
| `util/aes` | AES-CBC 加解密（支持自定义 IV、base64 标准/URL 编码） | `EncryptWithIv` / `DecryptWithIv` |
| `util/rsa` | RSA 公钥加密、私钥解密与密钥对生成 | `EncryptStrByPub` / `DecryptStrByPri` / `GenRsaKey` |
| `util/md5` | 字节流 / `io.Reader` 的 MD5 摘要 | `BytesMd5` / `ReaderMd5s` |
| `util/token` | 用户 token 的签名、AES 包装、校验与解析 | `GenerateSign` / `MakeEnAesToken` / `CheckToken` / `ParseToken` |
| `util/json` | 基于 jsoniter 的 JSON 封装与自定义类型解码扩展 | `Marshal` / `Unmarshal` |
| `util/mapstructure` | map/弱类型数据解码进 struct（支持 time、decimal、数值字符串互转） | `ParseJSONToStruct` |
| `util/date` | 日期计算（月份天数、按月累加跨年） | `GetNextMonthYearByRepeat` |
| `util/number` | 数字工具（中文数字、字符串数值互转）与 `decimal_cul` 精确四则运算 | `Str2Int` / `Str2Float` |
| `util/excel` | 基于 excelize 的流式 Excel 写入（并发安全、按列名排序落列） | `NewExcelWriter` / `WriteArray` / `SaveAndClose` |
| `util/gzip` | 对象 / 字节流的 gzip 压缩解压 | `EncodeBytes2BytesByGzip` |
| `util/mail` | 基于 go-mail 的 SMTP 邮件发送 | `SendMail` |
| `util/phonenum` | 菲律宾手机号校验与格式标准化 | — |
| `util/sort_queue/zset` | 仿 Redis ZSet 的并发安全有序集合（跳表实现；`sort_queue/skiplist` 为独立的基础跳表，对外入口是 zset） | `NewZSet` / `ZAdd` / `ZRange` / `ZRangeByScore` / `ZRank` |
| `util/yaml` | 读取本地 YAML 文件反序列化到结构体 | `YamlLoad` |
| `util/marshaller` | map 经 json 中转解码进结构体 | `MapToStruct` |
| `util/bool` | 布尔与数值互转 | `Bool2Num` |
| `util/operation` | 泛型三元表达式（含惰性求值版） | `Ternary` / `TernaryOp` |
| `util/system` | 金额分/元换算、PZ 用户名生成与反解析 | `FormatBalance` / `GetPzName` |
| `util/userx` | 按运行环境在第三方 Username 与我方 UserID 间互转 | `Username2UserID` / `UserID2Username` |
| `util/strings` | 泛型变参拼接、缓存 key 构造、随机串 | `StrCombine` / `KeyFormat` / `RandStr` |
| `util/ptr` | 泛型整型指针构造 | `IntegerPtr` |


根目录下的 `util/*.go` 提供无需单独拆包的轻量辅助函数，例如字符串处理、协程等待等。

## mocks

仓库根目录 `mocks/` 为 common-util 中的 interface 生成了 mockery mock，目录层级与源码一致，见 [testing](components/testing.md)。
