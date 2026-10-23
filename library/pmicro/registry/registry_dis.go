// Package registry 提供微服务注册中心统一接口
// @Author: yinwei
// @File: registry_dis
// @Version: 1.0.0
// @Date: 2024/8/22 15:36

package registry

import (
	"fmt"

	"github.com/go-micro/plugins/v4/registry/kubernetes"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/registry/etcd"
	istio2 "github.com/coderyw/gomicro-util/library/pmicro/registry/istio"
	"github.com/coderyw/gomicro-util/library/pmicro/registry/nacos"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/model/cst"
	"go-micro.dev/v4/registry"
)

// RegDis 注册中心和服务发现配置结构体
type RegDis struct {
	RDType      string            // 注册中心或者发现中心类型
	Opts        []registry.Option // 注册中心选项
	Istio       bool              // 是否启用Istio
	ServiceName string            // 服务名称
}

// registryAndDiscovery 注册中心和服务发现实现结构体
type registryAndDiscovery struct {
	reg   registry.Registry // 普通注册中心实例
	istio registry.Registry // Istio注册中心实例
	name  string            // 注册中心名称
}

// NewRegistry 创建新的注册中心实例
func NewRegistry(reg RegDis) (registry.Registry, cron.CronJob, error) {
	rd := new(registryAndDiscovery)
	var cjob cron.CronJob
	// 根据类型创建对应的注册中心
	switch reg.RDType {
	case cst.REGISTRY_TYPE_ETCD:
		rd.reg = etcd.NewRegistry(reg.Opts...)
	case cst.REGISTRY_TYPE_NACOS:
		rd.reg, cjob = nacos.NewRegistry(reg.ServiceName, reg.Opts...)
	case cst.REGISTRY_TYPE_K8S:
		rd.reg = kubernetes.NewRegistry(reg.Opts...)
	}
	// 如果启用Istio，创建Istio注册中心实例
	if reg.Istio {
		rd.istio = istio2.NewRegistry(reg.Opts...)
	}

	// 设置注册中心名称，优先使用普通注册中心
	if rd.reg != nil {
		rd.name = reg.RDType
	} else if rd.istio != nil {
		rd.name = "istio"
	} else {
		return nil, cjob, fmt.Errorf("registry is nil")
	}

	return rd, cjob, nil
}

// Init 初始化注册中心
func (n *registryAndDiscovery) Init(opts ...registry.Option) error {
	if n.reg != nil {
		return n.reg.Init(opts...)
	}
	return n.istio.Init(opts...)
}

// Options 获取注册中心选项
func (n *registryAndDiscovery) Options() registry.Options {
	return registry.Options{}
}

// Register 注册服务到注册中心
func (n *registryAndDiscovery) Register(s *registry.Service, opts ...registry.RegisterOption) error {
	if n.reg != nil {
		return n.reg.Register(s, opts...)
	}
	return n.istio.Register(s, opts...)
}

// Deregister 从注册中心注销服务
func (n *registryAndDiscovery) Deregister(s *registry.Service, opts ...registry.DeregisterOption) error {
	log.Info("从注册中心注销")
	if n.reg != nil {
		return n.reg.Deregister(s, opts...)
	}
	return n.istio.Deregister(s, opts...)
}

// GetService 根据服务名称获取服务信息
func (n *registryAndDiscovery) GetService(name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	// 优先使用Istio进行服务发现
	if n.istio != nil {
		return n.istio.GetService(name, opts...)
	}
	if n.reg != nil {

		return n.reg.GetService(name, opts...)
	}
	return nil, nil
}

// ListServices 列出所有注册的服务
func (n *registryAndDiscovery) ListServices(opts ...registry.ListOption) ([]*registry.Service, error) {
	// 优先使用Istio进行服务发现
	if n.istio != nil {
		return n.istio.ListServices(opts...)
	}
	if n.reg != nil {
		return n.reg.ListServices(opts...)
	}
	return nil, nil
}

// Watch 监听服务变化
func (n *registryAndDiscovery) Watch(opts ...registry.WatchOption) (registry.Watcher, error) {
	// 优先使用Istio进行服务监听
	if n.istio != nil {
		return n.istio.Watch(opts...)
	}
	if n.reg != nil {
		return n.reg.Watch(opts...)
	}
	return nil, fmt.Errorf("缺失注册中心")
}

// String 返回注册中心名称字符串
func (n *registryAndDiscovery) String() string {
	return n.name
}
