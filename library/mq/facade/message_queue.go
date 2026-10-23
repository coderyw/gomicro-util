package facade

import (
	"context"

	"github.com/coderyw/gomicro-util/library/mq/model"
)

type KafkaManager interface {
	// Push 异步推送消息
	//
	// 推送后，不会等待返回，直接成功。安全性比较低。推送错误信息将会在后续错误 channel 中打印出来
	//
	// Parameters:
	//   - ctx context.Context 上下文
	//   - key string 确定分区的 key
	//   - topic string 推送主题
	//   - obj interface{} 推送数据
	//   - instance ...string 实例名称（不传时会使用默认的 instance）
	//
	// Returns：
	//   - err 推送范湖错误信息
	Push(ctx context.Context, key, topic string, obj interface{}, instance ...string) (err error)
	// PushSync 同步推送 数据
	//
	//	同步推送，只有在 broker 确认了推送成功才会响应。具体参见推送者的 acsk 配置
	//
	// Parameters:
	//   - ctx context.Context 上下文
	//   - key string 确定分区的 key
	//   - topic string 推送主题
	//   - obj interface{} 推送数据
	//   - instance ...string 实例名称（不传时会使用默认的 instance）
	//
	// Returns：
	//   - err 推送范湖错误信息
	PushSync(ctx context.Context, key, topic string, obj interface{}, instance ...string) (err error)

	// GetProduceStats 获取推送统计
	//
	// Parameters:
	//
	// Returns：
	//   - []ProducerStats
	GetProduceStats() []model.ProducerStats
}

type RabbitMqManager interface {
	// PublishWithConfirm 在 Confirm 模式下发布消息，并等待 ack/nack 或超时。
	//
	// 使用独立 Channel 并开启 Confirm，通过 NotifyPublish 监听确认。
	//
	// Parameters:
	//   - ctx context.Context 上下文
	//   - exchange string 发送交换机 传""标识使用默认交换机(direct类型) 默认 exchange 特点 ：1. routing key = queue name 2. 自动路由到同名队列
	//   - key string 路由 key。具体使用取决于 exchange 类型.
	//     direct 完全匹配
	//     topic 模糊匹配
	//     fanout 无意义
	//     headers 无意义
	//   - mandatory bool 如果消息无法被路由 false → 直接丢弃 true → 返回给 producer
	//   - msg model.RabbitMqPublishin 推送消息
	//
	// Returns：
	//   - ack 是否确认推送成功
	//   - err 错误信息
	PublishWithConfirm(ctx context.Context, exchange, key string, mandatory bool, msg model.RabbitMqPublishing) (ack bool, err error)
	// Publish 简单发布（不等待 Confirm），使用池中的 Channel。
	//
	// Parameters:
	//   - ctx context.Context 上下文
	//   - exchange string 发送交换机 传""标识使用默认交换机(direct类型) 默认 exchange 特点 ：1. routing key = queue name 2. 自动路由到同名队列
	//   - key string 路由 key。具体使用取决于 exchange 类型.
	//     direct 完全匹配
	//     topic 模糊匹配
	//     fanout 无意义
	//     headers 无意义
	//   - mandatory bool 如果消息无法被路由 false → 直接丢弃 true → 返回给 producer
	//   - msg model.RabbitMqPublishin 推送消息
	//
	// Returns：
	//   - err 错误信息
	Publish(ctx context.Context, exchange, key string, mandatory bool, msg model.RabbitMqPublishing) error
}
