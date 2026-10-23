// Package mq 消息队列入口，当前主力为 Kafka（confluent-kafka-go）。
//
// 消费：通过 pmicro 的 option.MQHandler（单条）/ option.MQMulHandler（批量）注册，框架启动时自动初始化。
// handler 返回 error 或 panic 时只记日志，消息仍会提交，失败重试需业务自行处理。
//
// 推送：
//
//	// 异步推送；instance 省略时使用 default_producer
//	err := mq.GetKafkaManager(ctx).Push(ctx, "分区key", "topic.order.created", msg)
//	// 同步推送，等待投递结果
//	err = mq.GetKafkaManager(ctx).PushSync(ctx, "", "topic.audit", msg, "core")
//
// 在 pmicro 启动前调用 GetKafkaManager 会 panic。多租户下 topic 会按 ctx 中的租户自动加前缀。
// RabbitMQ 见 rabbitmq 子包，未接入 pmicro 自动初始化。
//
// 文档：docs/components/mq.md
package mq
