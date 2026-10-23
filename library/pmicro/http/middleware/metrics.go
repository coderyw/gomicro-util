// Package middleware
// @Author: yinwei
// @File: metrics
// @Version: 1.0.0
// @Date: 2026/8/12 10:00

package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests processed, partitioned by method, path and status",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request time in seconds, partitioned by method and path",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	for _, collector := range []prometheus.Collector{httpRequestsTotal, httpRequestDuration} {
		if err := prometheus.DefaultRegisterer.Register(collector); err != nil {
			// 已注册则忽略（多次 import 场景）
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				panic(err)
			}
		}
	}
}

// Metrics HTTP 请求指标中间件：请求量（method/path/status）与耗时分布（method/path）。
// path 使用注册的路由模板（如 /api/user/:id）而非原始 URL，避免高基数问题；
// 未匹配到路由的请求统一归入 unmatched，探测类路径不统计。
func Metrics() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		path := gctx.FullPath()
		// 探活、指标抓取自身不统计
		if strings.HasPrefix(gctx.Request.URL.Path, "/metrics") || strings.HasPrefix(gctx.Request.URL.Path, "/api/liveness") {
			gctx.Next()
			return
		}
		if path == "" {
			// 未匹配路由（404），不能用原始 URL 当 label，会被恶意路径打爆基数
			path = "unmatched"
		}
		start := time.Now()
		gctx.Next()

		method := gctx.Request.Method
		httpRequestsTotal.WithLabelValues(method, path, strconv.Itoa(gctx.Writer.Status())).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
	}
}
