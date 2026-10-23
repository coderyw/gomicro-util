// Package strings 字符串处理工具包
// @Author: yinwei
// @File: strings
// @Version: 1.0.0
// @Date: 2023/11/16 14:33

package strings

import (
	"math/rand"
	"strings"
	"time"
	"unsafe"

	"github.com/coderyw/gomicro-util/model/constrict"
)

// StrCombine 字符串组合函数，将多个字符串连接成一个字符串
// 使用泛型支持任何实现了String接口的类型
// 参数:
//   - strs: 可变参数，要组合的字符串切片
//
// 返回值:
//   - string: 组合后的字符串
func StrCombine[T constrict.String](strs ...T) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return string(strs[0])
	}
	b := strings.Builder{}
	for _, v := range strs {
		b.WriteString(string(v))
	}
	return b.String()
}

// KeyFormat 键格式化函数，将多个字符串用冒号(:)连接
// 常用于生成Redis键、缓存键等需要层级结构的键名
// 参数:
//   - strs: 可变参数，要格式化的字符串切片
//
// 返回值:
//   - string: 用冒号分隔的格式化字符串，如: "prefix:key:suffix"
func KeyFormat(strs ...string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	b := strings.Builder{}
	for i, v := range strs {
		if i > 0 {
			b.WriteString(":")
		}
		b.WriteString(v)
	}
	return b.String()
}

// letters 随机字符串生成使用的字符集，包含大小写字母和数字
const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// src 随机数生成源，使用当前时间纳秒作为种子
var src = rand.NewSource(time.Now().UnixNano())

const (
	// letterIdBits 用6位来表示一个字符索引
	letterIdBits = 6
	// letterIdMask 所有位都是1的掩码，位数等于letterIdBits
	letterIdMask = 1<<letterIdBits - 1
	// letterIdMax 一个63位随机数最多能表示的字符数量
	letterIdMax = 63 / letterIdBits
)

// RandStr 生成指定长度的随机字符串
// 使用高效的位运算算法，一个Int63()调用可以生成多个字符
// 参数:
//   - n: 要生成的随机字符串长度
//
// 返回值:
//   - string: 生成的随机字符串
func RandStr(n int) string {
	b := make([]byte, n)
	// rand.Int63()生成63位随机数，足够生成letterIdMax个字符
	for i, cache, remain := n-1, src.Int63(), letterIdMax; i >= 0; {
		if remain == 0 {
			// 缓存用完时重新生成随机数
			cache, remain = src.Int63(), letterIdMax
		}
		if idx := int(cache & letterIdMask); idx < len(letters) {
			// 如果索引有效，设置对应字符
			b[i] = letters[idx]
			i--
		}
		// 右移letterIdBits位，准备下一个字符
		cache >>= letterIdBits
		remain--
	}
	// 使用unsafe转换，避免内存拷贝
	return *(*string)(unsafe.Pointer(&b))
}
