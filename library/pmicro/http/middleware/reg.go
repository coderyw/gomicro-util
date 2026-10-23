// Package middleware
// @Author: yinwei
// @File: reg
// @Version: 1.0.0
// @Date: 2024/9/26 14:37

package middleware

import "github.com/gin-gonic/gin"

const (
	RESP_HANDLER = "_RespHandler"
)

// 获取默认的http中间件
func GetDefaultHttpMiddle() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		RESP_HANDLER: DefaultResp(),
	}
}
