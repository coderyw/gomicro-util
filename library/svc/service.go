// Package server_init
// @Author: yinwei
// @File: service
// @Version: 1.0.0
// @Date: 2023/11/2 16:51

package svc

import (
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/facade"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var signalNotify = signal.Notify

func Run(service facade.Service, sig ...os.Signal) error {
	// 华为云 golang 启动会重启一次，发现是启动后获取 nacos 配置错误，导致服务重启
	time.Sleep(5 * time.Second)
	if err := service.Init(); err != nil {
		return err
	}

	if err := service.Start(); err != nil {
		return err
	}

	if len(sig) == 0 {
		sig = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
	}

	signalChan := make(chan os.Signal, 1)
	signalNotify(signalChan, sig...)
	<-signalChan
	log.Info("收到停服信号")

	return service.Stop()
}
