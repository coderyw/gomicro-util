package rabbitmq

import (
	"sync"
	"sync/atomic"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ConnectionPool 连接池，维护多路 AMQP 连接并轮询分配，可扩展为多节点负载均衡。
type ConnectionPool struct {
	conns  []*conn
	next   uint32
	closed uint32
	mu     sync.RWMutex
}

// NewConnectionPool 创建连接池，size 为连接数（至少 1）。每个连接独立自动重连。
func NewConnectionPool(cfg Config, size int) (*ConnectionPool, error) {
	if size <= 0 {
		size = 1
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	pool := &ConnectionPool{conns: make([]*conn, 0, size)}
	for i := 0; i < size; i++ {
		c, err := newConn(cfg)
		if err != nil {
			_ = pool.Close()
			return nil, err
		}
		pool.conns = append(pool.conns, c)
	}
	return pool, nil
}

// GetConnection 轮询返回池中一个连接，供 Channel 创建使用。
func (p *ConnectionPool) GetConnection() *amqp.Connection {
	if atomic.LoadUint32(&p.closed) == 1 {
		return nil
	}
	p.mu.RLock()
	n := len(p.conns)
	p.mu.RUnlock()
	if n == 0 {
		return nil
	}
	idx := atomic.AddUint32(&p.next, 1) % uint32(n)
	p.mu.RLock()
	defer p.mu.RUnlock()
	if idx >= uint32(len(p.conns)) {
		return nil
	}
	c := p.conns[idx]
	if c.isClosed() {
		return nil
	}
	return c.connection()
}

// Connection 实现 ConnectionProvider：轮询返回一个可用连接。
func (p *ConnectionPool) Connection() *amqp.Connection {
	return p.GetConnection()
}

// IsClosed 是否已调用 Close 关闭池。
func (p *ConnectionPool) IsClosed() bool {
	return atomic.LoadUint32(&p.closed) == 1
}

// Close 关闭池内所有连接。
func (p *ConnectionPool) Close() error {
	if !atomic.CompareAndSwapUint32(&p.closed, 0, 1) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	var err error
	for _, c := range p.conns {
		if c != nil {
			if e := c.close(); e != nil && err == nil {
				err = e
			}
		}
	}
	p.conns = nil
	return err
}
