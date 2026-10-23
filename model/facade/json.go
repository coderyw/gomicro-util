// Package facade
// @Author: yinwei
// @File: json
// @Version: 1.0.0
// @Date: 2023/11/9 16:45

package facade

import "encoding/json"

type Marshal interface {
	json.Marshaler
	json.Unmarshaler
}
