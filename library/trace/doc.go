// Package trace traceId 生成与透传。
//
// traceId 保存在 context 中，经 gRPC metadata 双向透传；HTTP 只在入站读取 X-Trace-Id 等请求头，
// library/http 发出的请求不会自动携带。
//
//   - GetTraceCtx(ctx)：确保 ctx 携带 traceId，异步任务、定时任务入口必须调用
//   - GetTraceId(ctx) / MustGetTraceId(ctx)：读取 traceId
//   - SetTraceId / NewOrAddTraceId：设置或串联子任务 traceId
//
// 环境变量 TRACE_TYPE=orchestrion 时 traceId 取自 orchestrion span；skywalking 目前与 default 行为相同。
//
// 文档：docs/components/observability.md
package trace
