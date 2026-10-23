// Package nacos
// @Author: yinwei
// @File: consistent_hash
// @Version: 1.0.0
// @Date: 2025/6/12 18:18

package nacos

import (
	"errors"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	consistent_hash2 "github.com/coderyw/gomicro-util/library/consistent_hash"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/consistent_hash"
	"go.uber.org/zap"
)

type consistentHashWatch struct {
	ch          chan consistent_hash.Notice
	serviceName string
}

func (c *consistentHashWatch) OnChanged() chan consistent_hash.Notice {
	return c.ch
}

// 获取当前服务
//
// 只取 healthy 实例：非优雅退出的实例在 nacos 心跳超时前仍会出现在全量列表里，
// 若把它算进哈希环，分给它的分区会无人执行（幽灵节点）。
// 这里的过滤语义必须与订阅回调保持一致，否则推/拉两条通路会产生不同的环视图。
func (c *consistentHashWatch) OnNow() consistent_hash.Notice {
	all, err := client.SelectInstances(vo.SelectInstancesParam{
		ServiceName: c.serviceName,
		HealthyOnly: true,
	})
	if err != nil {
		return consistent_hash.Notice{
			Err: err,
		}
	}
	ins := buildInstances(all)
	// 本进程自身存活却拉到空列表，基本是注册中心抖动；
	// 返回错误让调用方保留当前环，避免瞬间清空所有分区分配
	if len(ins) == 0 {
		return consistent_hash.Notice{
			Err: errors.New("nacos returns no healthy instance, keep current hash ring"),
		}
	}
	return consistent_hash.Notice{
		ServiceName: c.serviceName,
		Err:         nil,
		Services:    ins,
	}
}

func (c *consistentHashWatch) consistentHashCallBackHandler(services []model.Instance, err error) {
	if err != nil {
		select {
		case c.ch <- consistent_hash.Notice{Err: err}:
		default:
			log.Warn("[Nacos] Notice consistent hash is full")
		}
		return
	}
	var serviceName string
	for _, v := range services {
		if len(serviceName) == 0 {
			serviceName = v.ServiceName
		}
	}
	ins := buildInstances(services)
	// 空列表不下发：订阅推送的空集大概率是注册中心抖动，
	// 且 10s 一次的 OnNow 拉取会兜底纠正真实的成员变化
	if len(ins) == 0 {
		log.Warn("[Nacos] subscribe push has no healthy instance, skip", zap.String("serviceName", serviceName))
		return
	}
	select {
	case c.ch <- consistent_hash.Notice{
		Err:         nil,
		ServiceName: serviceName,
		Services:    ins,
	}:
	default:
		log.Warn("[Nacos] Notice consistent hash is full")
	}

}

// buildInstances 统一推/拉两条通路的实例过滤规则：
// 只保留 healthy && enable 且 metadata id 非空的实例。
// id 为空的实例在哈希环里会互相折叠成同一节点，必须剔除。
func buildInstances(services []model.Instance) []consistent_hash.Instance {
	ins := make([]consistent_hash.Instance, 0, len(services))
	for _, v := range services {
		if !v.Healthy || !v.Enable {
			continue
		}
		id := v.Metadata["id"]
		if len(id) == 0 {
			log.Warn("[Nacos] instance metadata id is empty, skip",
				zap.String("ip", v.Ip),
				zap.String("serviceName", v.ServiceName))
			continue
		}
		ins = append(ins, consistent_hash.Instance{
			Id:   id,
			Host: v.Ip,
		})
	}
	return ins
}

func watch(serviceName string) {
	c := consistentHashWatch{ch: make(chan consistent_hash.Notice, 1024)}
	c.serviceName = serviceName
	consistent_hash2.Init(&c)

	param := &vo.SubscribeParam{
		ServiceName:       serviceName,
		SubscribeCallback: c.consistentHashCallBackHandler,
	}
	go func() {
		//time.Sleep(time.Minute)q
		client.Subscribe(param)
	}()

}
