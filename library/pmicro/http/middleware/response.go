// Package middleware
// @Author: yinwei
// @File: response
// @Version: 1.0.0
// @Date: 2024/9/26 10:50

package middleware

import (
	"github.com/gin-gonic/gin"
	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"github.com/coderyw/gomicro-util/library/log"
	ctx_cst "github.com/coderyw/gomicro-util/model/cst/ctx"
	httpresp "github.com/coderyw/gomicro-util/model/http"
	"go.uber.org/zap"
)

var defaultResp = httpresp.NsHttpResp{}

// 默认消息回复
func DefaultResp() gin.HandlerFunc {
	logger = log.GetGinLog()
	return func(gc *gin.Context) {
		gc.Next()

		// 判断是否使用默认返回
		if _, ok := gc.Get(ctx_cst.RESP_KEY_NO_DEFAULT); ok {
			return
		}

		resp := defaultResp
		var status error_code.StatusCode
		var code int
		// 获取处理错误码
		errI, ok := gc.Get(ctx_cst.RESP_KEY_ERROR)
		if !ok {
			status = error_code.SUCCESS
			code = 0
			data, ok := gc.Get(ctx_cst.RESP_KEY_DATA)
			if ok {
				resp.Data = data
			}
		} else {
			var err error
			status, err = error_code.Unwrap(errI.(error))
			if err != nil {
				resp.Detail = status.GetMessage() //对http-resp隐藏真实信息
			} else {
				resp.Detail = status.GetMessage()
			}
			code = int(status.Code())
		}
		resp.Code = uint32(code)
		resp.Msg = status.GetMessage()
		gc.JSON(status.HttpCode(), resp)
	}
}

// RespHeader 处理消息返回头
//
// Returns:
//   - gin.HandlerFunc
func RespHeader() gin.HandlerFunc {

	return func(c *gin.Context) {
		c.Next()
		// 获取是否有 header
		val, ok := c.Get(ctx_cst.RESP_KEY_HEADER)
		if !ok {
			return
		}
		if m, ok := val.(map[string]string); ok {
			for k, v := range m {
				c.Header(k, v)
			}
		} else {
			log.WarnCtx(c, "错误的返回值 header 类型", zap.Any("respHeader", val))
		}
	}
}
