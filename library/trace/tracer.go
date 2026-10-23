package trace

import (
	redis2 "github.com/DataDog/dd-trace-go/contrib/redis/go-redis.v9/v2"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/internal/apm/orchestrion"
)

var traceIdHeaders = []string{
	HTTP_TRACE_ID_1,
	HTTP_TRACE_ID_2,
	HTTP_TRACE_ID_3,
}

// GetGinTraceId 从 Gin 中获取 traceId
//
// Parameters:
//   - ctx *gin.Context
//
// Returns：
//   - string
func GetGinTraceId(ctx *gin.Context) string {
	var traceId string
	switch traceType {
	case TRACE_TYPE_ORCHESTRION:
		traceId, _ = orchestrion.GetSpanTraceId(ctx.Request.Context())
	default:
		// 从 header 获取
		//获取 ttraceId
		for _, v := range traceIdHeaders {
			if traceId = ctx.GetHeader(v); traceId != "" {
				break
			}
		}
	}
	if traceId == "" { //为空时候走默认
		traceId = uuid.NewString()
	}
	return traceId
}

// RedisTrace 配置Redis APM
//
// Parameters:
//   - rd redis.UniversalClient
//
// Returns：
func RedisTrace(rd redis.UniversalClient) {
	if traceType == TRACE_TYPE_ORCHESTRION {
		redis2.WrapClient(rd, redis2.WithService("redis"))
	}
}
