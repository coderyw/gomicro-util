// Package json
// @Author: yinwei
// @File: decode_int
// @Version: 1.0.0
// @Date: 2025/4/23 16:47

package json

import (
	jsoniter "github.com/json-iterator/go"
	"strconv"
	"unsafe"
)

type IntDecoder struct{}

func (d *IntDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("IntDecoder", "invalid int string: "+strVal)
				return
			}
			*(*int)(ptr) = intVal
		}
	case jsoniter.NumberValue:
		*(*int)(ptr) = iter.ReadInt()
	default:
		iter.ReportError("IntDecoder", "unsupported value type for int")
	}
}

type PtrIntDecoder struct{}

func (d *PtrIntDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("PtrIntDecoder", "invalid int string: "+strVal)
				return
			}
			*(**int)(ptr) = &intVal
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadInt()
		*(**int)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**int)(ptr) = nil
	default:
		iter.ReportError("PtrIntDecoder", "unsupported value type for int")
	}
}

type Int8Decoder struct{}

func (d *Int8Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("Int8Decoder", "invalid int8 string: "+strVal)
				return
			}
			*(*int8)(ptr) = int8(intVal)
		}
	case jsoniter.NumberValue:
		*(*int8)(ptr) = iter.ReadInt8()
	default:
		iter.ReportError("Int8Decoder", "unsupported value type for int")
	}
}

type PtrInt8Decoder struct{}

func (d *PtrInt8Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("PtrInt8Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := int8(intVal)
			*(**int8)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadInt8()
		*(**int8)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**int8)(ptr) = nil
	default:
		iter.ReportError("PtrInt8Decoder", "unsupported value type for int")
	}
}

type Int16Decoder struct{}

func (d *Int16Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("Int16Decoder", "invalid int16 string: "+strVal)
				return
			}
			*(*int16)(ptr) = int16(intVal)
		}
	case jsoniter.NumberValue:
		*(*int16)(ptr) = iter.ReadInt16()
	default:
		iter.ReportError("Int16Decoder", "unsupported value type for int")
	}
}

type PtrInt16Decoder struct{}

func (d *PtrInt16Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("PtrInt16Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := int16(intVal)
			*(**int16)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadInt16()
		*(**int16)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**int16)(ptr) = nil
	default:
		iter.ReportError("PtrInt16Decoder", "unsupported value type for int")
	}
}

type Int32Decoder struct{}

func (d *Int32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("Int32Decoder", "invalid int32 string: "+strVal)
				return
			}
			*(*int32)(ptr) = int32(intVal)
		}
	case jsoniter.NumberValue:
		*(*int32)(ptr) = iter.ReadInt32()
	default:
		iter.ReportError("Int32Decoder", "unsupported value type for int")
	}
}

type PtrInt32Decoder struct{}

func (d *PtrInt32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("PtrInt32Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := int32(intVal)
			*(**int32)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadInt32()
		*(**int32)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**int32)(ptr) = nil
	default:
		iter.ReportError("PtrInt32Decoder", "unsupported value type for int")
	}
}

type Int64Decoder struct{}

func (d *Int64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("Int64Decoder", "invalid int64 string: "+strVal)
				return
			}
			*(*int64)(ptr) = int64(intVal)
		}
	case jsoniter.NumberValue:
		*(*int64)(ptr) = iter.ReadInt64()
	default:
		iter.ReportError("Int64Decoder", "unsupported value type for int")
	}
}

type PtrInt64Decoder struct{}

func (d *PtrInt64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.Atoi(strVal)
			if err != nil {
				iter.ReportError("PtrInt64Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := int64(intVal)
			*(**int64)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadInt64()
		*(**int64)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**int64)(ptr) = nil
	default:
		iter.ReportError("PtrInt64Decoder", "unsupported value type for int")
	}
}

type UintDecoder struct{}

func (d *UintDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("UintDecoder", "invalid uint string: "+strVal)
				return
			}
			*(*uint)(ptr) = uint(intVal)
		}
	case jsoniter.NumberValue:
		*(*uint)(ptr) = iter.ReadUint()
	default:
		iter.ReportError("UintDecoder", "unsupported value type for uint")
	}
}

type PtrUintDecoder struct{}

func (d *PtrUintDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("PtrUintDecoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := uint(intVal)
			*(**uint)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadUint()
		*(**uint)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**uint)(ptr) = nil
	default:
		iter.ReportError("PtrUintDecoder", "unsupported value type for int")
	}
}

type Uint8Decoder struct{}

func (d *Uint8Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("Uint8Decoder", "invalid uint8 string: "+strVal)
				return
			}
			*(*uint8)(ptr) = uint8(intVal)
		}
	case jsoniter.NumberValue:
		*(*uint8)(ptr) = iter.ReadUint8()
	default:
		iter.ReportError("Uint8Decoder", "unsupported value type for uint8")
	}
}

type PtrUint8Decoder struct{}

func (d *PtrUint8Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("PtrUint8Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := uint8(intVal)
			*(**uint8)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadUint8()
		*(**uint8)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**uint8)(ptr) = nil
	default:
		iter.ReportError("PtrUint8Decoder", "unsupported value type for int")
	}
}

type Uint16Decoder struct{}

func (d *Uint16Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("Uint16Decoder", "invalid uint16 string: "+strVal)
				return
			}
			*(*uint16)(ptr) = uint16(intVal)
		}
	case jsoniter.NumberValue:
		*(*uint16)(ptr) = iter.ReadUint16()
	default:
		iter.ReportError("Uint16Decoder", "unsupported value type for uint16")
	}
}

type PtrUint16Decoder struct{}

func (d *PtrUint16Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("PtrUint16Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := uint16(intVal)
			*(**uint16)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadUint16()
		*(**uint16)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**uint16)(ptr) = nil
	default:
		iter.ReportError("PtrUint16Decoder", "unsupported value type for int")
	}
}

type Uint32Decoder struct{}

func (d *Uint32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("Uint32Decoder", "invalid uint32 string: "+strVal)
				return
			}
			*(*uint32)(ptr) = uint32(intVal)
		}
	case jsoniter.NumberValue:
		*(*uint32)(ptr) = iter.ReadUint32()
	default:
		iter.ReportError("Uint32Decoder", "unsupported value type for uint32")
	}
}

type PtrUint32Decoder struct{}

func (d *PtrUint32Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("PtrUint32Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := uint32(intVal)
			*(**uint32)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadUint32()
		*(**uint32)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**uint32)(ptr) = nil
	default:
		iter.ReportError("PtrUint32Decoder", "unsupported value type for int")
	}
}

type Uint64Decoder struct{}

func (d *Uint64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("Uint64Decoder", "invalid uint64 string: "+strVal)
				return
			}
			*(*uint64)(ptr) = uint64(intVal)
		}
	case jsoniter.NumberValue:
		*(*uint64)(ptr) = iter.ReadUint64()
	default:
		iter.ReportError("Uint64Decoder", "unsupported value type for uint64")
	}
}

type PtrUint64Decoder struct{}

func (d *PtrUint64Decoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	switch iter.WhatIsNext() {
	case jsoniter.StringValue:
		strVal := iter.ReadString()
		if len(strVal) > 0 {
			intVal, err := strconv.ParseUint(strVal, 10, 64)
			if err != nil {
				iter.ReportError("PtrUint64Decoder", "invalid int8 string: "+strVal)
				return
			}
			tmp := uint64(intVal)
			*(**uint64)(ptr) = &tmp
		}
	case jsoniter.NumberValue:
		tmp := iter.ReadUint64()
		*(**uint64)(ptr) = &tmp
	case jsoniter.NilValue:
		iter.ReadNil()
		*(**uint64)(ptr) = nil
	default:
		iter.ReportError("PtrUint64Decoder", "unsupported value type for int")
	}
}
