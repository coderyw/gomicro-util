// Package ctx
// @Author: yinwei
// @File: gin_ctx
// @Version: 1.0.0
// @Date: 2025/6/23 09:08

package metadata

import (
	"context"

	"github.com/gin-gonic/gin"
)

// setGinCtxData 设置 metadata 到 gin
//
//	仅仅支持 Metadata 对应的 metadata
//
// Parameters:
//   - ctx: *gin.Context
//   - key: string
//   - value: interface{}
func setGinCtxData(ctx *gin.Context, key string, value interface{}) context.Context {
	val := ctx.Value(_contextMetadataStr)
	if val == nil {
		val = &Metadata{}
	}
	md := val.(*Metadata)
	set(md, key, value)
	ctx.Set(_contextMetadataStr, md)
	return ctx
}
