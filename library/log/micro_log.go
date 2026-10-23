// Package log
// @Author: yinwei
// @File: micro_log
// @Version: 1.0.0
// @Date: 2024/6/28 14:04

package log

import (
	"fmt"
	logger2 "go-micro.dev/v4/logger"
)

func NewMicroLog() *microLog {
	return new(microLog)
}

type microLog struct {
	opts logger2.Options
}

func (m microLog) Init(options ...logger2.Option) error {
	return nil
}

func (m microLog) Options() logger2.Options {
	return m.opts
}

func (m microLog) Fields(fields map[string]interface{}) logger2.Logger {
	return m
}

func (m microLog) Log(level logger2.Level, v ...interface{}) {
	msg := fmt.Sprint(v...)

	switch level {
	case logger2.DebugLevel:
		defaultLogger.Debug(msg)
	case logger2.ErrorLevel:
		defaultLogger.Error(msg)
	case logger2.InfoLevel:
		defaultLogger.Info(msg)
	case logger2.FatalLevel:
		defaultLogger.Fatal(msg)
	default:
		fmt.Println(msg)
	}
}

func (m microLog) Logf(level logger2.Level, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)

	switch level {
	case logger2.DebugLevel:
		defaultLogger.Debug(msg)
	case logger2.ErrorLevel:
		defaultLogger.Error(msg)
	case logger2.InfoLevel:
		defaultLogger.Info(msg)
	case logger2.FatalLevel:
		defaultLogger.Fatal(msg)
	default:
		fmt.Println(msg)
	}
}

func (m microLog) String() string {
	return defaultLogger.Name()
}
