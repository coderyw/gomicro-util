# 可观测：日志、链路与指标

> 返回 [README](../../README.md)。

## log

日志组件，底层基于 [zap](https://github.com/uber-go/zap) 进行了包装。pmicro 启动时会按 `log` 配置调用 `InitDefaultLogger()` 初始化。

### 配置

```yaml
log:
  # DebugLevel  - 1
  # InfoLevel 0
  # WarnLevel 1
  # ErrorLevel 2
  # DPanicLevel 3
  # PanicLevel  4
  # FatalLevel 5
  # 会打印Level及以上级别的日志
  level: 0
  # 注意命名与行为相反：disableStacktrace 为 true 时才「开启」自动堆栈，为 false（默认）时不打印自动堆栈
  disableStacktrace: false
  # 仅在 disableStacktrace: true 时生效：该级别及以上的日志附带堆栈，不配置时为 3（DPanic）
  stacktrace_level: 2
```

### 使用

```go
// 打印 info 级别日志
log.Info("info log", zap.String("key", "value"), zap.Int("int", 1))
// 打印带traceId 的 info 日志
log.InfoCtx(ctx, "info log", zap.String("key", "value"), zap.Int("int", 1))

// 打印 warn 级别日志
log.Warn("info log", zap.String("key", "value"), zap.Int("int", 1))
// 打印带traceId 的 warn 日志
log.WarnCtx(ctx, "info log", zap.String("key", "value"), zap.Int("int", 1))
```

### 敏感值脱敏

除按字段名脱敏外，`library/log` 还支持**按值**脱敏：登记过的明文出现在日志的任何位置（message、字段、嵌套对象、堆栈）都会被替换为 `***`。[凭据引用](secret.md)解析出的值会自动登记，业务也可以调用 `log.RegisterRedactValue(v)` 登记（长度小于 6 的值忽略）。覆盖经本包输出的日志、Nacos SDK / gorm 等适配日志和标准库 `log`；直接 `fmt.Print` 与第三方自建 logger 不覆盖。

### 其他说明

业务代码优先使用带 `Ctx` 后缀的方法，会自动输出 traceId、用户、租户等元数据。注意：ctx 中没有 traceId 时，每行日志都会生成一个新的 uuid，同一请求的日志串不起来，异步任务入口请先调用 `trace.GetTraceCtx`。还提供 `InfoFmt`（虽然接收 ctx，但**不会**输出 traceId 等元数据）、`ErrorWithStack`，以及以 `*CtxLogger` 结尾的方法（使用指定 logger 输出）。`library/log` 同时内置 gorm、redis、nacos、cron、micro 等三方库的日志适配。

## trace

链路 id 工具：traceId 是保存在 context 中的字符串，gRPC 经 metadata（`Traceid` / `Sw8`）双向透传；HTTP 只在**入站**读取 `X-Trace-Id` / `Trace-Id` / `Traceid` 头并在响应头回写 `Traceid`，`library/http` 发出的请求**不会**自动带上 traceId。

- `GetTraceCtx(ctx)` 确保 ctx 携带 traceId（无则生成 uuid），异步任务/定时任务入口必调
- `GetTraceId(ctx)` 读取（无则返回空串）；`MustGetTraceId(ctx)` 读取（无则现场生成）
- `SetTraceId(ctx, id)` 强制设置；`NewOrAddTraceId(ctx, id)` 已有时拼接为 `父id:新id`（子任务链路串联）
- `trace/jaeger` 子包：jaeger tracer 构造与 span 工具（`NewTracer`、`StartChildSpanFromContext`、gin 中间件 `JaegerMiddlewareWithMicro`）；ctx 中无 `TraceCtxKey{}` 时全链路自动降级为 no-op

`TRACE_TYPE` 环境变量：只有 `orchestrion` 会改变 traceId 来源；`skywalking` 目前与 `default` 行为相同（sw8 透传不受该变量影响），见 [environment.md](../environment.md#可选功能)。APM span 接口 `StartSpan` / `FinishSpan` 在 trace 包的 `apm.go` 中，底层实现为 `internal/apm/orchestrion`。

## skywalking

skywalking 链路追踪 api，包装 `skywalking-go/toolkit/trace`，依赖 agent 编译期织入，无需初始化直接调用。

- 开 span：`Start()`（span 名自动取调用方函数名）/ `StartOperationName(name)` / `StartRedis()` / `StartMysql()`（带组件标记），配 `defer StopSpan()`；`SetTag` 打标签
- **跨 goroutine 传递链路**（异步任务必须，否则断链）：

```go
ref := skywalking.CaptureContext()
go func() {
    skywalking.ContinueContext(ref)
    skywalking.StartOperationName("asyncJob")
    defer skywalking.StopSpan()
    doWork()
}()
```

## metrics（监控指标采集与上报）

`library/pmicro/metrics`。指标统一注册在 prometheus 全局 registry，配置开启后由框架定时推送到 Truewatch DataWay（push），`/metrics` 拉取端点（pull，见 `http.enable_prometheus`）同时可用、数据一致。指标是旁路能力，全链路弱依赖：配置错误降级为不上报、打点路径 panic 兜底、上报失败只记日志，均不影响业务。

- **框架指标（零代码，随服务自动采集）**

  - Go 运行时：`go_goroutines`、`go_memstats_*`、`process_*`
  - gRPC 服务端/客户端：`micro_request_total`、`micro_request_duration_seconds` 等（按 endpoint、成功/失败分维度）
  - HTTP 服务端：`http_requests_total{method,path,status}`、`http_request_duration_seconds{method,path}`（path 为路由模板，未匹配路由归入 `unmatched`，探活与 `/metrics` 路径不统计）
  - 连接池状态（由 mysql/redis 的 `pool_stats_mode` 控制开关、`pool_stats_interval` 控制采集间隔，MySQL 默认 60s、Redis 默认 15s，详见 [cache/redis](redis.md#连接池统计) 与 [db/mysql](mysql.md#连接池统计) 章节）：MySQL `mysql_pool_*{instance}`（in_use/idle/open_connections/wait_count/wait_duration_ms 等）、Redis `redis_pool_*{instance}`（total_conns/idle_conns/busy_conns/hits/misses/timeouts/stale_conns）；metrics 上报未开启时这些数据直接丢弃、不进 `/metrics` 端点
  - Kafka librdkafka C 侧队列（由 kafka 的 `stats_mode` 控制开关、`statistics_interval_ms` 控制采集间隔，默认 60s，详见 [mq](mq.md#c-侧内存与再均衡) 章节）：生产端 `kafka_producer_queue_msg_cnt/msg_size{instance}`、消费端 `kafka_consumer_fetchq_cnt/fetchq_size{instance,group}`；丢弃语义同上

- **业务自定义打点**

  ```go
  import "github.com/coderyw/gomicro-util/library/pmicro/metrics"

  // 通常作为包级/服务级单例持有；service 会作为公共 tag 附加到每条指标
  var mc = metrics.NewClient("bmp-game-mw")

  // 计数类：请求量、失败次数、金额流水（分）等只增不减的值，value 必须 >= 0
  mc.Inc("requests_total", metrics.Tags{"endpoint": "place_bet", "ret": "0000"}, 1)

  // 耗时/分布类：需要看分位数的值，单位【毫秒】
  mc.Latency("downstream_latency_ms", metrics.Tags{"dependency": "base_balance"}, 83)
  ```

  无需初始化：框架启动时按 `metrics` 配置自动完成 Init 与优雅停机（停机前会补报最后一轮）；未配置上报时打点不会报错，但（除非设置 `disable_business: true`）指标仍会注册并累计，开启 `http.enable_prometheus` 时可在 `/metrics` 看到。

  > ⚠️ `Latency` 的接口注释写的是「单位秒」，但默认 Histogram 桶是按毫秒设计的（1ms ~ 300000ms），按秒传入时所有小于 1 的值都会落进第一个桶。请按**毫秒**传值，代码注释待修正。

- **瞬时值上报（SetGauge，框架内部为主）**

  ```go
  // Gauge 语义（Set 覆盖，不累加），用于连接池状态等瞬时值
  metrics.SetGauge("my_queue_size", metrics.Tags{"queue": "order"}, 12)

  // 查询上报循环是否已开启（Init 成功启动）
  metrics.Enabled()
  ```

  与 `Client.Inc/Latency` 的两点区别：**仅当 metrics 上报已开启时才写入**——未开启时直接丢弃，不注册指标、不进 `/metrics` 端点，调用方无需自行判断开关；**不附加 `service` label**——公共 tag（service/env/host）由上报器推送时统一附加。mysql/redis 连接池统计（`pool_stats_mode` 为 2/3 时）即通过该接口上报。

- **Tags 约定（重要）**

  - **tag value 只能来自可枚举白名单**（endpoint 名、返回码、依赖名等），不得把 userId、订单号、原始 URL 等高基数/无界值当 tag，否则时间序列爆炸拖垮存储与查询
  - **同一指标名的 tag key 集合必须一致**：以进程内首次打点为准，之后 key 集合不同的打点会被丢弃并记 warning，不影响业务
  - 指标名/tag key 中的非法字符（如连字符）会被自动替换为下划线

- **上报数据格式**：measurement 为服务名；每条指标自动附带 `service`/`env`/`host`（pod 名）公共 tag；field 名保持 prometheus 原名，可做跨服务统一 dashboard

- **配置**：见[configuration.md](../configuration.md)的 `metrics` 一节；`enable` 控制推送、`disable_business` 单独控制业务打点采集，两者独立
