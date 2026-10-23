// Package json
// @Author: yinwei
// @File: decode_string
// @Version: 1.0.0
// @Date: 2025/4/23 17:04

package json

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"unsafe"
)

type StrDecoder struct{}

func (d *StrDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		*(*string)(ptr) = iter.ReadString()
	case jsoniter.NumberValue:
		strVal := iter.Read()
		*(*string)(ptr) = fmt.Sprint(strVal)
	default:
		iter.ReportError("StrDecoder", "unsupported value type for string")
	}
}

type PtrStrDecoder struct{}

func (d *PtrStrDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		str := iter.ReadString()
		*(**string)(ptr) = &str
	case jsoniter.NumberValue:
		str := fmt.Sprint(iter.Read())
		*(**string)(ptr) = &str
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**string)(ptr) = nil
	default:
		iter.ReportError("PtrStrDecoder", "unsupported value type for *string")
	}
}
