package orchestrion

import (
	"context"
	"fmt"
	"testing"
)

func TestStartSpan(t *testing.T) {
	ctx := context.WithValue(context.TODO(), "123", "456")
	ctx = StartSpan(ctx, "test")
	fmt.Println(ctx.Value("123"))

}
