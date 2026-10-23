package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"
)

type customerConfig struct {
	value         atomic.Value
	defaultValue  any
	typ           reflect.Type // 底层结构体类型（非指针）
	isPtr         bool         // 初始化时是否是指针
	changeHandler func(value any)
}

var (
	cc *customerConfig
)

// 获取自定义配置
//
// 返回非指针类型配置
func GetCustomerConfig() any {
	return cc.value.Load()
}

// 初始化自定义配置
func initCustomerConfig(value any, c func(value any)) error {
	if value == nil {
		return fmt.Errorf("default config cannot be nil")
	}

	t := reflect.TypeOf(value)
	isPtr := false

	if t.Kind() == reflect.Ptr {
		isPtr = true
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return fmt.Errorf("config must be struct or *struct")
	}

	h := &customerConfig{
		typ:           t,
		isPtr:         isPtr,
		changeHandler: c,
	}

	// 统一存非指针值
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	h.value.Store(v.Interface())
	h.defaultValue = v.Interface()
	cc = h
	return nil
}

func updateFromJSON(str string) error {
	if cc == nil {
		return nil
	}
	v := reflect.New(cc.typ) // *T
	// 恢复默认值
	v.Elem().Set(reflect.ValueOf(cc.defaultValue))
	// 带有默认值的去反序列化
	if err := json.Unmarshal([]byte(str), v.Interface()); err != nil {
		return err
	}
	data := v.Elem().Interface()
	// 内部仍然存非指针
	cc.value.Store(data)
	if cc.changeHandler != nil {
		cc.changeHandler(data)
	}
	return nil
}
