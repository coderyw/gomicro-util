// Package mapstructure
// @Author: yinwei
// @File: decode_string
// @Version: 1.0.0
// @Date: 2025/4/23 11:42

package mapstructure

import (
	"fmt"
	"reflect"
	"strconv"
)

// 解析字符串，支持 string 和数字类型
func parseString(field reflect.Value, rawValue interface{}) error {
	var (
		du  string
		err error
	)
	switch v := rawValue.(type) {
	case string:
		du = v
	case float64:
		du = strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		du = strconv.FormatFloat(float64(v), 'f', -1, 64)
	case int:
		du = strconv.FormatInt(int64(v), 10)
	case int8:
		du = strconv.FormatInt(int64(v), 10)
	case int16:
		du = strconv.FormatInt(int64(v), 10)
	case int32:
		du = strconv.FormatInt(int64(v), 10)
	case int64:
		du = strconv.FormatInt(int64(v), 10)
	case uint:
		du = strconv.FormatUint(uint64(v), 10)
	case uint8:
		du = strconv.FormatUint(uint64(v), 10)
	case uint16:
		du = strconv.FormatUint(uint64(v), 10)
	case uint32:
		du = strconv.FormatUint(uint64(v), 10)
	case uint64:
		du = strconv.FormatUint(uint64(v), 10)
	default:
		return fmt.Errorf("string 格式化不支持类型：%v", v)
	}
	if err != nil {
		return err
	}
	field.SetString(du)

	return nil
}
