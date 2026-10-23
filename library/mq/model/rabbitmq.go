package model

import (
	"github.com/rabbitmq/amqp091-go"
)

// RabbitMqPublishing 推送消息参数
type RabbitMqPublishing struct {
	// 自定义元数据 请注意 headers 会增加消息体大小，不要放大对象
	Headers map[string]interface{}
	// MIME content type 默认application/json
	ContentType string
	// body 的编码方式。常见："gzip" "deflate" "utf-8" 默认值为空，大多数不需要使用
	ContentEncoding string // MIME content encoding
	// 投递模式，1: 临时消息( 吞吐率会更高，broker 重启后消息会丢失) 2: 持久消息(安全性会更高，吞吐率会低) 默认 2
	DeliveryMode uint8 //
	// 消息优先级。使用前提：队列必须声明时支持 priority(x-max-priority: 10)
	//
	// 数值越大优先级越高
	//
	// 使用场景
	//	•	高级别告警
	//	•	VIP 任务
	//	•	实时任务插队
	//
	// ⚠️会降低队列性能（内部排序）,不建议大量使用
	Priority uint8 // 0 to 9
	// 有多种用途：
	//
	// 1. 请求-响应模型
	//
	//	比如 RPC：
	// 	生产者发送：
	//	CorrelationId: uuid
	//	ReplyTo: "reply.queue"
	//
	//	消费者处理后：
	//	• 把结果发到 ReplyTo
	//	• 带同样 CorrelationId(用于标识消息)
	//
	//	生产者通过 CorrelationId 匹配响应。
	//
	// 2. 链路追踪
	//	可以放 traceId/requestId
	CorrelationId string
	// 用于 RPC。
	//	场景：
	//	A 服务发送请求：ReplyTo: "rpc.reply.queue"
	//	B 服务处理后：
	//	• 把结果发到这个队列
	//
	//⚠️ 普通异步场景基本不用
	ReplyTo string
	// 消息过期时间（以毫秒为单位）。 默认为 NeverExpire
	// 值“0”表示，如果消息到达其目的地并且当前有能力处理该消息的消费者未直接处理该消息，则消息将立即过期。
	// 如果您希望消息不会自行过期，请将此值设置为任何ttl值、空字符串或使用相应的常数 NeverExpire 和 ImmediatelyExpire。
	// 这不会影响队列配置的TTL值。
	Expiration string
	// 消息 id，用于幂等和日志追踪，（如果当做幂等使用请自定义）默认使用 traceId
	MessageId string
	// 消息类型标识
	//  比如："order.created"
	// 类似于 event name
	//
	// 和 routing key 的区别
	//	routing key 是用来路由消息的(exchange 级别)
	//	type 用于业务识别(消费者级别)
	//
	// 推荐用法是 routing key 控制路由，type 控制业务语义
	Type string
	//验证发送者身份。
	//
	//如果设置：
	//
	//RabbitMQ 会校验：
	//	•	必须和登录用户一致
	//
	//否则拒绝消息。
	//
	//⚠️几乎不用。
	UserId string
	//标识发送应用。
	//	比如：AppId: "order-service"
	//
	// 默认为发送端serverName
	AppId string
	// 发送消息体，编码与ContentEncoding 和 ContentType关联
	Body []byte
}

const (
	// 从不过期
	NeverExpire = amqp091.NeverExpire
	// 立即过期
	ImmediatelyExpire = amqp091.ImmediatelyExpire
)
