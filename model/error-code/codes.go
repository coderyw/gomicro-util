// Package error_code
//
// 框架内置错误码与业务错误码注册入口。
// 内置错误码的数值与 ca-common-model 保持一致，便于与存量服务之间通过 gRPC trailer 互通；
// 业务项目在自己的包里用 NewErrorCode 定义错误码，不需要改动本库。
package error_code

import (
	"fmt"
	"sync"
)

var (
	// SUCCESS 成功
	SUCCESS = newErrorCode(200, 1, "Success", "zh", "成功")
	// INVALID_PARAM 参数错误
	INVALID_PARAM = newErrorCode(200, 400000001, "Invalid param", "zh", "参数错误")
	// RECORD_NOT_FOUND 查询暂无记录
	RECORD_NOT_FOUND = newErrorCode(200, 404000001, "Record not found", "zh", "查询暂无记录")
	// GRPC_AUTHENTICATION_FAILURE 微服务认证失败
	GRPC_AUTHENTICATION_FAILURE = newErrorCode(200, 404000008, "Microservice authentication failed", "zh", "微服务认证失败")
	// INTERNAL_SERVER_ERROR 服务错误
	INTERNAL_SERVER_ERROR = newErrorCode(200, 500000000, "Internal Server error", "zh", "服务错误")
	// INVALID_PHONE_NUMBER 无效的手机号
	INVALID_PHONE_NUMBER = newErrorCode(200, 500000008, "Invalid phone number.", "zh", "无效的手机号")
)

var (
	registryMu   sync.RWMutex
	errorCodeMap = map[uint32]StatusCode{
		uint32(SUCCESS.code):                     SUCCESS,
		uint32(INVALID_PARAM.code):               INVALID_PARAM,
		uint32(RECORD_NOT_FOUND.code):            RECORD_NOT_FOUND,
		uint32(GRPC_AUTHENTICATION_FAILURE.code): GRPC_AUTHENTICATION_FAILURE,
		uint32(INTERNAL_SERVER_ERROR.code):       INTERNAL_SERVER_ERROR,
		uint32(INVALID_PHONE_NUMBER.code):        INVALID_PHONE_NUMBER,
	}
)

// NewErrorCode 定义并注册一个业务错误码，供 Code2Error / GrpcCode2Error 按数值还原。
//
// 应在包级变量初始化时调用；code 重复时 panic，避免两个错误码静默互相覆盖。
//
// Parameters:
//   - httpCode: HTTP 状态码
//   - code: 错误码数值
//   - defaultMsg: 默认提示
//   - msgs: 多语言提示，按 "语言", "提示" 成对传入，如 "zh", "参数错误"
func NewErrorCode(httpCode int, code Code, defaultMsg string, msgs ...string) StatusCode {
	s := newErrorCode(httpCode, code, defaultMsg, msgs...)
	registryMu.Lock()
	defer registryMu.Unlock()
	if old, ok := errorCodeMap[uint32(code)]; ok {
		panic(fmt.Sprintf("error_code: duplicate code %d (%q and %q)", code, old.message, defaultMsg))
	}
	errorCodeMap[uint32(code)] = s
	return s
}

func lookup(code uint32) (StatusCode, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	s, ok := errorCodeMap[code]
	return s, ok
}
