// Package log
// @Author: yinwei
// @File: gogoproto_log
// @Version: 1.0.0
// @Date: 2024/7/15 13:40

package log

import (
	"fmt"
	"github.com/coderyw/protobuf/log"
	"go.uber.org/zap"
)

type gogoProtoLog struct {
	logger *zap.Logger
}

func (g *gogoProtoLog) Info(msg string, args ...interface{}) {
	g.logger.Error(fmt.Sprintf(msg, args...))
}

func (g *gogoProtoLog) Fatalf(msg string, args ...interface{}) {
	g.logger.Fatal(fmt.Sprintf(msg, args...))
}

func (g *gogoProtoLog) Error(msg string, err error) {
	g.logger.Error(msg, zap.Error(err))
}

func setGogoLog(logger *zap.Logger) {
	logger = logger.WithOptions(zap.AddCaller(), zap.AddCallerSkip(2))

	log.WithLogger(&gogoProtoLog{logger: logger})
}
