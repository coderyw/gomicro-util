// Package config
// @Author: yinwei
// @File: lark
// @Version: 1.0.0
// @Date: 2025/6/4 09:25

package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

type LarkCfg struct {
	Groups map[string]LarkGroup
}

type LarkGroup struct {
	WebHook string `json:"webhook"`
	Sign    string `json:"sign"`
	Enabled *bool  `json:"enabled"`
}

var LarkConfig LarkCfg

func (s LarkCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	for l, v := range s.Groups {
		if err := enc.AddObject(l, v); err != nil {
			return err
		}
	}
	return nil
}

func (s LarkGroup) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("webhook", s.WebHook)
	enc.AddString("sign", util.Mask(s.Sign, 0, 4))
	var enabled = true
	if s.Enabled != nil && *s.Enabled == false {
		enabled = false
	}

	enc.AddBool("enabled", enabled)
	return nil
}
