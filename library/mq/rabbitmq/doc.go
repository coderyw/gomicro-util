// Package rabbitmq 基于 github.com/rabbitmq/amqp091-go 的 RabbitMQ 包装。
//
// 特性：
//   - 自动重连：连接断开后按配置间隔重试，支持指数退避上限
//   - Channel 管理：Channel 池（GetChannel / GetPooledChannel）、按需创建与归还
//   - Publisher Confirm：PublishWithConfirm 同步确认、ConfirmListener 异步监听 ack/nack
//   - 手动 ACK： AutoAck=false，在 DeliveryHandler 内调用 d.Ack/d.Nack/Reject
//   - QoS 控制：ConsumerOptions 的 PrefetchCount/PrefetchSize/Global，或 ApplyQos(ch, qos)
//   - 优雅关闭：Shutdown(ctx) 禁止新操作、等待在途 Channel 归还或超时后关闭
//   - 可扩展为连接池：ConnectionProvider 抽象、NewClientFromPool、ConnectionPool 多连接轮询
//
// 示例：
//
//	cfg := rabbitmq.FromRabbitMqCfg(serverCfg.Rabbitmq)
//	client, err := rabbitmq.NewClient(cfg)
//	if err != nil { ... }
//	defer client.Shutdown(context.Background())
//
//	// 发布（带 Confirm）
//	ack, err := client.PublishWithConfirm(ctx, "", "queue1", false, false, amqp.Publishing{Body: []byte("hello")})
//
//	// 消费（QoS + 手动 ACK + 自动恢复）
//	err = client.ConsumeWithAutoRecovery(ctx, rabbitmq.ConsumerOptions{
//	    Queue: "queue1", ManualAck: true, PrefetchCount: 10,
//	}, func(d amqp.Delivery) {
//	    log.Info(string(d.Body))
//	    _ = d.Ack(false)
//	})
//
// 连接池：
//
//	pool, _ := rabbitmq.NewConnectionPool(cfg, 4)
//	defer pool.Close()
//	client, _ := rabbitmq.NewClientFromPool(pool, cfg)
//
// 队列/交换机声明（Ensure 语义：不存在则创建，已存在则校验一致）：
//
//	queue, _ := client.EnsureQueueWithContext(ctx, rabbitmq.DefaultQueueOption("my-queue"))
//	_ = client.EnsureExchangeWithContext(ctx, rabbitmq.DefaultExchangeOption("my-ex", rabbitmq.ExchangeTopic))
//	_ = client.EnsureQueueBindWithContext(ctx, "my-queue", "order.#", "my-ex", nil)
//	// 或一次性声明拓扑
//	_ = client.EnsureTopologyWithContext(ctx, rabbitmq.Topology{
//	    Exchanges: []rabbitmq.ExchangeOption{rabbitmq.DefaultExchangeOption("ex", rabbitmq.ExchangeTopic)},
//	    Queues:    []rabbitmq.QueueOption{rabbitmq.DefaultQueueOption("q")},
//	    Binds:     []rabbitmq.QueueBindItem{{QueueName: "q", RoutingKey: "#", Exchange: "ex"}},
//	})
package rabbitmq
