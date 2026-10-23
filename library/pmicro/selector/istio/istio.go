// Package istio_registry
// @Author: yinwei
// @File: istio
// @Version: 1.0.0
// @Date: 2024/8/13 10:33

package istio

import (
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/registry/cache"
	"go-micro.dev/v4/selector"
	"time"
)

// 准确来说 这并不是注册发现中心
// 在使用Istio服务网格时，不需要在A的应用代码中显式写明B服务的IP地址。Istio通过服务发现和负载均衡机制，使得微服务A能够通过服务名直接调用微服务B，而无需关心B服务的实际IP地址。

// 在Kubernetes和Istio环境中，微服务之间的通信通常通过服务的DNS名称来实现。以下是调用B服务时的具体步骤：
//
//	1.	服务注册与发现：
//	•	每个微服务在Kubernetes中部署时，都会被注册为一个Kubernetes Service对象。
//	•	这个Service对象有一个固定的DNS名称，通常是<service-name>.<namespace>.svc.cluster.local。
//	2.	使用服务名调用：
//	•	在A的应用代码中，通过gRPC客户端调用B服务时，可以直接使用B服务的DNS名称，而不是硬编码IP地址。

// 例如，假设B服务的Kubernetes Service名称为grpc-server，并且它位于default命名空间中，A服务的gRPC客户端可以这样调用：
/**
conn, err := grpc.Dial("grpc-server.default.svc.cluster.local:50051", grpc.WithInsecure())
if err != nil {
    log.Fatalf("did not connect: %v", err)
}
defer conn.Close()

c := pb.NewYourServiceClient(conn)

ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

r, err := c.YourMethod(ctx, &pb.YourRequest{Name: "world"})
if err != nil {
    log.Fatalf("could not greet: %v", err)
}
log.Printf("Greeting: %s", r.GetMessage())

*/

// Istio的作用
//
//在使用Istio时，尽管A的应用代码中只是使用了B服务的DNS名称进行调用，但实际的请求流程如下：
//
//	1.	服务名解析：DNS请求将grpc-server.default.svc.cluster.local解析为B服务的Cluster IP（或直接路由到Envoy代理）。
//	2.	Envoy代理：A的Envoy代理会拦截这个请求，根据Istio的路由规则（如VirtualService和DestinationRule）决定将请求发送到哪个具体的Pod。
//	3.	负载均衡：Envoy代理根据从Istio Pilot获得的服务发现信息，选择B服务的一个Pod进行负载均衡，然后将请求转发给B服务的Envoy代理。
//	4.	服务调用：B服务的Envoy代理接收请求并将其转发到B服务的应用代码

type istioSelector struct {
	opts selector.Options
}

func (c *istioSelector) newCache() cache.Cache {
	opts := make([]cache.Option, 0, 1)

	if c.opts.Context != nil {
		if t, ok := c.opts.Context.Value("selector_ttl").(time.Duration); ok {
			opts = append(opts, cache.WithTTL(t))
		}
	}

	return cache.New(c.opts.Registry, opts...)
}

func (n *istioSelector) Init(opts ...selector.Option) error {
	for _, o := range opts {
		o(&n.opts)
	}
	return nil
}

func (n *istioSelector) Options() selector.Options {
	return n.opts
}

// grpc-server.default.svc.cluster.local
func (n *istioSelector) Select(service string, opts ...selector.SelectOption) (selector.Next, error) {
	var sopts selector.SelectOptions
	for _, opt := range opts {
		opt(&sopts)
	}
	services, err := n.opts.Registry.GetService(service)
	if err != nil {
		n.opts.Logger.Logf(logger.ErrorLevel, "获取service错误 %v,err=%v\n", service, err.Error())
		return nil, err
	}
	if len(services) == 0 || len(services[0].Nodes) == 0 {
		return nil, selector.ErrNotFound

	}
	return func() (*registry.Node, error) {
		return services[0].Nodes[0], nil
	}, nil
}

func (n *istioSelector) Mark(service string, node *registry.Node, err error) {
	return
}

func (n *istioSelector) Reset(service string) {
	return
}

func (n *istioSelector) Close() error {
	return nil
}

func (n *istioSelector) String() string {
	return "first"
}

func (n *istioSelector) next() {

}

func IstioSelector(opts ...selector.Option) selector.Selector {
	var sopts selector.Options
	for _, opt := range opts {
		opt(&sopts)
	}
	if sopts.Registry == nil {
		sopts.Registry = registry.DefaultRegistry
	}
	// 获取配置
	p := &istioSelector{
		opts: sopts,
	}

	return p
}

func (n *istioSelector) NewIstioSelector() selector.Strategy {
	return func(services []*registry.Service) selector.Next {
		return nil
	}
}
