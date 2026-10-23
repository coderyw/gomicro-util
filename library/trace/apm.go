package trace

import (
	"context"

	"github.com/coderyw/gomicro-util/internal/apm/orchestrion"
	"github.com/coderyw/gomicro-util/model/apm"
)

// StartSpan 开启链路追踪 Span
//
//	如果ctx 存在 parent span则会创建一个 child span，否则会创建一个 root span
//
// Parameters:
//   - ctx context.Context 上下文
//   - name string span 名称
//   - tags ...apm.ApmTag span 标签
//
// Returns：
//   - context.Context	开启链路追踪后的上下文
func StartSpan(ctx context.Context, name string, tags ...apm.ApmTag) context.Context {
	switch traceType {
	case TRACE_TYPE_ORCHESTRION:
		return orchestrion.StartSpan(ctx, name, tags...)
	default:
		return ctx
	}
}

func FinishSpan(ctx context.Context) {
	switch traceType {
	case TRACE_TYPE_ORCHESTRION:
		orchestrion.FinishSpan(ctx)
	default:
	}
}
