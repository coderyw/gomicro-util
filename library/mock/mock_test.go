package mock

import (
	"context"
	"fmt"
	"github.com/longbridgeapp/assert"
	"testing"
)

func TestGetMockResult(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "get nil t",
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMockCtxTestingValue(tt.args.ctx)
			assert.Nil(t, got)
		})
	}
}

func TestSetMockValue(t *testing.T) {
	type args struct {
		ctx context.Context
		t   *testing.T
		c   []interface{}
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "模拟实际使用场景",
			args: args{
				ctx: context.Background(),
				t:   &testing.T{},
				c:   []interface{}{"1", "2"},
			},
		},
	}
	type testStruct struct{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := MockSetUp(tt.args.ctx, tt.args.t)
			// set
			client := CheckAndGetMockClient[*testStruct](ctx, func(t *testing.T) *testStruct {
				return new(testStruct)
			})
			assert.NotNil(t, client)
			// get
			client2 := CheckAndGetMockClient[*testStruct](ctx, func(t *testing.T) *testStruct {
				return new(testStruct)
			})
			assert.Equal(t, client, client2)
		})
	}
}

func TypeA[T any](a T) T {
	var ret T
	return ret
}

func TestTy(t *testing.T) {
	type AB struct {
		A string
	}
	fmt.Println(TypeA(&AB{}) == nil)
}
