package nacos

import (
	"context"
	"testing"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"go-micro.dev/v4/registry"
)

func newTestWatcher() *watcher {
	return &watcher{
		wo:            registry.WatchOptions{Service: "svc"},
		exit:          make(chan bool),
		next:          make(chan *registry.Result, 100),
		cacheServices: make(map[string][]model.Instance),
	}
}

func inst(id, ip string) model.Instance {
	return model.Instance{InstanceId: id, Ip: ip, Port: 9804, ServiceName: "DEFAULT_GROUP@@svc", Healthy: true, Enable: true}
}

// drain 取出当前已下发的全部事件，格式为 "action ip:port"
func drain(nw *watcher) []string {
	var out []string
	for {
		select {
		case r := <-nw.next:
			out = append(out, r.Action+" "+r.Service.Nodes[0].Address)
		default:
			return out
		}
	}
}

func assertEvents(t *testing.T, got []string, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}

func TestCallBackHandle(t *testing.T) {
	t.Run("首次推送下发全部实例", func(t *testing.T) {
		nw := newTestWatcher()
		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1"), inst("b", "10.0.0.2")}, nil)
		assertEvents(t, drain(nw), "create 10.0.0.1:9804", "create 10.0.0.2:9804")
	})

	t.Run("新增、变更、删除一次推送里同时出现", func(t *testing.T) {
		nw := newTestWatcher()
		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1"), inst("b", "10.0.0.2")}, nil)
		drain(nw)

		changed := inst("a", "10.0.0.1")
		changed.Weight = 2
		nw.callBackHandle([]model.Instance{changed, inst("c", "10.0.0.3")}, nil)
		assertEvents(t, drain(nw), "update 10.0.0.1:9804", "create 10.0.0.3:9804", "delete 10.0.0.2:9804")
	})

	t.Run("未变化不下发", func(t *testing.T) {
		nw := newTestWatcher()
		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1")}, nil)
		drain(nw)
		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1")}, nil)
		assertEvents(t, drain(nw))
	})

	t.Run("空列表不 panic 并删除全部实例", func(t *testing.T) {
		nw := newTestWatcher()
		nw.callBackHandle(nil, nil)
		assertEvents(t, drain(nw))

		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1")}, nil)
		drain(nw)
		nw.callBackHandle([]model.Instance{}, nil)
		assertEvents(t, drain(nw), "delete 10.0.0.1:9804")

		nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1")}, nil)
		assertEvents(t, drain(nw), "create 10.0.0.1:9804")
	})

	t.Run("watcher 停止后不阻塞推送协程", func(t *testing.T) {
		nw := newTestWatcher()
		nw.next = make(chan *registry.Result) // 无缓冲且无人读取
		close(nw.exit)
		done := make(chan struct{})
		go func() {
			nw.callBackHandle([]model.Instance{inst("a", "10.0.0.1")}, nil)
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("callBackHandle blocked after watcher stopped")
		}
	})
}

func TestClientConfigFrom(t *testing.T) {
	t.Run("调用方只传账密时也不加载磁盘缓存", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), configKey{}, constant.ClientConfig{
			Username: "u", Password: "p", NamespaceId: "ns",
		})
		cc := clientConfigFrom(ctx)
		if !cc.NotLoadCacheAtStart {
			t.Fatal("NotLoadCacheAtStart = false, want true")
		}
		if cc.Username != "u" || cc.Password != "p" || cc.NamespaceId != "ns" {
			t.Fatalf("caller fields lost: %+v", cc)
		}
	})

	t.Run("没有传配置时用默认配置", func(t *testing.T) {
		cc := clientConfigFrom(context.Background())
		if !cc.NotLoadCacheAtStart || !cc.UpdateCacheWhenEmpty {
			t.Fatalf("defaults = %+v", cc)
		}
		if cc := clientConfigFrom(nil); !cc.NotLoadCacheAtStart { //nolint:staticcheck // 覆盖 ctx 为空
			t.Fatal("nil ctx: NotLoadCacheAtStart = false")
		}
	})
}
