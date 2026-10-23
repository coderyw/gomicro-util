// Package skywalking
// @Author: yinwei
// @File: trace_test.go
// @Version: 1.0.0
// @Date: 2025/4/15 18:17

package skywalking

import (
	"fmt"
	"github.com/coderyw/gomicro-util/model/constrict"
	"testing"
)

func Abc[T constrict.Integer](a T) {
	Getname()
}

func Getname() {
	a := getFuncName()
	fmt.Println(a)
}

func Test_getFuncName(t *testing.T) {
	Abc[int](1)
}
