// Package facade
// @Author: yinwei
// @File: zset_facade
// @Version: 1.0.0
// @Date: 2025/1/15 17:19

package facade

import "github.com/shopspring/decimal"

// 有序集合 操作
type ZSetAction interface {
	// Set 设置有序集合值
	//
	// Parameters:
	//  - key: string 有序集合key
	//  - score: decimal.Decimal 分数
	Set(key string, score decimal.Decimal)
	// Get 获取有序集合值
	//
	// Returns:
	//  - string 有序集合key
	//  - decimal.Decimal 分数
	Get() (string, decimal.Decimal)
}
