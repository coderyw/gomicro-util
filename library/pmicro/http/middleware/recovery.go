package middleware

import (
	"github.com/gin-gonic/gin"
	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
)

func Recovery() gin.HandlerFunc {
	return func(context *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.ErrorWithStack("Panic", zap.Any("error", err))
				context.JSONP(200, gin.H{"code": error_code.INTERNAL_SERVER_ERROR.Code(), "msg": error_code.INTERNAL_SERVER_ERROR.GetMessage(), "detail": "Panic Recovery"})
				context.Abort()
				return
			}
		}()
		context.Next()
	}
}
