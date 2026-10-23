# db/cache：Redis 与本地缓存

> 返回 [README](../../README.md)。

`library/db/cache/redis` 提供 Redis 多实例、带前缀命令、分布式锁、缓存读穿（防击穿）与分布式定时同步。pmicro 启动时按 `storage.redis` 配置自动初始化。

## 配置

配置结构为 `config.RedisConfig`，完整带注释示例见 [configuration.md](../configuration.md) 的 `storage.redis` 段。

```yaml
storage:
  redis:
    enable: true
    default: 'core'
    pool_stats_mode: 1
    pool_stats_interval: 15
    instance:
      core:
        enable: true
        host:
          - '<host>:6379'
        password: ''
        db: 0
        pool_size: 100
        max_idle_conns: 20
        # Redis 为集群模式且没有 proxy 时开启；开启后强制 TLS（跳过证书校验），db 配置无效
        is_cluster: false
```

## 使用

```go
// 默认实例
v, err := redis.GetRedis(ctx).GetValue(ctx, prefixenum.PrefixEnumCaPay, "key")

// 指定实例
v, err = redis.GetRedis(ctx, "activity").GetValue(ctx, prefixenum.PrefixEnumCaPay, "key")
```

`GetRedis` 返回 `facade.RedisInterface` 接口（`*Redis` 的全量抽象），业务依赖注入和 mock 都对着它（mock 实现见仓库根 `mocks/`）。多租户下 key 会自动加租户前缀，规则见 [multi-tenant.md](../multi-tenant.md)。

## key 前缀约定（prefixenum）

所有 API 的 prefix 参数类型是 `prefixenum.PrefixEnum`，最终 key 为 `<prefix>:<key>`（prefix 为空时就是 key 本身），多租户下再由 hook 加租户前缀；按服务划分命名空间（如 `PrefixEnumCaPay = "ca-pay"`）。新服务接入需在 `prefixenum/prefix_enum.go` 加常量，不要在业务侧硬编码前缀字符串。

## 分布式锁（lock.go）

| 方法 | 语义 |
|---|---|
| `Lock` / `UnLock` | SETNX 一次性尝试，拿不到立即返回 false；UnLock 用 Lua 比对 value 防误删（必须传加锁时的同一个 v） |
| `LockExpectKey` | 可重入：已持有（value 相同）则续期返回 true |
| `LockWithTimeOut` | 自旋抢锁直到超时 |
| `LockWithRenewal` / `LockWaitTimeOutWithRenewalV2` / `LockUntilSuccWithRenewal` | 抢锁 + 自动续约 + 执行回调 + 自动解锁，三者分别对应「只试一次 / 等待超时 / 无限重试」 |
| `RenewalLock` | 对已持有的锁开后台续期协程，返回停止函数 |

```go
// 推荐：参数对象版（NewLockParam 默认 等待5s/重试间隔100ms/锁过期5s/续约间隔3s）
p := redis.NewLockParam(prefixenum.PrefixEnumCaBalance, "user:1001")
ok, err := redis.GetRedis(ctx).LockWaitTimeOutWithRenewalV2(ctx, p, func() error {
    return doBusiness(ctx) // 期间自动续约，返回后自动解锁
})
```

等待类方法的超时按「每次重试扣减重试间隔」计算，不是墙钟时间；手动构造 `LockParam` 时 `RetryInterval` 不能为 0，否则会无限空转。推荐用 `NewLockParam` 获取默认值。

自动续约锁的生命周期：请求 ctx 取消后仍会继续续约，直到 `do()` 返回或 panic，随后停止续约并带持有者校验解锁。业务应使用原 ctx 响应取消并最终退出；Redis 故障或续约失败仍可能导致锁过期，续约不是绝对互斥保证。详见 [upgrade/v3.0.0.0.md](../upgrade/v3.0.0.0.md#redis-锁生命周期与超时)。

## 缓存读穿 / 防击穿（cache.go）

一组包级泛型函数：缓存 miss 时抢 `<key>:lock` 分布式锁再回源，拿到锁后二次查缓存，防止并发击穿。**各函数抢不到锁时的行为不同**，选用时注意：

| 函数 | 抢不到锁时 |
|-|-|
| `GetIntCache` / `GetStrCache` / `GetZSetList` | 只尝试一次，**直接返回零值且 err 为 nil**，调用方无法区分"无数据"和"没抢到锁" |
| `GetObjCacheT` / `GetListCacheT` / `GetObjHMCache` | 每 10ms 重试，**无上限**；加锁本身报错时不加锁直接回源 |
| `GetHMCacheField` | 最多等待 5s |
| `GetFromCacheOrDb` | 最多等待 20s（完全自定义三段式） |

```go
u, err := redis.GetObjCacheT[*model.User](ctx, redis.GetRedis(ctx), "user:1001",
	func(ctx context.Context) (*model.User, error) { return dao.GetUser(ctx, 1001) },
	10*time.Minute)
```

## 分布式定时同步（sync 子包）

把「待同步的业务 field」按 hash 分片写进 N 个 zset（score=时间戳），集群内多实例通过[一致性哈希](scheduling.md#consistent_hash)各自认领分区，定时扫到期数据回调业务同步函数。典型场景：余额等热点数据的延迟合并落库。

```go
rd := redis.GetRedis(ctx)
// 服务启动时注册一次（必须在 InitRedis 之后；框架服务在 AfterFunc 里注册即可）
_ = rd.InitSyncJob(ctx, model.RedisSyncOption{
	SyncKey:  "user_balance_sync", // 任务标识
	Ts:       10,                  // 同步周期 s，默认 5，最小 4
	QueueNum: 16,                  // 分片数，默认 10
	DoSync:   func(ctx context.Context, field string) error { return flushToDB(ctx, field) },
})
// 业务写入待同步数据（单条 / 批量）
_ = rd.PutWaitSyncKey(ctx, "user_balance_sync", "1001", time.Now().Unix())
_ = rd.PutWaitSyncKeys(ctx, "user_balance_sync", []string{"1001", "1002"}, time.Now().Unix())
```

要点：定时任务带随机抖动避免多实例同扫；分区有心跳兜底，无主分区会被其他实例抢占补扫，因此 **`DoSync` 必须幂等**。流程图见 [sync/flow.md](../../library/db/cache/redis/sync/flow.md)。

## 连接池统计

`InitRedis` 返回一个统计定时任务（pmicro 会自动注册；不用 pmicro 时需自行注册），按 `pool_stats_interval`（默认 15s，-1 不开启）采集每个实例的连接池状态，输出方式由 `pool_stats_mode` 控制，取值见 [configuration.md · 统计输出模式](../configuration.md#统计输出模式pool_stats_mode--stats_mode)。日志在状态无变化时不重复打印。

上报指标（tag 为 `instance=<实例名>`）：`redis_pool_total_conns` / `redis_pool_idle_conns` / `redis_pool_busy_conns` / `redis_pool_hits` / `redis_pool_misses` / `redis_pool_timeouts` / `redis_pool_stale_conns`。

## 子包

| 子包 | 职责 |
|-|-|
| `redis/facade` | Redis 抽象接口，供依赖注入和 mock |
| `redis/model` | 锁参数、同步任务等模型 |
| `redis/prefixenum` | key 前缀枚举 |
| `redis/sync` | 分布式定时同步实现 |

## 本地缓存（local-cache）

`library/db/cache/local-cache` 是进程内缓存统一入口，默认基于 `internal/cache/go-cache`，并结合 ctx 中的租户（`metadata.CONTEXT_NAME_SPACE`）做 key 隔离。详见 [local-cache/README.md](../../library/db/cache/local-cache/README.md)。
