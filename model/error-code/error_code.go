// Package error_code
// @Author: yinwei
// @File: error_code
// @Version: 1.0.0
// @Date: 2024/9/20 13:26

package error_code

import (
	"errors"
	"fmt"
	"io"
)

type Code uint32

// Code2Error
//
//	@Description: 通过code获取错误码
//	@Author	yw 2024-09-27 16:31:10
//	@param	code
//	@param	msg
//	@return	error
func Code2Error(code uint32, msg ...string) error {
	if val, ok := lookup(code); ok {
		return val.NewMsg(msg...)
	} else {
		return INTERNAL_SERVER_ERROR.NewMsg(msg...)
	}
}

// GrpcCode2Error 根据gRPC错误代码转换为自定义错误对象。
//
// 参数:
//
//	code: gRPC错误代码。
//	msg: 错误消息。
//	ext: 扩展信息，用于提供更多错误上下文。
//
// 返回值:
//
//	error: 根据错误代码和消息创建的错误对象。
//
// 该函数首先尝试根据code从errorCodeMap中查找对应的错误定义。
// 如果找到，则使用该错误定义创建一个新的错误对象，并附带额外的错误信息ext。
// 如果未找到，则使用默认状态码200和提供的错误代码、消息创建一个新的错误定义，并附带额外的错误信息ext。
func GrpcCode2Error(code uint32, msg string, ext string) error {
	// 检查errorCodeMap中是否存在对应的错误定义
	if val, ok := lookup(code); ok {
		// 存在则使用该错误定义创建新的错误对象
		return val.NewMsg(ext)
	} else {
		// 不存在则使用默认状态码和提供的信息创建新的错误定义
		return newErrorCode(200, Code(code), msg, "").NewMsg(ext)
	}
}

// Unwrap
//
//	@Description: 解析错误码
//	@Author	yw 2024-09-26 14:11:19
//	@param	err
//	@return	StatusCode
//	@return	error
func Unwrap(err error) (StatusCode, error) {
	if err == nil {
		return SUCCESS, nil
	}
	if ec, ok := err.(*ErrorCode); ok {
		return ec.StatusCode, ec.err
	}
	return INTERNAL_SERVER_ERROR, err
}

// MustUnwrap
//
//	@Description: 必须解析成错误码
//	@Author	yw 2024-09-27 16:02:58
//	@param	err
//	@return	StatusCode
//	@return	bool
//	@return	error
func MustUnwrap(err error) (StatusCode, bool, error) {
	if err == nil {
		return SUCCESS, true, nil
	}
	if ec, ok := err.(*ErrorCode); ok {
		return ec.StatusCode, true, ec.err
	}
	return StatusCode{}, false, err
}

const codeFormat = "{%v : %v}"
const codeFormat1 = "{%v : %v} = %v"

// 错误码结构
type StatusCode struct {
	httpCode int
	// 错误码
	code Code
	// 错误码默认提示
	message string
	// 多语言错误码提示
	msgs map[string]string
}

func (s StatusCode) Is(err error) bool {
	if err == nil {
		return false
	}
	if ec, ok := err.(*ErrorCode); ok {
		return ec.Code() == s.code
	}
	return s.message == err.Error()
}

func (s *StatusCode) HttpCode() int {
	return s.httpCode
}

func (s *StatusCode) Code() Code {
	return s.code
}

// GetMessage 获取错误码对应语言的提示信息
//
// Parameters:
//   - lang(string): 可选参数,中文：zh，英文：en；不传时，返回默认的提示信息；传入时根据对应的语言获取提示信息
//
// Returns:
//   - string: 提示信息
func (s *StatusCode) GetMessage(lang ...string) string {
	if len(lang) == 0 {
		return s.message
	}
	if s.msgs != nil {
		if val, ok := s.msgs[lang[0]]; ok {
			return val
		}
	}
	return s.message
}

type ErrorCode struct {
	StatusCode
	err error

	stack
}

// New
//
//	@Description: 根据传入的错误信息创建一个错误码
//	@Author	yw 2024-09-20 14:37:35
//	@param	errs
//	@return	error
func (s StatusCode) New(errs ...error) error {
	ec := ErrorCode{
		StatusCode: s,
	}
	if len(errs) > 0 {
		ec.err = errs[0]
	} else {
		ec.err = errors.New(s.message)
	}
	return &ec
}

// NewWithStack
//
//	@Description: 根据传入的错误信息创建一个带堆栈的错误码
//	@Author	yw 2024-09-20 14:37:55
//	@param	errs 需要被包装的错误
//	@return	error 带错误码的错误
func (s StatusCode) NewWithStack(errs ...error) error {
	ec := ErrorCode{
		StatusCode: s,
	}
	if len(errs) > 0 && errs[0] != nil {
		ec.err = errs[0]
	} else {
		ec.err = errors.New(s.message)
	}
	ec.stack = callers()
	return &ec
}

// NewMsg
//
//	@Description: 返回一个包装了msgs的错误码
//	@Author	yw 2024-09-20 14:40:09
//	@param	msgs
//	@return	error
func (s StatusCode) NewMsg(msgs ...string) error {
	ec := ErrorCode{
		StatusCode: s,
	}
	if len(msgs) > 0 {
		if len(msgs) > 1 {
			ec.err = fmt.Errorf(msgs[0], msgs[1:])
		} else {
			ec.err = errors.New(msgs[0])
		}
	} else {
		ec.err = errors.New(s.message)
	}
	return &ec
}

// NewReplaceMsg 创建一个自定义提示的错误码
//
// 传入自定义的提示信息，返回一个带错误码的错误
//
// Parameters:
//   - msg: 自定义提示信息
//
// Returns:
//   - error: 带错误码的错误
func (s StatusCode) NewReplaceMsg(msg string) error {
	ec := ErrorCode{
		StatusCode: s,
	}
	ec.message = msg
	ec.err = errors.New(s.message)
	return &ec
}

// NewMsgWithStack
//
//	@Description: 返回一个包装了msgs的错误码，包含堆栈
//	@Author	yw 2024-09-20 14:40:56
//	@param	msgs
//	@return	error
func (s StatusCode) NewMsgWithStack(msgs ...string) error {
	ec := ErrorCode{
		StatusCode: s,
	}
	if len(msgs) > 0 {
		if len(msgs) > 1 {
			ec.err = fmt.Errorf(msgs[0], msgs[1:])
		} else {
			ec.err = errors.New(msgs[0])
		}
	} else {
		ec.err = errors.New(s.message)
	}
	ec.stack = callers()
	return &ec
}

func (e ErrorCode) Is(target error) bool {
	if target == nil {
		return false
	}
	if val, ok := target.(*ErrorCode); ok {
		return e.Code() == val.Code()
	}
	return errors.Is(e.err, target)
}

func (e ErrorCode) Unwrap() error {
	return e.err
}

func (e ErrorCode) Error() string {
	if e.err != nil {
		return fmt.Sprintf(codeFormat1, e.code, e.message, e.err.Error())
	}
	return fmt.Sprintf(codeFormat, e.code, e.message)
}

func (w *ErrorCode) Cause() error { return w.err }

func (w *ErrorCode) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, codeFormat1, w.code, w.message, w.Cause())
			w.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func newErrorCode(httpCode int, code Code, defaultMsg string, msgs ...string) StatusCode {
	e := StatusCode{}
	e.code = code
	e.httpCode = httpCode
	e.message = defaultMsg
	if len(msgs) > 0 {
		e.msgs = make(map[string]string, len(msgs)/2+1)
	}
	for i := 1; i < len(msgs); {
		e.msgs[msgs[i-1]] = msgs[i]
		i += 2
	}
	return e
}
