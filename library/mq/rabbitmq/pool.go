package rabbitmq

import (
	"context"
	"sync"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"
)

// channelPool 限制并发 Channel 数量，从 ConnectionProvider 按需创建 Channel，支持优雅关闭时的在途计数。
type channelPool struct {
	provider ConnectionProvider
	size     int
	sem      chan struct{}
	inFlight int64
	closed   int32
}

func newChannelPool(provider ConnectionProvider, size int) *channelPool {
	if size <= 0 {
		size = 10
	}
	p := &channelPool{
		provider: provider,
		size:     size,
		sem:      make(chan struct{}, size),
	}
	for i := 0; i < size; i++ {
		p.sem <- struct{}{}
	}
	return p
}

func (p *channelPool) isClosed() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

// get 从当前连接创建一个 Channel，受池大小限制。若 ctx 取消或连接已关闭则返回错误。
func (p *channelPool) get(ctx context.Context) (*amqp.Channel, error) {
	if p.isClosed() {
		return nil, amqp.ErrClosed
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if p.provider.IsClosed() {
		return nil, amqp.ErrClosed
	}
	select {
	case <-p.sem:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	atomic.AddInt64(&p.inFlight, 1)
	c := p.provider.Connection()
	if c == nil {
		atomic.AddInt64(&p.inFlight, -1)
		p.sem <- struct{}{}
		return nil, amqp.ErrClosed
	}
	ch, err := c.Channel()
	if err != nil {
		atomic.AddInt64(&p.inFlight, -1)
		p.sem <- struct{}{}
		return nil, err
	}
	return ch, nil
}

// release 释放池槽并关闭 Channel。调用方在不再使用 Channel 时必须调用。
func (p *channelPool) release(ch *amqp.Channel) {
	atomic.AddInt64(&p.inFlight, -1)
	if ch != nil {
		_ = ch.Close()
	}
	select {
	case p.sem <- struct{}{}:
	default:
	}
}

// inFlightCount 当前未归还的 Channel 数量，用于优雅关闭等待。
func (p *channelPool) inFlightCount() int64 {
	return atomic.LoadInt64(&p.inFlight)
}

// close 标记池为已关闭，后续 get 将返回 ErrClosed。
func (p *channelPool) close() {
	atomic.StoreInt32(&p.closed, 1)
}

// PooledChannel 池化 Channel 封装，用完后必须调用 Close 以归还槽位并关闭底层 Channel。
type PooledChannel struct {
	ch   *amqp.Channel
	pool *channelPool
	once sync.Once
}

// Channel 返回底层 *amqp.Channel，供声明队列、发布、消费等使用。
func (c *PooledChannel) Channel() *amqp.Channel {
	return c.ch
}

// Close 关闭底层 Channel 并归还池槽，仅首次调用生效。
func (c *PooledChannel) Close() error {
	var err error
	c.once.Do(func() {
		if c.ch != nil {
			err = c.ch.Close()
		}
		c.pool.release(c.ch)
		c.ch = nil
	})
	return err
}
