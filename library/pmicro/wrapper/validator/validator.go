/**
 * @Author: yw
 * @Description: 检验插件。主要校验实现了接口Validator的服务.
 * @File: validator
 * @Version: 1.0.0
 * @Date: 2022/1/26 4:07 下午
 */
package validator

import (
	"context"
	commonResult_proto "github.com/coderyw/gomicro-util/model/common-result"
	"github.com/coderyw/gomicro-util/model/facade"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/errors"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
)

type Validator interface {
	Validate() error
}

// 在handler端进行校验
func NewHandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if v, ok := req.Body().(Validator); ok {
				if err := v.Validate(); err != nil {
					if val, ok := rsp.(facade.CommRespFacade); ok {
						val.SetErrorCode_(commonResult_proto.ERROR_CODE_INVALID_REQUEST)
						val.SetErrorMsg_(commonResult_proto.ERROR_CODE_INVALID_REQUEST.StringAlias())
						val.SetExtraMessage_(err.Error())
						return nil
					} else {
						return errors.BadRequest(req.Service(), "%v", err)
					}
				}
			}
			return fn(ctx, req, rsp)
		}
	}
}

// 在client call时校验
func NewCallWrapper() client.CallWrapper {
	return func(callFunc client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {

			if v, ok := req.Body().(Validator); ok {
				if err := v.Validate(); err != nil {
					if val, ok := rsp.(facade.CommRespFacade); ok {
						val.SetErrorCode_(commonResult_proto.ERROR_CODE_INVALID_REQUEST)
						val.SetErrorMsg_(commonResult_proto.ERROR_CODE_INVALID_REQUEST.StringAlias())
						val.SetExtraMessage_(err.Error())
						return nil
					} else {
						return errors.BadRequest(req.Service(), "%v", err)
					}
				}
			}
			return callFunc(ctx, node, req, rsp, opts)
		}
	}
}
