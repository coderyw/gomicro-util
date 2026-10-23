package jaeger

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	plog "log"
)

const SpanContext = "SpanContext"

var ginTracer opentracing.Tracer

/**
 * @Description: jaeger的gin中间件.与micro微服务一起使用时，直接使用global的tracer
 * @param jaegerOpen
 * @return gin.HandlerFunc
 */
func JaegerMiddlewareWithMicro(serverName, host string, jaegerOpen int) gin.HandlerFunc {
	tracer, _, err := newTracer(serverName, host)
	if err != nil {
		plog.Println(fmt.Sprintf("start tracer err =%v", err.Error()))
		tracer = opentracing.GlobalTracer()
	}
	ginTracer = tracer
	return func(c *gin.Context) {
		f(c, jaegerOpen)
		c.Next()
	}
}

var f = func(c *gin.Context, jaegerOpen int) {
	if jaegerOpen == 1 {
		var (
			parentSpan OpentracingSpan
			parentCtx  context.Context
		)
		tracer := ginTracer
		spCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(c.Request.Header))
		if err != nil {
			parentSpan, parentCtx = StartRootSpan(context.Background(), tracer, c.Request.URL.Path, "")
			defer parentSpan.Finish()
		} else {
			parentSpan, parentCtx = StartRootSpan(
				context.Background(), tracer, c.Request.URL.Path, "",
				opentracing.ChildOf(spCtx),
				opentracing.Tag{Key: string(ext.Component), Value: "HTTP"},
				ext.SpanKindRPCServer,
			)
			defer parentSpan.Finish()
		}
		parentSpan.LogFields(log.Object("data", c.Request))

		c.Set("Tracer", tracer)
		c.Set(SpanContext, parentCtx)
	}
}

/**
 * @Description: 判断context类型是否是gin.Context。如果是，则从中取出对应的span context,否则就返回本身
 * @param c
 * @return context.Context
 */
func GetContext(c context.Context) context.Context {
	if c == nil {
		return context.TODO()
	}
	if gc, ok := c.(*gin.Context); ok {
		if v, ok := gc.Get(SpanContext); ok {
			return v.(context.Context)
		}
		return context.Background()
	}
	return c
}
