// Package decimal
// @Author: yinwei
// @File: dec
// @Version: 1.0.0
// @Date: 2024/7/9 11:36

package decimal_cul

import (
	"github.com/shopspring/decimal"
	"github.com/coderyw/gomicro-util/model/constrict"
)

type Decimal struct {
	decimal.Decimal
}

func Mul[T constrict.Number, K constrict.Number](a T, b K) decimal.Decimal {
	return decimal.NewFromFloat(float64(a)).Mul(decimal.NewFromFloat(float64(b)))
}

func Div[T constrict.Number, K constrict.Number](a T, b K) decimal.Decimal {
	return decimal.NewFromFloat(float64(a)).Div(decimal.NewFromFloat(float64(b)))
}

func Add[T constrict.Number, K constrict.Number](a T, b K) decimal.Decimal {
	return decimal.NewFromFloat(float64(a)).Add(decimal.NewFromFloat(float64(b)))
}

func NewFromString(s string) (decimal.Decimal, error) {
	if len(s) == 0 {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}
