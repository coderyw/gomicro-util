package sync

import (
	"context"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pastCreatedAt 把 job 的 createdAt 拨回过去，跳过无主检查的冷启动静默期。
func pastCreatedAt(job *RedisSyncJob) {
	job.createdAt = time.Now().Add(-time.Duration(job.ts*hbTTLFactor)*time.Second - time.Second)
}

// 分区扫描完成后应写入心跳 key，TTL = ts*hbTTLFactor。
func TestSync_WritesHeartbeat(t *testing.T) {
	job, mr := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error { return nil })
	pastCreatedAt(job)

	job.sync(context.Background())

	for _, pt := range job.canSyncPt {
		hb := job.hbKeyOf(pt)
		require.True(t, mr.Exists(hb), "heartbeat missing: %s", hb)
		ttl := mr.TTL(hb)
		assert.Equal(t, time.Duration(job.ts*hbTTLFactor)*time.Second, ttl, "heartbeat ttl: %s", hb)
	}
}

// 无主分区（无心跳、无人认领）中的积压数据应被兜底接管消费。
func TestSync_OrphanPartitionTakenOver(t *testing.T) {
	var called int32
	got := make(chan string, 1)
	// 本实例只认领分区 0/1，分区 2/3 无主
	job, mr := newTestJob(t, 4, []string{"0", "1"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			got <- field
			return nil
		})
	pastCreatedAt(job)

	ctx := context.Background()
	// 找一个哈希到无主分区（2 或 3）的 field
	orphanField := ""
	for _, f := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if pt := hashMod(f, job.queueNum); pt == 2 || pt == 3 {
			orphanField = f
			break
		}
	}
	require.NotEmpty(t, orphanField, "no field hashes to orphan partition")
	require.NoError(t, job.putWaitSync(ctx, orphanField, time.Now().Unix()-10))

	job.sync(ctx)

	assert.Equal(t, int32(1), atomic.LoadInt32(&called), "orphan partition should be taken over and synced")
	assert.Equal(t, orphanField, <-got)
	// 接管后 zset 应被清空
	key := zsetKey(job, orphanField)
	assert.Equal(t, int64(0), job.universalClient.Client().ZCard(ctx, key).Val())
	// 代扫的分区也要写心跳
	pt := hashMod(orphanField, job.queueNum)
	assert.True(t, mr.Exists(job.hbKeyOf(strconv.FormatInt(pt, 10))))
}

// 心跳仍然存活的分区不应被接管（归属方还活着）。
func TestSync_AliveParitionNotTakenOver(t *testing.T) {
	var called int32
	job, _ := newTestJob(t, 4, []string{"0", "1"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			return nil
		})
	pastCreatedAt(job)

	ctx := context.Background()
	orphanField := ""
	for _, f := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if pt := hashMod(f, job.queueNum); pt == 2 || pt == 3 {
			orphanField = f
			break
		}
	}
	require.NotEmpty(t, orphanField)
	require.NoError(t, job.putWaitSync(ctx, orphanField, time.Now().Unix()-10))

	// 模拟"别的实例"仍在扫这个分区：预先写好心跳
	pt := strconv.FormatInt(hashMod(orphanField, job.queueNum), 10)
	require.NoError(t, job.universalClient.Client().
		Set(ctx, job.hbKeyOf(pt), "other-instance", time.Minute).Err())

	job.sync(ctx)

	assert.Equal(t, int32(0), atomic.LoadInt32(&called), "partition with live heartbeat must not be taken over")
}

// takeover 锁已被其他实例抢占时，本实例不应重复代扫。
func TestSync_TakeoverLockMutex(t *testing.T) {
	var called int32
	job, _ := newTestJob(t, 4, []string{"0", "1"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			return nil
		})
	pastCreatedAt(job)

	ctx := context.Background()
	orphanField := ""
	for _, f := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if pt := hashMod(f, job.queueNum); pt == 2 || pt == 3 {
			orphanField = f
			break
		}
	}
	require.NotEmpty(t, orphanField)
	require.NoError(t, job.putWaitSync(ctx, orphanField, time.Now().Unix()-10))

	// 模拟别的实例已抢到接管锁
	pt := strconv.FormatInt(hashMod(orphanField, job.queueNum), 10)
	require.NoError(t, job.universalClient.Client().
		SetNX(ctx, job.takeoverKeyOf(pt), "other-instance", time.Minute).Err())

	job.sync(ctx)

	assert.Equal(t, int32(0), atomic.LoadInt32(&called), "takeover lock held by another instance, must not double-scan")
}

// 冷启动静默期内不做无主检查（避免全量分区误报接管风暴）。
func TestSync_OrphanCheckSkippedDuringWarmup(t *testing.T) {
	var called int32
	job, _ := newTestJob(t, 4, []string{"0", "1"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			return nil
		})
	// 不拨 createdAt：newTestJob 刚创建，处于静默期

	ctx := context.Background()
	orphanField := ""
	for _, f := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if pt := hashMod(f, job.queueNum); pt == 2 || pt == 3 {
			orphanField = f
			break
		}
	}
	require.NotEmpty(t, orphanField)
	require.NoError(t, job.putWaitSync(ctx, orphanField, time.Now().Unix()-10))

	job.sync(ctx)

	assert.Equal(t, int32(0), atomic.LoadInt32(&called), "orphan check must be silent during warmup")
}

// canSyncPt 为空的实例（哈希环没分到分区）仍要参与无主接管。
func TestSync_EmptyAssignmentStillTakesOverOrphan(t *testing.T) {
	var called int32
	job, _ := newTestJob(t, 4, nil,
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			return nil
		})
	pastCreatedAt(job)

	ctx := context.Background()
	require.NoError(t, job.putWaitSync(ctx, "some-field", time.Now().Unix()-10))

	job.sync(ctx)

	assert.Equal(t, int32(1), atomic.LoadInt32(&called), "instance with empty assignment must still take over orphans")
}
