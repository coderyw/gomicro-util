// Package mapstructure
// @Author: yinwei
// @File: decode
// @Version: 1.0.0
// @Date: 2025/4/11 09:41
//
// 支持将 map[string]string 和 map[string]interface{},以及 struct，json 转为 struct
//

package mapstructure

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	"reflect"
	"time"
)

var (
	TIME_TYPE    = reflect.TypeOf(time.Time{})
	DECIMAL_TYPE = reflect.TypeOf(decimal.Decimal{})
)

// 自定义解码钩子函数
var hook = func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	// 优先处理 decimal.Decimal 类型
	if t == reflect.TypeOf(decimal.Decimal{}) {
		return decimal.NewFromString(fmt.Sprint(data)) // 字符串转 decimal
	} else if t.Kind() == reflect.Struct {
		return data, nil
	}

	// 处理所有数值类型的字符串转换
	if f.Kind() == reflect.String {
		strVal := data.(string)
		switch t.Kind() {
		case reflect.String:
			return strVal, nil
		case reflect.Int8:
			var tmp int64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return int8(tmp), err
		case reflect.Int16:
			var tmp int64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return int16(tmp), err
		case reflect.Int32:
			var tmp int64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return int32(tmp), err
		case reflect.Int64:
			var tmp int64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return tmp, err
		case reflect.Uint8:
			var tmp uint64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return uint8(tmp), err
		case reflect.Uint16:
			var tmp uint64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return uint16(tmp), err
		case reflect.Uint32:
			var tmp uint64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return uint32(tmp), err
		case reflect.Uint64:
			var tmp uint64
			_, err := fmt.Sscanf(strVal, "%d", &tmp)
			return tmp, err
		case reflect.Float32:
			var tmp float64
			_, err := fmt.Sscanf(strVal, "%f", &tmp)
			return float32(tmp), err
		case reflect.Float64:
			var tmp float64
			_, err := fmt.Sscanf(strVal, "%f", &tmp)
			return tmp, err
		case reflect.Struct:
			switch t {
			case TIME_TYPE:
				return time.ParseInLocation("2006-01-02 15:04:05", strVal, time.Local)
				//case DECIMAL_TYPE:
				//	return decimal.NewFromString(strVal)
			}
		}
	}

	// 默认处理
	return data, nil
}

func Decode(data interface{}, result interface{}) error {
	// 配置解码器
	config := &mapstructure.DecoderConfig{
		DecodeHook:       hook,
		Result:           result,
		WeaklyTypedInput: true,
		TagName:          "json",
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(data)
}

var (
	TimeType    = reflect.TypeOf(time.Time{})
	DecimalType = reflect.TypeOf(decimal.Decimal{})
)

// 将 JSON 字符串解析为目标结构体
func ParseJSONToStruct(jsonStr string, out interface{}) error {
	// 反序列化为 map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return fmt.Errorf("unmarshal to map failed: %w", err)
	}
	return fillStruct(data, reflect.ValueOf(out).Elem(), reflect.TypeOf(out).Elem())

}

func fillStruct(data map[string]interface{}, outVal reflect.Value, outType reflect.Type) error {
	// 遍历结构体字段
	for i := 0; i < outVal.NumField(); i++ {
		field := outVal.Field(i)
		structField := outType.Field(i)
		jsonKey := structField.Tag.Get("json")
		if jsonKey == "" {
			jsonKey = structField.Name
		}

		rawValue, exists := data[jsonKey]
		if !exists {
			continue
		}
		if err := fillField(context.Background(), field, rawValue, structField); err != nil {
			return err
		}
	}
	return nil
}

func fillField(ctx context.Context, field reflect.Value, rawValue interface{}, structField reflect.StructField) (err error) {
	// 设置值
	if field.CanSet() {
		kind := field.Kind()
		if reflect.String == kind {
			return parseString(field, rawValue)
		} else if kind >= reflect.Int && kind <= reflect.Int64 {
			return parseInt(field, rawValue)
		} else if kind >= reflect.Uint && kind <= reflect.Uint64 {
			return parseUInt(field, rawValue)
		} else if kind >= reflect.Float32 && kind <= reflect.Float64 {
			return parseFloat(field, rawValue)
		} else if kind == reflect.Struct {
			// 其他类型可按需扩展
			return parseStruct(field, structField, rawValue)
		} else if kind == reflect.Ptr {
			ptr := reflect.New(field.Type().Elem())
			newFiled := ptr.Elem()
			err := fillField(ctx, newFiled, rawValue, structField)
			if err != nil {
				return err
			}
			field.Set(newFiled.Addr())
			return nil
		} else {

		}
	}
	return nil
}
