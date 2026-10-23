package json

import (
	"reflect"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

// StructDecoder 处理结构体类型
type StructDecoder struct{}

func (decoder *StructDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// 处理结构体类型的解码逻辑
	if iter.WhatIsNext() == jsoniter.StringValue {
		// 如果是字符串，先解析成字符串
		str := iter.ReadString()
		// 然后尝试将字符串解析为结构体
		if err := json.UnmarshalFromString(str, (*interface{})(ptr)); err != nil {
			iter.Error = err
		}
	} else {
		// 直接解析结构体
		iter.ReadVal((*interface{})(ptr))
	}
}

// PtrStructDecoder 处理指针结构体类型
type PtrStructDecoder struct{}

func (decoder *PtrStructDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	// 处理指针结构体类型的解码逻辑
	if iter.WhatIsNext() == jsoniter.NilValue {
		iter.ReadNil()
		*((*unsafe.Pointer)(ptr)) = nil
		return
	}
	// 创建新的结构体实例
	newPtr := *((*unsafe.Pointer)(ptr))
	if newPtr == nil {
		// 如果指针为空，创建新的实例
		typ := reflect.TypeOf((*interface{})(ptr)).Elem()
		newPtr = unsafe.Pointer(reflect.New(typ).Pointer())
		*((*unsafe.Pointer)(ptr)) = newPtr
	}
	structDecoder := &StructDecoder{}
	structDecoder.Decode(newPtr, iter)
}

// ArrayDecoder 处理数组/切片类型
type ArrayDecoder struct {
	elemType reflect.Type
}

func (decoder *ArrayDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	if iter.WhatIsNext() == jsoniter.StringValue {
		str := iter.ReadString()

		// 使用预设的类型信息
		val := reflect.NewAt(decoder.elemType, ptr).Elem()
		elemType := decoder.elemType.Elem()
		isPtr := elemType.Kind() == reflect.Ptr
		if isPtr {
			elemType = elemType.Elem() // 获取指针指向的类型
		}

		// 根据元素类型进行不同的处理
		switch elemType.Kind() {
		case reflect.String:
			// 特殊处理字符串数组
			var strArray []string
			if err := json.UnmarshalFromString(str, &strArray); err != nil {
				iter.Error = err
				return
			}
			// 创建新的切片
			newSlice := reflect.MakeSlice(val.Type(), len(strArray), len(strArray))
			for i, s := range strArray {
				if isPtr {
					// 创建指针并设置值
					ptrValue := reflect.New(elemType)
					ptrValue.Elem().SetString(s)
					newSlice.Index(i).Set(ptrValue)
				} else {
					newSlice.Index(i).SetString(s)
				}
			}
			val.Set(newSlice)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// 处理整数类型
			var numbers []int64
			if err := json.UnmarshalFromString(str, &numbers); err != nil {
				iter.Error = err
				return
			}

			// 创建新的切片
			newSlice := reflect.MakeSlice(val.Type(), len(numbers), len(numbers))
			for i, num := range numbers {
				if isPtr {
					// 创建指针并设置值
					ptrValue := reflect.New(elemType)
					ptrValue.Elem().Set(reflect.ValueOf(num).Convert(elemType))
					newSlice.Index(i).Set(ptrValue)
				} else {
					newSlice.Index(i).Set(reflect.ValueOf(num).Convert(elemType))
				}
			}
			val.Set(newSlice)

		case reflect.Float32, reflect.Float64:
			// 处理浮点数类型
			var numbers []float64
			if err := json.UnmarshalFromString(str, &numbers); err != nil {
				iter.Error = err
				return
			}

			// 创建新的切片
			newSlice := reflect.MakeSlice(val.Type(), len(numbers), len(numbers))
			for i, num := range numbers {
				if isPtr {
					// 创建指针并设置值
					ptrValue := reflect.New(elemType)
					ptrValue.Elem().Set(reflect.ValueOf(num).Convert(elemType))
					newSlice.Index(i).Set(ptrValue)
				} else {
					newSlice.Index(i).Set(reflect.ValueOf(num).Convert(elemType))
				}
			}
			val.Set(newSlice)

		default:
			// 其他类型直接尝试解析
			if err := json.UnmarshalFromString(str, val.Interface()); err != nil {
				iter.Error = err
			}
		}
	} else {
		// 如果不是字符串，直接解析
		val := reflect.NewAt(decoder.elemType, ptr)
		iter.ReadVal(val.Interface())
	}
}

// PtrArrayDecoder 处理指针数组/切片类型
type PtrArrayDecoder struct {
	elemType reflect.Type
}

func (decoder *PtrArrayDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	if iter.WhatIsNext() == jsoniter.NilValue {
		iter.ReadNil()
		*((*unsafe.Pointer)(ptr)) = nil
		return
	}

	// 使用预设的类型信息
	val := reflect.NewAt(decoder.elemType, ptr).Elem()
	if val.IsNil() {
		// 如果是 nil，创建新的切片
		val.Set(reflect.New(decoder.elemType).Elem())
	}

	arrayDecoder := &ArrayDecoder{elemType: decoder.elemType}
	arrayDecoder.Decode(ptr, iter)
}

// MapDecoder 处理映射类型
type MapDecoder struct {
	elemType reflect.Type
}

func (decoder *MapDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	if iter.WhatIsNext() == jsoniter.StringValue {
		str := iter.ReadString()
		if str == "" || str == "null" {
			// 处理空值
			*((*unsafe.Pointer)(ptr)) = nil
			return
		}

		// 使用预设的类型信息
		val := reflect.NewAt(decoder.elemType, ptr)
		if val.Elem().IsNil() {
			val.Elem().Set(reflect.MakeMap(decoder.elemType))
		}

		// 使用全局 json 变量解析字符串
		if err := json.UnmarshalFromString(str, val.Interface()); err != nil {
			iter.Error = err
		}
	} else {
		// 如果不是字符串，使用预设的类型信息
		val := reflect.NewAt(decoder.elemType, ptr)
		if val.Elem().IsNil() {
			val.Elem().Set(reflect.MakeMap(decoder.elemType))
		}
		// 传入指针
		iter.ReadVal(val.Interface())
	}
}

// PtrMapDecoder 处理指针映射类型
type PtrMapDecoder struct {
	elemType reflect.Type
}

func (decoder *PtrMapDecoder) Decode(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
	if iter.WhatIsNext() == jsoniter.NilValue {
		iter.ReadNil()
		*((*unsafe.Pointer)(ptr)) = nil
		return
	}

	// 创建新的 map 实例
	val := reflect.NewAt(decoder.elemType, ptr)
	if val.Elem().IsNil() {
		val.Elem().Set(reflect.MakeMap(decoder.elemType))
	}

	mapDecoder := &MapDecoder{elemType: decoder.elemType}
	mapDecoder.Decode(ptr, iter)
}
