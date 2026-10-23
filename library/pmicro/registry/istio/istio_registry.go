// Package istio
// @Author: yinwei
// @File: istio_registry
// @Version: 1.0.0
// @Date: 2024/8/13 14:12

package istio

import (
	"context"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/util/cmd"
	"strings"
	"sync"
)

type istioRegistry struct {
	opts      registry.Options
	defaultMd metadata.Metadata

	env      string
	grpcPort string
	lock     sync.RWMutex
	cache    map[string][]*registry.Service
}

func init() {
	cmd.DefaultRegistries["istio"] = NewRegistry
}

// NewRegistry NewRegistry.
func NewRegistry(opts ...registry.Option) registry.Registry {

	n := &istioRegistry{
		opts: registry.Options{
			Context: context.Background(),
		},
		cache: make(map[string][]*registry.Service),
		lock:  sync.RWMutex{},
	}
	for _, o := range opts {
		o(&n.opts)
	}
	// 获取配置
	val := n.opts.Context.Value("env")
	if val == nil {
		panic("istio env is nil")
	}

	n.env = val.(string)

	val = n.opts.Context.Value("grpcport")
	if val == nil {
		panic("istio grpcport is nil")
	}
	n.grpcPort = val.(string)

	arr := strings.Split(n.grpcPort, ":")
	n.grpcPort = arr[len(arr)-1]

	md := make(metadata.Metadata, 5)
	md.Set("server", "grpc")
	md.Set("protocol", "grpc")
	md.Set("transport", "grpc")
	md.Set("broker", "grpc")
	md.Set("version", "v0.0.1")
	n.defaultMd = md

	return n
}

func (n *istioRegistry) Init(opts ...registry.Option) error {
	return nil
}

func (n *istioRegistry) Options() registry.Options {
	return n.opts
}

func (n *istioRegistry) Register(s *registry.Service, opts ...registry.RegisterOption) error {

	return nil
}

func (n *istioRegistry) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {

	return nil
}

func (n *istioRegistry) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {

	n.lock.RLock()
	registries, ok := n.cache[name]
	n.lock.RUnlock()
	if !ok {
		n.lock.Lock()
		// 加锁间隙可能会有保存cache，再次确认是否存在
		if registries, ok = n.cache[name]; !ok {
			bu := strings.Builder{}
			bu.WriteString(name)
			bu.WriteString("-svc")
			bu.WriteString(":9804")
			//bu.WriteString("9804")
			registries = []*registry.Service{
				&registry.Service{
					Name:      name,
					Version:   "v0.0.1",
					Metadata:  n.defaultMd,
					Endpoints: nil,
					Nodes: []*registry.Node{
						&registry.Node{
							Metadata: n.defaultMd,
							Id:       name,
							Address:  bu.String(),
						},
					},
				},
			}
			n.cache[name] = registries
		}
		n.lock.Unlock()
	}
	return registries, nil
}

func (n *istioRegistry) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {

	return nil, nil
}

func (n *istioRegistry) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	return newIstioWatcher(opts...)
}

func (n *istioRegistry) String() string {
	return "istio"
}
