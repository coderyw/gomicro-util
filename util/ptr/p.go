// Package ptr
// @Author: yinwei
// @File: p
// @Version: 1.0.0
// @Date: 2023/11/6 22:22

package ptr

import "github.com/coderyw/gomicro-util/model/constrict"

func Uint8[T constrict.Integer](val T) *uint8 {
	a := uint8(val)
	return &a
}

// IntegerPtr 获取整型指针
//
// Parameters:
//   - val: T
//
// Returns:
//   - *K
func IntegerPtr[T, K constrict.Integer](val T) *K {
	k := K(val)
	return &k
}

type TASK_TYPE int32

func (t TASK_TYPE) IsEqual(val any) bool {
	switch val.(type) {
	case TASK_TYPE:
		return val.(TASK_TYPE) == t
	case int:
		return int(t) == val
	case int8:
		return int8(t) == val
	case int32:
		return int32(t) == val
	case int64:
		return int64(t) == val
	case uint8:
		return uint8(t) == val
	case uint16:
		return uint16(t) == val
	case uint32:
		return uint32(t) == val
	case uint64:
		return uint64(t) == val
	default:
		return false

	}
}
