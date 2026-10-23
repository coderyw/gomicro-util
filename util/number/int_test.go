// Package integer
// @Author: yinwei
// @File: int_test.go
// @Version: 1.0.0
// @Date: 2024/5/24 14:52

package number

import (
	"github.com/coderyw/gomicro-util/model/constrict"
	"testing"
)

func TestNumToCN(t *testing.T) {
	type args struct {
		num int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "100", args: args{100}, want: "一百"},
		{name: "1", args: args{1}, want: "一"},
		{name: "109", args: args{109}, want: "一百零九"},
		{name: "10002321", args: args{10002321}, want: "一千万零两千三百二十一"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NumToCN(tt.args.num); got != tt.want {
				t.Errorf("NumToCN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotZero(t *testing.T) {
	type args[T constrict.Number] struct {
		a T
		b T
		c []T
	}
	type testCase[T constrict.Number] struct {
		name string
		args args[T]
		want T
	}
	tests := []testCase[int]{
		testCase[int]{
			name: "a is true,c has value",
			args: args[int]{
				a: 1,
				b: 2,
				c: []int{3},
			},
			want: 2,
		},
		testCase[int]{
			name: "a is false, c has value",
			args: args[int]{
				a: 0,
				b: 2,
				c: []int{3},
			},
			want: 3,
		},
		testCase[int]{
			name: "a is true, c has not a value",
			args: args[int]{
				a: 1,
				b: 2,
				c: nil,
			},
			want: 1,
		},
		testCase[int]{
			name: "a is false, c has not a value",
			args: args[int]{
				a: 0,
				b: 2,
				c: nil,
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//if got := NotZero(tt.args.a, tt.args.b, tt.args.c...); got != tt.want {
			//	t.Errorf("NotZero() = %v, want %v", got, tt.want)
			//}
		})
	}
}
