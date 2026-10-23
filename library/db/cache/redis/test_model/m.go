// Package test_model
// @Author: yinwei
// @File: m
// @Version: 1.0.0
// @Date: 2024/12/9 12:07

package test_model

import "github.com/shopspring/decimal"

type Hmtest struct {
	V1 int64           `json:"v1"`
	V2 string          `json:"v2"`
	V3 decimal.Decimal `json:"v3"`
}
