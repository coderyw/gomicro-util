package log

import "context"

type contextKey string

const (
	// noResponseLogKey 是用于在上下文中标记是否应抑制响应日志的键。
	noResponseLogKey contextKey = "no_response_log"
)

// WithNoResponseLog 返回一个新的上下文，其中包含一个标志，
// 用于指示日志记录器不应记录 gRPC 调用的响应体。
// 当调用方预知响应会非常大时，可以在发起 gRPC 调用前使用此函数。
//
// 示例:
//
//	client.GetHallGameList(log.WithNoResponseLog(ctx), req)
func WithNoResponseLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, noResponseLogKey, true)
}

// ShouldLogResponse 检查上下文，判断是否应该记录响应体。
// 如果上下文包含了由 WithNoResponseLog 设置的标志，则返回 false。
// 默认情况下返回 true。
func ShouldLogResponse(ctx context.Context) bool {
	val := ctx.Value(noResponseLogKey)
	if b, ok := val.(bool); ok {
		return !b
	}
	return true
}
