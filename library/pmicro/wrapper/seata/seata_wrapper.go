// Package seata
// @Author: yinwei
// @File: seata_wrapper
// @Version: 1.0.0
// @Date: 2024/7/26 13:09

package seata

import (
	"context"

	"github.com/seata/seata-go/pkg/constant"
	"github.com/seata/seata-go/pkg/tm"
	"github.com/coderyw/gomicro-util/library/log"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/registry"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

// 处理seata事物
func HandlerWrapper() server.HandlerWrapper {
	return func(handlerFunc server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			xid, ok := metadata.Get(ctx, constant.XidKey)
			if !ok {
				xid, _ = metadata.Get(ctx, constant.XidKeyLowercase)
			}

			if xid != "" {
				log.Info("Seata", zap.String("XID", xid))
				ctx = tm.InitSeataContext(ctx)
				tm.SetXID(ctx, xid)
			}

			err := handlerFunc(ctx, req, rsp)

			return err
		}
	}
}

// NewCallWrapper 在call后执行，处理错误问题
func CallWrapper() client.CallWrapper {
	return func(callFunc client.CallFunc) client.CallFunc {
		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
			// set the XID when intercepting a client request and release it directly when intercepting a response
			if tm.IsSeataContext(ctx) {
				xid := tm.GetXID(ctx)
				ctx = metadata.Set(ctx, constant.XidKey, xid)
				log.Info("Seata", zap.String("XID", xid))
			}
			err := callFunc(ctx, node, req, rsp, opts)
			return err
		}
	}
}
