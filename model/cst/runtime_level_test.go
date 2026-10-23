// Package cst
// @Author: yinwei
// @File: runtime_level_test.go
// @Version: 1.0.0
// @Date: 2025/6/30 13:21

package cst

import (
	"fmt"
	"testing"
	"time"
)

func TestGetHttpHandleLevel(t *testing.T) {
	type args struct {
		dur time.Duration
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "5000", args: args{dur: time.Millisecond * 5000}, want: fmt.Sprintf("%v:%v:%v:%v:%v:%v", RUNTIME_LEVEL_INSTANT, RUNTIME_LEVEL_VERY_NICE, RUNTIME_LEVEL_NICE, RUNTIME_LEVEL_GOOD, RUNTIME_LEVEL_SLOW, RUNTIME_LEVEL_BAD)},
		{name: "290", args: args{dur: time.Millisecond * 290}, want: fmt.Sprintf("%v:%v", RUNTIME_LEVEL_INSTANT, RUNTIME_LEVEL_VERY_NICE)},
		{name: "10", args: args{dur: time.Millisecond * 10}, want: fmt.Sprintf("%v", RUNTIME_LEVEL_INSTANT)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetHttpHandleLevel(tt.args.dur); got != tt.want {
				t.Errorf("GetHttpHandleLevel() = %v, want %v", got, tt.want)
			} else {
				t.Logf("%v", got)
			}
		})
	}
}

func TestGetOtherHandleLevel(t *testing.T) {
	type args struct {
		dur time.Duration
		msl []int64
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "120", args: args{
			dur: 120 * time.Millisecond,
			msl: []int64{1, 100, 200, 300, 500},
		}, want: fmt.Sprintf("%v:%v:%v", RUNTIME_LEVEL_INSTANT, RUNTIME_LEVEL_VERY_NICE, RUNTIME_LEVEL_NICE)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOtherHandleLevel(tt.args.dur, tt.args.msl); got.String != tt.want {
				t.Errorf("GetOtherHandleLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
