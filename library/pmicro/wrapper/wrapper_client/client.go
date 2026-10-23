package wrapper_client

import (
	"context"
	"fmt"

	trace2 "github.com/apache/skywalking-go/toolkit/trace"
	"github.com/coderyw/gomicro-util/library/log"
	config2 "github.com/coderyw/gomicro-util/library/pmicro/config"
	"github.com/coderyw/gomicro-util/library/trace"
	"go-micro.dev/v4/client"
	"go-micro.dev/v4/metadata"
	"go.uber.org/zap"
)

func ClientWrapper(cfg *config2.ServerCfg) client.Wrapper {
	return func(c client.Client) client.Client {
		return &wrapperClient{
			Client: c,
			cfg:    cfg,
		}
	}
}

type wrapperClient struct {
	client.Client
	cfg *config2.ServerCfg
}

func (c *wrapperClient) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	ctx = c.wrap(ctx, fmt.Sprintf("%v.%v", req.Service(), req.Endpoint()), req.Service())
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c *wrapperClient) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	ctx = c.wrap(ctx, fmt.Sprintf("%v.%v", req.Service(), req.Endpoint()), req.Service())
	return c.Client.Stream(ctx, req, opts...)
}

func (c *wrapperClient) Publish(ctx context.Context, p client.Message, opts ...client.PublishOption) error {
	ctx = c.wrap(ctx, p.Topic(), "")
	return c.Client.Publish(ctx, p, opts...)
}

func (c *wrapperClient) wrap(ctx context.Context, endpoint, srv string) context.Context {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		md = make(metadata.Metadata)
	}
	c.injectTraceHeader(ctx, endpoint, srv, md)
	c.makeAuth(ctx, endpoint, md)
	return metadata.NewContext(ctx, md)
}

// 注入 sw8 到 context metadata
func (c *wrapperClient) injectTraceHeader(ctx context.Context, endpoint, srv string, md map[string]string) {
	traceId := ""
	traceIdSky := ""
	trace2.CreateExitSpan(endpoint, srv, func(headerKey, headerValue string) error {
		log.DebugCtx(ctx, "skywalking inject traceId", zap.String(headerKey, headerValue))
		if headerKey == "sw8" {
			traceIdSky = headerValue
		}
		return nil
	})
	trace2.SetComponent(5008)
	defer trace2.StopSpan()

	traceId, _ = trace.MustGetTraceId(ctx)
	md[trace.METADATA_TRACE_ID] = traceId
	md[trace.METADATA_TRACE_ID_SKY] = traceIdSky
	return
}

func (c *wrapperClient) makeAuth(ctx context.Context, endpoint string, md map[string]string) {
	auc := c.cfg.MatchAuthClient(endpoint)
	if auc == nil {
		return
	}
	switch auc.AuthType {
	case config2.RPC_AUTH_TYPE_BASIC, config2.RPC_AUTH_TYPE_JWT:
		md[config2.RPC_METADATA_X_AUTH_TYPE] = auc.AuthType
		md[config2.RPC_METADATA_X_AUTH_VERSION] = auc.AuthVersion
		md[config2.RPC_METADATA_X_AUTH_SECRET] = auc.AuthSecret
	default:
		log.WarnCtx(ctx, "auth type 暂时不支持", zap.String("authType", auc.AuthType))
	}
	return

}
