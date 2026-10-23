/**
 * @Author: yinwei
 * @Description: 字符串工具包，提供字符串和字节数组之间的零拷贝转换
 * @File: strings
 * @Version: 1.0.0
 * @Date: 2023/10/24 16:40
 */
package util

import "unsafe"

// Str2Bytes 将字符串转换为字节数组，使用unsafe包进行零拷贝转换
// 注意：这种转换方式虽然性能高，但存在一定风险，需要确保返回的字节数组不被修改
func Str2Bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// Bytes2Str 将字节数组转换为字符串，不进行内存分配的零拷贝转换
// 注意：这种转换方式依赖于Go内部实现，在未来版本中可能会失效
// 如果字符串或切片的头部结构发生变化，此方法可能会出现问题
func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
