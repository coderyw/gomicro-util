package sync

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/consistent_hash"
	"github.com/coderyw/gomicro-util/library/cron"
	redis2 "github.com/coderyw/gomicro-util/library/db/cache/redis/facade"
	"github.com/coderyw/gomicro-util/library/db/cache/redis/model"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/library/trace"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

const (
	syncAddModeNormal int64 = 1
	syncAddModeRetry  int64 = 2

	// 分区心跳 TTL = ts * hbTTLFactor。
	// 心跳由每轮扫完分区的实例写入，TTL 覆盖约 5 个同步周期，
	// 吸收 cron 抖动（±ts/5）与单轮扫描耗时
	hbTTLFactor = 5
)

//go:embed sync_rem.lua
var syncRem string

//go:embed sync_range.lua
var syncRange string

//go:embed sync_add.lua
var syncAdd string

// 同步任务
type RedisSyncMgr struct {
	// 同步任务
	syncTask sync.Map
	// 同步协程池
	syncPool *ants.Pool
}

func NewRedisSyncMgr() (*RedisSyncMgr, error) {
	pool, err := ants.NewPool(100)
	if err != nil {
		return nil, err
	}
	return &RedisSyncMgr{
		syncPool: pool,
	}, nil
}

// 同步 job
type RedisSyncJob struct {
	ctx             context.Context // 注册时的租户上下文；sync 运行时复用，使 Lua 脚本 key 经前缀 hook 落到该租户，保证多租户隔离
	syncKey         string
	ts              int64               // 定时同步时间，单位 s
	doSync          model.RedisSyncFunc // 同步函数, 要同步的 key
	queueNum        int64
	canSyncPt       []string              // 可以同步的分区
	universalClient redis2.RedisInterface // 保存当时 redis 实例
	syncRemLua      *redis.Script
	syncRangeLua    *redis.Script
	syncAddLua      *redis.Script
	pool            *ants.Pool // 分区并发协程池
	// createdAt 用于无主分区检查的冷启动静默期：
	// 进程刚启动时全量分区的心跳还没建立，立刻检查会把所有分区误判为无主
	createdAt time.Time
}

func NewRedisSyncJob(ctx context.Context, opt model.RedisSyncOption, client redis2.RedisInterface, pool *ants.Pool) *RedisSyncJob {
	job := &RedisSyncJob{
		ctx:             ctx,
		syncKey:         opt.SyncKey,
		ts:              opt.Ts,
		doSync:          opt.DoSync,
		queueNum:        opt.QueueNum,
		universalClient: client,
		syncRemLua:      redis.NewScript(syncRem),
		syncRangeLua:    redis.NewScript(syncRange),
		syncAddLua:      redis.NewScript(syncAdd),
		pool:            pool,
		createdAt:       time.Now(),
	}
	return job
}

type SyncItem struct {
	Field   string
	Version string
}

// 添加 job
func (r *RedisSyncMgr) AddJob(ctx context.Context, opt model.RedisSyncOption, client redis2.RedisInterface) error {
	if opt.QueueNum == 0 {
		opt.QueueNum = 10
	}
	if opt.Ts == 0 {
		opt.Ts = 5
	}
	if opt.Ts < 4 {
		opt.Ts = 4
	}
	job := NewRedisSyncJob(ctx, opt, client, r.syncPool)
	// 按「租户 + syncKey」分键存储，保证同一 syncKey 的不同租户任务互不覆盖；
	// runOne / ServerChange / cron 均以该 taskKey 为标识。
	taskKey := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, opt.SyncKey)
	if _, loaded := r.syncTask.LoadOrStore(taskKey, job); loaded {
		return fmt.Errorf("syncKey %q already registered", taskKey)
	}
	// 计算所有分区任务
	vl := make([]string, opt.QueueNum)
	for i := range opt.QueueNum {
		vl[i] = fmt.Sprintf("%d", i)
	}
	// 开启监听一致性哈希环变化（合并后 OnChange 为 (key, values) (ok, ch, err) 签名）
	ok, ch, err := consistent_hash.OnChange(taskKey, vl)
	if err != nil {
		// 回滚已写入的 syncTask，避免后续重试注册被 LoadOrStore 卡住
		r.syncTask.Delete(taskKey)
		log.Error("Server 注册Redis错误", zap.Error(err))
		return err
	}
	if !ok {
		// 已经监听过，不需要再次监听
		return nil
	}
	// 启动监听变化
	go r.ServerChange(ch)
	// 对定时任务进行增加随机时间
	// 周期 30s，抖动 ±6s（即 24-35s 之间一个值
	// 最大抖动+-30s
	rd := min(max(opt.Ts/5, 1), 30)
	ts := opt.Ts + rand.Int63n(rd*2) - rd
	// 添加到定时任务
	cron.AddCron(cron2.CronJob{
		Name:  taskKey,
		DoJob: func() { r.runOne(taskKey) },
		// 打乱定时任务时间
		CronSpc:         fmt.Sprintf("@every %vs", ts),
		DoBeforeDone:    false,
		DelayBeforeDone: false,
	})
	return nil
}

func (r *RedisSyncMgr) PutSyncKey(ctx context.Context, syncKey string, field string, timestamp int64) error {
	taskKey := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, syncKey)
	val, ok := r.syncTask.Load(taskKey)
	if !ok {
		return errors.New("syncKey not exist")
	}
	return val.(*RedisSyncJob).putWaitSync(ctx, field, timestamp)
}

// 服务变动时候处理
func (r *RedisSyncMgr) ServerChange(ch <-chan consistent_hash.ChangeNotice) {
	// 监听变动
	for v := range ch {
		valI, ok := r.syncTask.Load(v.BusinessKey)
		if !ok {
			log.Warn("[Redis 同步] 收到未注册 syncKey 的变更，已跳过", zap.String("key", v.BusinessKey))
			continue
		}
		nv := *valI.(*RedisSyncJob)
		nv.canSyncPt = v.Jobs
		r.syncTask.Store(v.BusinessKey, &nv)
	}
}

// 运行单个 syncKey 的同步任务
func (r *RedisSyncMgr) runOne(taskKey string) {
	valI, ok := r.syncTask.Load(taskKey)
	if !ok {
		log.Warn("[Redis 同步] runOne 未找到 syncKey", zap.String("key", taskKey))
		// todo 未找到情况可以删除定时任务
		return
	}
	v := valI.(*RedisSyncJob)
	// 复用注册时的租户 ctx，使 sync 内 Lua 脚本 key 与分区心跳 key 经前缀 hook
	// 落到该租户，保证多租户隔离
	ctx := trace.GetTraceCtx(v.ctx)
	// 注意：canSyncPt 为空也要继续执行——
	// sync 内部除了扫自己认领的分区，还会做无主分区兜底检查。
	// 哈希环视图异常（如幽灵节点占位）时本实例可能分不到分区，
	// 但它仍然有责任参与无主接管
	log.InfoCtx(ctx, "[Redis 同步] Start sync", zap.String("key", v.syncKey), zap.Strings("values", v.canSyncPt))
	v.sync(ctx)
}

// parseSyncItems 解析 sync_range.lua 返回的扁平 [field, version, field, version, ...] 数组。
// Lua 侧约定都是 string，这里走类型断言；理论不可达的非 string 元素跳过，避免一行坏数据破坏整批。
func (r *RedisSyncJob) parseSyncItems(vals []any) []SyncItem {
	items := make([]SyncItem, 0, len(vals)/2)

	for i := 0; i+1 < len(vals); i += 2 {
		field, ok1 := vals[i].(string)
		version, ok2 := vals[i+1].(string)
		if !ok1 || !ok2 {
			continue
		}
		items = append(items, SyncItem{Field: field, Version: version})
	}

	return items
}

func (r *RedisSyncJob) rangeSyncItems(ctx context.Context, key string, now int64, count int64) ([]SyncItem, error) {
	versionKey := versionKeyOf(ctx, key)

	res, err := r.syncRangeLua.Run(
		ctx,
		r.universalClient.Client(),
		[]string{key, versionKey},
		strconv.FormatInt(now, 10),
		strconv.FormatInt(count, 10),
		strconv.FormatInt(r.ts*10, 10),
	).Result()
	if err != nil {
		return nil, err
	}

	vals, ok := res.([]any)
	if !ok {
		return nil, fmt.Errorf("sync range lua result type invalid: %T", res)
	}

	return r.parseSyncItems(vals), nil
}

func (r *RedisSyncJob) remSyncItem(ctx context.Context, key string, item SyncItem) error {
	versionKey := versionKeyOf(ctx, key)

	res, err := r.syncRemLua.Run(
		ctx,
		r.universalClient.Client(),
		[]string{key, versionKey},
		item.Field,
		item.Version,
	).Int()
	if err != nil {
		return err
	}

	if res == 0 {
		log.WarnCtx(ctx, "[Redis 同步] 删除跳过，版本已变化",
			zap.String("syncKey", r.syncKey),
			zap.String("key", key),
			zap.String("waitKey", item.Field),
			zap.String("version", item.Version),
		)
	}

	return nil
}

// 开始同步：各分区走协程池并发，池满则降级当前 goroutine 同步执行，避免漏跑
func (r *RedisSyncJob) sync(ctx context.Context) {
	start := time.Now()
	now := start.Unix()
	var wg sync.WaitGroup
	for _, pt := range r.canSyncPt {
		wg.Add(1)
		task := func() {
			defer wg.Done()
			r.syncPartition(ctx, pt, start, now)
		}
		if r.pool == nil {
			task()
			continue
		}
		if err := r.pool.Submit(task); err != nil {
			log.WarnCtx(ctx, "[Redis 同步] 分区入池失败，降级同步执行",
				zap.String("syncKey", r.syncKey), zap.String("pt", pt), zap.Error(err))
			task()
		}
	}
	wg.Wait()
	// 自己认领的分区扫完后，兜底检查无主分区
	r.checkOrphanPartitions(ctx, start, now)
}

// 单个分区的同步循环
func (r *RedisSyncJob) syncPartition(ctx context.Context, pt string, start time.Time, now int64) {
	const maxLoops = 10000
	key := r.syncKey + ":" + pt
	count := int64(50)
	total := 0
	log.InfoCtx(ctx, "[Redis 同步]", zap.String("syncKey", r.syncKey), zap.String("key", key))

	loops := 0
	for {
		loops++
		if loops > maxLoops {
			log.ErrorCtx(ctx, "[Redis 同步] 单分区轮次超限，强制退出", zap.String("key", key), zap.Int("loops", loops))
			return
		}
		arr, err := r.rangeSyncItems(ctx, key, now, count)
		if err != nil {
			log.ErrorCtx(ctx, "[Redis 同步] 获取待同步数据错误", zap.String("key", key), zap.Error(err))
			return
		}
		for _, k := range arr {
			log.InfoCtx(ctx, "[Redis 同步] 开始同步", zap.String("syncKey", r.syncKey), zap.String("key", key), zap.String("waitKey", k.Field))

			var err error
			if err = r.safeCall(ctx, key, k.Field); err != nil {
				log.ErrorCtx(ctx, "[Redis 同步] 同步报错", zap.String("syncKey", r.syncKey), zap.String("key", key), zap.Error(err))
			} else if err = r.remSyncItem(ctx, key, k); err != nil {
				log.ErrorCtx(ctx, "[Redis 同步] 删除同步数据错误", zap.String("syncKey", r.syncKey), zap.String("key", key), zap.Error(err))
			}
			if err != nil {
				// 同步或删除失败：把 field 重新入队（now+1 防止本轮再次扫到）
				if err1 := r.putWaitSyncRetry(ctx, k.Field, now+1); err1 != nil {
					log.ErrorCtx(ctx, "[Redis 同步] 移动待同步数据错误", zap.String("syncKey", r.syncKey), zap.String("key", key), zap.Error(err1))
					return
				}
			}
			total++
		}
		if int64(len(arr)) < count {
			// 本轮完整扫完才写心跳；出错提前返回不写，
			// 让别的实例能在心跳过期后接管这个分区
			r.writeHeartbeat(ctx, pt)
			log.InfoCtx(ctx, "[Redis 同步] 同步完成", zap.String("syncKey", r.syncKey), zap.String("key", key), zap.Int("total", total), zap.Duration("elapsed", time.Since(start)))
			return
		}
	}
}

// ==================== 无主分区兜底 ====================
//
// 分区归属依赖一致性哈希环，而环视图依赖服务发现的推/拉通知。
// 幽灵节点（实例死了但注册中心还没剔除）、通知丢失、视图分歧
// 都会造成某个分区被所有活着的实例认为"归别人管"，从而静默无主。
// 这里用与哈希环完全解耦的机制兜底：
//   - 每个分区扫描完成后写心跳 key（TTL = ts*hbTTLFactor）；
//   - 每轮同步结束后检查全部分区的心跳，缺失即视为无主，
//     用 SETNX 抢占锁后立即代扫一轮，并打 ERROR 日志供告警。
// 代扫是安全的：doSync 由业务方保证幂等（现有调用方均有分布式锁 +
// 差异对比），zset 删除走 version 校验，双扫最多浪费一次空转。

// hbKeyOf 分区心跳 key
func (r *RedisSyncJob) hbKeyOf(pt string) string {
	return r.syncKey + ":hb:" + pt
}

// takeoverKeyOf 无主接管锁 key
func (r *RedisSyncJob) takeoverKeyOf(pt string) string {
	return r.syncKey + ":takeover:" + pt
}

// writeHeartbeat 写分区心跳，失败仅告警（下一轮会重写）
func (r *RedisSyncJob) writeHeartbeat(ctx context.Context, pt string) {
	ttl := time.Duration(r.ts*hbTTLFactor) * time.Second
	if err := r.universalClient.Client().Set(ctx, r.hbKeyOf(pt), env.GetId(), ttl).Err(); err != nil {
		log.WarnCtx(ctx, "[Redis 同步] 分区心跳写入失败",
			zap.String("syncKey", r.syncKey), zap.String("pt", pt), zap.Error(err))
	}
}

// checkOrphanPartitions 检查并接管无主分区
func (r *RedisSyncJob) checkOrphanPartitions(ctx context.Context, start time.Time, now int64) {
	// 冷启动静默期：全量分区心跳尚未建立，跳过以免整批误报。
	// 静默期 = 心跳 TTL；滚动发布下老实例仍在写心跳，不受影响
	if time.Since(r.createdAt) < time.Duration(r.ts*hbTTLFactor)*time.Second {
		return
	}
	cli := r.universalClient.Client()
	pts := make([]string, r.queueNum)
	cmds := make([]*redis.IntCmd, r.queueNum)
	_, err := cli.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for i := range r.queueNum {
			pts[i] = strconv.FormatInt(i, 10)
			cmds[i] = pipe.Exists(ctx, r.hbKeyOf(pts[i]))
		}
		return nil
	})
	if err != nil {
		// 部分命令可能已成功，继续按单条结果判断；整体错误仅告警
		log.WarnCtx(ctx, "[Redis 同步] 分区心跳检查部分失败",
			zap.String("syncKey", r.syncKey), zap.Error(err))
	}
	for i, cmd := range cmds {
		// 单条出错不能当作"心跳缺失"，否则网络抖动会引发接管风暴
		if cmd.Err() != nil || cmd.Val() > 0 {
			continue
		}
		pt := pts[i]
		// 抢占接管锁：同一无主分区每个周期只允许一个实例代扫
		ok, err := cli.SetNX(ctx, r.takeoverKeyOf(pt), env.GetId(), time.Duration(r.ts)*time.Second).Result()
		if err != nil {
			log.WarnCtx(ctx, "[Redis 同步] 无主分区抢占失败",
				zap.String("syncKey", r.syncKey), zap.String("pt", pt), zap.Error(err))
			continue
		}
		if !ok {
			continue // 已被其他实例接管
		}
		// ERROR 级：分区无主属于必须告警的异常，正常情况不应出现
		log.ErrorCtx(ctx, "[Redis 同步] partition orphan detected, taking over",
			zap.String("syncKey", r.syncKey),
			zap.String("pt", pt),
			zap.String("takenBy", env.GetId()))
		r.syncPartition(ctx, pt, start, now)
	}
}

func (r *RedisSyncJob) safeCall(ctx context.Context, key, arg string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			log.ErrorCtx(ctx, "[Redis 同步] doSync panic", zap.Any("err", e), zap.String("key", key), zap.String("arg", arg))
			err = fmt.Errorf("doSync panic: %v", e)
		}
	}()
	return r.doSync(ctx, arg)
}

// 业务调用的写入路径：
// versionKey 不存在(空闲/排队中) → ZADD NX 保留首次入队时间
// versionKey 存在(正在被同步) → ZADD 覆盖 + 让本轮 rem 版本不匹配，留待下轮
func (r *RedisSyncJob) putWaitSync(ctx context.Context, field string, timestamp int64) error {
	return r.putWaitSyncWithMode(ctx, field, timestamp, syncAddModeNormal)
}

// 失败重试路径：score
//
//	用 GT，确保推到 now+1（下一轮 cron）
func (r *RedisSyncJob) putWaitSyncRetry(ctx context.Context, field string, timestamp int64) error {
	return r.putWaitSyncWithMode(ctx, field, timestamp, syncAddModeRetry)
}

// 保存到待同步队列
func (r *RedisSyncJob) putWaitSyncWithMode(ctx context.Context, field string, timestamp, mode int64) error {
	pt := hashMod(field, r.queueNum)
	key := r.syncKey + ":" + strconv.FormatInt(pt, 10)
	versionKey := versionKeyOf(ctx, key)
	return r.syncAddLua.Run(ctx, r.universalClient.Client(),
		[]string{key, versionKey},
		strconv.FormatInt(timestamp, 10),
		field,
		strconv.FormatInt(mode, 10),
	).Err()
}

// 兼容旧的 method 写法（测试中使用），转调包级实现
func (r *RedisSyncJob) hashMod(s string, mod int64) int64 {
	return hashMod(s, mod)
}

// versionKeyOf 返回 zset key 对应的 version hash key，用 hash tag {key} 让
// zset 与 version hash 落在同一 redis slot，保证 Lua 多 key 在集群下可用。
//
// 多租户前缀注入（key_prefix_hook_v3.go）对整串 key 做的是无脑前置拼接，
// 不识别 {}：裸 key 加前缀后整串参与哈希（前缀天然被算入），而 {key}:version
// 这种写法如果 tag 内仍是没加前缀的原始 key，前缀会被拼在 {} 外面，
// Redis 只用 {} 内的内容算哈希，导致两者哈希输入不一致，集群下报 CROSSSLOT。
//
// 这里预先用 hook 内部同一套前缀计算逻辑（namespace.FmtByTypeOnCtx）算出
// key 加前缀后的样子，塞进 {} 里；hook 之后再对整串做一次无脑前置拼接，
// 两次前缀叠加后 {} 内的内容和 key 最终的哈希输入就完全一致了。
// 不改 hook、不改 key 本身的命名（兼容线上已有数据），只改 versionKey 的构造方式；
// 无租户或兜底租户模式下 FmtByTypeOnCtx 原样返回 key，行为与改动前完全一致。
//
// 约束：key 不应自带 {}（hash tag）。若 key 已带 {}，本函数会产生嵌套 {}，
// Redis 的 tag 提取规则不支持嵌套，key 与 versionKey 会解析出不一致的 tag，
// 这是调用方传入了不合规 SyncKey 导致的既有问题，不在本函数内做兜底兼容。
func versionKeyOf(ctx context.Context, key string) string {
	tagged := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, key)
	return "{" + tagged + "}:version"
}

// hashMod 对字符串做 FNV-1a 32bit 哈希后取模。
// 内联实现避免每次调用 fnv.New32a 与 []byte(s) 的分配（PutWaitSyncKey 是业务高频路径）。
func hashMod(s string, mod int64) int64 {
	const (
		offset32 uint32 = 2166136261
		prime32  uint32 = 16777619
	)
	h := offset32
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime32
	}
	return int64(h) % mod
}

// PutSyncKeys 批量添加待同步数据（语义与逐条 PutSyncKey 完全一致，仅合并网络往返）
func (r *RedisSyncMgr) PutSyncKeys(ctx context.Context, syncKey string, fields []string, timestamp int64) error {
	taskKey := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_REDIS, syncKey)
	val, ok := r.syncTask.Load(taskKey)
	if !ok {
		return errors.New("syncKey not exist")
	}
	return val.(*RedisSyncJob).putWaitSyncs(ctx, fields, timestamp)
}

// putWaitSyncs 批量写入待同步队列：pipeline 中逐 field 执行与 putWaitSync 相同的 sync_add.lua，
// 每个 field 独立按 hash 取模分区、独立走 NX/覆盖与 versionKey 判定，语义与逐条调用完全一致，
// 仅把 N 次网络往返合并为一次（集群模式下 go-redis pipeline 自动按 slot 路由，脚本双 key 经 hash tag 同 slot）。
// pipeline 失败（如新实例脚本未缓存 NOSCRIPT、网络抖动）降级逐条重放：
// Script.Run 自带 EvalSha→Eval 兜底，且 sync_add.lua 同参重放幂等，部分成功后重放无副作用
func (r *RedisSyncJob) putWaitSyncs(ctx context.Context, fields []string, timestamp int64) error {
	if len(fields) == 0 {
		return nil
	}
	if len(fields) == 1 {
		return r.putWaitSync(ctx, fields[0], timestamp)
	}
	score := strconv.FormatInt(timestamp, 10)
	mode := strconv.FormatInt(syncAddModeNormal, 10)
	_, err := r.universalClient.Client().Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, field := range fields {
			key := r.syncKey + ":" + strconv.FormatInt(hashMod(field, r.queueNum), 10)
			r.syncAddLua.Run(ctx, pipe, []string{key, versionKeyOf(ctx, key)}, score, field, mode)
		}
		return nil
	})
	if err == nil {
		return nil
	}
	log.WarnCtx(ctx, "[Redis 同步] 批量写入待同步队列失败，降级逐条重放",
		zap.String("syncKey", r.syncKey), zap.Int("count", len(fields)), zap.Error(err))
	// 尽力送达：单条失败不拖累其余 field，返回首个失败供调用方告警
	var firstErr error
	for _, field := range fields {
		if err := r.putWaitSync(ctx, field, timestamp); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
