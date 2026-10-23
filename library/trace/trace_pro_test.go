// Package trace
// @Author: yinwei
// @File: trace_pro_test.go
// @Version: 1.0.0
// @Date: 2025/5/9 16:06

package trace

import (
	"context"
	"reflect"
	"testing"
)

func TestNewOrAddTraceId(t *testing.T) {
	type args struct {
		ctx             context.Context
		newOrAddTraceId string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "parent is nil", args: args{
			ctx:             nil,
			newOrAddTraceId: "123",
		}, want: "123"},
		{name: "parent is parentTraceId", args: args{
			ctx:             SetTraceId(context.Background(), "parentTraceId"),
			newOrAddTraceId: "123",
		}, want: "parentTraceId:123"},
		{name: "parent gets no id", args: args{
			ctx:             context.Background(),
			newOrAddTraceId: "123",
		}, want: "123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOrAddTraceId(tt.args.ctx, tt.args.newOrAddTraceId)
			gotId, _ := GetTraceId(got)
			if !reflect.DeepEqual(gotId, tt.want) {
				t.Errorf("NewOrAddTraceId() = %v, want %v", gotId, tt.want)
			}
		})
	}
}
