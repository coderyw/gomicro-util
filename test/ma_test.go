// Package test
// @Author: yinwei
// @File: ma_test
// @Version: 1.0.0
// @Date: 2024/11/28 17:51

package test

import (
	"fmt"
	"runtime"
	"testing"
)

func TestRem(t *testing.T) {
	data := make([][]byte, 0, 10)
	var memStats runtime.MemStats
	for i := range 9999999 {
		for _ = range 10 {
			data = append(data, make([]byte, 1024*1024*10))
		}
		runtime.ReadMemStats(&memStats)
		fmt.Printf("i=%v, Alloc: %v KB\n", i, memStats.Alloc/1024)
		data = data[:0]
		//time.Sleep(1 * time.Second)
	}
}

func TestRemMap(t *testing.T) {
	
}
