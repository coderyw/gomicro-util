// Package main
// @Author: yinwei
// @File: jwt_test
// @Version: 1.0.0
// @Date: 2024/9/27 09:49

package test

import (
	"fmt"
	"regexp"
	"testing"
)

func TestReg(t *testing.T) {
	s := "/login"
	_, err := regexp.Compile(s)
	if err == nil {
		fmt.Println("是正则")
	} else {
		fmt.Println("不是正则")
	}
}

func Benchmark_TestReg2(b *testing.B) {
	b.Run("reg", func(b *testing.B) {
		s := "/login"
		r, _ := regexp.Compile(s)
		for i := 0; i < b.N; i++ {
			r.MatchString("/login")
		}
	})

}
