package rabbitmq

import amqp "github.com/rabbitmq/amqp091-go"

// ConsumerOptions 消费者配置，支持 QoS、手动 ACK、自动恢复。
type ConsumerOptions struct {
	// Queue 队列名称 ⚠️必填
	Queue string
	// 当前消费者唯一标识。 默认为服务名
	Consumer string
	// true 表示自动确认；false 表示需在 handler 内手动 Ack/Nack/Reject 类似于 kafka 的消费者提交确认。
	AutoAck bool
	//如果为 true：
	//	• 这个队列只能被当前 connection 的这个 consumer 使用
	//	• 其他消费者会失败
	//
	//使用场景：
	//	• 临时队列
	//	• RPC reply 队列
	//	• 单机任务处理
	//
	// 默认 false
	Exclusive bool
	//如果为 true：
	//	• 客户端不等待服务器确认
	//	• 直接返回
	//
	//生产中基本不用。
	//
	//因为：
	//	• 可能隐藏错误
	NoWait bool
	//可以传一些扩展参数，例如：
	//
	// 	amqp.Table{
	//    "x-priority": 10,
	//	}
	//用于：
	//	•	消费者优先级
	//	•	特殊插件功能
	//
	//但日常很少使用。
	Args amqp.Table
	// QoS：使用下面三个；若仅需简单预取可只设 PrefetchCount
	// 如果不设置：默认无限制推送。
	//
	//会导致：
	//	•	消费端内存暴涨
	//	•	unacked 堆积
	//	•	触发流控
	//
	Qos Qos
}

// Qos 消费者流控
//
// 建议与消费者的 worker 数量去计算配置 一般是 workerNum*(1~0.8) workerNum数量根据对应服务区配置
type Qos struct {
	// consumer 消费 worker 数量（简单理解就是多少并发执行。） 默认 1
	WorkerNum int
	// consumer 可以最多同时处理未确认消息条数上限，0 表示不调用 Qos
	// 比如：
	//  20: consumer 最多同时处理 20 条未确认消息。
	PrefetchCount int
	// consumer 可以最多同时处理未确认字节数，0 表示不限制
	PrefetchSize int
	// 是否对本连接所有 consumer 生效
	Global bool
}

type ConsumerOption func(*ConsumerOptions)
