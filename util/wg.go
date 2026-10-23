// Package util 工具包
package util

import "sync"

// WaitGroupWrapper WaitGroup包装器，提供更方便的并发控制功能
// 嵌入了标准库的sync.WaitGroup，扩展了Wrap方法用于简化goroutine的使用
type WaitGroupWrapper struct {
	sync.WaitGroup
}

// Wrap 包装函数并在新的goroutine中执行
// 自动调用Add(1)和Done()，简化了WaitGroup的使用
// 参数:
//
//	f - 要在goroutine中执行的函数
func (w *WaitGroupWrapper) Wrap(f func()) {
	w.Add(1)
	go func() {
		f()
		w.Done()
	}()
}
