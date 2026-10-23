/**
 * @Author: dlan
 * @Description:
 * @File: span_test
 * @Version: 1.0.0
 * @Date: 2022/3/18 2:05 下午
 */
package jaeger

import (
	"context"
	"fmt"
	"testing"
)

func TestCtx(t *testing.T) {
	ctx := TraceCtx
	ctx = context.WithValue(ctx, "ewf", "wef")

	fmt.Println(ctx.Value(TraceCtxKey{}))
}
