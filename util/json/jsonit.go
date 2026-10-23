// Package json JSON处理工具包，基于jsoniter提供高性能的JSON序列化和反序列化
// @Author: yinwei
// @File: jsonit
// @Version: 1.0.0
// @Date: 2025/4/14 14:41

package json

import (
	"reflect"

	jsoniter "github.com/json-iterator/go"
)

// json 使用jsoniter的标准库兼容配置，提供更好的性能
var json = jsoniter.ConfigCompatibleWithStandardLibrary

// RegisterExtension 注册jsoniter扩展
// 参数:
//   - extension: 要注册的jsoniter扩展
func RegisterExtension(extension jsoniter.Extension) {
	json.RegisterExtension(extension)
}

// Json 自定义JSON扩展，用于兼容JSON字符串类型与Go数字类型之间的转换
// 继承自jsoniter.DummyExtension，提供类型转换功能
type Json struct {
	jsoniter.DummyExtension
}

// UpdateStructDescriptor 更新结构体描述符，为不同类型的字段设置相应的解码器
// 支持各种基本类型（int、float、string等）及其指针类型、复合类型（slice、map、struct）
// 参数:
//   - structDescriptor: 结构体描述符，包含字段信息
func (ext *Json) UpdateStructDescriptor(structDescriptor *jsoniter.StructDescriptor) {
	var fieldType reflect.Kind      // 字段的类型
	var isPtr bool                  // 是否为指针类型
	var decoder jsoniter.ValDecoder // 值解码器
	// 遍历结构体的所有字段
	for _, field := range structDescriptor.Fields {
		isPtr = false
		// 判断字段是否为指针类型
		if field.Field.Type().Kind() == reflect.Ptr {
			fieldType = field.Field.Type().Type1().Elem().Kind()
			isPtr = true
		} else {
			fieldType = field.Field.Type().Kind()
		}
		// 根据字段类型选择相应的解码器
		switch fieldType {
		case reflect.Int: // int类型
			if isPtr {
				decoder = &PtrIntDecoder{}
			} else {
				decoder = &IntDecoder{}
			}

		case reflect.Int8: // int8类型
			if isPtr {
				decoder = &PtrInt8Decoder{}
			} else {
				decoder = &Int8Decoder{}
			}
		case reflect.Int16: // int16类型
			if isPtr {
				decoder = &PtrInt16Decoder{}
			} else {
				decoder = &Int16Decoder{}
			}
		case reflect.Int32: // int32类型
			if isPtr {
				decoder = &PtrInt32Decoder{}
			} else {
				decoder = &Int32Decoder{}
			}
		case reflect.Int64: // int64类型
			if isPtr {
				decoder = &PtrInt64Decoder{}
			} else {
				decoder = &Int64Decoder{}
			}
		case reflect.Uint: // uint类型
			if isPtr {
				decoder = &PtrUintDecoder{}
			} else {
				decoder = &UintDecoder{}
			}
		case reflect.Uint8: // uint8类型
			if isPtr {
				decoder = &PtrUint8Decoder{}
			} else {
				decoder = &Uint8Decoder{}
			}
		case reflect.Uint16: // uint16类型
			if isPtr {
				decoder = &PtrUint16Decoder{}
			} else {
				decoder = &Uint16Decoder{}
			}
		case reflect.Uint32: // uint32类型
			if isPtr {
				decoder = &PtrUint32Decoder{}
			} else {
				decoder = &Uint32Decoder{}
			}
		case reflect.Uint64: // uint64类型
			if isPtr {
				decoder = &PtrUint64Decoder{}
			} else {
				decoder = &Uint64Decoder{}
			}
		case reflect.Float64: // float64类型
			if isPtr {
				decoder = &PtrFloat64Decoder{}
			} else {
				decoder = &Float64Decoder{}
			}
		case reflect.Float32: // float32类型
			if isPtr {
				decoder = &PtrFloat32Decoder{}
			} else {
				decoder = &Float32Decoder{}
			}
		case reflect.String: // string类型
			if isPtr {
				decoder = &PtrStrDecoder{}
			} else {
				decoder = &StrDecoder{}
			}
		case reflect.Struct: // struct结构体类型
			if isPtr {
				decoder = &PtrStructDecoder{}
			} else {
				decoder = &StructDecoder{}
			}
		case reflect.Array, reflect.Slice: // 数组和切片类型
			if isPtr {
				decoder = &PtrArrayDecoder{elemType: field.Field.Type().Type1()}
			} else {
				decoder = &ArrayDecoder{elemType: field.Field.Type().Type1()}
			}
		case reflect.Map: // map类型
			if isPtr {
				decoder = &PtrMapDecoder{
					elemType: field.Field.Type().Type1(),
				}
			} else {
				decoder = &MapDecoder{
					elemType: field.Field.Type().Type1(),
				}
			}
		default: // 未支持的类型跳过
			continue
		}
		// 将解码器分配给字段
		field.Decoder = decoder

	}
}
