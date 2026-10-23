// Package gin_util
// @Author: yinwei
// @File: response
// @Version: 1.0.0
// @Date: 2025/5/13 08:48

package gin

import (
	"context"

	"github.com/gin-gonic/gin"
	ctx_cst "github.com/coderyw/gomicro-util/model/cst/ctx"
)

// SetError sets the error in the gin context.
//
// Parameters:
//   - c: *gin.Context 上下文
//   - err: error  错误信息
func SetError(c *gin.Context, err error) {
	c.Set(ctx_cst.RESP_KEY_ERROR, err)
}

// SetResponseWithNoResponseLog 设置响应值，且不打印响应值
//
// Parameters:
//   - c *gin.Context
//   - data any
//
// Returns：
func SetResponseWithNoResponseLog(c *gin.Context, data any) {
	SetResponse(c, data)
	c.Set(ctx_cst.RESP_NO_RESPONSE_LOG, true)
}

// SetResponse sets the response data in the gin context.
//
// Parameters:
//   - c: *gin.Context 上下文
//   - data: any 响应数据
func SetResponse(c *gin.Context, data any) {
	c.Set(ctx_cst.RESP_KEY_DATA, data)
}

// SetNoDefaultResponse 设置不使用默认的响应体结构
//
// Parameters:
//   - c: *gin.Context
func SetNoDefaultResponse(c *gin.Context) {
	c.Set(ctx_cst.RESP_KEY_NO_DEFAULT, true)
}

// SetRespHeader 向 context 中设置 header
//
// 对于 context 处理方式有两个情况
//   - 如果是 *gin.Context 则会调用 Set 方法处理
//   - 如果不是 *gin.Context 则会调用 context.WithValue 方法处理
//
// Parameters:
//   - ctx: context.Context 上下文
//   - header: map[string]string header 参数
//
// Returns:
//   - context.Context 返回 context 接口
func SetRespHeader(ctx context.Context, header map[string]string) context.Context {
	if gc, ok := ctx.(*gin.Context); ok {
		gc.Set(ctx_cst.RESP_KEY_HEADER, header)
		return gc
	} else {
		return context.WithValue(ctx, ctx_cst.RESP_KEY_HEADER, header)
	}
}
