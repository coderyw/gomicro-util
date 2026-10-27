// Package redis
// @Author: yinwei
// @File: redis_test.go
// @Version: 1.0.0
// @Date: 2024/8/12 16:30

package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/coderyw/gomicro-util/internal/testenv"
	"sync"
	"testing"
	"time"

	"github.com/coderyw/gomicro-util/library/db/cache/redis/prefixenum"
	"github.com/coderyw/gomicro-util/library/metadata"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/util/strings"
	"github.com/longbridgeapp/assert"
	"github.com/redis/go-redis/v9"
)

func Init(t *testing.T) {
	rc := testenv.Require(t, "GOMICRO_TEST_REDIS_CLUSTER_ADDR", "GOMICRO_TEST_REDIS_CLUSTER_PASSWORD")
	_, _, err := InitRedis(config2.RedisConfig{
		Enable:  true,
		Default: "activity",
		Instance: map[string]config2.RedisInstanceConfig{
			"core": config2.RedisInstanceConfig{
				Enable:    true,
				Host:      []string{rc[0]},
				Password:  rc[1],
				Db:        0,
				Debug:     true,
				IsCluster: true,
			},
			"activity": config2.RedisInstanceConfig{
				Enable:    true,
				Host:      []string{rc[0]},
				Password:  rc[1],
				Db:        0,
				Debug:     true,
				IsCluster: true,
			},
		},
	})
	if err != nil {
		panic(err)
	}
}

func TestRedis_withLog(t *testing.T) {
	Init(t)
	cms, _ := GetRedis(context.Background()).Client().Command(context.Background()).Result()
	for _, v := range cms {
		fmt.Println(v.Name, " first key pos ", v.FirstKeyPos, " step ", v.StepCount, " lastPos ", v.LastKeyPos)
	}
	fmt.Println()
	ctx := WithRedisLog(context.Background())
	val, err := GetRedis(ctx).GetValue(ctx, "wefw", "test")
	if err != nil {
		t.Logf("got error: %v", err)
		return
	}

	t.Logf("got val: %v", val)
}
func TestRedis_KeyPrefixHook(t *testing.T) {
	Init(t)
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "test")
	t.Run("setnx", func(t *testing.T) {
		GetRedis(ctx).Setnx(ctx, "", "key", "value", time.Second*1024)
	})
	t.Run("get", func(t *testing.T) {
		val, _ := GetRedis(ctx).GetValue(ctx, "", "key")
		if !assert.Equal(t, "value", val) {
			t.Errorf("got val: %v, want: %v", val, "value")
		}
	})
	t.Run("get mul", func(t *testing.T) {
		_, _ = GetRedis(ctx).MGetValue(ctx, "", []string{"key", "key1"})
	})
}

func TestRedis_LockWithRenewal(t *testing.T) {
	Init(t)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		r := GetRedis(context.Background())
		bo, err := r.LockWithRenewal(context.Background(), "", "test_lock", 5*time.Second, 3*time.Second, func() error {
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
		bo, err := GetRedis(context.Background()).LockWithRenewal(context.Background(), "", "test_lock", 5*time.Second, 3*time.Second, func() error {
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
	Init(t)
	_, err := GetRedis(context.Background()).EvalSha(context.Background(), "234", []string{"dw"})
	fmt.Println(err)
}

func TestLockWithTimeOut(t *testing.T) {
	Init(t)
	key, err := GetRedis(context.Background()).LockWithTimeOut(context.Background(), "", "test_lock", 5*time.Second, time.Millisecond, 5*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println(key)
	key, err = GetRedis(context.Background()).LockWithTimeOut(context.Background(), "", "test_lock", 5*time.Second, time.Millisecond, 5*time.Second)
	fmt.Println("----------------")
	if err != nil {
		panic(err)
	}
	fmt.Println(key)
}

func TestRedis_IncrbyLimit(t *testing.T) {
	Init(t)
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
		{name: "增加1 设置下限0", args: args{
			prefix: "",
			key:    key,
			v:      1,
			limit:  0,
			up:     false,
			expire: []time.Duration{time.Second * 2},
		}, want: 1, wantErr: false},
		{name: "-3 设置下限0", args: args{
			prefix: "",
			key:    key,
			v:      -3,
			limit:  0,
			up:     false,
			expire: nil,
		}, want: 0, wantErr: true, wantErrIs: RedisIncExceed},
		{name: "+10 设置上限5", args: args{
			prefix: "",
			key:    key,
			v:      10,
			limit:  5,
			up:     true,
			expire: []time.Duration{time.Second * 2},
		}, want: 0, wantErr: true, wantErrIs: RedisIncExceed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := GetRedis(context.Background()).IncrbyLimit(context.Background(), tt.args.prefix, tt.args.key, tt.args.v, tt.args.limit, tt.args.up, tt.args.expire...)
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

func TestRedis_SetValueExist(t *testing.T) {
	Init(t)
	type args struct {
		prefix prefixenum.PrefixEnum
		key    string
		value  interface{}
		expire time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "更新一致", args: args{
			prefix: "",
			key:    "testKey",
			value:  123,
			expire: 12 * time.Second,
		}, want: true, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := GetRedis(context.Background()).SetValueExist(context.Background(), tt.args.prefix, tt.args.key, tt.args.value, tt.args.expire)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetValueExist() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SetValueExist() got = %v, want %v", got, tt.want)
			}
		})
	}
}
