// Package nacos
// @Author: yinwei
// @File: watcher
// @Version: 1.0.0
// @Date: 2024/6/26 18:00

package nacos

import (
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/consistent_hash"
	"go.uber.org/zap"

	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go-micro.dev/v4/registry"
)

type watcher struct {
	n  *nacosRegistry
	wo registry.WatchOptions

	next chan *registry.Result
	exit chan bool

	sync.RWMutex
	services      map[string][]*registry.Service
	cacheServices map[string][]model.Instance
	param         *vo.SubscribeParam
	Doms          []string
	watchNotice   chan consistent_hash.Notice
}

func newWatcher(nr *nacosRegistry, opts ...registry.WatchOption) (registry.Watcher, error) {
	var wo registry.WatchOptions
	for _, o := range opts {
		o(&wo)
	}
	nw := watcher{
		n:             nr,
		wo:            wo,
		exit:          make(chan bool),
		next:          make(chan *registry.Result, 10),
		services:      make(map[string][]*registry.Service),
		cacheServices: make(map[string][]model.Instance),
		param:         new(vo.SubscribeParam),
		Doms:          make([]string, 0),
	}

	param := &vo.SubscribeParam{
		ServiceName:       wo.Service,
		SubscribeCallback: nw.callBackHandle,
	}
	go nr.client.Subscribe(param)

	return &nw, nil
}

// callBackHandle 处理订阅推送：services 是该服务的全量实例列表，与上一次的列表按 InstanceId 比对，
// 逐个下发 create / update / delete。
//
// 列表可能为空（UpdateCacheWhenEmpty=true 且实例全部下线），不能按下标取服务名，否则 panic 会打死推送协程所在进程；
// 服务名用订阅时的 wo.Service 作为缓存键。
// 回调在 SDK 的推送协程里同步执行，下发时同时监听 exit，watcher 停止后不再阻塞推送协程。
func (nw *watcher) callBackHandle(services []model.Instance, err error) {
	if err != nil {
		log.Error("[Nacos] watcher call back handle error", zap.Error(err))
		return
	}
	key := nw.wo.Service

	nw.Lock()
	old := nw.cacheServices[key]
	nw.cacheServices[key] = append([]model.Instance(nil), services...)
	nw.Unlock()

	oldByID := make(map[string]model.Instance, len(old))
	for _, v := range old {
		oldByID[v.InstanceId] = v
	}
	for i := range services {
		cur := &services[i]
		prev, ok := oldByID[cur.InstanceId]
		switch {
		case !ok:
			log.Info("[Nacos] create", zap.String("serviceName", cur.ServiceName), zap.String("ip", cur.Ip), zap.Uint64("port", cur.Port))
			if !nw.send(&registry.Result{Action: "create", Service: buildRegistryService(cur)}) {
				return
			}
		case !reflect.DeepEqual(prev, *cur):
			if !nw.send(&registry.Result{Action: "update", Service: buildRegistryService(cur)}) {
				return
			}
		}
		delete(oldByID, cur.InstanceId)
	}
	for _, v := range old {
		if _, gone := oldByID[v.InstanceId]; !gone {
			continue
		}
		log.Info("[Nacos] del", zap.String("serviceName", v.ServiceName), zap.String("ip", v.Ip), zap.Uint64("port", v.Port))
		if !nw.send(&registry.Result{Action: "delete", Service: buildRegistryService(&v)}) {
			return
		}
	}
}

// send 下发一条变更；watcher 已停止时返回 false
func (nw *watcher) send(r *registry.Result) bool {
	select {
	case <-nw.exit:
		return false
	case nw.next <- r:
		return true
	}
}

func buildRegistryService(v *model.Instance) (s *registry.Service) {
	nodes := make([]*registry.Node, 0)
	nodes = append(nodes, &registry.Node{
		Id:       v.InstanceId,
		Address:  net.JoinHostPort(v.Ip, fmt.Sprintf("%d", v.Port)),
		Metadata: v.Metadata,
	})
	s = &registry.Service{
		Name:     v.ServiceName,
		Version:  "latest",
		Metadata: v.Metadata,
		Nodes:    nodes,
	}
	return
}

func (nw *watcher) Next() (r *registry.Result, err error) {
	select {
	case <-nw.exit:
		return nil, registry.ErrWatcherStopped
	case r, ok := <-nw.next:
		if !ok {
			return nil, registry.ErrWatcherStopped
		}
		return r, nil
	}
}

func (nw *watcher) Stop() {
	select {
	case <-nw.exit:
		return
	default:
		close(nw.exit)
		if len(nw.Doms) > 0 {
			for _, v := range nw.Doms {
				param := &vo.SubscribeParam{
					ServiceName:       v,
					SubscribeCallback: nw.callBackHandle,
				}
				_ = nw.n.client.Unsubscribe(param)
			}
		} else {
			_ = nw.n.client.Unsubscribe(nw.param)
		}
	}
}
