// Package redis Redis 多实例封装：带前缀命令、分布式锁、缓存读穿与分布式定时同步。
//
// pmicro 启动时按 storage.redis 配置调用 InitRedis 自动初始化。多租户下 key 会按 ctx 中的租户自动加前缀，
// 因此调用时务必传入请求链路上的 ctx。
//
// 使用方式：
//
//	// 默认实例 / 指定实例
//	v, err := redis.GetRedis(ctx).GetValue(ctx, prefixenum.PrefixEnumCaPay, "key")
//	v, err = redis.GetRedis(ctx, "activity").GetValue(ctx, prefixenum.PrefixEnumCaPay, "key")
//
//	// 等待抢锁 + 自动续约 + 执行完自动解锁
//	p := redis.NewLockParam(prefixenum.PrefixEnumCaBalance, "user:1001")
//	ok, err := redis.GetRedis(ctx).LockWaitTimeOutWithRenewalV2(ctx, p, func() error {
//		return doBusiness(ctx)
//	})
//
//	// 缓存读穿（防击穿）
//	u, err := redis.GetObjCacheT[*User](ctx, redis.GetRedis(ctx), "user:1001", loadUser, 10*time.Minute)
//
// GetRedis 返回 facade.RedisInterface，便于依赖注入与 mock。key 前缀请使用 prefixenum 中的常量。
//
// 文档：docs/components/redis.md
package redis
