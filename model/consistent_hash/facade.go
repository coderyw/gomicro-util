// Package consistent_hash
// @Author: yinwei
// @File: facade
// @Version: 1.0.0
// @Date: 2025/6/12 15:31

package consistent_hash

type Listener interface {
	// 服务发生变化时候 新的服务
	OnChanged() chan Notice
	// 获取当前服务
	OnNow() Notice
}

type Notice struct {
	ServiceName string
	Err         error
	Services    []Instance
}

type Instance struct {
	Host string //ip 地址加 port
	Id   string
}
