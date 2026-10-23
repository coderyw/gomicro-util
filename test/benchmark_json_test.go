// Package test
// @Author: yinwei
// @File: benchmark_json
// @Version: 1.0.0
// @Date: 2024/9/4 18:28

package test

import (
	"encoding/json"
	httpresp "github.com/coderyw/gomicro-util/model/http"
	"testing"
)

func Benchmark_test_marshal(b *testing.B) {
	type MsgBox struct {
		MsgType uint32 `json:"msgType"`
	}
	type NsHttpResp struct {
		Code   uint32      `json:"code"`
		Msg    string      `json:"msg"`
		Data   interface{} `json:"data"`
		MsgBox []MsgBox    `json:"msgBox"`
	}

	b.Run("原生", func(b *testing.B) {
		m1 := NsHttpResp{
			Code:   123,
			Msg:    "wef",
			Data:   nil,
			MsgBox: []MsgBox{{MsgType: 1}, {MsgType: 2}},
		}
		for i := 0; i < b.N; i++ {
			_, _ = json.Marshal(&m1)
		}
	})

	b.Run("easyjson", func(b *testing.B) {
		m1 := httpresp.NsHttpResp{
			Code: 123,
			Msg:  "wef",
			Data: nil,
		}
		for i := 0; i < b.N; i++ {
			_, _ = m1.MarshalJSON()
		}
	})

}

func Benchmark_test_unmarshal(b *testing.B) {
	type MsgBox struct {
		MsgType uint32 `json:"msgType"`
	}
	type NsHttpResp struct {
		Code   uint32      `json:"code"`
		Msg    string      `json:"msg"`
		Data   interface{} `json:"data"`
		MsgBox []MsgBox    `json:"msgBox"`
	}
	data := []byte(`{"code":123,"msg":"wef","data":null,"msgBox":[{"msgType":1},{"msgType":2}]}`)
	b.Run("原生", func(b *testing.B) {
		m1 := NsHttpResp{
			Code:   123,
			Msg:    "wef",
			Data:   nil,
			MsgBox: []MsgBox{{MsgType: 1}, {MsgType: 2}},
		}
		for i := 0; i < b.N; i++ {
			_ = json.Unmarshal(data, &m1)
		}
	})

	b.Run("easyjson", func(b *testing.B) {
		m1 := httpresp.NsHttpResp{
			Code: 123,
			Msg:  "wef",
			Data: nil,
		}
		for i := 0; i < b.N; i++ {
			_ = m1.UnmarshalJSON(data)
		}
	})

}
