// Package system
// @Author: yinwei
// @File: balance_test.go
// @Version: 1.0.0
// @Date: 2024/7/26 14:24

package system

import (
	"github.com/coderyw/gomicro-util/model/constrict"
	"testing"
)

func TestFormatBalance(t *testing.T) {
	type args[T constrict.Number] struct {
		balance T
		cent    []bool
	}
	type testCase[T constrict.Number] struct {
		name string
		args args[T]
		want float64
	}
	tests := []testCase[int64]{
		{
			name: "元-》分",
			args: args[int64]{
				balance: int64(1045),
				cent:    []bool{false},
			},
			want: 10.45,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBalance(tt.args.balance, tt.args.cent...); got != tt.want {
				t.Errorf("FormatBalance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatBalanceStr(t *testing.T) {
	type args[T constrict.Number] struct {
		balance T
		cent    []bool
	}
	type testCase[T constrict.Number] struct {
		name string
		args args[T]
		want string
	}
	tests := []testCase[int64]{
		{
			name: "元-》分",
			args: args[int64]{
				balance: int64(1045),
				cent:    []bool{false},
			},
			want: "10.45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBalanceStr(tt.args.balance, tt.args.cent...); got != tt.want {
				t.Errorf("FormatBalanceStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestFormatBalanceStr1(t *testing.T) {
	type args[T constrict.Number] struct {
		balance T
		cent    []bool
	}
	type testCase[T constrict.Number] struct {
		name string
		args args[T]
		want string
	}
	tests := []testCase[float64]{
		{
			name: "元-》分",
			args: args[float64]{
				balance: float64(1045.1),
				cent:    []bool{false},
			},
			want: "10.45",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatBalanceStr(tt.args.balance, tt.args.cent...); got != tt.want {
				t.Errorf("FormatBalanceStr() = %v, want %v", got, tt.want)
			}
		})
	}
}
