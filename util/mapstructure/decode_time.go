// Package mapstructure
// @Author: yinwei
// @File: decode_time
// @Version: 1.0.0
// @Date: 2025/4/22 10:03

package mapstructure

import (
	"fmt"
	"reflect"
	"time"
)

func parseTime(timeFormat string, field reflect.Value, rawValue interface{}) error {
	switch v := rawValue.(type) {
	case string:
		if timeFormat != "" {
			if t, err := time.Parse(timeFormat, v); err == nil {
				field.Set(reflect.ValueOf(t))
			}
		} else {
			// 默认 ISO 格式
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				field.Set(reflect.ValueOf(t))
			}
		}
	case float64:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}
	case float32:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}
	case int:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case int8:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case int16:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case int32:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case int64:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case uint:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case uint8:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case uint16:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case uint32:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}

	case uint64:
		// 支持时间戳，自动判断是否是秒或毫秒
		sec := int64(v)
		if sec > 1e12 {
			field.Set(reflect.ValueOf(time.UnixMilli(sec)))
		} else {
			field.Set(reflect.ValueOf(time.Unix(sec, 0)))
		}
	default:
		return fmt.Errorf("time 格式化不支持类型：%v", v)
	}
	return nil
}
