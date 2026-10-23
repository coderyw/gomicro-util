// Package log
// @Author: yinwei
// @File: go_mail_log
// @Version: 1.0.0
// @Date: 2024/12/9 19:32

package log

import (
	"fmt"

	"github.com/wneessen/go-mail/log"
	"go.uber.org/zap"
)

type GoMailLog struct {
}

func (g GoMailLog) Dir(dir int) string {
	switch dir {
	case 1:
		return "ClientToServer"
	case 0:
		return "ServerToClient"
	default:
		return "unknown"
	}
}

func (g GoMailLog) Debugf(log log.Log) {
	Debug(g.Dir(int(log.Direction)), zap.String("log", fmt.Sprintf(log.Format, log.Messages...)))
}

func (g GoMailLog) Infof(log log.Log) {
	Info(g.Dir(int(log.Direction)), zap.String("log", fmt.Sprintf(log.Format, log.Messages...)))
}

func (g GoMailLog) Warnf(log log.Log) {
	Warn(g.Dir(int(log.Direction)), zap.String("log", fmt.Sprintf(log.Format, log.Messages...)))
}

func (g GoMailLog) Errorf(log log.Log) {
	Error(g.Dir(int(log.Direction)), zap.String("log", fmt.Sprintf(log.Format, log.Messages...)))
}
