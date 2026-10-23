package orchestrion

import (
	"context"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	"github.com/coderyw/gomicro-util/model/apm"
)

func Init() {

	tracer.WithDebugStack(false)
}

// GetSpanTraceId 获取 orchestrion 的 traceId 和 spanId
//
// Parameters:
//   - ctx context.Context
//
// Returns：
//   - string traceId
//   - uint64 spanId
func GetSpanTraceId(ctx context.Context) (string, uint64) {
	if span, ok := tracer.SpanFromContext(ctx); ok {
		return span.Context().TraceID(), span.Context().SpanID()
	}
	return "", 0
}

// StartSpan 开启span
//
// Parameters:
//   - ctx context.Context
//   - name string
//   - tags ...apm.ApmTag
//
// Returns：
//   - context.Context
func StartSpan(ctx context.Context, name string, tags ...apm.ApmTag) context.Context {
	_, ctx = tracer.StartSpanFromContext(ctx, name, func(cfg *tracer.StartSpanConfig) {
		mp := make(map[string]interface{})
		for _, tag := range tags {
			mp[tag.Key] = tag.Value
		}
		cfg.Tags = mp
	})
	return ctx
}

func FinishSpan(ctx context.Context) {
	if span, ok := tracer.SpanFromContext(ctx); ok {
		span.Finish()
	}
}
