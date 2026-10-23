package local_cache

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/metadata"
)

func TestLocalCache(t *testing.T) {
	lc := NewLocalCache()
	type arg struct {
		ctx  context.Context
		args []interface{}
	}
	tests := []struct {
		name string
		args arg
		do   func(arg arg) interface{}
		want interface{}
	}{
		{name: "LocalSetOk", do: func(arg arg) interface{} {
			lc.Set(arg.ctx, arg.args[0].(string), arg.args[1], arg.args[2].(time.Duration))
			return nil
		}, args: arg{
			ctx:  metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok"),
			args: []interface{}{"key", "value", time.Duration(10) * time.Second},
		}, want: nil},
		{name: "LocalSetOk1", do: func(arg arg) interface{} {
			lc.Set(arg.ctx, arg.args[0].(string), arg.args[1], arg.args[2].(time.Duration))
			return nil
		}, args: arg{
			ctx:  metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok1"),
			args: []interface{}{"key", "value1", time.Duration(10) * time.Second},
		}, want: nil},
		{name: "LocalGet", do: func(arg arg) interface{} {
			val, ok := lc.Get(arg.ctx, arg.args[0].(string))
			return []interface{}{val, ok}
		}, args: arg{
			ctx:  metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok"),
			args: []interface{}{"key"},
		}, want: []interface{}{"value", true}},
		{name: "LocalGet", do: func(arg arg) interface{} {
			val, ok := lc.Get(arg.ctx, arg.args[0].(string))
			return []interface{}{val, ok}
		}, args: arg{
			ctx:  metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok1"),
			args: []interface{}{"key"},
		}, want: []interface{}{"value1", true}},
		{name: "LocalDelete", do: func(arg arg) interface{} {
			lc.Delete(arg.ctx, arg.args[0].(string))
			val, ok := lc.Get(arg.ctx, arg.args[0].(string))
			return []interface{}{val, ok}
		}, args: arg{
			ctx:  metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "ok"),
			args: []interface{}{"key"},
		}, want: []interface{}{nil, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.do(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewLocalCache() = %v, want %v", got, tt.want)
			}
		})
	}
}
