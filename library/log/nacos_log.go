// Package log
// @Author: yinwei
// @File: nacos_log
// @Version: 1.0.0
// @Date: 2024/7/2 15:10

package log

import (
	"fmt"
	nacos_logger "github.com/nacos-group/nacos-sdk-go/v2/common/logger"
	"go.uber.org/zap"
	"sync"
)

func GetNacosLogger() nacos_logger.Logger {
	l := defaultLogger.WithOptions(zap.AddCallerSkip(1))
	return &nacosLog{
		logger: l,
	}

}

type nacosLog struct {
	logger  *zap.Logger
	infoStr string
	lock    sync.Mutex
}

func (n *nacosLog) Info(args ...interface{}) {
	var str string
	if len(args) == 1 {
		str = fmt.Sprint(args[0])
	} else if len(args) > 1 {
		str = fmt.Sprintf(fmt.Sprint(args[0]), args[1:]...)
	} else {
		return
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	fmt.Println(RedactString(n.infoStr))
	fmt.Println(RedactString(str))
	fmt.Println()
	if n.infoStr != str {
		n.infoStr = str
		n.logger.Info(str)
	}
}

func (n *nacosLog) Warn(args ...interface{}) {
	if len(args) == 1 {
		n.logger.Warn(fmt.Sprint(args[0]))
	} else {
		n.logger.Warn(fmt.Sprintf(fmt.Sprint(args[0]), args[1:]...))
	}
}

func (n *nacosLog) Error(args ...interface{}) {
	if len(args) == 1 {
		n.logger.Error(fmt.Sprint(args[0]))
	} else {
		n.logger.Error(fmt.Sprintf(fmt.Sprint(args[0]), args[1:]...))
	}
}

func (n *nacosLog) Debug(args ...interface{}) {
	if len(args) == 1 {
		n.logger.Debug(fmt.Sprint(args[0]))
	} else {
		n.logger.Debug(fmt.Sprintf(fmt.Sprint(args[0]), args[1:]...))
	}
}

func (n *nacosLog) Infof(ff string, args ...interface{}) {

	str := fmt.Sprintf(ff, args...)
	n.lock.Lock()
	defer n.lock.Unlock()
	if n.infoStr != str {
		n.infoStr = str
		n.logger.Info(str)
	}
}

func (n *nacosLog) Warnf(f string, args ...interface{}) {
	n.logger.Warn(fmt.Sprintf(f, args...))
}

func (n *nacosLog) Errorf(f string, args ...interface{}) {
	n.logger.Error(fmt.Sprintf(f, args...))
}

func (n *nacosLog) Debugf(f string, args ...interface{}) {
	n.logger.Debug(fmt.Sprintf(f, args...))
}
