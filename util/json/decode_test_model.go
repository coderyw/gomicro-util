// Package json
// @Author: yinwei
// @File: decode_test_model
// @Version: 1.0.0
// @Date: 2025/4/23 17:14

package json

type TestAll struct {
	IntField       int  `json:"int_field"`         // 解析整型到整型
	IntPtrField    *int `json:"int_ptr_field"`     // 解析整型到ptr 整型
	IntStrField    int  `json:"int_str_field"`     // 解析字符串到整型
	IntStrPtrField *int `json:"int_str_ptr_field"` // 解析字符串到 ptr 整型
	IntPtrNilField *int `json:"int_ptr_nil_field"` // 解析空值到 ptr 整型

	Int8Field       int8  `json:"int8_field"`
	Int8PtrField    *int8 `json:"int8_ptr_field"`
	Int8StrField    int8  `json:"int8_str_field"`
	Int8StrPtrField *int8 `json:"int8_str_ptr_field"`
	Int8PtrNilField *int8 `json:"int8_ptr_nil_field"` //空值

	Int16Field       int16  `json:"int16_field"`
	Int16PtrField    *int16 `json:"int16_ptr_field"`
	Int16StrField    int16  `json:"int16_str_field"`
	Int16StrPtrField *int16 `json:"int16_str_ptr_field"`
	Int16PtrNilField *int16 `json:"int16_ptr_nil_field"` //空值

	Int32Field       int32  `json:"int32_field"`
	Int32PtrField    *int32 `json:"int32_ptr_field"`
	Int32StrField    int32  `json:"int32_str_field"`
	Int32StrPtrField *int32 `json:"int32_str_ptr_field"`
	Int32PtrNilField *int32 `json:"int32_ptr_nil_field"` //空值

	Int64Field       int64  `json:"int64_field"`
	Int64PtrField    *int64 `json:"int64_ptr_field"`
	Int64StrField    int64  `json:"int64_str_field"`
	Int64StrPtrField *int64 `json:"int64_str_ptr_field"`
	Int64PtrNilField *int64 `json:"int64_ptr_nil_field"` //空值

	UintField       uint  `json:"uint_field"`
	UintPtrField    *uint `json:"uint_ptr_field"`
	UintStrField    uint  `json:"uint_str_field"`
	UintStrPtrField *uint `json:"uint_str_ptr_field"`
	UintPtrNilField *uint `json:"uint_ptr_nil_field"` //空值

	Uint8Field       uint8  `json:"uint8_field"`
	Uint8PtrField    *uint8 `json:"uint8_ptr_field"`
	Uint8StrField    uint8  `json:"uint8_str_field"`
	Uint8StrPtrField *uint8 `json:"uint8_str_ptr_field"`
	Uint8PtrNilField *uint8 `json:"uint8_ptr_nil_field"` //空值

	Uint16Field       uint16  `json:"uint16_field"`
	Uint16PtrField    *uint16 `json:"uint16_ptr_field"`
	Uint16StrField    uint16  `json:"uint16_str_field"`
	Uint16StrPtrField *uint16 `json:"uint16_str_ptr_field"`
	Uint16PtrNilField *uint16 `json:"uint16_ptr_nil_field"` //空值

	Uint32Field       uint32  `json:"uint32_field"`
	Uint32PtrField    *uint32 `json:"uint32_ptr_field"`
	Uint32StrField    uint32  `json:"uint32_str_field"`
	Uint32StrPtrField *uint32 `json:"uint32_str_ptr_field"`
	Uint32PtrNilField *uint32 `json:"uint32_ptr_nil_field"` //空值

	Uint64Field       uint64  `json:"uint64_field"`
	Uint64PtrField    *uint64 `json:"uint64_ptr_field"`
	Uint64StrField    uint64  `json:"uint64_str_field"`
	Uint64StrPtrField *uint64 `json:"uint64_str_ptr_field"`
	Uint64PtrNilField *uint64 `json:"uint64_ptr_nil_field"` //空值

	Float64Field       float64  `json:"float64_field"`
	Float64PtrField    *float64 `json:"float64_ptr_field"`
	Float64StrField    float64  `json:"float64_str_field"`
	Float64StrPtrField *float64 `json:"float64_str_ptr_field"`
	Float64PtrNilField *float64 `json:"float64_ptr_nil_field"` //空值

	Float32Field       float32  `json:"float32_field"`
	Float32PtrField    *float32 `json:"float32_ptr_field"`
	Float32StrField    float32  `json:"float32_str_field"`
	Float32StrPtrField *float32 `json:"float32_str_ptr_field"`
	Float32PtrNilField *float32 `json:"float32_ptr_nil_field"` //空值

	StringField       string  `json:"string_field"`
	StringPtrField    *string `json:"string_ptr_field"`
	StringStrField    string  `json:"string_str_field"`
	StringStrPtrField *string `json:"string_str_ptr_field"`
	StringPtrNilField *string `json:"string_ptr_nil_field"` //空值
}

type TestSub struct {
	IntField       int  `json:"int_field"`
	IntPtrField    *int `json:"int_ptr_field"`
	IntStrField    int  `json:"int_str_field"`
	IntStrPtrField *int `json:"int_str_ptr_field"`
	IntPtrNilField *int `json:"int_ptr_nil_field"` //空值

	Int8Field       int8  `json:"int8_field"`
	Int8PtrField    *int8 `json:"int8_ptr_field"`
	Int8StrField    int8  `json:"int8_str_field"`
	Int8StrPtrField *int8 `json:"int8_str_ptr_field"`
	Int8PtrNilField *int8 `json:"int8_ptr_nil_field"` //空值

	Int16Field       int16  `json:"int16_field"`
	Int16PtrField    *int16 `json:"int16_ptr_field"`
	Int16StrField    int16  `json:"int16_str_field"`
	Int16StrPtrField *int16 `json:"int16_str_ptr_field"`
	Int16PtrNilField *int16 `json:"int16_ptr_nil_field"` //空值

	Int32Field       int32  `json:"int32_field"`
	Int32PtrField    *int32 `json:"int32_ptr_field"`
	Int32StrField    int32  `json:"int32_str_field"`
	Int32StrPtrField *int32 `json:"int32_str_ptr_field"`
	Int32PtrNilField *int32 `json:"int32_ptr_nil_field"` //空值

	Int64Field       int64  `json:"int64_field"`
	Int64PtrField    *int64 `json:"int64_ptr_field"`
	Int64StrField    int64  `json:"int64_str_field"`
	Int64StrPtrField *int64 `json:"int64_str_ptr_field"`
	Int64PtrNilField *int64 `json:"int64_ptr_nil_field"` //空值

	UintField       uint  `json:"uint_field"`
	UintPtrField    *uint `json:"uint_ptr_field"`
	UintStrField    uint  `json:"uint_str_field"`
	UintStrPtrField *uint `json:"uint_str_ptr_field"`
	UintPtrNilField *uint `json:"uint_ptr_nil_field"` //空值

	Uint8Field       uint8  `json:"uint8_field"`
	Uint8PtrField    *uint8 `json:"uint8_ptr_field"`
	Uint8StrField    uint8  `json:"uint8_str_field"`
	Uint8StrPtrField *uint8 `json:"uint8_str_ptr_field"`
	Uint8PtrNilField *uint8 `json:"uint8_ptr_nil_field"` //空值

	Uint16Field       uint16  `json:"uint16_field"`
	Uint16PtrField    *uint16 `json:"uint16_ptr_field"`
	Uint16StrField    uint16  `json:"uint16_str_field"`
	Uint16StrPtrField *uint16 `json:"uint16_str_ptr_field"`
	Uint16PtrNilField *uint16 `json:"uint16_ptr_nil_field"` //空值

	Uint32Field       uint32  `json:"uint32_field"`
	Uint32PtrField    *uint32 `json:"uint32_ptr_field"`
	Uint32StrField    uint32  `json:"uint32_str_field"`
	Uint32StrPtrField *uint32 `json:"uint32_str_ptr_field"`
	Uint32PtrNilField *uint32 `json:"uint32_ptr_nil_field"` //空值

	Uint64Field       uint64  `json:"uint64_field"`
	Uint64PtrField    *uint64 `json:"uint64_ptr_field"`
	Uint64StrField    uint64  `json:"uint64_str_field"`
	Uint64StrPtrField *uint64 `json:"uint64_str_ptr_field"`
	Uint64PtrNilField *uint64 `json:"uint64_ptr_nil_field"` //空值

	Float64Field       float64  `json:"float64_field"`
	Float64PtrField    *float64 `json:"float64_ptr_field"`
	Float64StrField    float64  `json:"float64_str_field"`
	Float64StrPtrField *float64 `json:"float64_str_ptr_field"`
	Float64PtrNilField *float64 `json:"float64_ptr_nil_field"` //空值

	Float32Field       float32  `json:"float32_field"`
	Float32PtrField    *float32 `json:"float32_ptr_field"`
	Float32StrField    float32  `json:"float32_str_field"`
	Float32StrPtrField *float32 `json:"float32_str_ptr_field"`
	Float32PtrNilField *float32 `json:"float32_ptr_nil_field"` //空值

	StringField       string  `json:"string_field"`
	StringPtrField    *string `json:"string_ptr_field"`
	StringStrField    string  `json:"string_str_field"`
	StringStrPtrField *string `json:"string_str_ptr_field"`
	StringPtrNilField *string `json:"string_ptr_nil_field"` //空值
}
