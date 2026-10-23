// Package log
// @Author: yinwei
// @File: encoder
// @Version: 1.0.0
// @Date: 2024/8/24 22:45

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// CustomEncoder 是一个自定义的 zapcore.Encoder 实现，用于过滤敏感信息
type CustomEncoder struct {
	zapcore.Encoder
}

// Clone 创建一个 CustomEncoder 的副本
func (e *CustomEncoder) Clone() zapcore.Encoder {
	return &CustomEncoder{Encoder: e.Encoder.Clone()}
}

// EncodeEntry 处理并过滤日志条目中的敏感信息
func (e *CustomEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	var filteredFields []zapcore.Field
	for _, field := range fields {
		//fmt.Println(field)
		if field.Key == "password" || field.Key == "secret" {
			// 如果字段是敏感字段，替换为 ****
			filteredFields = append(filteredFields, zap.String(field.Key, "****"))
		} else {
			filteredFields = append(filteredFields, field)
		}
	}
	// 调用原始的 EncodeEntry 方法，并使用过滤后的字段
	return e.Encoder.EncodeEntry(entry, filteredFields)
}
