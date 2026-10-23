// Package redis
// @Author: yinwei
// @File: redis_test.go
// @Version: 1.0.0
// @Date: 2024/8/12 16:30

package test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"

	"github.com/redis/go-redis/v9"
	redis2 "github.com/coderyw/gomicro-util/library/db/cache/redis"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/util/strings"
)

func Init() {
	_, _, err := redis2.InitRedis(config2.RedisConfig{
		Enable:  true,
		Default: "core",
		Instance: map[string]config2.RedisInstanceConfig{
			"core": {
				Enable:   true,
				Host:     []string{"127.0.0.1:6490"},
				Password: "123456",
				Db:       0,
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func TestRedis_LockWithRenewal(t *testing.T) {
	Init()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		bo, err := redis2.GetRedis(context.Background()).LockWithRenewal(context.Background(), "", "test_lock", 5*time.Second, 3*time.Second, func() error {
			fmt.Println("哈哈 我这里是获取到锁0")
			time.Sleep(10 * time.Second)
			return nil
		})
		if err != nil {
			fmt.Println("加锁失败: ", err)
			return
		}
		if !bo {
			fmt.Println("加锁失败：", bo)
			return
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		bo, err := redis2.GetRedis(context.Background()).LockWithRenewal(context.Background(), "", "test_lock", 5*time.Second, 3*time.Second, func() error {
			fmt.Println("哈哈 我这里是获取到锁1")
			time.Sleep(10 * time.Second)
			return nil
		})
		if err != nil {
			fmt.Println("加锁失败1: ", err)
			return
		}
		if !bo {
			fmt.Println("加锁失败1：", bo)
			return
		}
	}()
	wg.Wait()
}

func TestEvalSha(t *testing.T) {
	Init()
	_, err := redis2.GetRedis(context.Background()).EvalSha(context.Background(), "234", []string{"dw"})
	fmt.Println(err)
}

func TestLockWithTimeOut(t *testing.T) {
	Init()
	key, err := redis2.GetRedis(context.Background()).LockWithTimeOut(context.Background(), "", "test_lock", 5*time.Second, time.Millisecond, 5*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println(key)
	key, err = redis2.GetRedis(context.Background()).LockWithTimeOut(context.Background(), "", "test_lock", 5*time.Second, time.Millisecond, 5*time.Second)
	fmt.Println("----------------")
	if err != nil {
		panic(err)
	}
	fmt.Println(key)
}

func TestRedis_IncrbyLimit(t *testing.T) {
	Init()
	key := strings.RandStr(5)
	type fields struct {
		Client redis.UniversalClient
	}
	type args struct {
		prefix prefixenum.PrefixEnum
		key    string
		v      int64
		limit  int64
		up     bool
		expire []time.Duration
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		want      int64
		wantErr   bool
		wantErrIs error
	}{
		{name: "增加1 设置下限0", fields: fields{}, args: args{
			prefix: "",
			key:    key,
			v:      1,
			limit:  0,
			up:     false,
			expire: []time.Duration{time.Second * 2},
		}, want: 1, wantErr: false},
		{name: "-3 设置下限0", fields: fields{}, args: args{
			prefix: "",
			key:    key,
			v:      -3,
			limit:  0,
			up:     false,
			expire: nil,
		}, want: 0, wantErr: true, wantErrIs: redis2.RedisIncExceed},
		{name: "+10 设置上限5", fields: fields{}, args: args{
			prefix: "",
			key:    key,
			v:      10,
			limit:  5,
			up:     true,
			expire: []time.Duration{time.Second * 2},
		}, want: 0, wantErr: true, wantErrIs: redis2.RedisIncExceed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &redis2.Redis{
				UniversalClient: tt.fields.Client,
			}
			got, err := r.IncrbyLimit(context.Background(), tt.args.prefix, tt.args.key, tt.args.v, tt.args.limit, tt.args.up, tt.args.expire...)
			if (err != nil) != tt.wantErr {
				t.Errorf("IncrbyLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if (err != nil) && tt.wantErr && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("IncrbyLimit() error = %v, wantErr is %v", err, tt.wantErrIs)
				return
			}
			if got != tt.want {
				t.Errorf("IncrbyLimit() got = %v, want %v", got, tt.want)
			}
		})
	}
}
