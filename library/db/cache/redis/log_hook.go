// Package redis
// @Author: yinwei
// @File: log_hook
// @Version: 1.0.0
// @Date: 2024/11/30 17:08

package redis

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/cst"
	"go.uber.org/zap"
)

type logHook struct {
	logger   *zap.Logger
	instance string
	debug    bool
}

func (l *logHook) DialHook(next redis.DialHook) redis.DialHook {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if !l.isLogEnabled(ctx) {
			return next(ctx, network, addr)
		}
		return next(ctx, network, addr)
	}
}

func (l *logHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !l.isLogEnabled(ctx) {
			return next(ctx, cmd)
		}

		start := time.Now()
		err := next(ctx, cmd)
		if strings.HasPrefix(cmd.String(), "expire eonId:") {
			return nil
		}
		duration := time.Since(start)
		log.InfoCtxLogger(ctx, l.logger, "[Redis] Cmd", zap.String("instance", l.instance),
			zap.String("Cmd", cmd.String()),
			zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetRedisHandleLevel(duration)))
		return err
	}
}

func (l *logHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !l.isLogEnabled(ctx) {
			return next(ctx, cmds)
		}
		start := time.Now()
		err := next(ctx, cmds)
		duration := time.Since(start)
		for _, cmd := range cmds {
			log.InfoCtxLogger(ctx, l.logger, "[Redis] Cmd Pipeline", zap.String("instance", l.instance), zap.String("Cmd", cmd.String()))
		}
		log.InfoCtxLogger(ctx, l.logger, "[Redis] Cmd Pipeline", zap.String("instance", l.instance), zap.Duration("Duration", duration), zap.String(log.DUR_LOG_KEY, cst.GetRedisHandleLevel(duration)))

		return err
	}
}

type ctxKey struct {
}

// WithRedisLog 开启 redis log
//
//	在同 context 链路下会都开启日志
//
// Parameters:
//   - ctx: context.Context 上下文
//
// Returns:
//   - context.Context 返回新的上下文
func WithRedisLog(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, true)
}

// 是否开启日志
func (l *logHook) isLogEnabled(ctx context.Context) bool {
	if l.debug {
		return true
	}
	k := ctx.Value(ctxKey{})
	if k == nil {
		return false
	}
	val, ok := k.(bool)
	if !ok {
		return false
	}
	return val
}
