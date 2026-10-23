package kafka

import (
	"slices"
	"sync"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	tenant2 "github.com/coderyw/gomicro-util/model/tenant"
)

// 按租户列表生成精确 topic：前缀命名、去重、升序
func TestMakeTopicsForTenants(t *testing.T) {
	env.SetFallbackTenantName("")
	t.Cleanup(func() { env.SetFallbackTenantName("") })

	cases := []struct {
		name    string
		tenants []tenant2.Tenant
		want    []string
	}{
		{name: "大小写统一并去重", tenants: []tenant2.Tenant{{TenantName: "OK"}, {TenantName: "ok"}}, want: []string{"tenantok_pay-topic"}},
		{name: "空租户列表兜底原始topic", tenants: nil, want: []string{"pay-topic"}},
		{name: "多租户前缀命名且升序", tenants: []tenant2.Tenant{
			{TenantName: "ok"}, {TenantName: "ab"},
		}, want: []string{"tenantab_pay-topic", "tenantok_pay-topic"}},
		{name: "兜底租户空名即原始topic", tenants: []tenant2.Tenant{
			{TenantName: ""},
		}, want: []string{"pay-topic"}},
		{name: "重复租户去重", tenants: []tenant2.Tenant{
			{TenantName: "ok"}, {TenantName: "ok"},
		}, want: []string{"tenantok_pay-topic"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := makeTopicsForTenants("pay-topic", c.tenants)
			if !slices.Equal(got, c.want) {
				t.Errorf("makeTopicsForTenants() = %v, want %v", got, c.want)
			}
		})
	}
}

// 兜底租户模式：FmtByType 全部返回原始 topic，去重后只剩一个
func TestMakeTopicsForTenants_FallbackMode(t *testing.T) {
	env.SetFallbackTenantName("ns")
	t.Cleanup(func() { env.SetFallbackTenantName("") })

	got := makeTopicsForTenants("pay-topic", []tenant2.Tenant{
		{TenantName: "ok"}, {TenantName: "nwr"},
	})
	if !slices.Equal(got, []string{"pay-topic"}) {
		t.Errorf("makeTopicsForTenants() = %v, want [pay-topic]", got)
	}
}

// 并发投递安全：多个 sender 同时 pushResub 不丢投递（最终 channel 内必有一个目标）
func TestPushResub_ConcurrentSenders(t *testing.T) {
	cg := &ConsumerGroup{resubChan: make(chan []string, 1)}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				cg.pushResub([]string{"t"})
			}
		}(i)
	}
	wg.Wait()
	select {
	case <-cg.resubChan:
	default:
		t.Error("并发投递结束后 channel 内应保有最新目标")
	}
}

// resubChan 容量 1：重复投递只保留最新目标
func TestPushResub_KeepLatest(t *testing.T) {
	cg := &ConsumerGroup{resubChan: make(chan []string, 1)}
	cg.pushResub([]string{"a"})
	cg.pushResub([]string{"b"})
	cg.pushResub([]string{"c"})

	got := <-cg.resubChan
	if !slices.Equal(got, []string{"c"}) {
		t.Errorf("resubChan 取出 %v, want [c]", got)
	}
	select {
	case extra := <-cg.resubChan:
		t.Errorf("resubChan 不应残留旧目标, got %v", extra)
	default:
	}
}
