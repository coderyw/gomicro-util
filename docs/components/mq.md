# mq：消息队列

> 返回 [README](../../README.md)。

| 组件 | 包 | 状态 |
|-|-|-|
| Kafka | `library/mq`、`library/mq/kafka` | 主力，pmicro 自动初始化 |
| RabbitMQ | `library/mq/rabbitmq` | 可用，**未接入 pmicro 自动初始化** |
| RocketMQ | `library/mq/rocketmq` | 占位目录，暂未实现 |
| beanstalk | `library/beanstalk` | 轻量延迟队列客户端 |

## Kafka

底层驱动为 confluent-kafka-go（librdkafka）。配置结构为 `config.KafkaCfg`，完整带注释示例见 [configuration.md](../configuration.md) 的 `kafka` 段。更深入的设计与迁移说明：

- [kafka/ARCHITECTURE.md](../../library/mq/kafka/ARCHITECTURE.md)：架构
- [kafka/KAFKA_CONFIG.md](../../library/mq/kafka/KAFKA_CONFIG.md)：配置项详解
- [kafka/MIGRATION_GUIDE.md](../../library/mq/kafka/MIGRATION_GUIDE.md)：从 sarama 迁移
- [kafka/README_OPTIMIZATION.md](../../library/mq/kafka/README_OPTIMIZATION.md)：性能优化记录

### 消费

通过 pmicro option 注册，框架启动时自动初始化：

```go
p := pmicro.NewProgram(
	option.ServiceName("ca-xxx-srv"),
	// 单条消费：consumerInstance 对应 kafka.consumer[].name；groupId 省略时为服务名，
	// 但配置中仍须存在 group_id 等于服务名的 group，否则启动报错 group not found
	option.MQHandler("core1", "topic.order.created", func(ctx context.Context, msg model.KafkaConsumerMessage) error {
		return biz.OnOrderCreated(ctx, msg.Value)
	}),
	// 批量消费：按 max_mul_num 攒批或超时触发
	option.MQMulHandler("core1", "topic.log", func(ctx context.Context, msgs []model.KafkaConsumerMessage) (int, error) {
		return len(msgs), biz.BatchSave(ctx, msgs)
	}, "ca-xxx-srv-log"),
)
```

`model` 为 `library/mq/model`。消费行为要点：

- 一个 handler 对应一个 topic；同一 `instance|group` 下重复注册同一 topic 会被静默跳过
- 单条消费：traceId 从消息 header 还原（没有则生成新的）；租户从 **topic 名前缀**解析，生产端写入的租户 header 不参与
- 批量消费时，每条消息的 `msg.Ctx` 携带该条消息的 traceId 与租户，按条处理时请用 `msg.Ctx`；批量 handler 返回的 `int` 当前框架未使用
- **handler 返回 error 或 panic，消息都会照常提交**（批量时提交该批最后一条），只记日志；失败重试/落库需业务自行处理
- handler 内部有 recover，panic 不会打挂消费协程
- 停服过程中已拉取但未处理的批次会被跳过、不提交；分区被调走时，批量里未凑满的残留消息会被丢弃，由新 owner 重新消费
- ⚠️ `commit_type: 1` 当前实际不会提交偏移量，请使用默认的 0 或 2，见 [configuration.md](../configuration.md) 的 `kafka` 段

### 推送

```go
// 异步推送（不等 broker 确认）；instance 省略时用 default_producer（未配则第一个）
err := mq.GetKafkaManager(ctx).Push(ctx, "分区key", "topic.order.created", &OrderMsg{Id: 1})

// 同步推送（等待投递结果，30s 超时）；可指定生产者实例
err = mq.GetKafkaManager(ctx).PushSync(ctx, "", "topic.audit", data, "core")
```

⚠️ 通过 pmicro 启动时 Kafka 总会初始化：没有配置生产者时 `Push` 返回 `producer name is empty` 错误，不会 panic；只有在 pmicro 启动前（或不使用 pmicro）调用 `GetKafkaManager` 才会 panic。支持 mock（`mocks/library/mq/facade`）。多租户下 topic 自动加租户前缀，规则见 [multi-tenant.md](../multi-tenant.md)。

### C 侧内存与再均衡

- confluent-kafka-go 底层 librdkafka 的预取/发送队列在 **C 堆**中分配，Go heap profile（pprof）不可见；客户端内存占用 ≈ `持有分区数 × 每分区预取窗口`，与消费堆积（lag）总量无关——堆积存放在 broker 上。上述 `queued_min_messages` / `queued_max_messages_kbytes` / `queue_buffering_max_kbytes` 即用于封顶这部分内存，本库默认值已从 librdkafka 默认（50MB+/分区、1GB/producer）收敛到 ~8MB/分区、64MB/producer，特殊吞吐需求可按配置放大。
- `statistics_interval_ms` 控制 librdkafka 统计采集间隔，`stats_mode` 控制输出方式（-1 不统计 / 0 默认等同 1 / 1 仅日志 / 2 仅指标上报 / 3 都开启，语义与 mysql/redis 的 `pool_stats_mode` 一致）。日志形态为 `[Kafka] librdkafka stats`：消费端 `consumer_fetchq_cnt/fetchq_size`（C 侧预取队列条数/字节数）、生产端 `producer_queue_msg_cnt/msg_size`（发送队列）。指标形态（gauge，推到 `pmicro/metrics`，metrics 未开启则丢弃）：生产端 `kafka_producer_queue_msg_cnt` / `kafka_producer_queue_msg_size`（tag `instance`），消费端 `kafka_consumer_fetchq_cnt` / `kafka_consumer_fetchq_size`（tag `instance`+`group`）。这是观测 C 侧内存的唯一手段，建议接入监控告警。
- 消费端已注册 rebalance 回调：分区被调走时关闭对应的分区通道、回收消费 goroutine，并删除该分区待提交的偏移量。注意通道中已缓冲的消息仍会被处理完并重新记录偏移量，之后由定时任务提交，因此不能完全避免覆盖新 owner 的进度，消费逻辑需要幂等。分区回流时不会立即重建，而是在该分区下一条消息到达时懒创建。

`stats_mode` 取值见 [configuration.md · 统计输出模式](../configuration.md#统计输出模式pool_stats_mode--stats_mode)。

## RabbitMQ

基于 [amqp091-go](https://github.com/rabbitmq/amqp091-go) 的封装：自动重连、Channel 池、Publisher Confirm、手动 ACK、QoS 并发消费（ants 协程池）、拓扑声明（Ensure 语义）、优雅关闭。

⚠️ **尚未接入 pmicro 自动初始化**（`mq.InitRabbitMq` 是空实现），业务需自行构造 Client：

```go
cfg := rabbitmq.FromRabbitMqCfg(config.RabbitMqCfg{Host: "127.0.0.1:5672", Username: "guest", Pwd: "guest", VirtualHost: "/"})
client, _ := rabbitmq.NewClient(cfg)
defer client.Shutdown(context.Background())

// 带 confirm 的发布
ack, err := client.PublishWithConfirm(ctx, "", "example.queue", false, model.RabbitMqPublishing{
    ContentType: "text/plain",
    Body:        []byte("hello"),
})

// 自动重连消费；Qos.WorkerNum 控制并发 worker 数
_ = client.ConsumeWithAutoRecovery(ctx, func(d amqp.Delivery) {
    _ = d.Ack(false)
}, func(o *rabbitmq.ConsumerOptions) {
    o.Queue = "example.queue"
    o.AutoAck = false
    o.Qos.PrefetchCount = 10
    o.Qos.WorkerNum = 8
})
```

其他能力：`EnsureQueue`/`EnsureExchange`/`EnsureTopology` 声明拓扑；`NewConnectionPool` 多连接轮询；`RabbitMqPublishing` 默认 `ContentType=application/json`、`DeliveryMode=2`（持久化）、`MessageId=traceId`。

## beanstalk

极其轻量的延迟队列组件，server使用c写，客户端可通过go连接处理。

- 优点：轻量，便携。支持持久化binlog
- 缺点：server端单机，无法保证高可用。

[了解详情](https://zhangquan.me/2021/06/29/beanstalkd-xue-xi-yan-jiu/)

## 子包

| 子包 | 职责 |
|-|-|
| `mq/facade` | 消息队列接口抽象，便于依赖接口和 mock |
| `mq/model` | handler、消费消息、统计等通用模型 |
| `mq/option` | MQ 初始化和运行时选项 |
