// Package single
// @Author: yinwei
// @File: os_single_test.go
// @Version: 1.0.0
// @Date: 2025/6/9 09:10

package single

import (
	"fmt"
	"testing"
)

func TestOsKillSingle(t *testing.T) {

	got := OsKillSingle()
	got2 := OsKillSingle()
	sig := <-got
	fmt.Println("got :", sig.String())
	sig = <-got2
	fmt.Println("got2 :", sig.String())
}
