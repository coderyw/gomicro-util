// Package middleware
// @Author: yinwei
// @File: header
// @Version: 1.0.0
// @Date: 2024/9/5 13:21

package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/metadata"
)

var ipHeader = []string{
	// 已经与运维确认了CloudFlare 所传递的 ip 的请求头
	"True-Client-IP", "Cf-Connecting-Ip", "X-Forwarded-For", "X-Real-Ip",
}

func Header() gin.HandlerFunc {
	return func(gc *gin.Context) {
		if !canNext(gc.Request.URL.Path) {
			gc.Next()
			return
		}
		ipMake(gc)
		gc.Next()
	}
}

func ipMake(gc *gin.Context) {
	var ip string
	for _, hd := range ipHeader {
		if ip = gc.GetHeader(hd); ip != "" {
			break
		}
	}
	if ip == "" {
		//log.WarnCtx(gc, "Http Request Ip got failed")
		// 通过 gin 获取
		ip = gc.RemoteIP()
	}
	metadata.SetCtxData(gc, metadata.CONTEXT_REQ_IP, ip)
}
