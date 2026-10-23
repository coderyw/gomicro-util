// Package test
// @Author: yinwei
// @File: ants
// @Version: 1.0.0
// @Date: 2024/7/29 13:35

package test

import (
	"fmt"
	"github.com/panjf2000/ants/v2"
	"sync"
	"testing"
	"time"
)

func TestAnts(t *testing.T) {
	wg := sync.WaitGroup{}
	pools, _ := ants.NewPoolWithFunc(10, func(i interface{}) {
		defer wg.Done()
		fmt.Println(i)
		time.Sleep(time.Millisecond * 3000)
	}, ants.WithNonblocking(false))

	for i := 0; i < 100; i++ {
		wg.Add(1)
		err := pools.Invoke(i)
		if err != nil {
			t.Error(err)
			wg.Done()
		}
	}
	wg.Wait()

}
