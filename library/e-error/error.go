/**
 * @Author: yinwei
 * @Description: 错误处理工具包，提供错误类型判断、错误解析、错误码提取等功能
 * @File: error
 * @Version: 1.0.0
 * @Date: 2023/10/26 13:15
 */
// Package e_error 错误处理工具包（已废弃，建议使用common-model）
// 提供错误类型判断、错误解析、错误码提取等功能
package e_error

import (
	"errors"

	commonResult_proto "github.com/coderyw/gomicro-util/model/common-result"
	"github.com/coderyw/gomicro-util/library/db/cache/redis"
	"github.com/coderyw/gomicro-util/library/db/mysql"
	"gorm.io/gorm"
)

// Is 判断错误是否与目标错误匹配
// Deprecated: 请使用common-model
// 参数:
//   - err: 要检查的错误
//   - target: 目标错误
//
// 返回值:
//   - bool: 如果错误匹配返回true，否则返回false
func Is(err error, target error) bool {
	// 如果目标错误为空，直接比较
	if target == nil {
		return err == target
	}
	// 如果错误是ERROR_CODEError类型，解包获取原始错误
	if e1, ok := err.(*commonResult_proto.ERROR_CODEError); ok {
		err = e1.Unwrap()
	}
	// 如果目标错误是ERROR_CODEError类型，解包获取原始错误
	if e, ok := target.(*commonResult_proto.ERROR_CODEError); ok {
		target = e.Unwrap()
	}
	// 使用标准库的errors.Is进行比较
	return errors.Is(err, target)
}

// Unwrap 从错误中解析出错误码和原始错误
// 支持MySQL NotFound错误和Redis Nil错误的特殊处理
// Deprecated: 请使用common-model
// 参数:
//   - err: 要解析的错误
//   - d: 可选的默认错误码
//
// 返回值:
//   - commonResult_proto.ERROR_CODE: 解析出的错误码
//   - error: 原始错误
func Unwrap(err error, d ...commonResult_proto.ERROR_CODE) (commonResult_proto.ERROR_CODE, error) {
	if err != nil {
		// 如果是ERROR_CODEError类型，直接解包获取错误码和错误
		if e1, ok := err.(*commonResult_proto.ERROR_CODEError); ok {
			return e1.UnwrapCode()
		}
		// MySQL记录未找到错误
		if errors.Is(err, mysql.NotFound) {
			return commonResult_proto.ERROR_CODE_RECORD_NOT_FOUND, err
		}
		// Redis空值错误
		if errors.Is(err, redis.RedisNil) {
			return commonResult_proto.ERROR_CODE_RECORD_NOT_FOUND, err
		}
		// 如果提供了默认错误码，使用默认错误码
		if len(d) > 0 {
			return d[0], err
		}
		// 默认返回内部服务器错误
		return commonResult_proto.ERROR_CODE_INTERNAL_SERVER_ERROR, err
	}
	// 没有错误时返回成功状态
	return commonResult_proto.ERROR_CODE_SUCCESS, nil
}

// UnwrapCode 从错误中只解析错误码，不返回原始错误
// 支持MySQL NotFound错误和Redis Nil错误的特殊处理
// Deprecated: 请使用common-model
// 参数:
//   - err: 要解析的错误
//   - d: 可选的默认错误码
//
// 返回值:
//   - code: 解析出的错误码
//
// @Author yw 2024-08-05 11:41:14
func UnwrapCode(err error, d ...commonResult_proto.ERROR_CODE) (code commonResult_proto.ERROR_CODE) {
	if err != nil {
		// 如果是ERROR_CODEError类型，解包获取错误码
		if e1, ok := err.(*commonResult_proto.ERROR_CODEError); ok {
			code, _ = e1.UnwrapCode()
		} else if errors.Is(err, mysql.NotFound) {
			// MySQL记录未找到错误
			code = commonResult_proto.ERROR_CODE_RECORD_NOT_FOUND
		} else if errors.Is(err, redis.RedisNil) {
			// Redis空值错误
			code = commonResult_proto.ERROR_CODE_RECORD_NOT_FOUND
		} else if len(d) > 0 {
			// 使用提供的默认错误码
			code = d[0]
		} else {
			// 默认为内部服务器错误
			code = commonResult_proto.ERROR_CODE_INTERNAL_SERVER_ERROR
		}
	} else {
		// 没有错误时返回成功状态
		code = commonResult_proto.ERROR_CODE_SUCCESS
	}
	return
}

// IsNotFound 检测错误是否是数据库记录未找到错误
// 支持GORM和Redis的记录未找到错误判断
// Deprecated: 请使用common-model
// 参数:
//   - err: 要检查的错误
//
// 返回值:
//   - bool: 如果是记录未找到错误返回true，否则返回false
//
// @Author yw 2024-07-23 10:52:39
func IsNotFound(err error) bool {
	// 如果错误为空，返回false
	if err == nil {
		return false
	}
	// 检查是否为GORM记录未找到错误
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	} else if errors.Is(err, redis.RedisNil) {
		// 检查是否为Redis空值错误
		return true
	}
	// 其他错误返回false
	return false
}
