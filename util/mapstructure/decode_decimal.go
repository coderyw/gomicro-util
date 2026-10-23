// Package mapstructure
// @Author: yinwei
// @File: decode_decimal
// @Version: 1.0.0
// @Date: 2025/4/23 09:18

package mapstructure

import (
	"fmt"
	"github.com/shopspring/decimal"
	"reflect"
)

func parseDecimal(field reflect.Value, rawValue interface{}) error {
	var (
		de  decimal.Decimal
		err error
	)
	switch v := rawValue.(type) {
	case string:
		de, err = decimal.NewFromString(v)
	case float64:
		de = decimal.NewFromFloat(v)
	case float32:
		de = decimal.NewFromFloat32(v)
	case int:
		de = decimal.NewFromInt(int64(v))
	case int8:
		de = decimal.NewFromInt32(int32(v))
	case int16:
		de = decimal.NewFromInt32(int32(v))
	case int32:
		de = decimal.NewFromInt32(int32(v))
	case int64:
		de = decimal.NewFromInt(v)
	case uint:
		de = decimal.NewFromUint64(uint64(v))
	case uint8:
		de = decimal.NewFromUint64(uint64(v))
	case uint16:
		de = decimal.NewFromUint64(uint64(v))
	case uint32:
		de = decimal.NewFromUint64(uint64(v))
	case uint64:
		de = decimal.NewFromUint64(uint64(v))
	default:
		return fmt.Errorf("decimal 格式化不支持类型：%v", v)
	}
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(de))
	return nil
}
