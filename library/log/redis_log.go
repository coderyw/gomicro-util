// Package log
// @Author: yinwei
// @File: redis_log
// @Version: 1.0.0
// @Date: 2024/11/30 16:54

package log

import (
	"context"
	"fmt"
	"go.uber.org/zap"
)

type RedisLog struct {
	Logger *zap.Logger
}

func (r *RedisLog) Printf(ctx context.Context, format string, v ...interface{}) {
	InfoCtxLogger(ctx, r.Logger, fmt.Sprintf(format, v...))
}

func NewRedisLog() *RedisLog {

	zl := InitByDefaultLogger(zap.AddCallerSkip(6))
	return &RedisLog{Logger: zl}
}
