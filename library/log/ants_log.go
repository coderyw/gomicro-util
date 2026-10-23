// Package log
// @Author: yinwei
// @File: ants_log
// @Version: 1.0.0
// @Date: 2024/8/23 13:22

package log

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
)

type antsLog struct {
}

func (c antsLog) Printf(format string, args ...interface{}) {
	defaultLogger.Info(fmt.Sprintf(format, args...))
}

func GetAntsLogger() ants.Logger {
	return &antsLog{}
}
