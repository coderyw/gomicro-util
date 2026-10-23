package json

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type ComplexTestStruct struct {
	// 整数类型数组测试
	IntArray    []int    `json:"intArray"`
	Int8Array   []int8   `json:"int8Array"`
	Int16Array  []int16  `json:"int16Array"`
	Int32Array  []int32  `json:"int32Array"`
	Int64Array  []int64  `json:"int64Array"`
	UintArray   []uint   `json:"uintArray"`
	Uint8Array  []uint8  `json:"uint8Array"`
	Uint16Array []uint16 `json:"uint16Array"`
	Uint32Array []uint32 `json:"uint32Array"`
	Uint64Array []uint64 `json:"uint64Array"`

	// 浮点数类型数组测试
	Float32Array []float32 `json:"float32Array"`
	Float64Array []float64 `json:"float64Array"`

	// 指针类型数组测试
	PtrIntArray   []*int     `json:"ptrIntArray"`
	PtrFloatArray []*float64 `json:"ptrFloatArray"`

	// 字符串数组测试
	StringArray []string `json:"stringArray"`

	// 结构体测试
	SubStruct    SubStruct  `json:"subStruct"`
	PtrSubStruct *SubStruct `json:"ptrSubStruct"`

	// Map测试
	StringMap    map[string]string     `json:"stringMap"`
	IntMap       map[string]int        `json:"intMap"`
	StructMap    map[string]SubStruct  `json:"structMap"`
	PtrStructMap map[string]*SubStruct `json:"ptrStructMap"`
}

type SubStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestComplexDecoding(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected ComplexTestStruct
	}{
		{
			name: "字符串形式的数字数组",
			json: `{
				"intArray": "[1, 2, 3]",
				"int8Array": "[1, 2, 3]",
				"int16Array": "[1, 2, 3]",
				"int32Array": "[1, 2, 3]",
				"int64Array": "[1, 2, 3]",
				"uintArray": "[1, 2, 3]",
				"uint8Array": "[1, 2, 3]",
				"uint16Array": "[1, 2, 3]",
				"uint32Array": "[1, 2, 3]",
				"uint64Array": "[1, 2, 3]",
				"float32Array": "[1.1, 2.2, 3.3]",
				"float64Array": "[1.1, 2.2, 3.3]",
				"stringArray": ["a", "b", "c"],
				"subStruct": {"name": "test", "age": 20},
				"ptrSubStruct": {"name": "test2", "age": 30},
				"stringMap": {"key": "value"},
				"intMap": {"key": 123},
				"structMap": {"key": {"name": "test3", "age": 40}},
				"ptrStructMap": {"key": {"name": "test4", "age": 50}}
			}`,
			expected: ComplexTestStruct{
				IntArray:     []int{1, 2, 3},
				Int8Array:    []int8{1, 2, 3},
				Int16Array:   []int16{1, 2, 3},
				Int32Array:   []int32{1, 2, 3},
				Int64Array:   []int64{1, 2, 3},
				UintArray:    []uint{1, 2, 3},
				Uint8Array:   []uint8{1, 2, 3},
				Uint16Array:  []uint16{1, 2, 3},
				Uint32Array:  []uint32{1, 2, 3},
				Uint64Array:  []uint64{1, 2, 3},
				Float32Array: []float32{1.1, 2.2, 3.3},
				Float64Array: []float64{1.1, 2.2, 3.3},
				StringArray:  []string{"a", "b", "c"},
				SubStruct:    SubStruct{Name: "test", Age: 20},
				PtrSubStruct: &SubStruct{Name: "test2", Age: 30},
				StringMap:    map[string]string{"key": "value"},
				IntMap:       map[string]int{"key": 123},
				StructMap:    map[string]SubStruct{"key": {Name: "test3", Age: 40}},
				PtrStructMap: map[string]*SubStruct{"key": {Name: "test4", Age: 50}},
			},
		},
		{
			name: "常规数组形式",
			json: `{
				"intArray": [1, 2, 3],
				"int8Array": [1, 2, 3],
				"int16Array": [1, 2, 3],
				"int32Array": [1, 2, 3],
				"int64Array": [1, 2, 3],
				"uintArray": [1, 2, 3],
				"uint8Array": [1, 2, 3],
				"uint16Array": [1, 2, 3],
				"uint32Array": [1, 2, 3],
				"uint64Array": [1, 2, 3],
				"float32Array": [1.1, 2.2, 3.3],
				"float64Array": [1.1, 2.2, 3.3],
				"ptrIntArray": [1, 2, 3],
				"ptrFloatArray": [1.1, 2.2, 3.3],
				"stringArray": ["a", "b", "c"],
				"subStruct": {"name": "test", "age": 20},
				"ptrSubStruct": {"name": "test2", "age": 30},
				"stringMap": {"key": "value"},
				"intMap": {"key": 123},
				"structMap": {"key": {"name": "test3", "age": 40}},
				"ptrStructMap": {"key": {"name": "test4", "age": 50}}
			}`,
			expected: ComplexTestStruct{
				IntArray:     []int{1, 2, 3},
				Int8Array:    []int8{1, 2, 3},
				Int16Array:   []int16{1, 2, 3},
				Int32Array:   []int32{1, 2, 3},
				Int64Array:   []int64{1, 2, 3},
				UintArray:    []uint{1, 2, 3},
				Uint8Array:   []uint8{1, 2, 3},
				Uint16Array:  []uint16{1, 2, 3},
				Uint32Array:  []uint32{1, 2, 3},
				Uint64Array:  []uint64{1, 2, 3},
				Float32Array: []float32{1.1, 2.2, 3.3},
				Float64Array: []float64{1.1, 2.2, 3.3},
				StringArray:  []string{"a", "b", "c"},
				SubStruct:    SubStruct{Name: "test", Age: 20},
				PtrSubStruct: &SubStruct{Name: "test2", Age: 30},
				StringMap:    map[string]string{"key": "value"},
				IntMap:       map[string]int{"key": 123},
				StructMap:    map[string]SubStruct{"key": {Name: "test3", Age: 40}},
				PtrStructMap: map[string]*SubStruct{"key": {Name: "test4", Age: 50}},
			},
		},
		{
			name: "nil测试",
			json: `{
				"intArray": null,
				"float64Array": null,
				"ptrSubStruct": null,
				"stringMap": null,
				"ptrStructMap": null
			}`,
			expected: ComplexTestStruct{},
		},
	}

	// 注册扩展
	RegisterExtension(&Json{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ComplexTestStruct
			err := json.UnmarshalFromString(tt.json, &got)
			if err != nil {
				t.Errorf("解析错误: %v", err)
				return
			}

			// 比较结果
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("结果不匹配\n获得: %+v\n期望: %+v", got, tt.expected)
			}
		})
	}
}

// 测试边界情况
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		shouldError bool
	}{
		{
			name: "超出范围的整数",
			json: `{
				"int8Array": "[256, 257]"
			}`,
			shouldError: true,
		},
		{
			name: "无效的JSON字符串",
			json: `{
				"intArray": "[1, 2, 3"
			}`,
			shouldError: true,
		},
		{
			name: "类型不匹配",
			json: `{
				"intArray": ["a", "b", "c"]
			}`,
			shouldError: true,
		},
		{
			name: "空数组",
			json: `{
				"intArray": "[]",
				"float64Array": "[]"
			}`,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got ComplexTestStruct
			err := json.UnmarshalFromString(tt.json, &got)

			if tt.shouldError && err == nil {
				t.Error("期望出错但没有错误")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("不期望出错但出现错误: %v", err)
			}
		})
	}
}

// 添加新的测试结构体，用于测试所有基本类型
type AllTypesStruct struct {
	// 整数类型
	Int    int    `json:"int"`
	Int8   int8   `json:"int8"`
	Int16  int16  `json:"int16"`
	Int32  int32  `json:"int32"`
	Int64  int64  `json:"int64"`
	Uint   uint   `json:"uint"`
	Uint8  uint8  `json:"uint8"`
	Uint16 uint16 `json:"uint16"`
	Uint32 uint32 `json:"uint32"`
	Uint64 uint64 `json:"uint64"`

	// 浮点类型
	Float32 float32 `json:"float32"`
	Float64 float64 `json:"float64"`

	// 字符串类型
	String string `json:"string"`

	// 指针类型
	PtrInt     *int     `json:"ptrInt"`
	PtrInt8    *int8    `json:"ptrInt8"`
	PtrInt16   *int16   `json:"ptrInt16"`
	PtrInt32   *int32   `json:"ptrInt32"`
	PtrInt64   *int64   `json:"ptrInt64"`
	PtrUint    *uint    `json:"ptrUint"`
	PtrUint8   *uint8   `json:"ptrUint8"`
	PtrUint16  *uint16  `json:"ptrUint16"`
	PtrUint32  *uint32  `json:"ptrUint32"`
	PtrUint64  *uint64  `json:"ptrUint64"`
	PtrFloat32 *float32 `json:"ptrFloat32"`
	PtrFloat64 *float64 `json:"ptrFloat64"`
	PtrString  *string  `json:"ptrString"`

	// 指针元素的数组/切片
	IntPtrArray     []*int     `json:"intPtrArray"`
	Int8PtrArray    []*int8    `json:"int8PtrArray"`
	Int16PtrArray   []*int16   `json:"int16PtrArray"`
	Int32PtrArray   []*int32   `json:"int32PtrArray"`
	Int64PtrArray   []*int64   `json:"int64PtrArray"`
	UintPtrArray    []*uint    `json:"uintPtrArray"`
	Uint8PtrArray   []*uint8   `json:"uint8PtrArray"`
	Uint16PtrArray  []*uint16  `json:"uint16PtrArray"`
	Uint32PtrArray  []*uint32  `json:"uint32PtrArray"`
	Uint64PtrArray  []*uint64  `json:"uint64PtrArray"`
	Float32PtrArray []*float32 `json:"float32PtrArray"`
	Float64PtrArray []*float64 `json:"float64PtrArray"`

	// 指针类型数组测试
	PtrIntArray   []*int     `json:"ptrIntArray"`
	PtrFloatArray []*float64 `json:"ptrFloatArray"`
}

// 测试所有基本类型的字符串转换
func TestAllTypesDecoding(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected AllTypesStruct
	}{
		{
			name: "字符串形式的数字",
			json: `{
				"int": "123",
				"int8": "127",
				"int16": "32767",
				"int32": "2147483647",
				"int64": "9223372036854775807",
				"uint": "123",
				"uint8": "255",
				"uint16": "65535",
				"uint32": "4294967295",
				"uint64": "18446744073709551615",
				"float32": "3.14",
				"float64": "3.141592653589793",
				"string": "test string",
				"ptrInt": "456",
				"ptrInt8": "100",
				"ptrInt16": "1000",
				"ptrInt32": "10000",
				"ptrInt64": "100000",
				"ptrUint": "456",
				"ptrUint8": "200",
				"ptrUint16": "2000",
				"ptrUint32": "20000",
				"ptrUint64": "200000",
				"ptrFloat32": "2.718",
				"ptrFloat64": "2.718281828459045",
				"ptrString": "test ptr string",
				"intPtrArray": "[1, 2, 3]",
				"int8PtrArray": "[1, 2, 3]",
				"int16PtrArray": "[1, 2, 3]",
				"int32PtrArray": "[1, 2, 3]",
				"int64PtrArray": "[1, 2, 3]",
				"uintPtrArray": "[1, 2, 3]",
				"uint8PtrArray": "[1, 2, 3]",
				"uint16PtrArray": "[1, 2, 3]",
				"uint32PtrArray": "[1, 2, 3]",
				"uint64PtrArray": "[1, 2, 3]",
				"float32PtrArray": "[1.1, 2.2, 3.3]",
				"float64PtrArray": "[1.1, 2.2, 3.3]",
				"ptrIntArray": "[1, 2, 3]",
				"ptrFloatArray": "[1.1, 2.2, 3.3]"
			}`,
			expected: func() AllTypesStruct {
				i := 456
				i8 := int8(100)
				i16 := int16(1000)
				i32 := int32(10000)
				i64 := int64(100000)
				u := uint(456)
				u8 := uint8(200)
				u16 := uint16(2000)
				u32 := uint32(20000)
				u64 := uint64(200000)
				f32 := float32(2.718)
				f64 := 2.718281828459045
				s := "test ptr string"
				return AllTypesStruct{
					Int:             123,
					Int8:            127,
					Int16:           32767,
					Int32:           2147483647,
					Int64:           9223372036854775807,
					Uint:            123,
					Uint8:           255,
					Uint16:          65535,
					Uint32:          4294967295,
					Uint64:          18446744073709551615,
					Float32:         3.14,
					Float64:         3.141592653589793,
					String:          "test string",
					PtrInt:          &i,
					PtrInt8:         &i8,
					PtrInt16:        &i16,
					PtrInt32:        &i32,
					PtrInt64:        &i64,
					PtrUint:         &u,
					PtrUint8:        &u8,
					PtrUint16:       &u16,
					PtrUint32:       &u32,
					PtrUint64:       &u64,
					PtrFloat32:      &f32,
					PtrFloat64:      &f64,
					PtrString:       &s,
					IntPtrArray:     []*int{&i},
					Int8PtrArray:    []*int8{&i8},
					Int16PtrArray:   []*int16{&i16},
					Int32PtrArray:   []*int32{&i32},
					Int64PtrArray:   []*int64{&i64},
					UintPtrArray:    []*uint{&u},
					Uint8PtrArray:   []*uint8{&u8},
					Uint16PtrArray:  []*uint16{&u16},
					Uint32PtrArray:  []*uint32{&u32},
					Uint64PtrArray:  []*uint64{&u64},
					Float32PtrArray: []*float32{&f32},
					Float64PtrArray: []*float64{&f64},
					PtrIntArray:     []*int{&i},
					PtrFloatArray:   []*float64{&f64},
				}
			}(),
		},
		{
			name: "常规形式的数字",
			json: `{
				"int": 123,
				"int8": 127,
				"int16": 32767,
				"int32": 2147483647,
				"int64": 9223372036854775807,
				"uint": 123,
				"uint8": 255,
				"uint16": 65535,
				"uint32": 4294967295,
				"uint64": 18446744073709551615,
				"float32": 3.14,
				"float64": 3.141592653589793,
				"string": "test string",
				"ptrInt": 456,
				"ptrInt8": 100,
				"ptrInt16": 1000,
				"ptrInt32": 10000,
				"ptrInt64": 100000,
				"ptrUint": 456,
				"ptrUint8": 200,
				"ptrUint16": 2000,
				"ptrUint32": 20000,
				"ptrUint64": 200000,
				"ptrFloat32": 2.718,
				"ptrFloat64": 2.718281828459045,
				"ptrString": "test ptr string",
				"intPtrArray": [1, 2, 3],
				"int8PtrArray": [1, 2, 3],
				"int16PtrArray": [1, 2, 3],
				"int32PtrArray": [1, 2, 3],
				"int64PtrArray": [1, 2, 3],
				"uintPtrArray": [1, 2, 3],
				"uint8PtrArray": [1, 2, 3],
				"uint16PtrArray": [1, 2, 3],
				"uint32PtrArray": [1, 2, 3],
				"uint64PtrArray": [1, 2, 3],
				"float32PtrArray": [1.1, 2.2, 3.3],
				"float64PtrArray": [1.1, 2.2, 3.3]",
				"ptrIntArray": [1, 2, 3],
				"ptrFloatArray": [1.1, 2.2, 3.3]
			}`,
			expected: func() AllTypesStruct {
				i := 456
				i8 := int8(100)
				i16 := int16(1000)
				i32 := int32(10000)
				i64 := int64(100000)
				u := uint(456)
				u8 := uint8(200)
				u16 := uint16(2000)
				u32 := uint32(20000)
				u64 := uint64(200000)
				f32 := float32(2.718)
				f64 := 2.718281828459045
				s := "test ptr string"
				return AllTypesStruct{
					Int:             123,
					Int8:            127,
					Int16:           32767,
					Int32:           2147483647,
					Int64:           9223372036854775807,
					Uint:            123,
					Uint8:           255,
					Uint16:          65535,
					Uint32:          4294967295,
					Uint64:          18446744073709551615,
					Float32:         3.14,
					Float64:         3.141592653589793,
					String:          "test string",
					PtrInt:          &i,
					PtrInt8:         &i8,
					PtrInt16:        &i16,
					PtrInt32:        &i32,
					PtrInt64:        &i64,
					PtrUint:         &u,
					PtrUint8:        &u8,
					PtrUint16:       &u16,
					PtrUint32:       &u32,
					PtrUint64:       &u64,
					PtrFloat32:      &f32,
					PtrFloat64:      &f64,
					PtrString:       &s,
					IntPtrArray:     []*int{&i},
					Int8PtrArray:    []*int8{&i8},
					Int16PtrArray:   []*int16{&i16},
					Int32PtrArray:   []*int32{&i32},
					Int64PtrArray:   []*int64{&i64},
					UintPtrArray:    []*uint{&u},
					Uint8PtrArray:   []*uint8{&u8},
					Uint16PtrArray:  []*uint16{&u16},
					Uint32PtrArray:  []*uint32{&u32},
					Uint64PtrArray:  []*uint64{&u64},
					Float32PtrArray: []*float32{&f32},
					Float64PtrArray: []*float64{&f64},
					PtrIntArray:     []*int{&i},
					PtrFloatArray:   []*float64{&f64},
				}
			}(),
		},
	}

	// 注册扩展
	RegisterExtension(&Json{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AllTypesStruct
			err := json.UnmarshalFromString(tt.json, &got)
			if err != nil {
				t.Errorf("解析错误: %v", err)
				return
			}

			// 比较结果
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("结果不匹配\n获得: %+v\n期望: %+v", got, tt.expected)
			}
		})
	}
}

// 添加一个辅助函数来安全地打印结构体
func prettyPrint(v interface{}) string {
	if v == nil {
		return "nil"
	}

	// 使用反射来安全地处理结构体
	val := reflect.ValueOf(v)
	typ := val.Type()

	// 如果是指针，获取其指向的元素
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "nil"
		}
		val = val.Elem()
		typ = val.Type()
	}

	// 创建一个安全的副本
	result := make(map[string]interface{})

	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// 获取 json tag
		jsonTag := field.Tag.Get("json")
		if jsonTag == "" {
			jsonTag = field.Name
		}
		if idx := strings.Index(jsonTag, ","); idx != -1 {
			jsonTag = jsonTag[:idx]
		}

		// 安全地获取字段值
		var value interface{}
		switch fieldValue.Kind() {
		case reflect.Ptr:
			if fieldValue.IsNil() {
				value = nil
			} else {
				value = fieldValue.Elem().Interface()
			}
		case reflect.Slice:
			if fieldValue.IsNil() {
				value = nil
			} else {
				slice := make([]interface{}, fieldValue.Len())
				for i := 0; i < fieldValue.Len(); i++ {
					elem := fieldValue.Index(i)
					if elem.Kind() == reflect.Ptr && !elem.IsNil() {
						slice[i] = elem.Elem().Interface()
					} else {
						slice[i] = elem.Interface()
					}
				}
				value = slice
			}
		case reflect.Map:
			if fieldValue.IsNil() {
				value = nil
			} else {
				mapValue := make(map[string]interface{})
				iter := fieldValue.MapRange()
				for iter.Next() {
					k := iter.Key().Interface()
					v := iter.Value()
					if v.Kind() == reflect.Ptr && !v.IsNil() {
						mapValue[fmt.Sprint(k)] = v.Elem().Interface()
					} else {
						mapValue[fmt.Sprint(k)] = v.Interface()
					}
				}
				value = mapValue
			}
		default:
			value = fieldValue.Interface()
		}

		result[jsonTag] = value
	}

	// 将结果转换为 JSON
	data, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		return fmt.Sprintf("error marshaling: %v", err)
	}
	return string(data)
}

// 扩展边界测试
func TestExtendedEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		json        string
		shouldError bool
	}{
		{
			name:        "int8溢出-正数",
			json:        `{"int8": "128"}`,
			shouldError: true,
		},
		{
			name:        "int8溢出-负数",
			json:        `{"int8": "-129"}`,
			shouldError: true,
		},
		{
			name:        "uint8溢出",
			json:        `{"uint8": "256"}`,
			shouldError: true,
		},
		{
			name:        "uint负数",
			json:        `{"uint": "-1"}`,
			shouldError: true,
		},
		{
			name:        "float32精度",
			json:        `{"float32": "3.4028235E38"}`,
			shouldError: true,
		},
		{
			name:        "无效的数字字符串",
			json:        `{"int": "not a number"}`,
			shouldError: true,
		},
		{
			name:        "指针类型的无效值",
			json:        `{"ptrInt": "invalid"}`,
			shouldError: true,
		},
		{
			name:        "空字符串转数字",
			json:        `{"int": ""}`,
			shouldError: true,
		},
		{
			name:        "布尔值转数字",
			json:        `{"int": true}`,
			shouldError: true,
		},
		{
			name: "null值处理",
			json: `{
				"int": null,
				"ptrInt": null,
				"string": null,
				"ptrString": null
			}`,
			shouldError: false,
		},
		{
			name: "指针数组的null值处理",
			json: `{
				"ptrIntArray": null,
				"ptrFloatArray": null
			}`,
			shouldError: false,
		},
		{
			name: "指针数组的空数组处理",
			json: `{
				"ptrIntArray": [],
				"ptrFloatArray": []
			}`,
			shouldError: false,
		},
		{
			name: "指针数组的无效值",
			json: `{
				"ptrIntArray": ["invalid"],
				"ptrFloatArray": ["not a number"]
			}`,
			shouldError: true,
		},
		{
			name: "指针数组的类型不匹配",
			json: `{
				"ptrIntArray": [true, false],
				"ptrFloatArray": ["string"]
			}`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got AllTypesStruct
			err := json.UnmarshalFromString(tt.json, &got)

			if tt.shouldError && err == nil {
				t.Error("期望出错但没有错误")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("不期望出错但出现错误: %v", err)
			}
		})
	}
}
