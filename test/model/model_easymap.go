package model

import (
	fmt "fmt"
	easy_facade "github.com/coderyw/easymap/gen/facade"
	strconv "strconv"
	unsafe "unsafe"
)

func str2Bytes_model_easymap(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
func (v *Target1) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["d"]; ok {
		{
			pv := val
			v.D = string(pv)
		}
	}

	return nil
}
func (v *Target1) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["d"]; ok {
		switch val.(type) {
		case string:
			v.D = string(val.(string))
		}
	}

	return nil
}

func (v *Target1) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["d"] = v.D
	return m, nil
}

func (v *Target1) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["d"] = fmt.Sprint(v.D)
	return m, nil
}

type Target1Field string

func (v Target1Field) MarshalBinary() (data []byte, err error) {
	return str2Bytes_model_easymap(string(v)), nil
}

const (
	Target1_D Target1Field = "d"
)

func (v *Target) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["a"]; ok {
		if pv, err := strconv.ParseInt(val, 10, 64); err != nil {
			return err
		} else {
			v.A = int(pv)
		}
	}
	if val, ok = m["b"]; ok {
		if pv, err := strconv.ParseInt(val, 10, 64); err != nil {
			return err
		} else {
			v.B = int(pv)
		}
	}

	return nil
}
func (v *Target) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["a"]; ok {
		switch val.(type) {
		case int16:
			v.A = int(val.(int16))
		case int32:
			v.A = int(val.(int32))
		case int64:
			v.A = int(val.(int64))
		case uint8:
			v.A = int(val.(uint8))
		case uint16:
			v.A = int(val.(uint16))
		case int:
			v.A = int(val.(int))
		case int8:
			v.A = int(val.(int8))
		case uint:
			v.A = int(val.(uint))
		case uint32:
			v.A = int(val.(uint32))
		case uint64:
			v.A = int(val.(uint64))
		}
	}
	if val, ok = m["b"]; ok {
		switch val.(type) {
		case int16:
			v.B = int(val.(int16))
		case int32:
			v.B = int(val.(int32))
		case int64:
			v.B = int(val.(int64))
		case uint16:
			v.B = int(val.(uint16))
		case uint32:
			v.B = int(val.(uint32))
		case uint64:
			v.B = int(val.(uint64))
		case int:
			v.B = int(val.(int))
		case int8:
			v.B = int(val.(int8))
		case uint:
			v.B = int(val.(uint))
		case uint8:
			v.B = int(val.(uint8))
		}
	}
	if val, ok = m["d"]; ok {
		var i interface{} = v.D
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *Target) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["a"] = v.A
	m["b"] = v.B
	return m, nil
}

func (v *Target) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["a"] = fmt.Sprint(v.A)
	m["b"] = fmt.Sprint(v.B)
	return m, nil
}

type TargetField string

func (v TargetField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_model_easymap(string(v)), nil
}

const (
	Target_A TargetField = "a"
	Target_B TargetField = "b"
	Target_C TargetField = "c"
	Target_D TargetField = "d"
)

func (v *Source1) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["d"]; ok {
		{
			pv := val
			v.D = string(pv)
		}
	}

	return nil
}
func (v *Source1) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["d"]; ok {
		switch val.(type) {
		case string:
			v.D = string(val.(string))
		}
	}

	return nil
}

func (v *Source1) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["d"] = v.D
	return m, nil
}

func (v *Source1) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["d"] = fmt.Sprint(v.D)
	return m, nil
}

type Source1Field string

func (v Source1Field) MarshalBinary() (data []byte, err error) {
	return str2Bytes_model_easymap(string(v)), nil
}

const (
	Source1_D Source1Field = "d"
)

func (v *Source) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["a"]; ok {
		if pv, err := strconv.ParseInt(val, 10, 64); err != nil {
			return err
		} else {
			v.A = int(pv)
		}
	}
	if val, ok = m["b"]; ok {
		if pv, err := strconv.ParseInt(val, 10, 64); err != nil {
			return err
		} else {
			v.B = int(pv)
		}
	}

	return nil
}
func (v *Source) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["a"]; ok {
		switch val.(type) {
		case int:
			v.A = int(val.(int))
		case int16:
			v.A = int(val.(int16))
		case int64:
			v.A = int(val.(int64))
		case uint8:
			v.A = int(val.(uint8))
		case uint32:
			v.A = int(val.(uint32))
		case uint64:
			v.A = int(val.(uint64))
		case int8:
			v.A = int(val.(int8))
		case int32:
			v.A = int(val.(int32))
		case uint:
			v.A = int(val.(uint))
		case uint16:
			v.A = int(val.(uint16))
		}
	}
	if val, ok = m["b"]; ok {
		switch val.(type) {
		case int8:
			v.B = int(val.(int8))
		case int16:
			v.B = int(val.(int16))
		case int64:
			v.B = int(val.(int64))
		case uint64:
			v.B = int(val.(uint64))
		case uint32:
			v.B = int(val.(uint32))
		case int:
			v.B = int(val.(int))
		case int32:
			v.B = int(val.(int32))
		case uint:
			v.B = int(val.(uint))
		case uint8:
			v.B = int(val.(uint8))
		case uint16:
			v.B = int(val.(uint16))
		}
	}
	if val, ok = m["d"]; ok {
		var i interface{} = v.D
		if m1, ok := val.(map[string]interface{}); ok {
			if b, ok := i.(easy_facade.EasyMapInter); ok {
				if err := b.UnMarshalMapInterface(m1); err != nil {
					return err
				}
			}
		} else if m2, ok := val.(map[string]string); ok {
			if b, ok := i.(easy_facade.EasyMapString); ok {
				if err := b.UnMarshalMap(m2); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (v *Source) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["a"] = v.A
	m["b"] = v.B
	return m, nil
}

func (v *Source) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["a"] = fmt.Sprint(v.A)
	m["b"] = fmt.Sprint(v.B)
	return m, nil
}

type SourceField string

func (v SourceField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_model_easymap(string(v)), nil
}

const (
	Source_A SourceField = "a"
	Source_B SourceField = "b"
	Source_C SourceField = "c"
	Source_D SourceField = "d"
)
