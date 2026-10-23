// Package ca_common_util
// @Author: yinwei
// @File: init
// @Version: 1.0.0
// @Date: 2024/6/18 14:07

package ca_common_util

import (
	"github.com/shopspring/decimal"
)

func init() {
	// 设置json序列化成数字
	decimal.MarshalJSONWithoutQuotes = true
}
