// Package log
// @Author: yinwei
// @File: cron_log
// @Version: 1.0.0
// @Date: 2024/8/23 13:10

package log

import (
	"fmt"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

func GetCronLogger() cron.Logger {
	return new(cronFlogger)
}

type cronFlogger struct {
}

func (c cronFlogger) Info(msg string, keysAndValues ...interface{}) {
	// fixme info日志暂时不要
	//return
	var fields []zap.Field = make([]zap.Field, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)/2; i++ {
		fields[i] = zap.Any(fmt.Sprint(keysAndValues[i*2]), keysAndValues[i*2+1])
	}
	defaultLogger.Debug(msg, fields...)
}

func (c cronFlogger) Error(err error, msg string, keysAndValues ...interface{}) {
	var fields []zap.Field = make([]zap.Field, 1+len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues)/2; i++ {
		fields[i] = zap.Any(fmt.Sprint(keysAndValues[i*2]), keysAndValues[i*2+1])
	}
	fields[len(fields)-1] = zap.Error(err)
	defaultLogger.Error(msg, fields...)
}
