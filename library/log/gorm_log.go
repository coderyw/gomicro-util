// Package log
// @Author: yinwei
// @File: gorm_log
// @Version: 1.0.0
// @Date: 2024/7/31 14:19

package log

import (
	"context"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/coderyw/gomicro-util/model/cst"
	"go.uber.org/zap"
	"gorm.io/gorm"
	logger2 "gorm.io/gorm/logger"
)

type gormLog struct {
	logLevel logger2.LogLevel
	logger   *zap.Logger
	driver   string
}

var getStatus func()

func NewGormLog(LogLevel logger2.LogLevel, driver string) *gormLog {
	if driver != "" {

		runes := []rune(driver)
		runes[0] = unicode.ToUpper(runes[0])
		for i := 1; i < len(runes); i++ {
			runes[i] = unicode.ToLower(runes[i])
		}
		driver = string(runes)
	}
	return &gormLog{
		logLevel: LogLevel,
		logger:   defaultLogger.WithOptions(zap.AddCallerSkip(2)),
		driver:   fmt.Sprintf("[%v] Exec", driver),
	}
}

func (g *gormLog) LogMode(level logger2.LogLevel) logger2.Interface {
	g.logLevel = level
	return g
}

func (g *gormLog) Info(ctx context.Context, s string, i ...interface{}) {
	if g.logLevel >= logger2.Info {

		g.logger.Info(fmt.Sprintf(s, i...), getZap(ctx)...)
	}
}

func (g *gormLog) Warn(ctx context.Context, s string, i ...interface{}) {
	if g.logLevel >= logger2.Warn {
		g.logger.Warn(fmt.Sprintf(s, i...), getZap(ctx)...)
	}
}

func (g *gormLog) Error(ctx context.Context, s string, i ...interface{}) {
	if g.logLevel >= logger2.Error {
		g.logger.Error(fmt.Sprintf(s, i...), getZap(ctx)...)
	}
}

func (g *gormLog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if g.logLevel < logger2.Silent {
		return
	}
	sql, eff := fc()
	sub := time.Now().Sub(begin)
	var zaps []zap.Field = make([]zap.Field, 0, 10)
	zaps = append(zaps, zap.String("Duration", sub.String()), zap.String(DUR_LOG_KEY, cst.GetMysqlHandleLevel(sub)), zap.String("SQL", sql))
	zaps = append(zaps, getZap(ctx)...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zaps = append(zaps, zap.String("Reason", err.Error()))
			g.logger.Warn(g.driver, zaps...)
		} else {
			zaps = append(zaps, zap.Error(err))
			g.logger.Error(g.driver, zaps...)
		}
		return
	}
	zaps = append(zaps, zap.Int64("RowsAffected", eff))
	g.logger.Info(g.driver, zaps...)

}
