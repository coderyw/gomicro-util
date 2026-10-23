// Package ctx
// @Author: yinwei
// @File: ctx_test
// @Version: 1.0.0
// @Date: 2025/6/16 15:06

package metadata

import (
	"context"
	"testing"
	"time"
)

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	ctx = context.WithValue(ctx, "key", "value")
	go func() {
		time.Sleep(5 * time.Second)
		cancel()

	}()
	<-ctx.Done()
}
