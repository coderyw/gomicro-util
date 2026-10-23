// Package test
// @Author: yinwei
// @File: pprof_test
// @Version: 1.0.0
// @Date: 2023/11/22 17:54

package test

import (
	"bytes"
	"fmt"
	pprof2 "runtime/pprof"
	"testing"
)

func TestPProf(t *testing.T) {
	buf := new(bytes.Buffer)
	err := pprof2.Lookup("goroutine").WriteTo(buf, 1)
	fmt.Println(err)
	fmt.Println(buf.String())

}
