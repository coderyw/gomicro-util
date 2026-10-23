// Package metadata
// @Author: yinwei
// @File: gin_ctx_test.go
// @Version: 1.0.0
// @Date: 2025/6/26 16:02

package metadata

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetGinCtxData(t *testing.T) {
	Request, _ := http.NewRequest("GET", "/api", nil)
	gc := &gin.Context{Request: Request}
	SetCtxData(gc, CONTEXT_USER_ID, int64(123))
	val := gc.Value(_contextMetadataStr)
	fmt.Println(val)
}
