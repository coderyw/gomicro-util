// Package http
// @Author: yinwei
// @File: ns
// @Version: 1.0.0
// @Date: 2024/9/4 18:21

package httpresp

import "go.uber.org/zap/zapcore"

// 默认消息返回体
type NsHttpResp struct {
	Code   uint32      `json:"code"`
	Msg    string      `json:"msg"`
	Detail string      `json:"detail"`
	Data   interface{} `json:"data"`
}

func (s *NsHttpResp) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddUint32("code", s.Code)
	enc.AddString("msg", s.Msg)
	enc.AddString("detail", s.Detail)
	if s.Data != nil {
		if val, ok := s.Data.(zapcore.ObjectMarshaler); ok {
			if err := enc.AddObject("data", val); err != nil {
				return err
			}
		} else {
			return enc.AddReflected("data", s.Data)
		}
	}

	return nil
}

//type MsgBox struct {
//	MsgType uint32 `json:"msgType"`
//}
