// Package constrict
// @Author: yinwei
// @File: int
// @Version: 1.0.0
// @Date: 2023/11/11 11:38

package constrict

// 整型
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 |
		~uint16 | ~uint32 | ~uint64
}

type Float interface {
	~float32 | ~float64
}

type Number interface {
	Integer | Float
}

type String interface {
	~string
}
