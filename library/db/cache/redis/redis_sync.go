package redis

import (
	"context"
	"errors"

	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
	sync2 "github.com/coderyw/gomicro-util/library/db/cache/redis/sync"
)

var syncMgr *sync2.RedisSyncMgr

// InitSyncJob 初始化同步任务
//
// Parameters:
//   - opt RedisSyncOption 同步参数
func (r *Redis) InitSyncJob(ctx context.Context, opt model.RedisSyncOption) error {
	if opt.DoSync == nil {
		return errors.New("doSync is nil")
	}
	if opt.SyncKey == "" {
		return errors.New("syncKey is empty")
	}
	r1 := *r
	return syncMgr.AddJob(ctx, opt, &r1)
}

// PutWaitSyncKey 添加待同步数据
//
//	使用前需要先通过 InitSyncJob 进行初始化，然后会根据分区数量，以及 field 的 hash 值进行取模计算最终待同步的 key(zset):
//	syncKey:partition
//
//	zset 中 field 存储的是函数传递进来的 filed，score 传入的是时间戳
//
// Parameters:
//   - syncKey string 同步 key
//   - field string 同步字段
//   - timestamp int64 秒时间戳
//
// Returns：
//   - error
func (r *Redis) PutWaitSyncKey(ctx context.Context, syncKey string, field string, timestamp int64) error {
	return syncMgr.PutSyncKey(ctx, syncKey, field, timestamp)
}

// PutWaitSyncKeys 批量添加待同步数据
//
//	语义与逐条调用 PutWaitSyncKey 完全一致（每个 field 独立按 hash 取模分区、独立执行写入脚本），
//	仅将 N 次网络往返合并为一次 pipeline，供单次需投递大量 field 的调用方（如批量消费）使用
//
// Parameters:
//   - syncKey string 同步 key
//   - fields []string 同步字段列表
//   - timestamp int64 秒时间戳
//
// Returns：
//   - error
func (r *Redis) PutWaitSyncKeys(ctx context.Context, syncKey string, fields []string, timestamp int64) error {
	return syncMgr.PutSyncKeys(ctx, syncKey, fields, timestamp)
}
