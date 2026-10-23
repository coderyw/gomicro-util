// Package number 数字处理工具包
// @Author: yinwei
// @File: int
// @Version: 1.0.0
// @Date: 2023/11/11 11:36

package number

import (
	"context"
	"strconv"

	"github.com/shopspring/decimal"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/constrict"
	"go.uber.org/zap"
)

// Equal 判断两个整数是否相等
// 参数:
//   - v1: 第一个整数
//   - v2: 第二个整数
//
// 返回值:
//   - bool: 如果两个整数相等返回true，否则返回false
func Equal[T constrict.Integer](v1, v2 T) bool {

	return v1 == v2
}

// szdw 数字单位数组，用于数字转中文时的单位表示
var szdw = []string{"十", "百", "千", "万", "十万", "百万", "千万", "亿"}

// sz 数字对应的中文字符，用于数字转中文
var sz = []string{"零", "一", "二", "三", "四", "五", "六", "七", "八", "九"}

// NumToCN 将阿拉伯数字转换为中文数字表示
// 参数:
//   - num: 要转换的整数
//
// 返回值:
//   - string: 对应的中文数字字符串
func NumToCN(num int) string {
	// 1、处理数字为0的情况
	if num == 0 {
		return "零"
	}
	var ans string
	// 存储结果的字符串切片

	res := make([]string, 0)

	// 数字单位索引，从-1开始
	idx := -1
	for num > 0 {
		// 获取当前位数的值
		x := num % 10
		// 2、处理数字大于等于10的情况
		// 插入数字单位，只有当数字单位索引在范围内，且当前数字不为0时才有效
		if idx >= 0 && idx < len(szdw) && x != 0 {
			res = append([]string{szdw[idx]}, res...)
		}
		// 3、处理数字中间有多个0的情况
		// 当前数字为0，且后一位也为0时，为避免重复删除一个零字符
		if x == 0 && len(res) != 0 && res[0] == "零" {
			res = res[1:]
		}
		// 插入数字对应的中文字符
		res = append([]string{sz[x]}, res...)
		num /= 10
		idx++
	}
	// 4、处理个位数为0的情况
	if len(res) > 1 && res[len(res)-1] == "零" {
		res = res[:len(res)-1]
	}
	// 将字符串切片合并为最终结果
	for i := 0; i < len(res); i++ {
		ans = ans + res[i]
	}
	return ans
}

// Ternary 三元运算函数，根据条件选择返回值
// 获取非0的数。如果传了c，则就是三元运算。如果不传c则是二元运算
// 逻辑规则:
//  1. 如果a是0，c有值则返回c; c没值则返回b
//  2. 如果a不是0，c有值则返回b，c没值则返回a
//
// 参数:
//   - a: 条件参数
//   - b: 第一个候选返回值
//   - c: 可选的第二个候选返回值
//
// 返回值:
//   - T: 根据条件选择的返回值
//
// @Author yw 2024-07-09 09:20:50
func Ternary[T constrict.Number, TA constrict.Number](a TA, b T, c ...T) T {
	if a == 0 {
		if len(c) == 0 {
			return b
		} else {
			return c[0]
		}
	} else {
		if len(c) == 0 {
			return T(a)
		} else {
			return b
		}
	}
}

// TernaryB 基于布尔值的三元运算函数
// 逻辑规则:
//  1. 如果a是false，返回c
//  2. 如果a是true，返回b
//
// 参数:
//   - a: 布尔条件
//   - b: 条件为true时的返回值
//   - c: 条件为false时的返回值
//
// 返回值:
//   - T: 根据布尔条件选择的返回值
//
// @Author yw 2024-07-09 09:20:50
func TernaryB[T constrict.Number](a bool, b T, c T) T {
	if !a {
		return c
	} else {
		return b
	}
}

// Abs 计算数字的绝对值
// 参数:
//   - a: 输入的数字
//
// 返回值:
//   - T: 输入数字的绝对值
func Abs[T constrict.Number](a T) T {
	if a < 0 {
		return -a
	}
	return a
}

// Str2Float 将字符串转换为浮点数
// 使用decimal包进行精确转换，避免浮点数精度问题
// 参数:
//   - s: 要转换的字符串
//
// 返回值:
//   - float64: 转换后的浮点数，转换失败时返回0
//
// @Author yw 2024-07-12 17:32:43
func Str2Float(s string) float64 {
	dec, err := decimal.NewFromString(s)
	if err != nil {
		log.Warn("转换字符串到浮点数错误", zap.Error(err))
	}
	return dec.InexactFloat64()
}

// Str2Int 将字符串转换为整数
// 参数:
//   - ctx: 上下文对象，用于日志记录
//   - s: 要转换的字符串
//
// 返回值:
//   - int: 转换后的整数，转换失败时返回0
func Str2Int(ctx context.Context, s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.ErrorCtx(ctx, "转换字符串到整数错误", zap.String("s", s), zap.Error(err))
	}
	return i
}
