// Package bool
// @Author: yinwei
// @File: boolean
// @Version: 1.0.0
// @Date: 2024/7/15 09:16

package bool

// Bool2Num
//
//	@Description: boolean转uint8
//	@Author yw 2024-07-15 09:17:00
//	@param b
//	@return uint8
func Bool2Num(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
