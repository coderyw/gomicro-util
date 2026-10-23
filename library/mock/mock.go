// mock.go
// mock测试
//
// copyright (c) 2022-2023

package mock

import (
	"context"
	"github.com/coderyw/gomicro-util/library/env"
	"testing"
)

// 上下文中的 key，用于确保其他代码无法引用
type mockCtxKey struct{}

// mock 数据
type mockCtxValue struct {
	t       *testing.T    // 单测函数的上下文,用于mock开启判断和初始化client
	clients []interface{} // mock的rpc服务的client集合,兼容同时mock多个rpc的场景
}

// MockSetUp 开启 mock，仅在开发和测试环境可以使用
func MockSetUp(ctx context.Context, t *testing.T) context.Context {
	// 只有测试环境和开发环境可以使用mock
	if env.IsRelease() || env.IsPre() {
		return ctx
	}
	return context.WithValue(ctx, mockCtxKey{}, &mockCtxValue{t: t})
}

// CheckAndGetMockClient 用于判断是否开启 mock，以及在开启 mock 情况下获取类型为 T 的 mock 客户端
//
// 获取 mock 客户端，仅在开发和测试环境可以使用
//
// 如果不存在，则会保存
func CheckAndGetMockClient[T any](ctx context.Context, f func(t *testing.T) T) T {
	var ret T
	// 只有测试环境和开发环境可以使用mock
	if env.IsRelease() || env.IsPre() {
		return ret
	}
	// 获取上下文中的 mock 数据
	mockInfo := getMockCtxTestingValue(ctx)
	if mockInfo == nil || mockInfo.t == nil {
		return ret
	}
	// 遍历所有 mock 客户端
	for _, v := range mockInfo.clients {
		// ctx中已存储对应类型的client,直接返回
		if client, ok := v.(T); ok {
			return client
		}
	}
	// ctx中不存在入参类型client,将client set进ctx
	c := f(mockInfo.t)
	mockInfo.clients = append(mockInfo.clients, c)
	return c
}

// 获取 mock 测试函数
func getMockCtxTestingValue(ctx context.Context) *mockCtxValue {
	m := ctx.Value(mockCtxKey{})
	if m != nil {
		val, ok := m.(*mockCtxValue)
		if ok {
			return val
		}
	}
	return nil
}
