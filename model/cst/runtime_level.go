// Package cst
// @Author: yinwei
// @File: runtime_level
// @Version: 1.0.0
// @Date: 2025/5/9 23:34

package cst

import (
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

// 运行时长的日志告警等级
// 为了方便日志查询，比如需要查询耗时>500ms的日志，这里需要对运行时长进行分类：
//
//		比如：请求超过3000ms,日志应为：l_instant:l_very_nice:l_nice:l_good:l_slow 这里需要包含从l_instant~l_slow的所有标识，方便查询
//	 比如：请求超过500ms,日志应为：l_instant:l_very_nice:l_nice 这里需要包含从 l_instant~l_nice 的所有标识，方便查询
const (
	RUNTIME_LEVEL_INSTANT   = "l_instant"
	RUNTIME_LEVEL_VERY_NICE = "l_very_nice"
	// 运行时长很短，非常 nice
	RUNTIME_LEVEL_NICE = "l_nice"
	// 运行时长较好，good
	RUNTIME_LEVEL_GOOD = "l_good"
	// 运行时长还可接受 slow
	RUNTIME_LEVEL_SLOW = "l_slow"
	// 运行很慢 bad
	RUNTIME_LEVEL_BAD = "l_bad"
)

var defaultMsl []int64 = []int64{5, 10, 20, 50, 100}

// 对象池
var pool = sync.Pool{
	New: func() any {
		return &strings.Builder{}
	},
}

// GetHttpHandleLevel 获取http响应时长等级
//
// Parameters:
//   - dur: time.Duration 响应时间，请求与处理都有响应时间
//
// Returns:
//   - string
func GetHttpHandleLevel(dur time.Duration) string {
	mil := dur.Milliseconds() // 获取毫秒
	// 使用对象池，避免频繁创建对象
	bu := pool.Get().(*strings.Builder)
	defer func() {
		bu.Reset()
		pool.Put(bu)
	}()
	write(bu, RUNTIME_LEVEL_INSTANT)
	if mil >= 200 {
		write(bu, RUNTIME_LEVEL_VERY_NICE)
	}
	if mil >= 500 {
		write(bu, RUNTIME_LEVEL_NICE)
	}
	if mil >= 1000 {
		write(bu, RUNTIME_LEVEL_GOOD)
	}
	if mil >= 3000 {
		write(bu, RUNTIME_LEVEL_SLOW)
	}
	if mil >= 5000 {
		write(bu, RUNTIME_LEVEL_BAD)
	}

	return bu.String()
}

func write(bu *strings.Builder, i string) {
	if bu.Len() > 0 {
		bu.WriteString(":")
	}
	bu.WriteString(i)
}

// GetKafkaConsumerHandleLevel 获取 kafka 消费耗时等级
//
// Parameters:
//   - dur: time.Duration 消费耗时
//
// Returns:
//   - string 日志等级
func GetKafkaConsumerHandleLevel(dur time.Duration) string {
	mil := dur.Milliseconds() // 获取毫秒
	// 使用对象池，避免频繁创建对象
	bu := pool.Get().(*strings.Builder)
	defer func() {
		bu.Reset()
		pool.Put(bu)
	}()
	write(bu, RUNTIME_LEVEL_INSTANT)
	if mil >= 10 {
		write(bu, RUNTIME_LEVEL_VERY_NICE)
	}
	if mil >= 30 {
		write(bu, RUNTIME_LEVEL_NICE)
	}
	if mil >= 50 {
		write(bu, RUNTIME_LEVEL_GOOD)
	}
	if mil >= 100 {
		write(bu, RUNTIME_LEVEL_SLOW)
	}
	if mil >= 200 {
		write(bu, RUNTIME_LEVEL_BAD)
	}

	return bu.String()
}

// GetRedisHandleLevel 获取Redis处理的响应日志
//
// Parameters:
//   - dur: time.Duration
//
// Returns:
//   - string
func GetRedisHandleLevel(dur time.Duration) string {
	mil := dur.Milliseconds() // 获取毫秒

	// 使用对象池，避免频繁创建对象
	bu := pool.Get().(*strings.Builder)
	defer func() {
		bu.Reset()
		pool.Put(bu)
	}()
	write(bu, RUNTIME_LEVEL_INSTANT)
	if mil >= 5 {
		write(bu, RUNTIME_LEVEL_VERY_NICE)
	}
	if mil >= 10 {
		write(bu, RUNTIME_LEVEL_NICE)
	}
	if mil >= 20 {
		write(bu, RUNTIME_LEVEL_GOOD)
	}
	if mil >= 50 {
		write(bu, RUNTIME_LEVEL_SLOW)
	}
	if mil >= 100 {
		write(bu, RUNTIME_LEVEL_BAD)
	}
	return bu.String()
}

// GetMysqlHandleLevel 获取mysql处理时间日志
//
// Parameters:
//   - dur: time.Duration
//
// Returns:
//   - string
func GetMysqlHandleLevel(dur time.Duration) string {
	mil := dur.Milliseconds() // 获取毫秒

	// 使用对象池，避免频繁创建对象
	bu := pool.Get().(*strings.Builder)
	defer func() {
		bu.Reset()
		pool.Put(bu)
	}()
	write(bu, RUNTIME_LEVEL_INSTANT)
	if mil >= 20 {
		write(bu, RUNTIME_LEVEL_VERY_NICE)
	}
	if mil >= 50 {
		write(bu, RUNTIME_LEVEL_NICE)
	}
	if mil >= 100 {
		write(bu, RUNTIME_LEVEL_GOOD)
	}
	if mil >= 200 {
		write(bu, RUNTIME_LEVEL_SLOW)
	}
	if mil >= 500 {
		write(bu, RUNTIME_LEVEL_BAD)
	}
	return bu.String()
}

// GetOtherHandleLevel 获取自定义"处理时间"日志表示
//
// mls长度必须是5，且从小到大排序。如果未从小到大排序，会导致错误的时间日志
//
// Parameters:
//   - dur: time.Duration 处理耗时
//   - mls: []int64 耗时等级划分（需要从小到大排序），默认为[5,10,20,50,100]，且单位为 ms
//
// Returns:
//   - zap.Field 最终格式化的时间，比如："time_level" : "l_instant:l_very_nice:l_nice"
func GetOtherHandleLevel(dur time.Duration, msl []int64) zap.Field {
	if len(msl) != 5 {
		msl = defaultMsl
	}

	mil := dur.Milliseconds() // 获取毫秒

	// 使用对象池，避免频繁创建对象
	bu := pool.Get().(*strings.Builder)
	defer func() {
		bu.Reset()
		pool.Put(bu)
	}()
	write(bu, RUNTIME_LEVEL_INSTANT)
	if mil >= msl[0] {
		write(bu, RUNTIME_LEVEL_VERY_NICE)
	}
	if mil >= msl[1] {
		write(bu, RUNTIME_LEVEL_NICE)
	}
	if mil >= msl[2] {
		write(bu, RUNTIME_LEVEL_GOOD)
	}
	if mil >= msl[3] {
		write(bu, RUNTIME_LEVEL_SLOW)
	}
	if mil >= msl[4] {
		write(bu, RUNTIME_LEVEL_BAD)
	}
	return zap.String("time_level", bu.String())
}
