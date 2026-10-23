package grpc

import (
	"context"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

type pool struct {
	size int
	ttl  int64

	//  max streams on a *poolConn
	maxStreams int
	//  max idle conns
	maxIdle int

	sync.Mutex
	conns map[string]*streamsPool
}

type streamsPool struct {
	//  head of list
	head *poolConn
	//  busy conns list
	busy *poolConn
	//  the size of list
	count int
	//  idle conn
	idle int
}

type poolConn struct {
	//  grpc conn
	*grpc.ClientConn
	err  error
	addr string

	//  pool and streams pool
	pool    *pool
	sp      *streamsPool
	streams int
	created int64

	//  list
	pre  *poolConn
	next *poolConn
	in   bool
}

func newPool(size int, ttl time.Duration, idle int, ms int) *pool {
	if ms <= 0 {
		ms = 1
	}
	if idle < 0 {
		idle = 0
	}
	return &pool{
		size:       size,
		ttl:        int64(ttl.Seconds()),
		maxStreams: ms,
		maxIdle:    idle,
		conns:      make(map[string]*streamsPool),
	}
}

func (p *pool) getConn(dialCtx context.Context, addr string, opts ...grpc.DialOption) (*poolConn, error) {
	now := time.Now().Unix()
	p.Lock()
	sp, ok := p.conns[addr]
	if !ok {
		sp = &streamsPool{head: &poolConn{}, busy: &poolConn{}, count: 0, idle: 0}
		p.conns[addr] = sp
	}
	//  while we have conns check streams and then return one
	//  otherwise we'll create a new conn
	conn := sp.head.next
	for conn != nil {
		//  check conn state
		// https://github.com/grpc/grpc/blob/master/doc/connectivity-semantics-and-api.md
		switch conn.GetState() {
		case connectivity.Connecting:
			conn = conn.next
			continue
		case connectivity.Shutdown:
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				sp.idle--
			}
			conn = next
			continue
		case connectivity.TransientFailure:
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				conn.ClientConn.Close()
				sp.idle--
			}
			conn = next
			continue
		case connectivity.Ready:
		case connectivity.Idle:
		}
		//  a old conn
		if now-conn.created > p.ttl {
			next := conn.next
			if conn.streams == 0 {
				removeConn(conn)
				conn.ClientConn.Close()
				sp.idle--
			}
			conn = next
			continue
		}
		//  a busy conn
		if conn.streams >= p.maxStreams {
			next := conn.next
			removeConn(conn)
			addConnAfter(conn, sp.busy)
			conn = next
			continue
		}
		//  a idle conn
		if conn.streams == 0 {
			sp.idle--
		}
		//  a good conn
		conn.streams++
		p.Unlock()
		return conn, nil
	}
	p.Unlock()

	//  create new conn
	cc, err := grpc.DialContext(dialCtx, addr, opts...)
	if err != nil {
		return nil, err
	}
	conn = &poolConn{cc, nil, addr, p, sp, 1, time.Now().Unix(), nil, nil, false}

	//  add conn to streams pool
	p.Lock()
	if sp.count < p.size {
		addConnAfter(conn, sp.head)
	}
	p.Unlock()

	return conn, nil
}

func (p *pool) release(addr string, conn *poolConn, err error) {
	p.Lock()
	p, sp, created := conn.pool, conn.sp, conn.created
	//  try to add conn
	if !conn.in && sp.count < p.size {
		addConnAfter(conn, sp.head)
	}
	if !conn.in {
		p.Unlock()
		conn.ClientConn.Close()
		return
	}
	//  a busy conn
	if conn.streams >= p.maxStreams {
		removeConn(conn)
		addConnAfter(conn, sp.head)
	}
	conn.streams--
	//  if streams == 0, we can do something
	if conn.streams == 0 {
		//  1. it has errored
		//  2. too many idle conn or
		//  3. conn is too old
		now := time.Now().Unix()
		if isConnErr(err) || sp.idle >= p.maxIdle || now-created > p.ttl {
			removeConn(conn)
			p.Unlock()
			conn.ClientConn.Close()
			return
		}
		sp.idle++
	}
	p.Unlock()
}

func (conn *poolConn) Close() {
	conn.pool.release(conn.addr, conn, conn.err)
}

// isConnErr 判断是否为连接级错误，只有这类错误才需要丢弃连接。
// 收到服务端返回的 gRPC 状态码（含业务错误使用的 Internal 等）说明 HTTP/2 流已完整走完，
// 底层连接是健康的，不能因此关闭连接，否则每个业务错误都会触发一次连接重建。
// codes.Unavailable：连接被拒/GOAWAY/传输中断，连接不可用。
// 非 status 错误（如 go-micro 的超时错误）属于流级别取消，连接仍可复用，
// 真正坏死的连接会在 getConn 的状态检查（TransientFailure/Shutdown）中被剔除。
func isConnErr(err error) bool {
	if err == nil {
		return false
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DataLoss:
		return true
	default:
		return false
	}
}

func removeConn(conn *poolConn) {
	if conn.pre != nil {
		conn.pre.next = conn.next
	}
	if conn.next != nil {
		conn.next.pre = conn.pre
	}
	conn.pre = nil
	conn.next = nil
	conn.in = false
	conn.sp.count--
	return
}

func addConnAfter(conn *poolConn, after *poolConn) {
	conn.next = after.next
	conn.pre = after
	if after.next != nil {
		after.next.pre = conn
	}
	after.next = conn
	conn.in = true
	conn.sp.count++
	return
}
