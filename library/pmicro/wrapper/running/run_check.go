// Package running
// @Author: yinwei
// @File: run_check
// @Version: 1.0.0
// @Date: 2024/12/9 20:30

package running

import (
	"context"
	error_code "github.com/coderyw/gomicro-util/model/error-code"
	"github.com/coderyw/gomicro-util/library/pmicro/wait_stop"
	"go-micro.dev/v4/server"
)

// 校验服务是否运行中

// RunCheckHandlerWrapper 检查服务是否已经停止运行
//
// 通过server.HandlerWrapper 可以在服务处理之前进行校验，如果服务已经停止运行，则直接返回错误信息
//
// Returns
//   - HandlerWrapper: 返回 server.HandlerWrapper 中间件
func RunCheckHandlerWrapper() server.HandlerWrapper {
	return func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if wait_stop.IsStop() {
				return error_code.INTERNAL_SERVER_ERROR.NewMsg("服务已停止")
			}
			wait_stop.Add()
			defer wait_stop.Done()
			return handlerFunc(ctx, req, rsp)
		}
	}
}
