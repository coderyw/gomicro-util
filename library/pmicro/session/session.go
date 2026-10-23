// Package session
// @Author: yinwei
// @File: session
// @Version: 1.0.0
// @Date: 2023/11/17 10:47

package session

import (
	"context"
	"fmt"
)

type UserIdKey struct{}
type ClientTypeKey struct{}

// 必须获取userId 否则panic
func MustUserId(ctx context.Context) uint64 {
	val := ctx.Value(UserIdKey{})
	if val == nil {
		panic(fmt.Errorf("userId缺失"))
	}
	return val.(uint64)
}

func UserId(ctx context.Context) (uint64, bool) {
	val := ctx.Value(UserIdKey{})
	if val == nil {
		return 0, false
	}
	return val.(uint64), true
}

func MustClientType(ctx context.Context) int {
	val := ctx.Value(ClientTypeKey{})
	if val == nil {
		panic(fmt.Errorf("userId缺失"))
	}
	return val.(int)
}

func ClientType(ctx context.Context) (int, bool) {
	val := ctx.Value(ClientTypeKey{})
	if val == nil {
		return 0, false
	}
	return val.(int), true
}
