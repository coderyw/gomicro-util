package sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	redisfacade "github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
)

// fakeRedisInterface 只实现 sync 模块用到的 Client()。
// 通过嵌入 RedisInterface（nil），其他方法被触达时会 panic，让测试快速暴露问题。
type fakeRedisInterface struct {
	redisfacade.RedisInterface
	client redis.UniversalClient
}

func (f *fakeRedisInterface) Client() redis.UniversalClient { return f.client }

// 构造一个跳过 AddJob（即跳过 cron / consistent_hash）的最小 RedisSyncJob。
func newTestJob(t *testing.T, queueNum int64, parts []string, doSync model.RedisSyncFunc) (*RedisSyncJob, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cli.Close() })

	pool, err := ants.NewPool(8)
	require.NoError(t, err)
	t.Cleanup(pool.Release)

	job := NewRedisSyncJob(context.Background(), model.RedisSyncOption{
		SyncKey:  "test-sync",
		Ts:       5,
		DoSync:   doSync,
		QueueNum: queueNum,
	}, &fakeRedisInterface{client: cli}, pool)
	job.canSyncPt = parts
	return job, mr
}

func zsetKey(job *RedisSyncJob, field string) string {
	pt := job.hashMod(field, job.queueNum)
	return fmt.Sprintf("%v:%v", job.syncKey, pt)
}

func versionHashKey(zsetKey string) string {
	return fmt.Sprintf("{%s}:version", zsetKey)
}

// 成功路径：putWaitSync → sync → ZSET / version hash 应清空。
func TestSync_HappyPath(t *testing.T) {
	var called int32
	got := make(chan string, 8)
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			got <- field
			return nil
		})

	ctx := context.Background()
	fields := []string{"field-a", "field-b", "field-c", "field-d"}
	ts := time.Now().Unix() - 10
	for _, f := range fields {
		require.NoError(t, job.putWaitSync(ctx, f, ts))
	}

	job.sync(ctx)

	assert.Equal(t, int32(len(fields)), atomic.LoadInt32(&called))

	collected := make([]string, 0, len(fields))
	for range fields {
		select {
		case f := <-got:
			collected = append(collected, f)
		default:
		}
	}
	assert.ElementsMatch(t, fields, collected)

	cli := job.universalClient.Client()
	for _, pt := range job.canSyncPt {
		key := fmt.Sprintf("%v:%v", job.syncKey, pt)
		assert.Equal(t, int64(0), cli.ZCard(ctx, key).Val(), "zset not cleaned: %s", key)
		assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(), "version not cleaned: %s", versionHashKey(key))
	}
}

// doSync 返回 error → 数据被重入队，score 改写为 now+1。
func TestSync_DoSyncError_Requeues(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return errors.New("boom")
		})

	ctx := context.Background()
	const field = "field-fail"
	originScore := time.Now().Unix() - 10
	require.NoError(t, job.putWaitSync(ctx, field, originScore))

	beforeSync := time.Now().Unix()
	job.sync(ctx)
	afterSync := time.Now().Unix()

	// 同一条数据本轮只会被取出一次（version 已自增，下一次 range 取不到，因为 score 已改成 now+1）
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))

	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	// 数据应仍在 zset 中，score = now+1（来自本轮 sync 内部的 now := start.Unix()）
	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	scoreI := int64(score)
	assert.Greater(t, scoreI, originScore, "重入后 score 必须大于原始 score")
	assert.GreaterOrEqual(t, scoreI, beforeSync+1, "score 至少为 now+1")
	assert.LessOrEqual(t, scoreI, afterSync+1, "score 不应超过 sync 结束时间+1")

	// 版本 hash 应被 sync_add.lua 中的 HDEL 清掉
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())
}

// doSync panic → safeCall 兜底，不影响同一轮中其他 item。
func TestSync_DoSyncPanic_DoesNotStopOthers(t *testing.T) {
	var panicCalls, okCalls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			if field == "panic-key" {
				atomic.AddInt32(&panicCalls, 1)
				panic("oops")
			}
			atomic.AddInt32(&okCalls, 1)
			return nil
		})

	ctx := context.Background()
	ts := time.Now().Unix() - 10
	require.NoError(t, job.putWaitSync(ctx, "panic-key", ts))
	require.NoError(t, job.putWaitSync(ctx, "ok-key", ts))

	job.sync(ctx)

	assert.Equal(t, int32(1), atomic.LoadInt32(&panicCalls))
	assert.Equal(t, int32(1), atomic.LoadInt32(&okCalls))

	cli := job.universalClient.Client()

	// ok-key：成功同步后应被 ZREM
	okKey := zsetKey(job, "ok-key")
	_, err := cli.ZScore(ctx, okKey, "ok-key").Result()
	assert.ErrorIs(t, err, redis.Nil, "ok-key 应已从 zset 中删除")

	// panic-key：失败后被重入队 score = now+1
	panicKey := zsetKey(job, "panic-key")
	score, err := cli.ZScore(ctx, panicKey, "panic-key").Result()
	require.NoError(t, err)
	assert.Greater(t, int64(score), ts, "panic-key 应被重入队，score 大于原始 score")
}

// 并发写入：worker 取到 version=1 后，业务侧再次 putWaitSync 会 HDEL 版本号；
// 此时 remSyncItem 应当跳过 ZREM 并返回 nil，避免误删新数据。
func TestRemSyncItem_VersionMismatch_SkipsZREM(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"},
		func(ctx context.Context, field string) error { return nil })

	ctx := context.Background()
	const field = "race-field"
	require.NoError(t, job.putWaitSync(ctx, field, time.Now().Unix()-10))

	key := zsetKey(job, field)

	items, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "1", items[0].Version, "首次 HINCRBY 后版本号应为 1")

	// 模拟"业务侧并发再次入队"：sync_add.lua 中的 HDEL 让旧 version 失效
	require.NoError(t, job.putWaitSync(ctx, field, time.Now().Unix()-5))

	// 此时 worker 才执行 remSyncItem：版本失配 → 跳过 ZREM、不报错
	require.NoError(t, job.remSyncItem(ctx, key, items[0]))

	cli := job.universalClient.Client()

	// ZSET 中数据应保留（业务侧新写入未被误删）
	_, err = cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err, "并发写入的数据不应被误删")

	// 版本 hash 此刻应为空（最后一次 putWaitSync 的 HDEL 之后没有任何 HINCRBY）
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())
}

// PutSyncKey 对未注册的 syncKey 应直接返回 error，不走 redis。
func TestPutSyncKey_UnknownSyncKey(t *testing.T) {
	mgr, err := NewRedisSyncMgr()
	require.NoError(t, err)

	err = mgr.PutSyncKey(context.Background(), "not-registered", "f", time.Now().Unix())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not exist")
}

// rangeSyncItems 整体失败时：当前分区应跳过、其他分区继续；不应 panic。
func TestSync_RangeError_SkipsPartitionAndContinues(t *testing.T) {
	var calls int32
	job, mr := newTestJob(t, 1, []string{"0"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})

	ctx := context.Background()
	require.NoError(t, job.putWaitSync(ctx, "f1", time.Now().Unix()-10))

	mr.SetError("simulated redis error")
	require.NotPanics(t, func() { job.sync(ctx) })
	mr.SetError("")

	assert.Equal(t, int32(0), atomic.LoadInt32(&calls), "range 失败时不应调用 doSync")

	cli := job.universalClient.Client()
	_, err := cli.ZScore(ctx, zsetKey(job, "f1"), "f1").Result()
	require.NoError(t, err, "数据应仍在 zset 中")
}

// remSyncItem 真实网络错误（非版本失配）→ 进入失败重入分支；
// 同步过程中通过 SetError 模拟 doSync 后所有 redis 命令失败，
// 期望：不 panic、数据保留在 zset 中。
func TestSync_RemError_DataPreserved(t *testing.T) {
	var doCalls int32
	var mr *miniredis.Miniredis
	job, mrRef := newTestJob(t, 1, []string{"0"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&doCalls, 1)
			mr.SetError("simulated rem/putWait error")
			return nil
		})
	mr = mrRef

	ctx := context.Background()
	const field = "rem-err-field"
	require.NoError(t, job.putWaitSync(ctx, field, time.Now().Unix()-10))

	require.NotPanics(t, func() { job.sync(ctx) })
	mr.SetError("")

	assert.Equal(t, int32(1), atomic.LoadInt32(&doCalls), "doSync 应被调用 1 次")

	cli := job.universalClient.Client()
	_, err := cli.ZScore(ctx, zsetKey(job, field), field).Result()
	require.NoError(t, err, "rem 失败后数据应保留在 zset 中")
}

// 单分区数据量超过内部 batch count(50) 时，内层循环应分页处理直至全部完成。
func TestSync_BatchPagination(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 1, []string{"0"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})

	ctx := context.Background()
	const total = 120 // 超过内部 count=50，至少需要 3 轮 range
	ts := time.Now().Unix() - 10
	for i := range total {
		require.NoError(t, job.putWaitSync(ctx, fmt.Sprintf("k-%d", i), ts))
	}

	job.sync(ctx)

	assert.Equal(t, int32(total), atomic.LoadInt32(&calls))

	cli := job.universalClient.Client()
	key := fmt.Sprintf("%v:%v", job.syncKey, "0")
	assert.Equal(t, int64(0), cli.ZCard(ctx, key).Val(), "全部数据应已从 zset 中清除")
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(), "version hash 应已清空")
}

// 空数据：没有 putWaitSync 时 sync 应当无任何 doSync 调用，且不报错。
func TestSync_EmptyData_NoOp(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})

	require.NotPanics(t, func() { job.sync(context.Background()) })
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls))
}

// parseSyncItems 对异常长度（奇数）应跳过末尾未配对元素，不 panic。
func TestParseSyncItems_OddLength(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	items := job.parseSyncItems([]any{"a", "1", "b"})
	require.Len(t, items, 1, "末尾未配对元素应被丢弃")
	assert.Equal(t, "a", items[0].Field)
	assert.Equal(t, "1", items[0].Version)

	assert.Empty(t, job.parseSyncItems([]any{}), "空输入应返回空切片")
}

// runOne：syncKey 未注册时不应 panic，应当作 no-op。
func TestRunOne_UnknownSyncKey_NoOp(t *testing.T) {
	mgr, err := NewRedisSyncMgr()
	require.NoError(t, err)
	require.NotPanics(t, func() { mgr.runOne("not-registered") })
}

// runOne：canSyncPt 为空（一致性哈希尚未分配分区）时应直接返回，
// 即便 zset 中已有数据也不应触发 doSync。
func TestRunOne_EmptyCanSyncPt_SkipsSync(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 1, nil, // canSyncPt 显式为空
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})
	require.NoError(t, job.putWaitSync(context.Background(), "f", time.Now().Unix()-10))

	mgr, err := NewRedisSyncMgr()
	require.NoError(t, err)
	mgr.syncTask.Store(job.syncKey, job)

	mgr.runOne(job.syncKey)
	assert.Equal(t, int32(0), atomic.LoadInt32(&calls),
		"canSyncPt 为空时不应调到 doSync")
}

// runOne：正常路径应派发到 v.sync，doSync 被调用对应次数。
func TestRunOne_DispatchesToSync(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})

	ctx := context.Background()
	ts := time.Now().Unix() - 10
	fields := []string{"a", "b", "c"}
	for _, f := range fields {
		require.NoError(t, job.putWaitSync(ctx, f, ts))
	}

	mgr, err := NewRedisSyncMgr()
	require.NoError(t, err)
	mgr.syncTask.Store(job.syncKey, job)
	mgr.runOne(job.syncKey)

	assert.Equal(t, int32(len(fields)), atomic.LoadInt32(&calls))
}

// 分区并发：每个分区一条数据 + doSync sleep 80ms。
// 4 个分区串行需 ~320ms，分区池化并发应远低于此。
func TestSync_PartitionsRunConcurrently(t *testing.T) {
	const sleepEach = 80 * time.Millisecond
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			time.Sleep(sleepEach)
			return nil
		})

	// 直接给每个分区 ZADD 一条数据，绕开 hashMod 的不均匀分布，
	// 让每个分区在 syncPartition 中都至少触发一次 sleep。
	ctx := context.Background()
	cli := job.universalClient.Client()
	ts := time.Now().Unix() - 10
	for _, pt := range job.canSyncPt {
		key := fmt.Sprintf("%v:%v", job.syncKey, pt)
		versionKey := fmt.Sprintf("{%s}:version", key)
		field := "v-" + pt
		require.NoError(t, cli.HSet(ctx, versionKey, field, "1").Err())
		require.NoError(t, cli.ZAdd(ctx, key, redis.Z{Score: float64(ts), Member: field}).Err())
	}

	start := time.Now()
	job.sync(ctx)
	elapsed := time.Since(start)

	// 串行 4*80=320ms；并发理想 80ms。阈值 200ms 给 CI 足够余量。
	assert.Less(t, elapsed, 200*time.Millisecond,
		"分区应并发执行（实际耗时=%v，串行参考=%v）", elapsed, 4*sleepEach)
}

// 协程池关闭后 Submit 必返回错误，sync 必须走"当前 goroutine 同步执行"的降级路径，
// 且所有分区数据仍应被正确处理（不丢数据）。
func TestSync_PoolClosed_FallbackCompletesAll(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})

	// 替换为已关闭的池：Submit 必然失败 → 100% 触发降级路径。
	closedPool, err := ants.NewPool(4)
	require.NoError(t, err)
	closedPool.Release()
	job.pool = closedPool

	ctx := context.Background()
	ts := time.Now().Unix() - 10
	fields := []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8"}
	for _, f := range fields {
		require.NoError(t, job.putWaitSync(ctx, f, ts))
	}

	require.NotPanics(t, func() { job.sync(ctx) })

	assert.Equal(t, int32(len(fields)), atomic.LoadInt32(&calls),
		"池关闭降级后仍应处理全部数据")

	cli := job.universalClient.Client()
	for _, pt := range job.canSyncPt {
		key := fmt.Sprintf("%v:%v", job.syncKey, pt)
		assert.Equal(t, int64(0), cli.ZCard(ctx, key).Val(), "zset 未清空: %s", key)
		assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(),
			"version 未清空: %s", versionHashKey(key))
	}
}

// pool == nil 兜底：构造函数允许传 nil（或字段被外部清空），
// sync 应退化为串行执行，不 panic、数据完整。
func TestSync_NilPool_FallbackSerial(t *testing.T) {
	var calls int32
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&calls, 1)
			return nil
		})
	job.pool = nil

	ctx := context.Background()
	ts := time.Now().Unix() - 10
	fields := []string{"a", "b", "c", "d"}
	for _, f := range fields {
		require.NoError(t, job.putWaitSync(ctx, f, ts))
	}

	require.NotPanics(t, func() { job.sync(ctx) })
	assert.Equal(t, int32(len(fields)), atomic.LoadInt32(&calls))
}

// 多次 putWaitSync 同一 field：每次 HDEL 应让 version hash 中该 field 回到不存在状态，
// HINCRBY 重新从 1 开始 —— 这是版本号机制能够区分"新一代待同步"的基础。
func TestPutWaitSync_ResetsVersion(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSync(ctx, field, time.Now().Unix()-10))
	items1, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.Len(t, items1, 1)
	assert.Equal(t, "1", items1[0].Version, "首次 HINCRBY 应当返回 1")

	require.NoError(t, job.putWaitSync(ctx, field, time.Now().Unix()-5))
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(),
		"再次 putWaitSync 后 version hash 应被 HDEL 清空")

	items2, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.Len(t, items2, 1)
	assert.Equal(t, "1", items2[0].Version,
		"HDEL 之后 HINCRBY 应重新从 1 开始，而非累加到 2")
}

// 排队中重复 PutSyncKey：versionKey 不存在 → 走 NX 分支，
// score 必须保留首次入队时间，避免高频写把同步时间一推再推。
// 这是 30s 攒批语义的核心不变式。
func TestPutWaitSync_NX_PreservesEarliestScore(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSync(ctx, field, 100))
	require.NoError(t, job.putWaitSync(ctx, field, 200))

	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(100), score,
		"排队中 PutSyncKey 应保留首次入队 score（NX 语义）")

	// 版本哈希仍为空（NX 没有触达版本号）
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())
}

// 正在被同步（versionKey 存在）时再次 PutSyncKey：走 ZADD 覆盖分支，
// score 必须被新值替换，且 HDEL 让 rem 必然版本失配。
func TestPutWaitSync_HExists_Overwrites(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSync(ctx, field, 100))

	// 模拟"正在被某次 sync 持有"：range 会 HINCRBY 版本号
	items, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(1), cli.HLen(ctx, versionHashKey(key)).Val(),
		"前置：版本哈希应有 field")

	require.NoError(t, job.putWaitSync(ctx, field, 200))

	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(200), score,
		"versionKey 存在时 PutSyncKey 应覆盖 score")

	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(),
		"HDEL 应清掉版本号，让随后的 rem 必然失配")
}

// 重试路径：传入的 score 大于原 score → GT 应把 score 推大。
func TestPutWaitSyncRetry_GT_AdvancesSmallerScore(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSync(ctx, field, 100))
	require.NoError(t, job.putWaitSyncRetry(ctx, field, 200))

	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(200), score, "GT 应把较小的 score 推大")
}

// 重试路径：传入的 score 小于原 score → GT 应保留原 score，
// 防止异常调用把同步时间往前拉、本应跳过的轮次又被拽回来。
func TestPutWaitSyncRetry_GT_KeepsLargerScore(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSync(ctx, field, 300))
	require.NoError(t, job.putWaitSyncRetry(ctx, field, 200))

	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(300), score, "GT 不应把较大的 score 拉小")
}

// 批量写入（pipeline 路径）：落库结果必须与逐条 putWaitSync 完全一致——
// 每个 field 独立按 hash 落到自己的分区 zset、score 为传入时间戳、版本哈希不被触达。
func TestPutWaitSyncs_BatchMatchesSingleSemantics(t *testing.T) {
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"}, nil)

	ctx := context.Background()
	cli := job.universalClient.Client()
	// 预载脚本，确保走 pipeline EvalSha 主路径而非 NOSCRIPT 降级
	require.NoError(t, cli.ScriptLoad(ctx, syncAdd).Err())

	fields := []string{"field-a", "field-b", "field-c", "field-d"}
	require.NoError(t, job.putWaitSyncs(ctx, fields, 100))

	for _, f := range fields {
		key := zsetKey(job, f)
		score, err := cli.ZScore(ctx, key, f).Result()
		require.NoError(t, err, "field %s 应已入队", f)
		assert.Equal(t, float64(100), score)
		assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())
	}
}

// 批量写入的 NX 语义：重复批量投递时 score 保留首次入队时间，与逐条行为一致。
func TestPutWaitSyncs_NX_PreservesEarliestScore(t *testing.T) {
	job, _ := newTestJob(t, 2, []string{"0", "1"}, nil)

	ctx := context.Background()
	cli := job.universalClient.Client()
	require.NoError(t, cli.ScriptLoad(ctx, syncAdd).Err())

	fields := []string{"f1", "f2"}
	require.NoError(t, job.putWaitSyncs(ctx, fields, 100))
	require.NoError(t, job.putWaitSyncs(ctx, fields, 200))

	for _, f := range fields {
		score, err := cli.ZScore(ctx, zsetKey(job, f), f).Result()
		require.NoError(t, err)
		assert.Equal(t, float64(100), score, "批量重复投递应保留首次入队 score（NX 语义）")
	}
}

// 批量写入遇 versionKey 存在（正在被同步）：覆盖 score 并 HDEL 版本号，与逐条行为一致。
func TestPutWaitSyncs_VersionExists_Overwrites(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	const field = "f"
	key := zsetKey(job, field)
	cli := job.universalClient.Client()
	require.NoError(t, cli.ScriptLoad(ctx, syncAdd).Err())

	require.NoError(t, job.putWaitSyncs(ctx, []string{field, "f-other"}, 100))
	// 模拟"正在被某次 sync 持有"：range 会 HINCRBY 版本号
	_, err := job.rangeSyncItems(ctx, key, time.Now().Unix(), 10)
	require.NoError(t, err)
	require.NotEqual(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(), "前置：版本哈希应有 field")

	require.NoError(t, job.putWaitSyncs(ctx, []string{field, "f-other"}, 200))

	score, err := cli.ZScore(ctx, key, field).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(200), score, "versionKey 存在时批量投递应覆盖 score")
	assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val(), "HDEL 应清掉版本号")
}

// 脚本未缓存（新实例 NOSCRIPT）：pipeline 失败后降级逐条重放，数据一条不少。
func TestPutWaitSyncs_NoScriptFallback(t *testing.T) {
	job, _ := newTestJob(t, 2, []string{"0", "1"}, nil)

	ctx := context.Background()
	cli := job.universalClient.Client()
	// 不预载脚本：pipeline 内 EvalSha 将报 NOSCRIPT

	fields := []string{"f1", "f2", "f3"}
	require.NoError(t, job.putWaitSyncs(ctx, fields, 100))

	for _, f := range fields {
		score, err := cli.ZScore(ctx, zsetKey(job, f), f).Result()
		require.NoError(t, err, "降级重放后 field %s 应已入队", f)
		assert.Equal(t, float64(100), score)
	}
}

// 空列表与单元素：空列表零操作不报错；单元素直接走逐条路径。
func TestPutWaitSyncs_EmptyAndSingle(t *testing.T) {
	job, _ := newTestJob(t, 1, []string{"0"}, nil)

	ctx := context.Background()
	cli := job.universalClient.Client()

	require.NoError(t, job.putWaitSyncs(ctx, nil, 100))
	require.Empty(t, cli.Keys(ctx, "*").Val(), "空列表不应产生任何键")

	require.NoError(t, job.putWaitSyncs(ctx, []string{"only"}, 100))
	score, err := cli.ZScore(ctx, zsetKey(job, "only"), "only").Result()
	require.NoError(t, err)
	assert.Equal(t, float64(100), score)
}

// 未注册的 syncKey：与 PutSyncKey 同口径报错。
func TestPutSyncKeys_UnknownSyncKey(t *testing.T) {
	mgr := &RedisSyncMgr{}
	err := mgr.PutSyncKeys(context.Background(), "not-exist", []string{"f"}, 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syncKey not exist")
}

// 批量投递 → sync 全链路：doSync 对每个 field 恰好回调一次，队列与版本哈希清空。
func TestPutWaitSyncs_ThenSync_HappyPath(t *testing.T) {
	var called int32
	got := make(chan string, 8)
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"},
		func(ctx context.Context, field string) error {
			atomic.AddInt32(&called, 1)
			got <- field
			return nil
		})

	ctx := context.Background()
	cli := job.universalClient.Client()
	require.NoError(t, cli.ScriptLoad(ctx, syncAdd).Err())

	fields := []string{"field-a", "field-b", "field-c", "field-d"}
	require.NoError(t, job.putWaitSyncs(ctx, fields, time.Now().Unix()-10))

	job.sync(ctx)

	assert.Equal(t, int32(len(fields)), atomic.LoadInt32(&called))
	for _, f := range fields {
		key := zsetKey(job, f)
		_, err := cli.ZScore(ctx, key, f).Result()
		assert.ErrorIs(t, err, redis.Nil, "同步完成后 zset 应清空")
		assert.Equal(t, int64(0), cli.HLen(ctx, versionHashKey(key)).Val())
	}
}

// 并发批量投递（配合 go test -race）：多 goroutine 同时 putWaitSyncs，
// 模拟批量消费按分区并发回调的真实调用形态；断言无数据竞争且所有 field 均入队。
func TestPutWaitSyncs_ConcurrentBatches(t *testing.T) {
	job, _ := newTestJob(t, 4, []string{"0", "1", "2", "3"}, nil)

	ctx := context.Background()
	cli := job.universalClient.Client()
	require.NoError(t, cli.ScriptLoad(ctx, syncAdd).Err())

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			fields := make([]string, 0, 25)
			for i := 0; i < 25; i++ {
				fields = append(fields, fmt.Sprintf("g%d-f%d", g, i))
			}
			assert.NoError(t, job.putWaitSyncs(ctx, fields, 100))
		}(g)
	}
	wg.Wait()

	total := int64(0)
	for pt := int64(0); pt < 4; pt++ {
		total += cli.ZCard(ctx, fmt.Sprintf("%v:%v", job.syncKey, pt)).Val()
	}
	assert.Equal(t, int64(100), total, "4 goroutine × 25 field 应全部入队")
}
