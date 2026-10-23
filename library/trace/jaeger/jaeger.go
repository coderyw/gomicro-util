package jaeger

import (
	"context"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
	jaegerConfig "github.com/uber/jaeger-client-go/config"
	"io"
	"runtime"
	"strings"
	"sync"
	"time"
)

var globalTracer = new(sync.Map)
var tracerHost string

//	func init() {
//		tracerHost = config.GetTracerHostConfig()
//	}
type RootSpanId struct{}
type NotTraceKey struct{}

// 用于判断ctx是否包含了链路数据
type TraceCtxKey struct{}

const span_Context_Key = "spanContextKey"
const ROOT_SPAN_TAG = "root_span_tag"

var noTrace = newSpan(nil)
var NotTraceCtx = context.WithValue(context.Background(), NotTraceKey{}, noTrace)
var TraceCtx = context.WithValue(context.Background(), TraceCtxKey{}, TraceCtxKey{})

/**
 * @Description: 新建一个tracer
 * @param serverName 服务名
 * @param host agent地址
 * @return opentracing.Tracer
 * @return io.Closer
 * @return error-code
 */
func NewTracer(serverName, host string) (opentracing.Tracer, io.Closer, error) {
	//jaeger
	if tracerHost == "" {
		tracerHost = host
	}
	return newTracer(serverName, tracerHost)

}

func newTracer(serverName, host string) (opentracing.Tracer, io.Closer, error) {
	//jaeger
	cfg := jaegerConfig.Configuration{
		ServiceName: serverName, //自定义服务名称
		Sampler: &jaegerConfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegerConfig.ReporterConfig{
			LogSpans:            true,
			BufferFlushInterval: 1 * time.Second,
			//LocalAgentHostPort:  host, //jaeger agent
		},
	}
	sender, err := jaeger.NewUDPTransport(host, 0)
	if err != nil {
		return nil, nil, err
	}
	reporter := jaeger.NewRemoteReporter(sender)
	// Initialize tracer with a logger and a metrics factory
	tracer, closer, err := cfg.NewTracer(
		jaegerConfig.Reporter(reporter),
	)
	return tracer, closer, err
}

// 用来代替空closer
type CloserToDo struct {
}

func (c *CloserToDo) Close() error {
	return nil
}

/**
 * @Description: 从父级ctx，创建一个span。其中span的名称，是调用这个函数的函数的名称。
 * @param ctx	父级context
 * @return opentracing.Span
 * @return context.Context
 */
func StartChildSpanFromContextWithoutName(ctx context.Context) (OpentracingSpan, context.Context) {
	if ctx == nil || ctx.Value(TraceCtxKey{}) == nil { //如果ctx 是空，或者ctx包含了这值，则表明不需要走trace
		return noTrace, ctx
	}
	ctx = GetContext(ctx)
	return StartChildSpanFromContext(ctx, GetFuncName())
}

/*
*
  - @Description: 从ctx中获取span，如果span为空或者创建span的tracer为空，则走global的tracer，
    否则使用创建span的tracer进行创建新的span,和上一个span 的关系为父子
  - @param name	span名称
  - @param ctx	上下文
  - @return opentracing.Span
  - @return context.Context
*/
func StartChildSpanFromContext(ctx context.Context, name string) (OpentracingSpan, context.Context) {
	if ctx == nil || ctx.Value(TraceCtxKey{}) == nil { //如果ctx 是空，或者ctx包含了这值，则表明不需要走trace
		return noTrace, ctx
	}
	ctx = GetContext(ctx)
	parentSpan := opentracing.SpanFromContext(ctx)
	var tracer opentracing.Tracer
	if parentSpan == nil {
		tracer = opentracing.GlobalTracer()
	} else {
		tracer = parentSpan.Tracer()
	}
	span, ccc := opentracing.StartSpanFromContextWithTracer(ctx, tracer, name)
	return newSpan(span), ccc
}

func StartChildSpanFromContextWithTrace(ctx context.Context, name string, tracer opentracing.Tracer) (OpentracingSpan, context.Context) {
	if ctx == nil || ctx.Value(TraceCtxKey{}) == nil { //如果ctx 是空，或者ctx包含了这值，则表明不需要走trace
		return noTrace, ctx
	}
	ctx = GetContext(ctx)
	parentSpan := opentracing.SpanFromContext(ctx)
	var tc opentracing.Tracer
	if parentSpan != nil {
		tc = parentSpan.Tracer()
	}
	if tc == nil {
		tc = tracer
	}
	if tc == nil {
		tc = opentracing.GlobalTracer()
	}
	span, ccc := opentracing.StartSpanFromContextWithTracer(ctx, tracer, name)
	return newSpan(span), ccc
}

/**
 * @Description: 获取一个root span
 * @param tracer tracer必填
 * @param name   span名称
 * @param traceId traceId 可选
 * @return opentracing.Span
 * @return context.Context
 */
func StartRootSpan(ctx context.Context, tracer opentracing.Tracer, name, traceId string, opts ...opentracing.StartSpanOption) (OpentracingSpan, context.Context) {
	if ctx == nil || ctx.Value(TraceCtxKey{}) == nil { //如果ctx 是空，或者ctx包含了这值，则表明不需要走trace
		return noTrace, ctx
	}
	ctx = GetContext(ctx)
	if traceId != "" {
		ctx = context.WithValue(ctx, RootSpanId{}, traceId)
	}
	span, resCtx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, name, opts...)
	if traceId != "" {
		span = span.SetTag(ROOT_SPAN_TAG, traceId)
	}
	return newSpan(span), resCtx
}

func StartRootSpanWithoutTracer(ctx context.Context, name, traceId string, opts ...opentracing.StartSpanOption) (OpentracingSpan, context.Context) {
	if ctx == nil || ctx.Value(TraceCtxKey{}) == nil { //如果ctx 是空，或者ctx包含了这值，则表明不需要走trace
		return noTrace, ctx
	}
	tracer := opentracing.GlobalTracer()
	ctx = GetContext(ctx)
	if traceId != "" {
		ctx = context.WithValue(ctx, RootSpanId{}, traceId)
	}
	span, resCtx := opentracing.StartSpanFromContextWithTracer(ctx, tracer, name, opts...)
	if traceId != "" {
		span = span.SetTag(ROOT_SPAN_TAG, traceId)
	}
	return newSpan(span), resCtx
}

/**
 * @Description: 从ctx中获取traceId。如果ctx中没有span，则返回空
 * @param ctx
 * @return string
 */
func GetTraceIdFromCtx(ctx context.Context) string {
	ctx = GetContext(ctx)
	if ctx != nil {
		s := ctx.Value(RootSpanId{})
		if res, ok := s.(string); ok {
			return res
		}
	}
	return ""
}

func SetTraceIdToCtx(ctx context.Context, t string) context.Context {
	ctx = GetContext(ctx)
	return context.WithValue(ctx, RootSpanId{}, t)
}

func GetFuncName() string {
	pc, _, _, ok := runtime.Caller(2)
	if ok {
		name := runtime.FuncForPC(pc).Name()
		index := strings.LastIndex(name, ".")
		name = name[index+1:]
		return name
	}
	return "GetFuncName"
}

/**
 * @Author yinwei
 * @Description //结束span
 * @Date 10:19 上午 2021/8/3
 * @Param
 * @return
 **/
func FinishWithError(span OpentracingSpan, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.LogFields(log.Error(err))
	}
	span.Finish()
}

func FinishWithMsg(span OpentracingSpan, k, msg string) {
	if span == nil {
		return
	}
	span.LogFields(log.String(k, msg))
	span.Finish()
}

func Finish(span OpentracingSpan) {
	if span == nil {
		return
	}
	span.Finish()
}
