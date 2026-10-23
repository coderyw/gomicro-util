package rabbitmq

import (
	"context"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Client RabbitMQ 包装客户端，支持自动重连、Channel 管理、Publisher Confirm、手动 ACK、QoS、优雅关闭，可基于连接池扩展。
type Client struct {
	cfg     Config
	conn    *conn
	pool    *channelPool
	closing uint32
	closed  uint32
}

// NewClient 使用给定配置创建客户端并建立单连接。
func NewClient(cfg Config) (*Client, error) {
	c, err := newConn(cfg)
	if err != nil {
		return nil, err
	}
	return NewClientWithProvider(c, cfg)
}

// NewClientWithProvider 使用已有 ConnectionProvider（单连接或连接池）创建客户端，便于扩展为连接池。
func NewClientWithProvider(provider ConnectionProvider, cfg Config) (*Client, error) {
	size := cfg.ChannelPoolSize
	if size <= 0 {
		size = 10
	}
	pool := newChannelPool(provider, size)
	cli := &Client{cfg: cfg, pool: pool}
	if c, ok := provider.(*conn); ok {
		cli.conn = c
	}
	return cli, nil
}

// NewClientFromPool 从连接池创建客户端，Channel 将从池中轮询取连接创建。
func NewClientFromPool(pool *ConnectionPool, cfg Config) (*Client, error) {
	return NewClientWithProvider(pool, cfg)
}

// Close 关闭连接与池（若为单连接则关闭连接；若为池则仅关闭本 Client 的池逻辑，不关池）。
func (c *Client) Close() error {
	atomic.StoreUint32(&c.closed, 1)
	c.pool.close()
	if c.conn != nil {
		return c.conn.close()
	}
	return nil
}

// Shutdown 优雅关闭：先禁止新操作，等待在途 Channel 归还或超时，再关闭连接。
// ctx 超时使用 Config.ShutdownTimeout，若 ctx 未设 deadline 则用 ShutdownTimeout。
func (c *Client) Shutdown(ctx context.Context) error {
	atomic.StoreUint32(&c.closing, 1)
	timeout := c.cfg.ShutdownTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	for c.pool.inFlightCount() > 0 {
		select {
		case <-ctx.Done():
			return c.Close()
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	return c.Close()
}

// GetChannel 从池中获取一个 Channel，用完后必须调用返回的 release。
func (c *Client) GetChannel(ctx context.Context) (ch *amqp.Channel, release func(), err error) {
	if atomic.LoadUint32(&c.closing) != 0 || atomic.LoadUint32(&c.closed) != 0 {
		return nil, nil, amqp.ErrClosed
	}
	ch, err = c.pool.get(ctx)
	if err != nil {
		return nil, nil, err
	}
	release = func() { c.pool.release(ch) }
	return ch, release, nil
}

// GetPooledChannel 从池中获取一个 PooledChannel，用完后调用 PooledChannel.Close 归还并关闭底层 Channel。
func (c *Client) GetPooledChannel(ctx context.Context) (*PooledChannel, error) {
	if atomic.LoadUint32(&c.closing) != 0 || atomic.LoadUint32(&c.closed) != 0 {
		return nil, amqp.ErrClosed
	}
	ch, err := c.pool.get(ctx)
	if err != nil {
		return nil, err
	}
	return &PooledChannel{ch: ch, pool: c.pool}, nil
}
