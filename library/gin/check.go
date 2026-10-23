// Package gin
// @Author: yinwei
// @File: check
// @Version: 1.0.0
// @Date: 2025/5/13 08:43

package gin

import (
	"github.com/gin-gonic/gin"
	error_code "github.com/coderyw/gomicro-util/model/error-code"
)

// BindAndCheck 绑定并验证请求参数（只支持解析 body）
//
// Parameters:
//   - c: *gin.Context gin 上下文
//   - obj: interface{} 解析到对象
//
// Returns:
//   - error 错误信息
func BindAndCheck(c *gin.Context, obj interface{}) error {
	if obj == nil {
		return error_code.INTERNAL_SERVER_ERROR.NewMsg("解析对象为空")
	}
	if err := c.Bind(obj); err != nil {
		return error_code.INVALID_PARAM.New(err)
	}
	if val, ok := obj.(Validator); ok {
		if err := val.Validate(); err != nil {
			return error_code.INVALID_PARAM.New(err)
		}
	}
	if val, ok := obj.(CValidator); ok {
		if err := val.DoValidate(); err != nil {
			return error_code.INVALID_PARAM.New(err)
		}
	}

	return nil
}

type Validator interface {
	Validate() error
}

// 可以自定义参数校验验证器
type CValidator interface {
	DoValidate() error
}
