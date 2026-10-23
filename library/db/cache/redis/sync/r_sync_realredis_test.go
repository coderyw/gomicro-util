package sync

// 真实 Redis 冒烟测试（自测用，不进 CI）：
// 需设置环境变量 REAL_REDIS_ADDR（如 127.0.0.1:16379）才会执行，否则跳过。
// 验证 miniredis 覆盖不到的真服务器语义：pipeline 内 EvalSha 的 NOSCRIPT 降级、
// 脚本缓存后的 pipeline 主路径、批量与逐条落库结果逐字段一致。

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
)

func newRealRedisJob(t *testing.T, queueNum int64) (*RedisSyncJob, redis.UniversalClient) {
	t.Helper()
	addr := os.Getenv("REAL_REDIS_ADDR")
	if addr == "" {
		t.Skip("REAL_REDIS_ADDR 未设置，跳过真实 Redis 冒烟")
	}
	cli := redis.NewClient(&redis.Options{Addr: addr, DB: 15})
	require.NoError(t, cli.FlushDB(context.Background()).Err())
	t.Cleanup(func() {
		_ = cli.FlushDB(context.Background()).Err()
		_ = cli.Close()
	})

	pool, err := ants.NewPool(8)
	require.NoError(t, err)
	t.Cleanup(pool.Release)

	job := NewRedisSyncJob(context.Background(), model.RedisSyncOption{
		SyncKey:  fmt.Sprintf("real-sync-%d", time.Now().UnixNano()),
		Ts:       5,
		QueueNum: queueNum,
	}, &fakeRedisInterface{client: cli}, pool)
	return job, cli
}

// 冷启动（脚本未缓存）批量写入 → NOSCRIPT 降级逐条；随后二次批量走 pipeline 主路径。
// 两轮之后逐字段与逐条写入的另一 syncKey 对账，语义必须一致。
func TestRealRedis_PutWaitSyncs_ColdAndWarm(t *testing.T) {
	job, cli := newRealRedisJob(t, 4)
	ctx := context.Background()

	fields := []string{"u-1001", "u-1002", "u-1003", "u-1004", "u-1005"}

	// 冷启动：SCRIPT FLUSH 保证脚本一定不在缓存，命中 NOSCRIPT 降级路径
	require.NoError(t, cli.ScriptFlush(ctx).Err())
	require.NoError(t, job.putWaitSyncs(ctx, fields, 100))
	for _, f := range fields {
		score, err := cli.ZScore(ctx, zsetKey(job, f), f).Result()
		require.NoError(t, err, "冷启动降级后 field %s 应已入队", f)
		assert.Equal(t, float64(100), score)
	}

	// 暖缓存：降级路径的 Eval 已把脚本载入缓存，此轮应走 pipeline EvalSha 主路径。
	// NX 语义：score 维持 100 不被 200 覆盖
	require.NoError(t, job.putWaitSyncs(ctx, fields, 200))
	for _, f := range fields {
		score, err := cli.ZScore(ctx, zsetKey(job, f), f).Result()
		require.NoError(t, err)
		assert.Equal(t, float64(100), score, "NX 语义下二次批量投递不应覆盖 score")
	}

	// 对账：同一批 field 用逐条 putWaitSync 写到另一个 job（不同 syncKey，共用同一 redis 连接，
	// 注意不能再走 newRealRedisJob——它会 FlushDB 清掉 job 已写入的数据），
	// 逐字段比对分区归属与 score，验证批量与逐条语义一致
	pool2, err := ants.NewPool(8)
	require.NoError(t, err)
	t.Cleanup(pool2.Release)
	job2 := NewRedisSyncJob(context.Background(), model.RedisSyncOption{
		SyncKey:  job.syncKey + "-single",
		Ts:       5,
		QueueNum: 4,
	}, &fakeRedisInterface{client: cli}, pool2)
	for _, f := range fields {
		require.NoError(t, job2.putWaitSync(ctx, f, 100))
	}
	for _, f := range fields {
		batchPt := job.hashMod(f, job.queueNum)
		singlePt := job2.hashMod(f, job2.queueNum)
		require.Equal(t, batchPt, singlePt, "同一 field 批量与逐条应落入同号分区")
		s1 := cli.ZScore(ctx, zsetKey(job, f), f).Val()
		s2 := cli.ZScore(ctx, zsetKey(job2, f), f).Val()
		assert.Equal(t, s2, s1, "field %s 批量与逐条 score 不一致", f)
	}
}

// versionKey 存在（正在被同步）时的批量覆盖语义，在真实 Redis 上复核。
func TestRealRedis_PutWaitSyncs_VersionOverwrite(t *testing.T) {
	job, cli := newRealRedisJob(t, 1)
	ctx := context.Background()
	const field = "u-2001"
	key := zsetKey(job, field)

	require.NoError(t, job.putWaitSyncs(ctx, []string{field, "u-2002"}, 100))
	_, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.NotEqual(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())

	require.NoError(t, job.putWaitSyncs(ctx, []string{field, "u-2002"}, 200))
	assert.Equal(t, float64(200), cli.ZScore(ctx, key, field).Val(), "versionKey 存在时应覆盖 score")
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(), "HDEL 应清掉版本号")
}
