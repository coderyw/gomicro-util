// Package metadata
// @Author: yinwei
// @File: md
// @Version: 1.0.0
// @Date: 2023/11/20 14:34

package metadata

import (
	"context"

	"github.com/coderyw/gomicro-util/library/log"
	ctx2 "github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/pmicro/config"
	"github.com/coderyw/gomicro-util/util"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

// 请求数据前 存放metadata
func NewCallWrapper() client.CallWrapper {
	return func(callFunc client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			val := ctx2.GetMicroMetadata(ctx)
			if val != nil {
				ctx = metadata.MergeContext(ctx, val, true)
				log.InfoCtx(ctx, "[GRPC] Call metadata", zap.Any("metadata", val))
			}
			return callFunc(ctx, node, req, rsp, opts)
		}
	}
}
func NewHandlerWrapper() server.HandlerWrapper {
	return func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			md, ok := metadata.FromContext(ctx)
			// 对部分元数据脱敏
			md = mask(md)
			log.InfoCtx(ctx, "[GRPC] Handler metadata", zap.Any("metadata", md))
			if ok {
				var err error
				ctx, err = ctx2.SetFromMicroMetadata(ctx, md)
				if err != nil {
					log.ErrorCtx(ctx, "[GRPC] metadata set error", zap.Any("metadata", md), zap.Error(err))
					return err
				}
			}

			return handlerFunc(ctx, req, rsp)
		}
	}
}

func mask(md metadata.Metadata) metadata.Metadata {
	if val, ok := md[config.RPC_METADATA_X_AUTH_SECRET_MICRO]; ok {
		md[config.RPC_METADATA_X_AUTH_SECRET_MICRO] = util.Mask(val, 2, 3)
	}
	return md
}
