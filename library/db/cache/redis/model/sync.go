package model

import "context"

// 基于 redis 的同步
//
// 通过有序集合，在时间尺度上进行同步。

type RedisSyncOption struct {
	SyncKey  string        //
	Ts       int64         //定时同步时间，单位 s
	DoSync   RedisSyncFunc //同步函数
	QueueNum int64         //同步队列分片数量
}

// RedisSyncFunc
//
// 参数:
//   - ctx: 上下文
//   - string: 待同步集合的 field
type RedisSyncFunc func(context.Context, string) error
