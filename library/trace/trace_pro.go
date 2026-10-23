// Package trace
// @Author: yinwei
// @File: trace_pro
// @Version: 1.0.0
// @Date: 2024/11/28 09:22

package trace

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/coderyw/gomicro-util/internal/apm/orchestrion"
)

// 字符串的 traceidkey
const CtxTraceIdKey = "trace_id_key"

const (
	// traceID 头，兼容前端可能传不同的头
	HTTP_TRACE_ID_1 = "X-Trace-Id"
	HTTP_TRACE_ID_2 = "Trace-Id"
	HTTP_TRACE_ID_3 = "Traceid"

	// ctx的traceId key
	TRACE_ID_KEY = "traceID"
	SPAN_ID_KEY  = "spanID"

	// 微服务 元数据传递过程中 traceId
	METADATA_TRACE_ID     = "Traceid"
	METADATA_TRACE_ID_SKY = "Sw8"
	// traceId 类型，default，skywalking,orchestrion
	TRACE_TYPE_ENV         = "TRACE_TYPE"
	TRACE_TYPE_DEFAULT     = "default"
	TRACE_TYPE_SKYWALKING  = "skywalking"
	TRACE_TYPE_ORCHESTRION = "orchestrion"
)

var traceType string // trace 使用类型，默认 0 业务自定义，1 skywalking

func init() {
	//初始化 trace
	t := os.Getenv(TRACE_TYPE_ENV)
	switch t {
	case TRACE_TYPE_DEFAULT:
		traceType = t
	case TRACE_TYPE_SKYWALKING:
		traceType = t
	case TRACE_TYPE_ORCHESTRION:
		traceType = t
	default:
		traceType = "default"
	}
}

// GetTraceCtx 通过传入的父 context 获取带有 traceId 的 context
//
// 说明：如果 cxt 为空，则会自定义生成一个
//   - 如果 ctx 中有 traceId，则直接返回
//   - 如果 ctx 中没有 traceId，则会从 skywalking 中获取，如果 skywalking 也没有，则会生成 uuid，然后放入  ctx 中
//
// Parameters:
//   - ctx: context.Context 上下文
//
// Returns:
//   - context.Context 返回带有 traceId 的上下文
func GetTraceCtx(ctx context.Context, traceId ...string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	var id string
	if traceType == TRACE_TYPE_ORCHESTRION {
		id, _ = orchestrion.GetSpanTraceId(ctx)
	}
	if id != "" {
		return ctx
	}
	val := ctx.Value(CtxTraceIdKey)
	if val == nil {
		if len(traceId) == 0 || len(traceId[0]) == 0 {
			id = uuid.NewString()
		} else {
			id = traceId[0]
		}
		ctx = context.WithValue(ctx, CtxTraceIdKey, id)
	}

	return ctx
}

// MustGetTraceId 从 context 中获取 traceId
//
//	如果 context 中没有，则会从 skywalking 中获取，如果 skywalking 也没有，则会生成 uuid
//
// Parameters:
//   - ctx: context.Context 上下文
//
// Returns:
//   - string traceId
func MustGetTraceId(ctx context.Context) (string, uint64) {
	if ctx == nil {
		return uuid.NewString(), 0
	}
	var (
		traceId string
		spanId  uint64
	)
	if traceType == TRACE_TYPE_ORCHESTRION {
		traceId, spanId = orchestrion.GetSpanTraceId(ctx)
	}
	if traceId == "" { //没有从 APM 中获取到，使用默认的
		var id string
		ti := ctx.Value(CtxTraceIdKey)
		if ti != nil {
			if val, ok := ti.(string); ok {
				id = val
			}
		}
		if len(id) == 0 {
			id = uuid.NewString()
		}
		return id, 0
	}
	return traceId, spanId
}

// 从上下文中获取 traceId
func GetTraceId(ctx context.Context) (string, uint64) {
	if ctx == nil {
		return "", 0
	}
	var (
		traceId string
		spanId  uint64
	)
	if traceType == TRACE_TYPE_ORCHESTRION {
		traceId, spanId = orchestrion.GetSpanTraceId(ctx)
	}
	if traceId == "" {
		var id string
		ti := ctx.Value(CtxTraceIdKey)
		if ti != nil {
			if val, ok := ti.(string); ok {
				id = val
			}
		}
		return id, 0
	}
	return traceId, spanId
}

// SetTraceId 为context 设置 traceId
//
// Parameters:
//   - ctx: context.Context 上下文，可为空
//   - traceId: string traceId
//
// Returns:
//   - context.Context 返回新的上下文
func SetTraceId(ctx context.Context, traceId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, CtxTraceIdKey, traceId)
}

// NewOrAddTraceId 添加一个新的或则拼接 traceId
//
// - 如果当前 context 中没有  traceId，则会使用 newOrAddTraceId 当做 traceId
// - 如果当前 context 中有 traceId，则会使用 context 中的 traceId:newOrAddTraceId 作为新的 traceId
//
// Parameters:
//   - ctx: context.Context 上下文 可为空
//   - newOrAddTraceId: string 要添加的或则新的  traceId
//
// Returns:
//   - context.Context 返回新的 context
func NewOrAddTraceId(ctx context.Context, newOrAddTraceId string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	parentId, _ := GetTraceId(ctx)
	if len(parentId) == 0 {
		parentId = newOrAddTraceId
	} else {
		parentId = fmt.Sprintf("%v:%v", parentId, newOrAddTraceId)
	}
	return SetTraceId(ctx, parentId)
}

func NewOrAddTraceIdCtx(ctx context.Context, addCtx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	parentId, _ := GetTraceId(ctx)
	newOrAddTraceId, _ := GetTraceId(addCtx)
	if len(parentId) == 0 {
		if len(newOrAddTraceId) == 0 {
			return ctx
		}
		return SetTraceId(ctx, newOrAddTraceId)
	}

	if len(newOrAddTraceId) == 0 {
		return ctx
	}
	parentId = fmt.Sprintf("%v:%v", parentId, newOrAddTraceId)
	return SetTraceId(ctx, parentId)
}
