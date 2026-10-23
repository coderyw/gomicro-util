package rabbitmq_test

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/coderyw/gomicro-util/library/mq/model"
	rabbitmq "github.com/coderyw/gomicro-util/library/mq/rabbitmq"
	"github.com/coderyw/gomicro-util/model/config"
)

// ExampleNewClient 展示如何从全局配置创建 RabbitMQ Client。
func ExampleNewClient() {
	// 一般从服务配置中获取 RabbitMqCfg，这里直接构造演示
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)

	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		// 连接失败时进行错误处理（此处仅示意）
		return
	}
	defer client.Shutdown(context.Background())

	// Output:
}

// ExampleClient_PublishWithConfirm 展示如何使用发布确认发送消息。
func ExampleClient_PublishWithConfirm() {
	// 初始化 rabbitmq
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 推送 普通消息
	_, _ = client.PublishWithConfirm(ctx, "", "example.queue", false, model.RabbitMqPublishing{
		ContentType: "text/plain",
		Headers: map[string]interface{}{
			"tenantId": 123,
		},
		Body: []byte("hello"),
	})

	// rpc 消息
	_, _ = client.PublishWithConfirm(ctx, "", "example.queue", false, model.RabbitMqPublishing{
		ContentType: "application/json",
		Headers: map[string]interface{}{
			"tenantId": 123,
		},
		ReplyTo:       "rpc.replay.getage.queue",
		CorrelationId: "123",
		Body:          []byte("{\"user_id\":123}"),
	})

	// Output:
}

func ExampleClient_Publish() {
	// 初始化 rabbitmq
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 推送 普通消息
	_ = client.Publish(ctx, "", "example.queue", false, model.RabbitMqPublishing{
		ContentType: "text/plain",
		Headers: map[string]interface{}{
			"tenantId": 123,
		},
		Body: []byte("hello"),
	})

	// rpc 消息
	_ = client.Publish(ctx, "", "example.queue", false, model.RabbitMqPublishing{
		ContentType: "application/json",
		Headers: map[string]interface{}{
			"tenantId": 123,
		},
		ReplyTo:       "rpc.replay.getage.queue",
		CorrelationId: "123",
		Body:          []byte("{\"user_id\":123}"),
	})

	// Output:
}

// ExampleClient_PublishWithExpiration 展示如何推送带过期时间（TTL）的消息。
func ExampleClient_publishWithExpiration() {
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Expiration 单位为毫秒，这里表示 60 秒后过期
	_, _ = client.PublishWithConfirm(ctx, "", "example.ttl.queue", false, model.RabbitMqPublishing{
		ContentType: "text/plain",
		Body:        []byte("expired in 60s"),
		Expiration:  "60000",
	})

	// Output:
}

// ExampleClient_ConsumeWithAutoRecovery 展示如何开启自动恢复的消费者（手动 ACK）。
func ExampleClient_ConsumeWithAutoRecovery() {
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = client.ConsumeWithAutoRecovery(ctx, func(d amqp.Delivery) {
		// 处理消息后手动 ACK
		_ = d.Ack(false)
	}, func(options *rabbitmq.ConsumerOptions) {
		options.Queue = "example.queue"
		options.AutoAck = false
		options.Qos.PrefetchCount = 10
	})

	// Output:
}

// ExampleTopology 展示如何一次性声明交换机、队列和绑定。
func ExampleTopology() {
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	top := rabbitmq.Topology{
		Exchanges: []rabbitmq.ExchangeOption{
			rabbitmq.DefaultExchangeOption("example.ex", rabbitmq.ExchangeTopic),
		},
		Queues: []rabbitmq.QueueOption{
			rabbitmq.DefaultQueueOption("example.queue"),
		},
		Binds: []rabbitmq.QueueBindItem{
			{
				QueueName:  "example.queue",
				RoutingKey: "example.#",
				Exchange:   "example.ex",
			},
		},
	}
	_ = client.EnsureTopologyWithContext(ctx, top)

	// Output:
}

// ExampleClient_DeadLetterQueue 展示如何配置死信队列并消费死信消息。
func ExampleClient_deadLetterQueue() {
	raw := config.RabbitMqCfg{
		Enable:      true,
		Host:        "127.0.0.1:5672",
		Username:    "guest",
		Pwd:         "guest",
		VirtualHost: "/",
	}
	cfg := rabbitmq.FromRabbitMqCfg(raw)
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		return
	}
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 声明正常队列 + 死信交换机 + 死信队列
	top := rabbitmq.Topology{
		Exchanges: []rabbitmq.ExchangeOption{
			// 死信交换机
			rabbitmq.DefaultExchangeOption("example.dlx.ex", rabbitmq.ExchangeDirect),
		},
		Queues: []rabbitmq.QueueOption{
			// 正常业务队列，配置死信交换机和路由键
			{
				Name:       "example.normal.queue",
				Durable:    true,
				AutoDelete: false,
				Exclusive:  false,
				Args: amqp.Table{
					"x-dead-letter-exchange":    "example.dlx.ex",
					"x-dead-letter-routing-key": "example.dlx.routing",
				},
			},
			// 死信队列
			rabbitmq.DefaultQueueOption("example.dlq.queue"),
		},
		Binds: []rabbitmq.QueueBindItem{
			{
				QueueName:  "example.dlq.queue",
				RoutingKey: "example.dlx.routing",
				Exchange:   "example.dlx.ex",
			},
		},
	}
	_ = client.EnsureTopologyWithContext(ctx, top)

	// 消费死信队列上的消息（如业务队列 Nack/Reject 或 TTL 过期进入 DLQ）
	_ = client.ConsumeWithAutoRecovery(ctx, func(d amqp.Delivery) {
		// 在这里处理死信消息，然后手动 ACK
		_ = d.Ack(false)
	}, func(options *rabbitmq.ConsumerOptions) {
		options.Queue = "example.dlq.queue"
		options.AutoAck = false
	})

	// Output:
}
