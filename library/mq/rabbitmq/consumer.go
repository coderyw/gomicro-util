package rabbitmq

import (
	"context"
	"time"

	"github.com/panjf2000/ants/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// DeliveryHandler 处理单条投递。当 AutoAck=false 或 ManualAck=true 时，必须在处理完成后调用 d.Ack(false) 或 d.Nack/Reject，否则消息会堆积并可能重复投递。
type DeliveryHandler func(d amqp.Delivery)

// ConsumeWithAutoRecovery 在独立 Channel 上消费，连接或 Channel 关闭时自动重连并重新 Consume。
// 支持 QoS（PrefetchCount/PrefetchSize/Global）与手动 ACK（ManualAck 或 AutoAck=false）。
func (c *Client) ConsumeWithAutoRecovery(ctx context.Context, handler DeliveryHandler, opts ...ConsumerOption) error {
	opt := new(ConsumerOptions)
	for _, v := range opts {
		v(opt)
	}
	if opt.Qos.WorkerNum <= 0 {
		opt.Qos.WorkerNum = 1
	}
	autoAck := opt.AutoAck
	for {
		// 判断是否超时错误
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// 获取 channel
		ch, release, err := c.GetChannel(ctx)
		if err != nil {
			log.WarnCtx(ctx, "get channel error, will retry", zap.Error(err))
			// 重试
			time.Sleep(c.cfg.ReconnectInterval)
			continue
		}
		// 设置 QoS
		if opt.Qos.PrefetchCount > 0 || opt.Qos.PrefetchSize > 0 {
			err = ch.Qos(opt.Qos.PrefetchCount, opt.Qos.PrefetchSize, opt.Qos.Global)
			if err != nil {
				log.WarnCtx(ctx, "qos error, will retry", zap.Error(err))
				release()
				time.Sleep(c.cfg.ReconnectInterval)
				continue
			}
		}
		// 获取消费的 delivery
		deliveries, err := ch.Consume(opt.Queue, opt.Consumer, autoAck, opt.Exclusive, false, opt.NoWait, opt.Args)
		if err != nil {
			release()
			time.Sleep(c.cfg.ReconnectInterval)
			continue
		}
		consumerDone := make(chan struct{})

		go func() {
			defer close(consumerDone)
			defer release()
			// 消费方法
			var consume = func(param interface{}) {
				handler(param.(amqp.Delivery))
			}
			// 使用协程池
			pool, _ := ants.NewPoolWithFunc(opt.Qos.WorkerNum, consume)
			defer pool.Release()
			// 遍历数据处理
			for d := range deliveries {
				pool.Invoke(d)
			}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-consumerDone:
			// channel 关闭或 cancel，deliveries 已关闭，等待后重连并重新消费
		}
		time.Sleep(c.cfg.ReconnectInterval)
	}
}
