// Package metadata
// @Author: yinwei
// @File: ctx_cst_test.go
// @Version: 1.0.0
// @Date: 2025/6/26 09:56

package metadata

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/coderyw/gomicro-util/util/mapstructure"
	"go-micro.dev/v4/metadata"
)

func TestMapStruct(t *testing.T) {
	m := map[string]string{
		"Userid":   "149371427764080646",
		"Terminal": "0",
	}
	md := &Metadata{}
	err := mapstructure.Decode(m, md)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%+v\n", md)
}

func TestGetInt64(t *testing.T) {
	type args struct {
		ctx context.Context
		key string
	}
	type testCase[T any] struct {
		name string
		args args
		want T
	}
	tests := []testCase[int64]{
		{name: "userId", args: args{
			ctx: context.Background(),
			key: CONTEXT_USER_ID,
		}, want: 0},
		{name: "terminal", args: args{
			ctx: SetCtxData(context.Background(), string(CONTEXT_TERMINAL), int64(128)),
			key: CONTEXT_TERMINAL,
		}, want: 128},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get[int64](tt.args.ctx, string(tt.args.key))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCustom(t *testing.T) {
	type args struct {
		ctx context.Context
		key string
	}
	type testCase[T any] struct {
		name string
		args args
		want T
	}
	tests := []testCase[string]{
		{name: "Custom_userId", args: args{
			ctx: context.Background(),
			key: string(CONTEXT_USER_ID),
		}, want: ""},
		{name: "Custom_terminal", args: args{
			ctx: SetCtxData(context.Background(), fmt.Sprintf("%s%s", CONTEXT_CUSTOM_PREFIX, CONTEXT_TERMINAL), "128"),
			key: fmt.Sprintf("%s%s", CONTEXT_CUSTOM_PREFIX, CONTEXT_TERMINAL),
		}, want: "128"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Get[string](tt.args.ctx, tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadata_FromMicroMetadata(t *testing.T) {

	type args struct {
		md2 metadata.Metadata
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "decode", args: args{md2: map[string]string{"Userid": "1234", "Reqip": "192.168.1.1", "Terminalname": "terminalName", "Terminal": "128"}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := &Metadata{}
			if err := md.FromMicroMetadata(tt.args.md2); (err != nil) != tt.wantErr {
				t.Errorf("FromMicroMetadata() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Logf("md = %+v", md)
			}
		})
	}
}
