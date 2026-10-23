// Package system
// @Author: yinwei
// @File: balance
// @Version: 1.0.0
// @Date: 2024/7/3 16:38

package system

import (
	"github.com/shopspring/decimal"
	"github.com/coderyw/gomicro-util/model/constrict"
	"github.com/coderyw/gomicro-util/util/number/decimal_cul"
)

// FormatBalance
//
//	@Description: 格式化余额
//		默认返回分
//	@Author yw 2024-07-03 16:38:57
//	@param balance 余额
//	@param coin true:返回分; false:  返回元，
//	@return float64
func FormatBalance[T constrict.Number](balance T, cent ...bool) float64 {
	if len(cent) == 0 || (len(cent) > 0 && !cent[0]) {
		return decimal_cul.Div(balance, 100).Round(2).InexactFloat64()
	}
	return decimal.NewFromFloat(float64(balance)).Round(2).InexactFloat64()
}

func FormatBalanceStr[T constrict.Number](balance T, cent ...bool) string {
	if len(cent) == 0 || (len(cent) > 0 && !cent[0]) {
		return decimal_cul.Div(balance, 100).Round(2).String()
	}
	return decimal.NewFromFloat(float64(balance)).Round(2).String()
}
