// Package mapstructure
// @Author: yinwei
// @File: decode_int
// @Version: 1.0.0
// @Date: 2025/4/23 09:40

package mapstructure

import (
	"fmt"
	"reflect"
	"strconv"
)

type Integer64 interface {
	int64 | uint64
}

// 解析无符号整型，支持 string 和数字类型
func parseUInt(field reflect.Value, rawValue interface{}) error {
	var (
		du  uint64
		err error
	)
	switch v := rawValue.(type) {
	case string:
		if len(v) == 0 {
			du = 0
		} else {
			du, err = strconv.ParseUint(v, 10, 64)
		}
	case float64:
		du = uint64(v)
	case float32:
		du = uint64(v)
	case int:
		du = uint64(v)
	case int8:
		du = uint64(v)
	case int16:
		du = uint64(v)
	case int32:
		du = uint64(v)
	case int64:
		du = uint64(v)
	case uint:
		du = uint64(v)
	case uint8:
		du = uint64(v)
	case uint16:
		du = uint64(v)
	case uint32:
		du = uint64(v)
	case uint64:
		du = uint64(v)
	default:
		return fmt.Errorf("uint 格式化不支持类型：%v", v)
	}
	if err != nil {
		return err
	}
	field.SetUint(du)

	return nil
}

// 解析有符号整型，支持 string 和数字类型
func parseInt(field reflect.Value, rawValue interface{}) error {
	var (
		du  int64
		err error
	)
	switch v := rawValue.(type) {
	case string:
		if len(v) == 0 {
			du = 0
		} else {
			du, err = strconv.ParseInt(v, 10, 64)
		}
	case float64:
		du = int64(v)
	case float32:
		du = int64(v)
	case int:
		du = int64(v)
	case int8:
		du = int64(v)
	case int16:
		du = int64(v)
	case int32:
		du = int64(v)
	case int64:
		du = int64(v)
	case uint:
		du = int64(v)
	case uint8:
		du = int64(v)
	case uint16:
		du = int64(v)
	case uint32:
		du = int64(v)
	case uint64:
		du = int64(v)
	default:
		return fmt.Errorf("int 格式化不支持类型：%v", v)
	}
	if err != nil {
		return err
	}
	field.SetInt(du)

	return nil
}

// 解析浮点数，支持 string 和数字类型
func parseFloat(field reflect.Value, rawValue interface{}) error {
	var (
		du  float64
		err error
	)
	switch v := rawValue.(type) {
	case string:
		if len(v) == 0 {
			du = 0
		} else {
			du, err = strconv.ParseFloat(v, 10)
		}
	case float64:
		du = float64(v)
	case float32:
		du = float64(v)
	case int:
		du = float64(v)
	case int8:
		du = float64(v)
	case int16:
		du = float64(v)
	case int32:
		du = float64(v)
	case int64:
		du = float64(v)
	case uint:
		du = float64(v)
	case uint8:
		du = float64(v)
	case uint16:
		du = float64(v)
	case uint32:
		du = float64(v)
	case uint64:
		du = float64(v)
	default:
		return fmt.Errorf("float64 格式化不支持类型：%v", v)
	}
	if err != nil {
		return err
	}
	field.SetFloat(du)

	return nil
}
