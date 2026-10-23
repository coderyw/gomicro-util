// Package wait_stop
// @Author: yinwei
// @File: ws
// @Version: 1.0.0
// @Date: 2024/12/9 20:25

package wait_stop

import (
	"github.com/coderyw/gomicro-util/library/log"
	"sync"
)

var (
	wg   = &sync.WaitGroup{}
	lock sync.RWMutex
	stop = false
)

func Add() {
	log.Debug("add")
	wg.Add(1)
}

func Done() {
	log.Debug("done")
	wg.Done()
}

func Wait() {
	wg.Wait()
}

func Stop() {
	lock.Lock()
	defer lock.Unlock()
	stop = true
}

func IsStop() bool {
	lock.RLock()
	defer lock.RUnlock()
	return stop
}
