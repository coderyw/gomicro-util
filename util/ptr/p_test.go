// Package ptr
// @Author: yinwei
// @File: p_test.go
// @Version: 1.0.0
// @Date: 2023/11/7 22:58

package ptr

import "testing"

func TestTASK_TYPE_IsEqual(t *testing.T) {
	type args struct {
		val any
	}
	tests := []struct {
		name string
		t    TASK_TYPE
		args args
		want bool
	}{
		{name: "not equal to int", t: TASK_TYPE(1), args: args{val: 2}, want: false},
		{name: "equal to int", t: TASK_TYPE(1), args: args{val: 1}, want: true},
		{name: "equal to int32", t: TASK_TYPE(1), args: args{val: int32(1)}, want: true},
		{name: "equal to int64", t: TASK_TYPE(1), args: args{val: int64(1)}, want: true},
		{name: "not equal to int64", t: TASK_TYPE(1), args: args{val: int64(2)}, want: false},
		{name: "not equal to string", t: TASK_TYPE(1), args: args{val: "1"}, want: false},
		{name: "not equal to uin8", t: TASK_TYPE(1), args: args{val: uint8(2)}, want: false},
		{name: "equal to uin8", t: TASK_TYPE(1), args: args{val: uint8(1)}, want: true},
		{name: "equal to TASK_TYPE", t: TASK_TYPE(1), args: args{val: TASK_TYPE(1)}, want: true},
		{name: "Not equal to TASK_TYPE", t: TASK_TYPE(1), args: args{val: TASK_TYPE(2)}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.IsEqual(tt.args.val); got != tt.want {
				t.Errorf("IsEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}
