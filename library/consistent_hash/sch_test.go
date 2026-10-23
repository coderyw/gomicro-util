// Package consistent_hash
// @Author: yinwei
// @File: sch_test
// @Version: 1.0.0
// @Date: 2025/6/12 16:55

package consistent_hash

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stathat/consistent"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/model/consistent_hash"
	"go.uber.org/zap"
)

type listener struct {
	ch chan consistent_hash.Notice
}

func (l *listener) OnNow() consistent_hash.Notice {
	//TODO implement me
	panic("implement me")
}

func (l *listener) OnChanged() chan consistent_hash.Notice {
	return l.ch
}

func (l *listener) Init(ctx context.Context) {
	l.ch = make(chan consistent_hash.Notice)
	go func() {
		t := time.NewTicker(3 * time.Second)
		notice := consistent_hash.Notice{
			ServiceName: "server-test",
			Err:         nil,
			Services: []consistent_hash.Instance{
				{
					Host: "127.0.0.1:8080",
					Id:   env.GetId(),
				},
				{
					Host: "127.0.0.3:8080",
					Id:   uuid.NewString(),
				},
				{
					Host: "127.0.0.2:8080",
					Id:   uuid.NewString(),
				},
			},
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				l.ch <- notice
			}

		}

	}()
}

func TestInit(t *testing.T) {
	l := &listener{}
	ctx, cancel := context.WithCancel(context.Background())
	l.Init(ctx)
	//开始
	Init(l)

	_, cn, _ := OnChange("key1", []string{"0", "1", "2", "3"})
	_, cn2, _ := OnChange("key2", []string{"12", "33", "45", "66"})

	go func() {
		for {
			select {
			case notice := <-cn:
				log.Info("key1", zap.Any("notice", notice))
			case notice := <-cn2:
				log.Info("key2", zap.Any("notice", notice))
			}
		}
	}()
	<-single.OsKillSingle()
	cancel()
}

func TestMaxHash(t *testing.T) {

	nc := consistent.New()
	nc.NumberOfReplicas = 20

	//	{"appName": "redis-sync", "version": "latest", "services": [{"Host":"172.21.0.3","Id":"2242fc03-8751-469a-95e8-1ba71784ad07"},{"Host":"172.21.0.4","Id":"d60661d6-ddc6-4c71-a8f1-d18293f9cbb7"},{"Host":"172.21.0.2","Id":"5cb853c7-a387-4bd2-8ff9-0a7b174b9174"}], "current": "2242fc03-8751-469a-95e8-1ba71784ad07"}
	svc := []string{"720c5ce7-fdfa-4b50-ba75-9d883b8b8b5c", "bef37faa-5821-4f96-92eb-ac4fb80b45aa", "48c1a857-6a94-4847-a1a0-770e8b323004", "795571a0-72dc-45f3-bd20-dffa31f7d93a"}

	// 遍历所有 服务
	for _, v := range svc {

		nc.Add(v)

	}

	m := make(map[string]int)

	for i := range 10 {
		val, _ := nc.Get(fmt.Sprint(i))
		//log.Info("get", zap.String("val", val))
		for _, v := range svc {
			if strings.HasPrefix(val, v) {
				if val, ok := m[v]; ok {
					m[v] = val + 1
				} else {
					m[v] = 1
				}
			}
		}
	}
	log.Info("m", zap.Any("m", m))
}
