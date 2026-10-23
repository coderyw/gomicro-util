// Package json
// @Author: yinwei
// @File: decode_float
// @Version: 1.0.0
// @Date: 2025/4/23 17:09

package json

import (
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"unsafe"
)

type Float64Decoder struct{}

func (d *Float64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				iter.ReportError("Float64Decoder", "invalid float64 string: "+strVal)
				return
			}
			*(*float64)(ptr) = float64(intVal)
		}
	case jsoniter.NumberValue:
		*(*float64)(ptr) = iter.ReadFloat64()
	default:
		iter.ReportError("Float64Decoder", "unsupported value type for float64")
	}
}

type PtrFloat64Decoder struct{}

func (d *PtrFloat64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				iter.ReportError("PtrFloat64Decoder", "invalid float64 string: "+strVal)
				return
			}
			tmp := float64(intVal)
			*(**float64)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadFloat64()
		*(**float64)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**float64)(ptr) = nil
	default:
		iter.ReportError("PtrUint64Decoder", "unsupported value type for int")
	}
}

type Float32Decoder struct{}

func (d *Float32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				iter.ReportError("Float32Decoder", "invalid float64 string: "+strVal)
				return
			}
			*(*float32)(ptr) = float32(intVal)
		}
	case jsoniter.NumberValue:
		*(*float32)(ptr) = iter.ReadFloat32()
	default:
		iter.ReportError("Float32Decoder", "unsupported value type for float64")
	}
}

type PtrFloat32Decoder struct{}

func (d *PtrFloat32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				iter.ReportError("PtrFloat32Decoder", "invalid float64 string: "+strVal)
				return
			}
			tmp := float32(intVal)
			*(**float32)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadFloat32()
		*(**float32)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**float32)(ptr) = nil
	default:
		iter.ReportError("PtrFloat32Decoder", "unsupported value type for int")
	}
}
