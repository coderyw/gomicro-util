// Package cst
// @Author: yinwei
// @File: registry
// @Version: 1.0.0
// @Date: 2024/8/13 09:19

package cst

// 注册中心类型
const (
	REGISTRY_TYPE_ETCD  = "etcd"
	REGISTRY_TYPE_NACOS = "nacos"
	REGISTRY_TYPE_K8S   = "k8s"
)

// 使用istio服务网格进行服务调用
// 严格来说这并不是注册中心，而是一层服务代理。
// 调用服务时，通过dns名称去调用，然后istio通过代理，会将dns名对应的service上，同时支持微服务治理
// 所以这里并不需要服务的注册和发现。会使用默认的registry，然后selector会使用istio进行处理
const DISCOVERY_TYPE_ISTIO = "istio"
