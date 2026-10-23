package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/metadata"
)

const tenantHeader = "X-Tenant-Name"

func ParseNamespace() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		hd := ctx.GetHeader(tenantHeader)
		if hd != "" {
			metadata.SetCtxData(ctx, metadata.CONTEXT_NAME_SPACE, hd)
		}
	}
}
