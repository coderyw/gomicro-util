// Package common_result_proto
//
// gRPC 响应公共结果的错误码枚举与结构体，替代 ca-common-model 中由 protoc 生成的同名包。
// 这里是手写的普通 Go 类型，不向 protobuf 全局注册表注册，因此可以与仍依赖 ca-common-model 的模块共存。
// 枚举数值与 ca-common-model 保持一致。
package common_result_proto

import (
	"strconv"

	"github.com/pkg/errors"
)

type ERROR_CODE int32

const (
	ERROR_CODE_SUCCESS ERROR_CODE = 0
	// 请求无效
	ERROR_CODE_INVALID_REQUEST ERROR_CODE = 1034
	// 服务错误
	ERROR_CODE_INTERNAL_SERVER_ERROR ERROR_CODE = 1200
	ERROR_CODE_INVALID_TOKEN         ERROR_CODE = 10003
	// 查询暂无记录
	ERROR_CODE_RECORD_NOT_FOUND ERROR_CODE = 1000000
	// 参数错误
	ERROR_CODE_INVALID_PARAM ERROR_CODE = 1000001
)

var ERROR_CODE_alias = map[int32]string{
	0:       "SUCCESS",
	1034:    "Invalid request",
	1200:    "Internal server error",
	10003:   "登录过期,请重新登录",
	1000000: "Record not found",
	1000001: "Invalid param",
}

// StringAlias 返回错误码的提示文案，未定义时返回数值字符串
func (x ERROR_CODE) StringAlias() string {
	s, ok := ERROR_CODE_alias[int32(x)]
	if ok {
		return s
	}
	return strconv.Itoa(int(x))
}

type ERROR_CODEError struct {
	code ERROR_CODE
	err  error
}

func (e ERROR_CODEError) Error() string {
	if e.err != nil {
		return e.code.StringAlias() + ":" + e.err.Error()
	}
	return e.code.StringAlias()
}

// UnwrapCode 解析错误码与错误
func (e ERROR_CODEError) UnwrapCode() (ERROR_CODE, error) {
	err := e.err
	if err == nil {
		err = errors.New(e.code.StringAlias())
	}
	return e.code, err
}

// Unwrap 返回被包装的错误
func (e ERROR_CODEError) Unwrap() error {
	return e.err
}

func (e ERROR_CODEError) Is(target error) bool {
	return errors.Is(e.err, target)
}

func Is(err error, target error) bool {
	if target == nil {
		return err == target
	}
	if e1, ok := err.(*ERROR_CODEError); ok {
		err = e1.Unwrap()
	}
	if e, ok := target.(*ERROR_CODEError); ok {
		target = e.Unwrap()
	}
	return errors.Is(err, target)
}

// New 用可选的 err 创建带堆栈的错误
func (c ERROR_CODE) New(errs ...error) error {
	var err error
	if len(errs) == 0 {
		err = errors.New(c.StringAlias())
	} else {
		err = errors.WithStack(errs[0])
	}
	return &ERROR_CODEError{code: c, err: err}
}

// NewMsg 用可选的 err 创建不带堆栈的错误
func (c ERROR_CODE) NewMsg(errs ...error) error {
	var err error
	if len(errs) == 0 {
		err = errors.New(c.StringAlias())
	} else {
		err = errs[0]
	}
	return &ERROR_CODEError{code: c, err: err}
}

// With 用 err 创建带堆栈的错误，err 为 nil 时返回 nil
func (c ERROR_CODE) With(err error) error {
	if err == nil {
		return nil
	}
	return &ERROR_CODEError{code: c, err: errors.WithStack(err)}
}

// CommonResult 所有返回信息的公共结构
type CommonResult struct {
	// 错误码
	ErrorCode ERROR_CODE `json:"error_code" form:"error_code"`
	// 错误信息
	ErrorMsg string `json:"error_msg" form:"error_msg"`
	// 错误信息扩展，用于详细信息
	ExtraMessage string `json:"extra_message" form:"extra_message"`
}

func (m *CommonResult) GetErrorCode() ERROR_CODE {
	if m != nil {
		return m.ErrorCode
	}
	return ERROR_CODE_SUCCESS
}

func (m *CommonResult) GetErrorMsg() string {
	if m != nil {
		return m.ErrorMsg
	}
	return ""
}

func (m *CommonResult) GetExtraMessage() string {
	if m != nil {
		return m.ExtraMessage
	}
	return ""
}

func (m *CommonResult) SetErrorCode_(val ERROR_CODE) { m.ErrorCode = val }

func (m *CommonResult) SetErrorMsg_(val string) { m.ErrorMsg = val }

func (m *CommonResult) SetExtraMessage_(val string) { m.ExtraMessage = val }

func (m *CommonResult) IsNil() bool {
	return m == nil
}
