// Package log 基于 zap 的统一日志门面。
//
// pmicro 启动时按 log 配置调用 InitDefaultLogger 初始化。带 Ctx 后缀的方法会从 ctx 中取出 traceId、
// 用户、租户等元数据一并输出，业务代码应优先使用：
//
//	log.InfoCtx(ctx, "order created", zap.Int64("order_id", id))
//	log.ErrorCtx(ctx, "pay failed", zap.Error(err))
//
// 包内同时提供 gorm、redis、nacos、cron、go-micro 等三方库的日志适配。
//
// 文档：docs/components/observability.md
package log
