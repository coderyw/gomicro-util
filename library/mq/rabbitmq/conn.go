package rabbitmq

import (
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

// ConnectionProvider 提供 AMQP 连接，用于单连接或连接池。实现方需保证自动重连由外部或自身完成。
type ConnectionProvider interface {
	Connection() *amqp.Connection
	IsClosed() bool
}

// conn 维护底层 AMQP 连接，并在断开时自动重连。
type conn struct {
	cfg    Config
	url    string
	amqp   *amqp.Connection
	mu     sync.RWMutex
	closed bool
	done   chan struct{}
}

func newConn(cfg Config) (*conn, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	url := cfg.AMQPURL()
	ac := amqp.Config{
		Vhost:     cfg.VirtualHost,
		Heartbeat: cfg.Heartbeat,
	}
	if ac.Heartbeat == 0 {
		ac.Heartbeat = 10 * time.Second
	}
	c, err := amqp.DialConfig(url, ac)
	if err != nil {
		return nil, err
	}
	conn := &conn{
		cfg:  cfg,
		url:  url,
		amqp: c,
		done: make(chan struct{}),
	}
	go conn.runNotifyClose()
	return conn, nil
}

func (c *conn) runNotifyClose() {
	closeCh := c.amqp.NotifyClose(make(chan *amqp.Error, 1))
	select {
	case err, ok := <-closeCh:
		if ok && err != nil {
			log.Warn("rabbitmq connection closed", zap.Error(err))
			c.reconnect()
		}
	case <-c.done:
		return
	}
}

func (c *conn) reconnect() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	_ = c.amqp.Close()
	c.mu.Unlock()

	interval := c.cfg.ReconnectInterval
	maxWait := c.cfg.MaxReconnectWait
	if maxWait == 0 {
		maxWait = 30 * time.Second
	}
	if interval == 0 {
		interval = 3 * time.Second
	}
	wait := interval
	for {
		select {
		case <-c.done:
			return
		default:
		}
		ac := amqp.Config{
			Vhost:     c.cfg.VirtualHost,
			Heartbeat: c.cfg.Heartbeat,
		}
		if ac.Heartbeat == 0 {
			ac.Heartbeat = 10 * time.Second
		}
		nc, err := amqp.DialConfig(c.url, ac)
		if err != nil {
			log.Warn("rabbitmq reconnect failed", zap.Error(err), zap.Duration("next", wait))
			time.Sleep(wait)
			if wait < maxWait {
				wait += interval
				if wait > maxWait {
					wait = maxWait
				}
			}
			continue
		}
		c.mu.Lock()
		if c.closed {
			_ = nc.Close()
			c.mu.Unlock()
			return
		}
		c.amqp = nc
		c.mu.Unlock()
		wait = interval
		log.Info("rabbitmq reconnected")
		go c.runNotifyClose()
		return
	}
}

func (c *conn) connection() *amqp.Connection {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.amqp
}

// Connection 实现 ConnectionProvider。
func (c *conn) Connection() *amqp.Connection {
	return c.connection()
}

// IsClosed 实现 ConnectionProvider。
func (c *conn) IsClosed() bool {
	return c.isClosed()
}

func (c *conn) close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	if c.amqp != nil {
		return c.amqp.Close()
	}
	return nil
}

func (c *conn) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}
