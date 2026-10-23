package rabbitmq

import (
	"context"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/coderyw/gomicro-util/library/mq/model"
)

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
func (c *Client) PublishWithConfirm(ctx context.Context, exchange, key string, mandatory bool, msg model.RabbitMqPublishing) (ack bool, err error) {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return false, err
	}

	defer release()
	if err = ch.Confirm(false); err != nil {
		return false, err
	}
	mqMsg := RabbitMqPublishing2AmqpPublishing(ctx, msg)
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	// todo 当 mandatory =true 时候需要监听 ch.NotifyReturn(make(chan amqp.Return)) 用于处理消息无法被路由时候producer 的处理
	if err = ch.PublishWithContext(ctx, exchange, key, mandatory, false, mqMsg); err != nil {
		return false, err
	}
	select {
	case conf, ok := <-confirms:
		if !ok {
			return false, amqp.ErrClosed
		}
		return conf.Ack, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}

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
func (c *Client) Publish(ctx context.Context, exchange, key string, mandatory bool, msg model.RabbitMqPublishing) error {
	ch, release, err := c.GetChannel(ctx)
	if err != nil {
		return err
	}
	defer release()
	mqMsg := RabbitMqPublishing2AmqpPublishing(ctx, msg)
	return ch.PublishWithContext(ctx, exchange, key, mandatory, false, mqMsg)
}

// ConfirmListener 用于在单独 goroutine 中监听发布确认（ack/nack）。
type ConfirmListener struct {
	Ack  chan uint64
	Nack chan uint64
	done chan struct{}
	once sync.Once
}

// NewConfirmListener 创建确认监听器，容量建议不小于未确认发布数。
func NewConfirmListener(cap int) *ConfirmListener {
	if cap <= 0 {
		cap = 64
	}
	return &ConfirmListener{
		Ack:  make(chan uint64, cap),
		Nack: make(chan uint64, cap),
		done: make(chan struct{}),
	}
}

// Close 关闭监听器通道。
func (l *ConfirmListener) Close() {
	l.once.Do(func() { close(l.done) })
}

// Run 在给定 Channel 上开启 Confirm 并循环将 ack/nack 投递到 Ack/Nack chan，直到 Channel 关闭或 ctx 取消。
// 调用方需保证该 Channel 仅用于发布且不再被他人关闭；Run 返回时不会关闭 Channel。
func (l *ConfirmListener) Run(ctx context.Context, ch *amqp.Channel) error {
	if err := ch.Confirm(false); err != nil {
		return err
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, cap(l.Ack)+cap(l.Nack)))
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-l.done:
			return nil
		case c, ok := <-confirms:
			if !ok {
				return amqp.ErrClosed
			}
			if c.Ack {
				select {
				case l.Ack <- c.DeliveryTag:
				default:
				}
			} else {
				select {
				case l.Nack <- c.DeliveryTag:
				default:
				}
			}
		}
	}
}
