// Package zset
// @Author: yinwei
// @File: zset_test
// @Version: 1.0.0
// @Date: 2025/1/24 13:41

package zset

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestZset(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	zset := NewZSet()

	zset.ZAdd("member1", 10.5)
	zset.ZAdd("member2", 5.5)
	zset.ZAdd("member3", 15.5)

	fmt.Println("ZRange(0, -1):", zset.ZRange(0, -1))
	fmt.Println("ZRangeByScore(5, 15):", zset.ZRangeByScore(5, 15))
	fmt.Println("ZRank(member3):", zset.ZRank("member3"))
	fmt.Println("ZCount(5, 15):", zset.ZCount(5, 15))

	zset.ZRemRangeByRank(0, 1)
	fmt.Println("After ZRemRangeByRank(0, 1):", zset.ZRange(0, -1))

	zset.ZRemRangeByScore(10, 20)
	fmt.Println("After ZRemRangeByScore(10, 20):", zset.ZRange(0, -1))
}

func Init() *ZSet {
	zset := NewZSet()

	zset.ZAdd("member1", 10.5)
	zset.ZAdd("member2", 5.5)
	zset.ZAdd("member3", 1)
	zset.ZAdd("member4", 22)
	zset.ZAdd("member5", 14)
	zset.ZAdd("member6", 99)
	zset.ZAdd("member7", 1231)
	zset.ZAdd("member8", 234332)
	zset.Print(false)
	return zset
}

func TestZSet_ZRemRangeByScore(t *testing.T) {

	t.Run("1~30", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(1, 30))
		zset.Print(false)
	})
	t.Run("21~30", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(21, 30))
		zset.Print(false)
	})
	t.Run("1~22222", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(1, 22222))
		zset.Print(false)
	})
	t.Run("1~234332", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(1, 234332))
		zset.Print(false)
	})
	t.Run("1~234332", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(1, 234332))
		zset.Print(false)
	})
	t.Run("7~7", func(t *testing.T) {
		zset := Init()
		t.Logf("remove: %v", zset.ZRemRangeByScore(7, 7))
		zset.Print(false)
	})
}

func TestZSet_ZRem(t *testing.T) {
	zsl := Init()
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "移除member8", args: args{
			key: "member8",
		}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := zsl.ZRem(tt.args.key); got != tt.want {
				t.Errorf("ZRem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func RandomInit(num int) *ZSet {
	zset := NewZSet()
	for i := range num {
		zset.ZAdd(fmt.Sprintf("member%v", i), float64(rand.Intn(100000)))
	}
	return zset
}

func Benchmark_ZRemRangeByScore(b *testing.B) {
	//初始化100000个元素

	b.Run("运行10次", func(b *testing.B) {
		for i := 0; i < 10; i++ {
			b.StopTimer()
			set := RandomInit(100000)
			b.StartTimer()
			set.ZRemRangeByScore(1, 10000)
		}
	})
	b.Run("运行100次", func(b *testing.B) {
		for i := 0; i < 100; i++ {
			b.StopTimer()
			set := RandomInit(100000)
			b.StartTimer()
			set.ZRemRangeByScore(1, 10000)
		}
	})

	b.Run("运行100次", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				set := RandomInit(100000)
				set.ZRemRangeByScore(1, 10000)
			}
		})

	})

}
