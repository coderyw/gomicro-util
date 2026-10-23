// Package test
// @Author: yinwei
// @File: decimal
// @Version: 1.0.0
// @Date: 2024/9/6 16:30

package test

import (
	"encoding/json"
	"fmt"
	"github.com/shopspring/decimal"
	"testing"
)

type DecimalS struct {
	A int `json:"a"`
	B decimal.Decimal
	C decimal.Decimal
}

func TestDecimalMarshal(t *testing.T) {
	a := decimal.NewFromFloat(1.233)
	data, err := json.Marshal(a)
	fmt.Println(err)
	fmt.Println(string(data))
}

func TestDecimalSMarshL(t *testing.T) {
	ds := DecimalS{
		A: 1,
		B: decimal.New(123, 3),
		C: decimal.NewFromFloat(1.23),
	}
	data, err := json.Marshal(ds)
	fmt.Println(err)
	fmt.Println(string(data))
}

func TestUnmarshal(t *testing.T) {
	str := `{"a":1,"B":123000,"C":1.23}`
	var ds DecimalS
	err := json.Unmarshal([]byte(str), &ds)
	fmt.Println(err)
	fmt.Println(ds)
}

func TestNewFromString(t *testing.T) {
	str := ""
	val, err := decimal.NewFromString(str)
	fmt.Println(err)
	fmt.Println(val)
}
