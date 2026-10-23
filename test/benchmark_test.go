// Package test
// @Author: yinwei
// @File: benchmark_test
// @Version: 1.0.0
// @Date: 2024/6/25 09:09

package test

import (
	"fmt"
	"github.com/coderyw/protobuf/keyreflect"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

type ABC struct {
	A int
	B string
	C bool
}

func BenchmarkReflectNew(b *testing.B) {
	r := reflect.TypeOf(ABC{})
	keyreflect.Register("abc", r)

	b.Run("system new", func(b *testing.B) {
		_ = new(ABC)
	})
	b.Run("keyreflect new", func(b *testing.B) {
		_, _ = keyreflect.New("abc")
	})
	b.Run("reflect new", func(b *testing.B) {
		_ = reflect.New(r).Interface()
	})
}

func TestString(t *testing.T) {
	ss := StringStrc{B: "abc", BB: StringStrc2{BB: "bb"}}
	log.Debug("msg", zap.Any("ss", ss))
}

type StringStrc struct {
	B  string      `json:"b"`
	BB StringStrc2 `json:"bb"`
}

type StringStrc2 struct {
	BB string `json:"bb"`
}

func (a2 StringStrc2) String() string {
	return fmt.Sprintf("string2:BB:%s", a2.BB)
}

func (a StringStrc) String() string {
	return fmt.Sprintf("string:B:%s, BB:%v", a.B, a.BB)
}
