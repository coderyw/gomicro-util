// Package single
// @Author: yinwei
// @File: os_single
// @Version: 1.0.0
// @Date: 2025/6/9 09:08

package single

import (
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// OsKillSingle 监听系统停服的信号
//
// Returns:
//   - chan 停服通知通道
func OsKillSingle() chan os.Signal {

	sig := []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, sig...)
	return signalChan
}

var readinessProbe atomic.Bool

func GetReadinessProbe() bool {
	return readinessProbe.Load()
}

func SetReadinessProbe(probe bool) {
	readinessProbe.Store(probe)
}
