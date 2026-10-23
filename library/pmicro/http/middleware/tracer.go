// Package middleware
// @Author: yinwei
// @File: tracer
// @Version: 1.0.0
// @Date: 2024/12/31 10:57

package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/trace"
)

func OriginalTracer() gin.HandlerFunc {

	return func(gc *gin.Context) {
		path := gc.Request.URL.Path
		method := gc.Request.Method
		if method == http.MethodOptions || strings.HasPrefix(path, "/api/liveness") || strings.HasPrefix(path, "/metrics") {
			gc.Next()
			return
		}
		traceId := trace.GetGinTraceId(gc)
		gc.Header(trace.HTTP_TRACE_ID_3, traceId)
		gc.Set(trace.CtxTraceIdKey, traceId)
		gc.Next()
	}
}
