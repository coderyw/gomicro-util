// Package test
// @Author: yinwei
// @File: map_test
// @Version: 1.0.0
// @Date: 2024/7/24 08:52

package test

import (
	"log"
	"runtime"
	"strconv"
	"testing"
)

const cnt = 8192

type AB struct {
	BBB int
}

var mGlobal = make(map[int]string)

func TestMapDel(t *testing.T) {
	var m = make(map[int]string)
	memStatus("未初始化map时候分配内存")

	initMap(m)
	runtime.GC()
	memStatus("初始化map后分配内存")

	delMap(m)
	runtime.GC()
	memStatus("删除map元素后分配内存")

	m = nil
	runtime.GC()
	memStatus("map设置为nil后")

}

func initMap(m map[int]string) {
	for i := 0; i < cnt; i++ {
		m[i] = strconv.Itoa(i)
		mGlobal[i] = strconv.Itoa(i)
		//m[i] = i
	}
}

func delMap(m map[int]string) {
	for i := 0; i < cnt/2; i++ {
		delete(m, i)
		delete(mGlobal, i)
	}
}

func memStatus(s string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// Alloc is bytes of allocated heap objects.

	// TotalAlloc is cumulative bytes allocated for heap objects.
	// Sys is the total bytes of memory obtained from the OS.
	// NumGC is the number of completed GC cycles.
	log.Printf("%v Alloc: %v, TotalAlloc: %v, Sys: %v, NumGC: %v", s, m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
}

func TestT(t *testing.T) {
	var a chan int

	a <- 1
}
