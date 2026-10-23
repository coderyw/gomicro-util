package etcd

import (
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/model/cst"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/registry/cache"
	"go-micro.dev/v4/selector"
	"go.uber.org/atomic"
	"sync"
	"time"
)

type pollNodeSelector struct {
	opts selector.Options
	srv  sync.Map
	rc   cache.Cache
	mu   sync.RWMutex
}

func (c *pollNodeSelector) newCache() cache.Cache {
	opts := make([]cache.Option, 0, 1)

	if c.opts.Context != nil {
		if t, ok := c.opts.Context.Value("selector_ttl").(time.Duration); ok {
			opts = append(opts, cache.WithTTL(t))
		}
	}

	return cache.New(c.opts.Registry, opts...)
}

func (n *pollNodeSelector) Init(opts ...selector.Option) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, o := range opts {
		o(&n.opts)
	}
	if n.rc != nil {
		n.rc.Stop()
	}
	n.rc = n.newCache()
	return nil
}

func (n *pollNodeSelector) Options() selector.Options {
	return n.opts
}

func (n *pollNodeSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var sopts selector.SelectOptions
	for _, opt := range opts {
		opt(&sopts)
	}

	services, err := n.rc.GetService(service)
	if err != nil {
		n.opts.Logger.Logf(logger.ErrorLevel, "获取service错误 %v,err=%v\n", service, err.Error())
		return nil, err
	}

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}
	//TODO 配置传入Filter
	for _, filter := range sopts.Filters {
		services = filter(services)
	}

	services = selector.FilterLabel(cst.ENV_ENVIRON, env.GetEnv())(services)

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}
	if _, ok := n.srv.Load(service); !ok {
		n.srv.Store(service, atomic.NewUint64(0))
	}
	val, _ := n.srv.Load(service)
	c := val.(*atomic.Uint64)
	cc := c.Inc()

	s := services[0] //轮训servers
	nodeCount := len(s.Nodes)
	if nodeCount == 0 {
		return nil, selector.ErrNotFound
	}
	return func() (*registry.Node, error) {
		return s.Nodes[cc%uint64(nodeCount)], nil
	}, nil
}

func (n *pollNodeSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (n *pollNodeSelector) Reset(service string) {
	return
}

func (n *pollNodeSelector) Close() error {
	return nil
}

func (n *pollNodeSelector) String() string {
	return "first"
}

func PollNodeSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options
	for _, opt := range opts {
		opt(&sopts)
	}
	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	return &pollNodeSelector{opts: sopts, srv: sync.Map{}}
}
