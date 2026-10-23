// Package mapstructure
// @Author: yinwei
// @File: decode_struct
// @Version: 1.0.0
// @Date: 2025/4/23 13:13

package mapstructure

import (
	"reflect"
	"time"
)

func parseStruct(field reflect.Value, structField reflect.StructField, rawValue interface{}) error {
	switch field.Type() {
	case TimeType: // 处理 time.Time 类型
		timeFormat := structField.Tag.Get("time_format")
		if len(timeFormat) == 0 {
			timeFormat = time.DateTime
		}
		return parseTime(timeFormat, field, rawValue)
	case DecimalType:
		return parseDecimal(field, rawValue)
	default:
		// 嵌套结构体
		if subMap, ok := rawValue.(map[string]interface{}); ok {
			if err := fillStruct(subMap, field, field.Type()); err != nil {
				return err
			}
		}
	}
	return nil
}
