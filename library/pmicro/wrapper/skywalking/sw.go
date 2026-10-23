// Package skywalking
// @Author: yinwei
// @File: sw
// @Version: 1.0.0
// @Date: 2024/7/16 09:35

package skywalking

import (
	"context"
	"fmt"
	"github.com/apache/skywalking-go/toolkit/trace"
	"github.com/coderyw/gomicro-util/library/log"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/server"
	"go.uber.org/zap"
)

//
//func SwClientWrapper() client.CallWrapper {
//	return func(callFunc client.CallFunc) client.CallFunc {
//		return func(ctx context.Context, node *registry.Node, req client.Request, rsp interface{}, opts client.CallOptions) error {
//
//			trace.CreateExitSpan(req.Endpoint(), node.Address, func(headerKey, headerValue string) error {
//				ctx = metadata.Set(ctx, headerKey, headerValue)
//				return nil
//			})
//			trace.SetComponent(5008)
//			defer trace.StopSpan()
//			return callFunc(ctx, node, req, rsp, opts)
//		}
//	}
//}

func SwClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &skyClient{c}
	}
}

func SwHandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			md, _ := metadata.FromContext(ctx)
			log.Info("metadata", zap.Any("m", md))
			trace.CreateEntrySpan(req.Endpoint(), func(headerKey string) (string, error) {
				val, bo := metadata.Get(ctx, "Sw8")
				if !bo {
					log.Warn("缺失traceId", zap.String("headerKey", headerKey))
					return "", fmt.Errorf("miss traceId")
				}
				return val, nil
			})
			trace.SetComponent(5009)
			defer trace.StopSpan()
			return fn(ctx, req, rsp)
		}
	}
}

type skyClient struct {
	client.Client
}

func (c *skyClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	var sp *trace.SpanRef
	ctx, sp = injectTraceHeader(ctx, req.Endpoint(), req.Service())
	err := c.Client.Call(ctx, req, rsp, opts...)
	if err != nil && sp != nil {
		//sp.AddEvent(trace.ErrorEventType, err.Error())
	}
	return err
}

func (c *skyClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	var sp *trace.SpanRef
	ctx, sp = injectTraceHeader(ctx, req.Endpoint(), req.Service())
	st, err := c.Client.Stream(ctx, req, opts...)
	if err != nil && sp != nil {
		//sp.AddEvent(trace.ErrorEventType, err.Error())
	}
	return st, err
}

func (c *skyClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	var sp *trace.SpanRef
	ctx, sp = injectTraceHeader(ctx, p.Topic(), "")
	err := c.Client.Publish(ctx, p, opts...)
	if err != nil && sp != nil {
		//sp.AddEvent(trace.ErrorEventType, err.Error())
	}
	return err
}

// 注入 sw8 到 context metadata
func injectTraceHeader(ctx context.Context, endpoint, srv string) (context.Context, *trace.SpanRef) {

	sp, err := trace.CreateExitSpan(endpoint, srv, func(headerKey, headerValue string) error {
		ctx = metadata.Set(ctx, headerKey, headerValue)
		return nil
	})
	if err != nil {
		log.WarnCtx(ctx, "[GRPC] Create ExitSpan error", zap.Error(err))
	}
	trace.SetComponent(5008)

	defer trace.StopSpan()
	return ctx, sp
}
