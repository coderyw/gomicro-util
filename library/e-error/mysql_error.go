// Package e_error
// @Author: yinwei
// @File: mysql_error
// @Version: 1.0.0
// @Date: 2023/11/8 10:06

package e_error

import (
	"errors"
	commonResult "github.com/coderyw/gomicro-util/model/common-result"
	"github.com/coderyw/gomicro-util/library/db/mysql"
)

// Deprecated: 请使用 common-model
// CheckSqlError 校验error是否是notfound错误
func CheckSqlError(err error, code ...commonResult.ERROR_CODE) error {
	if err == nil {
		return err
	}
	if errors.Is(err, mysql.NotFound) {
		if len(code) != 0 {
			return code[0].New(err)
		}
		return commonResult.ERROR_CODE_RECORD_NOT_FOUND.New(err)
	}
	return commonResult.ERROR_CODE_INTERNAL_SERVER_ERROR.New(err)
}
