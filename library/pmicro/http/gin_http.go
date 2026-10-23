// Package http
// @Author: yinwei
// @File: gin_http
// @Version: 1.0.0
// @Date: 2024/8/14 15:49

package http

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/http/middleware"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
	"github.com/coderyw/gomicro-util/library/single"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
)

var defaultMiddlewares = []string{
	middleware.RESP_HANDLER,
}

// 处理http路由配置
func HttpServer(cfg srv_config.HttpCfg, op option.Options) (*http.Server, error) {
	var httpServer *http.Server
	if cfg.Host != "" {
		gh := gin.New()
		if env.IsRelease() || env.IsPre() {
			// 开启 release 模式
			gin.SetMode(gin.ReleaseMode)
		}
		// 配置使用官方 context
		gh.ContextWithFallback = true
		// recovery
		gh.Use(middleware.Recovery())
		// 请求指标统计
		gh.Use(middleware.Metrics())
		// trace 链路
		gh.Use(middleware.OriginalTracer())
		// 请求头处理
		gh.Use(middleware.Header())
		// 自定义日志
		gh.Use(middleware.CustomLogger(cfg.LogHeaders))
		// 跨域
		gh.Use(Cors(cfg.CorsHeaders)) //
		// namespace处理
		gh.Use(middleware.ParseNamespace())

		// 返回值请求头处理
		gh.Use(middleware.RespHeader())

		// 注册路由
		if op.RegisterHttpHandler != nil {
			if op.HttpHandlerDefaultResp { //是否使用默认的消息返回操作
				gh.Use(middleware.DefaultResp())
			}
			op.RegisterHttpHandler(gh)
		}

		// 探活接口(兼容没改动的配置)
		gh.GET("/api/liveness", ServiceAliveHandler)

		if cfg.EnablePrometheus {
			gh.GET("/metrics", gin.WrapH(promhttp.Handler()))
			pprof.Register(gh)
		}
		// http server 配置
		httpServer = &http.Server{
			Addr:              cfg.Host,
			Handler:           gh.Handler(),
			ReadTimeout:       time.Duration(cfg.HttpServerCfg.ReadTimeoutS) * time.Second,
			ReadHeaderTimeout: time.Duration(cfg.HttpServerCfg.ReadHeaderTimeoutS) * time.Second,
			WriteTimeout:      time.Duration(cfg.HttpServerCfg.WriteTimeoutS) * time.Second,
			IdleTimeout:       time.Duration(cfg.HttpServerCfg.IdleTimeoutS) * time.Second,
			MaxHeaderBytes:    cfg.HttpServerCfg.MaxHeaderBytes,
		}
		log.Info("启动http服务.....", zap.String("host", cfg.Host))
		ln, err := net.Listen("tcp", httpServer.Addr)
		if err != nil {
			return nil, err
		}
		go func() {
			if err := httpServer.Serve(ln); err != nil {
				if errors.Is(err, http.ErrServerClosed) {
					log.Info("停止 HTTP Server")
				} else {
					log.Error("Serve", zap.Error(err))
				}
			}
		}()
	}

	return httpServer, nil
}

// 主要用于探活相关接口
func LivenessServer(cfg srv_config.LivenessCfg) (*http.Server, error) {
	if cfg.Host == "" {
		return nil, nil
	}
	gh := gin.New()
	if env.IsRelease() || env.IsPre() {
		// 开启 release 模式
		gin.SetMode(gin.ReleaseMode)
	}
	auth := func(gctx *gin.Context) {
		if cfg.Secret == "" {
			gctx.Next()
			return
		}
		str := gctx.GetHeader("Authorization")
		if str == cfg.Secret {
			gctx.Next()
		} else {
			gctx.AbortWithStatus(http.StatusUnauthorized)
		}
	}
	// 探活接口
	gh.GET("/api/liveness", ServiceAliveHandler)
	// 就绪接口
	gh.GET("/api/readiness", ServiceReadiness)
	// 通知服务不需要就绪 准备服务开始下线
	gh.POST("/api/set/unreadiness", auth, SetUnReadiness)
	// 通知服务设置就绪. 可用于恢复
	gh.POST("/api/set/readiness", auth, SetReadiness)
	// http server 配置
	httpServer := &http.Server{
		Addr:    cfg.Host,
		Handler: gh.Handler(),
	}
	log.Info("启动http探活服务.....", zap.String("host", cfg.Host))
	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		return nil, err
	}
	go func() {
		if err := httpServer.Serve(ln); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("停止 HTTP探活 Server")
			} else {
				log.Error("Serve", zap.Error(err))
			}
		}
	}()
	return httpServer, nil
}

func SetUnReadiness(gctx *gin.Context) {
	log.Info("收到取消就绪消息。服务开始准备下线")
	single.SetReadinessProbe(false)
	gctx.Status(http.StatusOK)
}

func SetReadiness(gctx *gin.Context) {
	log.Info("收到设置就绪消息")
	single.SetReadinessProbe(true)
	gctx.Status(http.StatusOK)
}

// ServiceAliveHandler
//
//	@Description: 运维使用 探活接口
//	@Author yw 2024-08-14 15:53:44
//	@param gctx
func ServiceAliveHandler(gctx *gin.Context) {
	gctx.JSON(http.StatusOK, gin.H{})
}

func ServiceReadiness(gctx *gin.Context) {
	if !single.GetReadinessProbe() {
		gctx.Status(http.StatusServiceUnavailable)
		gctx.Abort()
		return
	}
	gctx.Status(http.StatusOK)
}

// Cors 处理跨域请求
func Cors(headersList []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			// 只设置一次 Access-Control-Allow-Origin
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Max-Age", "172800")
			c.Header("Access-Control-Allow-Credentials", "true")

			// 使用 strings.Builder 提高性能
			defaultHeaders := []string{
				"Authorization", "Content-Length", "X-CSRF-Token", "Token", "session",
				"X_Requested_With", "Accept", "Origin", "Host", "Connection",
				"Accept-Encoding", "Accept-Language", "DNT", "X-CustomHeader",
				"Keep-Alive", "User-Agent", "X-Requested-With", "If-Modified-Since",
				"Cache-Control", "Content-Type", "Pragma", "lang", "Front", "terminal",
				"roadshow", "grayscale",
			}

			// 合并默认 headers 和传入的 headersList
			allHeaders := append(defaultHeaders, headersList...)
			c.Header("Access-Control-Allow-Headers", strings.Join(allHeaders, ","))
			c.Header("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers,Cache-Control,Content-Language,Content-Type,Expires,Last-Modified,Pragma,FooBar")
		}

		// 处理 OPTIONS 请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
