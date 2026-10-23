// Package cron 基于 robfig/cron（WithSeconds，6 段表达式：秒 分 时 日 月 周）的定时任务。
//
// 推荐通过 pmicro 的 option.Cron 注册任务，框架负责初始化与停机。不使用 pmicro 时直接调用 AddCron：
// 包初始化时已创建并启动全局 cron 实例，Init 重复调用无效果。
//
// 多租户定时任务使用 TenantCron：对当前部署下每个租户各执行一次，并以带租户的锁 key 保证集群内只执行一次。
//
//	tc := cron.NewTenantCron("bmp-activity", cron.WithLockFunc(redisLock))
//	option.Cron(model_cron.CronJob{
//		Name:    "warmup",
//		CronSpc: "0 */5 * * * ?",
//		DoJob:   tc.Wrap("warmup_ticket", func(ctx context.Context) { /* ctx 已带租户 */ }),
//	})
//
// 未传 WithLockFunc 时默认无锁执行。
//
// 文档：docs/components/scheduling.md
package cron
