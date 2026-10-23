// Package log
// @Author: yinwei
// @File: core
// @Version: 1.0.0
// @Date: 2024/8/24 22:31

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogCore struct {
	zapcore.Core
}

func (c *LogCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// 这里可以添加额外的检查逻辑
	return c.Core.Check(entry, ce)
}

// Write 处理日志条目并过滤敏感字段
func (c *LogCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var filteredFields []zapcore.Field
	for _, field := range fields {
		if field.Key == "password" || field.Key == "secret" {
			// 如果字段是敏感字段，替换为 ****
			filteredFields = append(filteredFields, zap.String(field.Key, "****"))
		} else {
			filteredFields = append(filteredFields, field)
		}
	}
	// 调用原始的 Write 方法，并使用过滤后的字段
	return c.Core.Write(entry, filteredFields)
}
