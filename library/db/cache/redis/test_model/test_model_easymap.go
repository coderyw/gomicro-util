package test_model

import (
	fmt "fmt"
	decimal "github.com/shopspring/decimal"
	strconv "strconv"
	unsafe "unsafe"
)

func str2Bytes_test_model_easymap(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}
func (v *Hmtest) UnMarshalMap(m map[string]string) error {
	var (
		ok  bool
		val string
	)
	if val, ok = m["v1"]; ok {
		if pv, err := strconv.ParseInt(val, 10, 64); err != nil {
			return err
		} else {
			v.V1 = int64(pv)
		}
	}
	if val, ok = m["v2"]; ok {
		{
			pv := val
			v.V2 = string(pv)
		}
	}
	if val, ok = m["v3"]; ok {
		var err error
		v.V3, err = decimal.NewFromString(val)
		if err != nil {
			return err
		}
	}

	return nil
}
func (v *Hmtest) UnMarshalMapInterface(m map[string]interface{}) error {
	var (
		ok  bool
		val interface{}
	)
	if val, ok = m["v1"]; ok {
		switch acval := val.(type) {
		case int64:
			v.V1 = int64(acval)
		case uint32:
			v.V1 = int64(acval)
		case int32:
			v.V1 = int64(acval)
		case int8:
			v.V1 = int64(acval)
		case int16:
			v.V1 = int64(acval)
		case uint:
			v.V1 = int64(acval)
		case uint8:
			v.V1 = int64(acval)
		case uint16:
			v.V1 = int64(acval)
		case uint64:
			v.V1 = int64(acval)
		case string:
			if len(acval) == 0 {
				v.V1 = 0
			} else {
				pvv, err := strconv.ParseInt(acval, 10, 64)
				if err != nil {
					return err
				}
				v.V1 = int64(pvv)
			}
		case int:
			v.V1 = int64(acval)
		}
	}
	if val, ok = m["v2"]; ok {
		switch acval := val.(type) {
		case int:
			v.V2 = strconv.FormatInt(int64(acval), 10)
		case int8:
			v.V2 = strconv.FormatInt(int64(acval), 10)
		case int16:
			v.V2 = strconv.FormatInt(int64(acval), 10)
		case int32:
			v.V2 = strconv.FormatInt(int64(acval), 10)
		case int64:
			v.V2 = strconv.FormatInt(int64(acval), 10)
		case uint16:
			v.V2 = strconv.FormatUint(uint64(acval), 10)
		case uint32:
			v.V2 = strconv.FormatUint(uint64(acval), 10)
		case string:
			v.V2 = acval
		case uint:
			v.V2 = strconv.FormatUint(uint64(acval), 10)
		case uint8:
			v.V2 = strconv.FormatUint(uint64(acval), 10)
		case uint64:
			v.V2 = strconv.FormatUint(uint64(acval), 10)
		case float64:
			v.V2 = strconv.FormatFloat(float64(acval), 'f', -1, 64)
		case float32:
			v.V2 = strconv.FormatFloat(float64(acval), 'f', -1, 64)
		}
	}
	if val, ok = m["v3"]; ok {
		switch acval := val.(type) {
		case int8:
			v.V3 = decimal.NewFromInt(int64(acval))
		case uint:
			v.V3 = decimal.NewFromInt(int64(acval))
		case uint32:
			v.V3 = decimal.NewFromInt(int64(acval))
		case uint64:
			v.V3 = decimal.NewFromInt(int64(acval))
		case int:
			v.V3 = decimal.NewFromInt(int64(acval))
		case int16:
			v.V3 = decimal.NewFromInt(int64(acval))
		case int32:
			v.V3 = decimal.NewFromInt(int64(acval))
		case int64:
			v.V3 = decimal.NewFromInt(int64(acval))
		case uint8:
			v.V3 = decimal.NewFromInt(int64(acval))
		case uint16:
			v.V3 = decimal.NewFromInt(int64(acval))
		case float64:
			v.V3 = decimal.NewFromFloat(float64(acval))
		case float32:
			v.V3 = decimal.NewFromFloat(float64(acval))
		case string:
			var err error
			v.V3, err = decimal.NewFromString(acval)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *Hmtest) MarshalMap() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	m["v1"] = v.V1
	m["v2"] = v.V2
	m["v3"] = v.V3.String()
	return m, nil
}

func (v *Hmtest) MarshalMapString() (map[string]string, error) {
	m := make(map[string]string)
	m["v1"] = fmt.Sprint(v.V1)
	m["v2"] = fmt.Sprint(v.V2)
	m["v3"] = v.V3.String()
	return m, nil
}

func (v *Hmtest) CheckField(m map[string]string) error {
	if _, ok := m["v1"]; !ok {
		return fmt.Errorf("field v1 is miss")
	}
	if _, ok := m["v2"]; !ok {
		return fmt.Errorf("field v2 is miss")
	}
	if _, ok := m["v3"]; !ok {
		return fmt.Errorf("field v3 is miss")
	}
	return nil
}

type HmtestField string

func (v HmtestField) MarshalBinary() (data []byte, err error) {
	return str2Bytes_test_model_easymap(string(v)), nil
}

const (
	Hmtest_V1 HmtestField = "v1"
	Hmtest_V2 HmtestField = "v2"
	Hmtest_V3 HmtestField = "v3"
)
