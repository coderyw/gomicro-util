// Package system 系统工具包
// @Author: yinwei
// @File: name
// @Version: 1.0.0
// @Date: 2024/7/3 16:08

// Package system 提供系统相关的工具函数，包括用户名处理等功能
package system

import (
	"errors"
	"strconv"
	"strings"
)

// pz PZ用户名前缀常量
const pz = "PZ"

// GetPzName 根据用户ID生成PZ格式的用户名
// 生成格式: "PZ" + 用户ID
// 参数:
//   - id: 用户ID
//
// 返回值:
//   - string: 生成的PZ格式用户名，例如: "PZ123456"
//
// @Author yw 2024-07-03 16:11:11
func GetPzName(id int64) string {
	// 使用StringBuilder高效拼接字符串
	builder := strings.Builder{}
	// 添加PZ前缀
	builder.WriteString(pz)
	// 添加用户ID（转换为十进制字符串）
	builder.WriteString(strconv.FormatInt(id, 10))
	return builder.String()
}

// GetUserIdFromPzName 从PZ格式的用户名中解析出用户ID
// 解析格式: "PZ123456" -> 123456
// 参数:
//   - name: PZ格式的用户名
//
// 返回值:
//   - int64: 解析出的用户ID
//   - error: 解析过程中的错误，如果用户名格式不正确
//
// @Author yw 2024-07-03 17:34:58
func GetUserIdFromPzName(name string) (int64, error) {
	// 检查用户名长度是否有效（至少需要"PZ"前缀加一位数字）
	if len(name) < 3 {
		return 0, errors.New("invalid pz name")
	}
	// 提取除"PZ"前缀外的ID部分
	idStr := name[2:]
	// 将字符串转换为int64类型的用户ID
	return strconv.ParseInt(idStr, 10, 64)
}
