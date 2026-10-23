// Package middleware
// @Author: yinwei
// @File: log
// @Version: 1.0.0
// @Date: 2024/9/5 20:36

package middleware

import (
	"bytes"
	"io"
	"net/textproto"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/model/cst"
	ctx_cst "github.com/coderyw/gomicro-util/model/cst/ctx"
	"go.uber.org/zap"
)

var logger *zap.Logger

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
}

// CustomLogger 记录 HTTP 请求的中间件
//
// logHeaders 为需要打印的 header 配置：不配置则不打印 header；配置 * 打印所有 header；
// 否则按 , 拼接需要打印的 header 名，如 X-Request-Id,User-Agent
func CustomLogger(logHeaders ...string) gin.HandlerFunc {
	logger = log.GetGinLog()
	var (
		currentRequests int64
		//maxBodySize     = 1024 // 最大打印 body 长度
	)

	// 启动时解析一次 header 打印配置
	var headerCfg string
	if len(logHeaders) > 0 {
		headerCfg = logHeaders[0]
	}
	printAllHeaders, allowHeaders := parseLogHeaders(headerCfg)

	return func(c *gin.Context) {
		// 去掉部分不打印接口日志
		if !canNext(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		active := atomic.AddInt64(&currentRequests, 1)
		defer atomic.AddInt64(&currentRequests, -1)

		// 包装 ResponseWriter 以便捕获响应
		//blw := &bodyLogWriter{body: bytes.NewBuffer(nil), ResponseWriter: c.Writer}
		//c.Writer = blw

		// 读取并恢复请求 body（文件上传不打印）
		//body := readRequestBody(c, maxBodySize)

		// 请求开始日志
		startFields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("ip", metadata.Get[string](c, metadata.CONTEXT_REQ_IP)),
			zap.String("path", path),
			zap.Int64("current_requests", active),
		}
		if headers := collectLogHeaders(c, printAllHeaders, allowHeaders); len(headers) > 0 {
			startFields = append(startFields, zap.Any("headers", headers))
		}
		log.InfoCtx(c, "HTTP Handle start", startFields...)

		// 执行请求
		c.Next()

		// 请求结束日志
		duration := time.Since(start)
		//response := readResponseBody(c, blw, maxBodySize)

		log.InfoCtx(c, "HTTP Handle end",
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", duration),
			zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)),
			zap.Int("size", c.Writer.Size()),
		)
	}
}

// parseLogHeaders 解析 header 打印配置
//
// 配置为 * 时打印所有 header；否则按 , 拆分出可打印的 header 名；不配置则不打印
func parseLogHeaders(cfg string) (printAll bool, allowHeaders []string) {
	cfg = strings.TrimSpace(cfg)
	if cfg == "" {
		return false, nil
	}
	if cfg == "*" {
		return true, nil
	}
	for h := range strings.SplitSeq(cfg, ",") {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		allowHeaders = append(allowHeaders, textproto.CanonicalMIMEHeaderKey(h))
	}
	return false, allowHeaders
}

// collectLogHeaders 根据配置收集需要打印的 header
func collectLogHeaders(c *gin.Context, printAll bool, allowHeaders []string) map[string][]string {
	if printAll {
		return c.Request.Header
	}
	if len(allowHeaders) == 0 {
		return nil
	}
	headers := make(map[string][]string, len(allowHeaders))
	for _, h := range allowHeaders {
		if values := c.Request.Header.Values(h); len(values) > 0 {
			headers[h] = values
		}
	}
	return headers
}

// readRequestBody 读取并恢复请求 body
func readRequestBody(c *gin.Context, maxSize int) string {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return "[file upload or binary data]"
	}
	if c.Request.Body == nil {
		return ""
	}
	if c.Request.ContentLength > int64(maxSize*10) { // 大于10倍限制，直接跳过
		return "[large body skipped]"
	}

	old := c.Request.Body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.ErrorCtx(c, "Error reading request body", zap.Error(err))
		return "[error reading body]"
	}
	defer func() {
		old.Close()
	}()
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	if len(bodyBytes) > maxSize {
		return string(bodyBytes[:maxSize]) + "... (truncated)"
	}
	return string(bodyBytes)
}

// readResponseBody 读取响应内容
func readResponseBody(c *gin.Context, blw *bodyLogWriter, maxSize int) string {
	// 1. 如果服务端主动要求不打印响应 body
	if data, ok := c.Get(ctx_cst.RESP_NO_RESPONSE_LOG); ok {
		if skip, _ := data.(bool); skip {
			return "ignore response body by server"
		}
	}

	// 2. 如果响应是文件下载（根据 Content-Type / Content-Disposition 判断）
	ct := c.Writer.Header().Get("Content-Type")
	cd := c.Writer.Header().Get("Content-Disposition")
	if strings.Contains(cd, "attachment") ||
		strings.Contains(ct, "application/octet-stream") ||
		strings.Contains(ct, "application/zip") {
		return "[file download or binary data]"
	}

	// 3. 正常响应内容
	respBytes := blw.body.Bytes()
	if len(respBytes) > maxSize {
		return string(respBytes[:maxSize]) + "... (truncated)"
	}
	return string(respBytes)
}
func canNext(path string) bool {
	if strings.HasPrefix(path, "/api/liveness") {
		return false
	}
	if strings.HasPrefix(path, "/metrics") {
		return false
	}
	return true
}
