// Package mapstructure
// @Author: yinwei
// @File: decode_slice
// @Version: 1.0.0
// @Date: 2025/4/23 13:14

package mapstructure

import (
	"fmt"
	"reflect"
)

func parseSlice(field reflect.Value, rawValue interface{}) error {
	switch v := rawValue.(type) {
	case []int:
		fmt.Println(v)
	}
	return nil

}
