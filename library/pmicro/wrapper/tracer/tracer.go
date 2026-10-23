/**
 * @Author: dlan
 * @Description:
 * @File: micro
 * @Version: 1.0.0
 * @Date: 2022/6/28 5:11 下午
 */
package tracer

import (
	"context"

	trace2 "github.com/apache/skywalking-go/toolkit/trace"
	"github.com/coderyw/gomicro-util/library/trace"
	"go-micro.dev/v4/metadata"
	"go-micro.dev/v4/server"
)

//943334e65dfa11f09d48163c323ecd2b.2115.39639635709270409, 1-OTQzMzM0ZTY1ZGZhMTFmMDlkNDgxNjNjMzIzZWNkMmIuNzkwNDEuMzk2Mzk2MzU3MDk0MzAwMDM=-OTQzMzM0ZTY1ZGZhMTFmMDlkNDgxNjNjMzIzZWNkMmIuNzkwNDEuMzk2Mzk2MzU3MDk0MzAwMDQ=-0-bnMtdGVzdC5jYS1vcGVuLWFwaS50ZXN0-OTQzMzRmMGI1ZGZhMTFmMDlkNDgxNjNjMzIzZWNkMmJAMTkyLjE2OC41MS4yMDQ=-Y2EtYWN0aXZpdHkuU3BpbkFjdGl2aXR5SHR0cFNlcnZpY2UuR2V0U3BpbkFjdGl2aXR5Q29uZmln-Y2EtYWN0aXZpdHktc3ZjOjk4MDQ=

// NewHandlerWrapper accepts an opentracing Tracer and returns a Handler Wrapper
func NewHandlerWrapper() server.HandlerWrapper {
	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			//先获取 sw8
			sky, _ := metadata.Get(ctx, trace.METADATA_TRACE_ID_SKY)
			trace2.CreateEntrySpan(req.Endpoint(), func(headerKey string) (string, error) {
				return sky, nil
			})
			trace2.SetComponent(5009)
			defer trace2.StopSpan()
			val, _ := metadata.Get(ctx, trace.METADATA_TRACE_ID)
			ctx = trace.GetTraceCtx(ctx, val)
			return h(ctx, req, rsp)
		}
	}
}
