// Package log
// @File: redact
// 凭据明文脱敏：按"值"而不是"字段名"判定
package log

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"sync"

	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// secret 组件把从 CSMS / AWS Secrets Manager 解析出的明文登记到这里，日志条目编码完成后再把登记值整行替换为掩码。
// message、任意字段、MarshalLogObject 嵌套对象、堆栈里出现的明文一律覆盖，与按字段名的脱敏叠加而非替代。
// 值经 zap 编码后可能带 JSON 转义（引号、反斜杠、换行），登记时同时登记转义后的形态。
// 覆盖范围：经本包默认 logger 输出的日志（含 Nacos SDK、gorm 等经本包适配器转接的日志）与标准库 log 包的输出；
// 直接 fmt.Print 到标准输出、第三方库自建的 logger 不经过这里，不在覆盖范围内。

const (
	redactMask = "***"
	// 过短的值（如 "1234"）登记后会误伤正常日志里的数字与短串，这类值交给按字段名的脱敏兜底
	redactMinLen = 6
)

var redactor = &redactTable{seen: map[string]struct{}{}}

type redactTable struct {
	mu   sync.RWMutex
	seen map[string]struct{}
	// 按长度降序：一个值是另一个值的子串时先替换长的，避免残留半截明文
	values [][]byte
}

// RegisterRedactValue 登记一个需要在日志中隐去的明文值；重复登记幂等，短于阈值的值忽略
func RegisterRedactValue(v string) {
	if len(v) < redactMinLen {
		return
	}
	redactor.add(v)
	if escaped := jsonEscape(v); escaped != v {
		redactor.add(escaped)
	}
}

// ResetRedactValues 清空登记表（测试用）
func ResetRedactValues() {
	redactor.mu.Lock()
	defer redactor.mu.Unlock()
	redactor.seen = map[string]struct{}{}
	redactor.values = nil
}

// RedactString 把字符串中已登记的明文替换为掩码
func RedactString(s string) string {
	out, changed := redactor.apply([]byte(s))
	if !changed {
		return s
	}
	return string(out)
}

func (r *redactTable) add(v string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.seen[v]; dup {
		return
	}
	r.seen[v] = struct{}{}
	r.values = append(r.values, []byte(v))
	sort.SliceStable(r.values, func(i, j int) bool { return len(r.values[i]) > len(r.values[j]) })
}

// apply 返回替换结果与是否发生替换；无登记值或未命中时不分配
func (r *redactTable) apply(b []byte) ([]byte, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	changed := false
	for _, v := range r.values {
		if bytes.Contains(b, v) {
			b = bytes.ReplaceAll(b, v, []byte(redactMask))
			changed = true
		}
	}
	return b, changed
}

// jsonEscape 返回 v 作为 JSON 字符串内容的转义形态（不含外层引号、不转义 HTML 字符，与 zap 的编码一致）
func jsonEscape(v string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return v
	}
	out := bytes.TrimRight(buf.Bytes(), "\n")
	if len(out) >= 2 {
		out = out[1 : len(out)-1]
	}
	return string(out)
}

// redactEncoder 包装 zapcore.Encoder：条目编码完成后对整行做明文替换
type redactEncoder struct {
	zapcore.Encoder
}

func (e *redactEncoder) Clone() zapcore.Encoder {
	return &redactEncoder{Encoder: e.Encoder.Clone()}
}

func (e *redactEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf, err := e.Encoder.EncodeEntry(entry, fields)
	if err != nil || buf == nil {
		return buf, err
	}
	if out, changed := redactor.apply(buf.Bytes()); changed {
		buf.Reset()
		_, _ = buf.Write(out)
	}
	return buf, nil
}

// redactWriter 标准库 log 包的输出包一层按值脱敏：log.Output 每次写入一整行，替换不会被写入边界截断
type redactWriter struct{ w io.Writer }

func (r redactWriter) Write(p []byte) (int, error) {
	out, changed := redactor.apply(p)
	if !changed {
		return r.w.Write(p)
	}
	if _, err := r.w.Write(out); err != nil {
		return 0, err
	}
	return len(p), nil
}
