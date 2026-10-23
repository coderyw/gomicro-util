// Package consistent_hash
// @Author: yinwei
// @File: server_cst_hash
// @Version: 1.0.0
// @Date: 2025/6/12 15:20

package consistent_hash

import (
	"errors"
	"sync"
	"time"

	"github.com/coderyw/gomicro-util/library/consistent_hash/hash_circle"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/consistent_hash"
	"go.uber.org/zap"
)

var ErrNotInit = errors.New("consistent hash not init")

// 整个节点数 量
const (
	loadServerInterval = 10 //定时刷新服务列表
	// 每 N 次定时刷新强制重下发一次分区分配（约 1 分钟）。
	// 分配通知本身是 change-only 的，一旦某次通知丢失或消费方状态漂移，
	// 没有重下发就会永远保持错误分配；周期重下发让这类问题在一分钟内自愈
	forceNoticeEveryTicks = 6
	// 每 N 次定时刷新打印一次环快照（约 5 分钟），用于事后对账：
	// 环成员 + 本实例认领的分区
	snapshotEveryTicks = 30
)

// 微服务端的一致性 hash
type ServerCstHash struct {
	// 服务变动监听器
	l consistent_hash.Listener
	// 一致性哈希环计算器
	c hash_circle.ConsistentHash
	// 需要通知的业务
	notice     sync.Map //map[string]keyInfo
	serversMap map[string]struct{}
}

// 变化通知
type ChangeNotice struct {
	// 当前业务 key
	BusinessKey string
	// 执行 jobs
	Jobs []string
}

// 业务信息
type keyInfo struct {
	// 业务 key
	businessKey string
	// 需要执行的 job
	jobs []string
	// 通知通道
	chNotice chan ChangeNotice
}

var (
	s    *ServerCstHash
	once sync.Once
)

// Init 初始化一致性哈希，不需要调用，底层初始化
//
// Parameters:
//   - listener consistent_hash.Listener 服务变动监听器
func Init(listener consistent_hash.Listener) {
	// 只初始化一次
	once.Do(func() {
		c := hash_circle.NewSortHash()
		s1 := &ServerCstHash{
			l:          listener,
			notice:     sync.Map{},
			c:          c,
			serversMap: make(map[string]struct{}),
		}
		s = s1
		// 启动协程，监听服务变动
		go s.change()
	})

}

func (s *ServerCstHash) change() {
	//

	log.Info("[Consistent Hash] change start")
	// 获取监听通道
	ch := s.l.OnChanged()
	defer s.close()
	//定时加载变动
	ticker := time.NewTicker(time.Second * loadServerInterval)
	defer func() {
		ticker.Stop()
		<-ticker.C
	}()
	tick := 0
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				log.Info("[Consistent Hash] change stop")
				return
			}
			s.onChange(v)
			ticker.Reset(time.Second * loadServerInterval)
		case <-ticker.C:
			// 重写拉取当前服务节点
			s.onChange(s.l.OnNow())
			tick++
			if tick%forceNoticeEveryTicks == 0 {
				s.renoticeAll()
			}
			if tick%snapshotEveryTicks == 0 {
				s.logSnapshot()
			}
		}
	}

}

// renoticeAll 无条件重算并下发所有 businessKey 的分区分配。
// onChange 只在成员变化时下发；这里补上无变化时的周期性重下发
func (s *ServerCstHash) renoticeAll() {
	s.notice.Range(func(key, value any) bool {
		s.noticeJob(value.(keyInfo))
		return true
	})
}

// logSnapshot 打印当前环成员与本实例分区分配的快照
func (s *ServerCstHash) logSnapshot() {
	members := s.c.Members()
	jobs := make(map[string][]string)
	s.notice.Range(func(key, value any) bool {
		v := value.(keyInfo)
		jobs[v.businessKey] = s.getCurJob(v.jobs)
		return true
	})
	log.Info("[Consistent Hash] ring snapshot",
		zap.String("current", env.GetId()),
		zap.Int("memberCount", len(members)),
		zap.Any("members", members),
		zap.Any("myJobs", jobs))
}

// 服务发生变化处理
func (s *ServerCstHash) onChange(v consistent_hash.Notice) {
	if v.Err != nil {
		log.Error("[Consistent Hash] change error", zap.Error(v.Err))
		return
	}
	add, del := s.serverHasDiff(v.Services)
	if len(add) == 0 && len(del) == 0 {
		// 服务没有变化
		log.Debug("[Consistent Hash] service no change")
		return
	}

	// 根据变化后的 server 重新初始化一致性哈希环
	s.setService(add, del)

	log.Info("[Consistent Hash] service change", zap.Any("services", v.Services), zap.Any("add", add), zap.Any("del", del), zap.String("current", env.GetId()))

	//遍历所有需要通知的 businessKey
	s.notice.Range(func(key, value any) bool {
		v := value.(keyInfo)

		s.noticeJob(v)
		return true
	})
}

func (s *ServerCstHash) noticeJob(v keyInfo) {
	vals := s.getCurJob(v.jobs)
	log.Info("[Consistent Hash] job change", zap.String("businessKey", v.businessKey), zap.Any("jobs", vals))
	// 非阻塞下发：消费方停摆时阻塞在这里会冻结整个 change 循环，
	// 导致本实例的环视图永久过期。丢弃的通知由周期性 renoticeAll 补发
	select {
	case v.chNotice <- ChangeNotice{
		BusinessKey: v.businessKey,
		Jobs:        vals,
	}:
	default:
		log.Warn("[Consistent Hash] notice channel full, dropped (will be re-sent by periodic renotice)",
			zap.String("businessKey", v.businessKey))
	}

}

// 设置运行中的服务节点
func (s *ServerCstHash) setService(add, del []string) {
	//新增
	for _, v := range add {
		s.c.Add(v)
		s.serversMap[v] = struct{}{}
	}
	//删除
	for _, v := range del {
		s.c.Remove(v)
		delete(s.serversMap, v)
	}
	//log.Info("[Consistent Hash] change now consistent", zap.Any("now", s.c.NumberOfReplicas))

	// 重新设置一致性哈希环,根据节点数量去处理
	//if len(add) != 0 || len(del) != 0 {
	//	mem := s.c.Members() //获取当前所有节点
	//	n := len(mem)
	//	rep := s.c.NumberOfReplicas
	//	// 副本数总为 1000上限
	//	for i := 10; i <= 100; i = i + 10 {
	//		if n <= i {
	//			rep = 1000 / i
	//			break
	//		}
	//	}
	//	if s.c.NumberOfReplicas != rep {
	//		log.Info("[Consistent Hash] change new consistent", zap.Any("old", s.c.NumberOfReplicas), zap.Int("new", rep))
	//		nn := consistent.New()
	//		nn.NumberOfReplicas = rep
	//		nn.Set(mem)
	//		s.c = nn
	//	}
	//}

}

// 判断服务是否发生变更
func (s *ServerCstHash) serverHasDiff(svcs []consistent_hash.Instance) ([]string, []string) {
	// 先校验新的 svcs 是否与当前不一致
	var add []string
	mp := make(map[string]struct{})
	for _, v := range svcs {
		_, ok := s.serversMap[v.Id]
		if !ok {
			add = append(add, v.Id)
		}
		mp[v.Id] = struct{}{}
	}
	// 删除的服务
	var del []string
	for v := range s.serversMap {
		_, ok := mp[v]
		if !ok {
			del = append(del, v)
		}
	}
	return add, del
}

// 关闭
func (s *ServerCstHash) close() {
	log.Info("[Consistent Hash] close")
	s.notice.Range(func(key, value any) bool {
		v := value.(keyInfo)
		close(v.chNotice)
		return true
	})
}

// 获取当前服务可执行任务
func (s *ServerCstHash) getCurJob(arr []string) []string {
	return s.c.Gets(env.GetId(), arr)
}

func (s *ServerCstHash) add(key string, values []string) (bool, <-chan ChangeNotice) {
	if _, ok := s.notice.Load(key); ok {
		return false, nil
	}
	ch := make(chan ChangeNotice, 1)
	v := keyInfo{
		businessKey: key,
		jobs:        values,
		chNotice:    ch,
	}
	s.notice.Store(key, v)
	s.noticeJob(v)
	return true, ch
}

// OnChange 监听当前服务可执行的任务
//
// 不会处理多租户情况
//
// Parameters:
//   - businessKey string 当前任务标识
//   - jobs []string 所有待执行任务
//
// Returns：
//   - chan ChangeNotice 通知可执行任务变动。有可能返回可执行任务为空
//   - error
func OnChange(key string, values []string) (bool, <-chan ChangeNotice, error) {
	if s == nil {
		return false, nil, ErrNotInit
	}
	bo, ch := s.add(key, values)
	return bo, ch, nil
}
