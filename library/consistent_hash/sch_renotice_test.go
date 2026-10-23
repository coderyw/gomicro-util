// Package consistent_hash
// 周期性重下发与非阻塞通知的单元测试
package consistent_hash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/coderyw/gomicro-util/library/consistent_hash/hash_circle"
	"github.com/coderyw/gomicro-util/library/env"
)

func newTestHash(t *testing.T, nodes ...string) *ServerCstHash {
	t.Helper()
	s1 := &ServerCstHash{
		c:          hash_circle.NewSortHash(),
		serversMap: make(map[string]struct{}),
	}
	s1.setService(nodes, nil)
	return s1
}

// renoticeAll 应在成员无变化时也重新下发分区分配。
func TestRenoticeAll_ResendsAssignment(t *testing.T) {
	s1 := newTestHash(t, env.GetId(), "node-b")
	_, ch := s1.add("biz", []string{"0", "1", "2", "3"})

	// add 时的初始通知
	first := <-ch
	require.Equal(t, "biz", first.BusinessKey)

	// 成员没有任何变化，renoticeAll 也必须重发
	s1.renoticeAll()
	select {
	case second := <-ch:
		assert.Equal(t, first.Jobs, second.Jobs, "无变化重下发的分配应与初始一致")
	case <-time.After(time.Second):
		t.Fatal("renoticeAll did not resend assignment")
	}
}

// 通知通道满且无人消费时，noticeJob 不允许阻塞（否则会冻结 change 循环）。
func TestNoticeJob_NonBlockingWhenChannelFull(t *testing.T) {
	s1 := newTestHash(t, env.GetId(), "node-b")
	_, _ = s1.add("biz", []string{"0", "1", "2", "3"}) // 初始通知占满 cap=1 的通道，且无人消费

	done := make(chan struct{})
	go func() {
		s1.renoticeAll()
		s1.renoticeAll()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("noticeJob blocked on full channel")
	}
}

// 快照日志在有/无 businessKey 时都不应 panic。
func TestLogSnapshot_NoPanic(t *testing.T) {
	s1 := newTestHash(t, env.GetId(), "node-b")
	s1.logSnapshot()
	_, ch := s1.add("biz", []string{"0", "1"})
	<-ch
	s1.logSnapshot()
}
