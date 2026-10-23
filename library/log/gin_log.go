// Package log
// @Author: yinwei
// @File: gin_log
// @Version: 1.0.0
// @Date: 2024/9/5 20:49

package log

import "go.uber.org/zap"

var ginLog *zap.Logger

func setGinLog(l *zap.Logger) {
	ginLog = l.WithOptions(zap.WithCaller(false))
}

func GetGinLog() *zap.Logger {
	return ginLog
}
