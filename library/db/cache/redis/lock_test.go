// Package redis
// @Author: yinwei
// @File: lock_test.go
// @Version: 1.0.0
// @Date: 2025/4/12 12:45

package redis

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

func TestRedis_LockUntilSuccWithRenewal(t *testing.T) {
	Init()

	type args struct {
		enum            prefixenum.PrefixEnum
		k               string
		retryInterval   time.Duration
		timeOut         time.Duration
		renewalInterval time.Duration
		do              func() error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "成功", args: args{
			enum:            "test",
			k:               "test_lock_succ",
			retryInterval:   100 * time.Millisecond,
			timeOut:         5 * time.Second,
			renewalInterval: 3 * time.Second,
			do: func() error {
				log.Info("开始执行 do")
				time.Sleep(10 * time.Millisecond)
				return nil
			},
		}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if err := GetRedis(context.Background()).LockUntilSuccWithRenewal(context.Background(), tt.args.enum, tt.args.k, tt.args.retryInterval, tt.args.timeOut, tt.args.renewalInterval, tt.args.do); (err != nil) != tt.wantErr {
				t.Errorf("LockUntilSuccWithRenewal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_Betch_LockUntilSuccWithRenewal(t *testing.T) {
	Init()
	wg := sync.WaitGroup{}
	for i := range 2 {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			if err := GetRedis(context.Background()).LockUntilSuccWithRenewal(context.Background(), "test", "test_lock_succ", 100*time.Millisecond, 1000*time.Second, 3*time.Second, func() error {
				log.Info("DO", zap.Any("i", i))
				return nil
			}); err != nil {
				t.Errorf("LockUntilSuccWithRenewal() error = %v", err)
			}

		}(i)
	}
	wg.Wait()
}

func TestRedis_UnLock(t *testing.T) {
	Init()
	type fields struct {
		Client redis.UniversalClient
	}
	type args struct {
		ctx  context.Context
		enum prefixenum.PrefixEnum
		k    string
		v    interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{name: "succ", fields: fields{Client: GetRedis(context.Background()).Client()}, args: args{
			ctx:  context.Background(),
			enum: "test",
			k:    "123",
			v:    12,
		}, want: false, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redis{
				UniversalClient: tt.fields.Client,
			}
			got, err := r.UnLock(tt.args.ctx, tt.args.enum, tt.args.k, tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnLock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnLock() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedis_LockWaitTimeOutWithRenewal(t *testing.T) {
	Init()
	type fields struct {
		Client redis.UniversalClient
	}
	type args struct {
		ctx             context.Context
		enum            prefixenum.PrefixEnum
		k               string
		waitTimeOut     time.Duration
		retryInterval   time.Duration
		timeOut         time.Duration
		renewalInterval time.Duration
		do              func() error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{name: "succ", fields: fields{Client: GetRedis(context.Background()).Client()}, args: args{
			ctx:             context.Background(),
			enum:            "test",
			k:               "123",
			waitTimeOut:     5 * time.Second,
			retryInterval:   10 * time.Millisecond,
			timeOut:         100 * time.Second,
			renewalInterval: 3 * time.Second,
			do: func() error {
				return nil
			},
		}, want: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redis{
				UniversalClient: tt.fields.Client,
			}
			got, err := r.LockWaitTimeOutWithRenewal(tt.args.ctx, tt.args.enum, tt.args.k, tt.args.waitTimeOut, tt.args.retryInterval, tt.args.timeOut, tt.args.renewalInterval, tt.args.do)
			if (err != nil) != tt.wantErr {
				t.Errorf("LockWaitTimeOutWithRenewal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("LockWaitTimeOutWithRenewal() got = %v, want %v", got, tt.want)
			}
		})
	}
}
