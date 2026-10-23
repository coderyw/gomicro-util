// Package log
// @Author: yinwei
// @File: ctx_log
// @Version: 1.0.0
// @Date: 2025/6/10 17:15

package log

import (
	"context"
	"fmt"
	"math"
	"time"

	ctx2 "github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getZap(ctx context.Context) []zap.Field {
	var fields []zap.Field
	val, spanId := trace.MustGetTraceId(ctx)
	if len(val) != 0 {
		fields = append(fields, zap.String(trace.TRACE_ID_KEY, val), zap.Uint64(trace.SPAN_ID_KEY, spanId))
	}
	return append(fields, ctx2.GetLogMd(ctx)...)
}

func getFmt(ctx context.Context) []string {
	zaps := getZap(ctx)
	arr := make([]string, 0, len(zaps))
	for _, f := range zaps {
		arr = append(arr, zapFmt(f))
	}
	return arr
}

func zapFmt(f zap.Field) string {
	var val interface{}
	switch f.Type {
	case zapcore.ArrayMarshalerType:
		val = f.Interface
	case zapcore.ObjectMarshalerType:
		val = f.Interface
	case zapcore.InlineMarshalerType:
		val = f.Interface
	case zapcore.BinaryType:
		val = f.Interface
	case zapcore.BoolType:
		val = f.Integer
	case zapcore.ByteStringType:
		val = f.Interface
	case zapcore.Complex128Type:
		val = f.Interface
	case zapcore.Complex64Type:
		val = f.Interface
	case zapcore.DurationType:
		val = time.Duration(f.Integer).String()
	case zapcore.Float64Type:
		val = math.Float64frombits(uint64(f.Integer))
	case zapcore.Float32Type:
		val = math.Float32frombits(uint32(f.Integer))
	case zapcore.Int64Type:
		val = f.Integer
	case zapcore.Int32Type:
		val = int32(f.Integer)
	case zapcore.Int16Type:
		val = int16(f.Integer)
	case zapcore.Int8Type:
		val = int8(f.Integer)
	case zapcore.StringType:
		val = f.String
	case zapcore.TimeType:
		if f.Interface != nil {
			val = time.Unix(0, f.Integer).In(f.Interface.(*time.Location))
		} else {
			val = time.Unix(0, f.Integer)
		}
	case zapcore.TimeFullType:
		val = f.Interface.(time.Time)
	case zapcore.Uint64Type:
		val = uint64(f.Integer)
	case zapcore.Uint32Type:
		val = uint32(f.Integer)
	case zapcore.Uint16Type:
		val = uint16(f.Integer)
	case zapcore.Uint8Type:
		val = uint8(f.Integer)
	case zapcore.UintptrType:
		val = uintptr(f.Integer)
	case zapcore.ReflectType:
		val = f.Interface
	case zapcore.NamespaceType:
		val = f.Key
	case zapcore.StringerType:
		val = f.Interface
	case zapcore.ErrorType:
		val = f.Interface
	case zapcore.SkipType:
		break
	default:
	}
	return fmt.Sprintf("%v:%v", f.Key, val)

}
