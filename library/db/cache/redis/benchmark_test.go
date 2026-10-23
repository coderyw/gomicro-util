// Package redis
// @Author: yinwei
// @File: branchmark_test
// @Version: 1.0.0
// @Date: 2024/7/8 18:07

package redis

import (
	"testing"
)

func Benchmark_Ab(b *testing.B) {
	b.Run("reflect", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Te[*Ab]()
		}
	})
	b.Run("new", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = new(Ab)
		}
	})
}
