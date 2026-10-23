package nacos

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/registry/cache"
	"go-micro.dev/v4/selector"
	"go.uber.org/atomic"
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
		fmt.Println("获取服务错误。", err.Error())
		return nil, err
	}

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}
	//TODO 配置传入Filter
	for _, filter := range sopts.Filters {
		services = filter(services)
	}
	// nacos 不需要过滤
	//services = selector.FilterLabel(cst.ENV_ENVIRON, env.GetEnv())(services)

	if len(services) == 0 {
		return nil, selector.ErrNotFound
	}

	return func() (*registry.Node, error) {

		if _, ok := n.srv.Load(service); !ok {
			n.srv.Store(service, atomic.NewUint64(0))
		}
		val, _ := n.srv.Load(service)
		c := val.(*atomic.Uint64)
		cc := c.Inc()
		sc := len(services)
		node := services[cc%uint64(sc)].Nodes[0]
		// 兼容java的grpc，java会注册微服务到nacos，但是端口是web的端口，除非java不开启web端口。如果开启了web端口，需要这里替换一下端口。
		// 正则的grpc端口在node的metadata的gRPC_port
		if len(node.Metadata["gRPC_port"]) != 0 {
			//替换端口
			idnex := strings.LastIndex(node.Address, ":")
			re := strings.NewReplacer(node.Address[idnex:], ":"+node.Metadata["gRPC_port"])
			node.Address = re.Replace(node.Address)
		}

		return node, nil
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

func (n *pollNodeSelector) next() {

}

func PollNodeSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options
	for _, opt := range opts {
		opt(&sopts)
	}
	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}

	p := &pollNodeSelector{opts: sopts, srv: sync.Map{}}
	p.rc = p.newCache()
	return p
}

func (n *pollNodeSelector) NewPollNodeSelector() selector.Strategy {
	return func(services []*registry.Service) selector.Next {
		return nil
	}
}
