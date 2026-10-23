// Package operation
// @Author: yinwei
// @File: op
// @Version: 1.0.0
// @Date: 2024/9/9 13:19

package operation

// TernaryOp
//
//	@Description: 三元运算
//	@Author yw 2024-09-09 13:20:40
//	@param a
//	@param b
//	@param c
//	@return T
func TernaryOp[T any](a bool, b func() T, c func() T) T {
	if a {
		return b()
	} else {
		return c()
	}
}

// Ternary
//
//	@Description: 三元运算
//	@Author yw 2024-09-09 13:25:56
//	@param a
//	@param b
//	@param c
//	@return T
func Ternary[T any](a bool, b, c T) T {
	if a {
		return b
	} else {
		return c
	}
}
