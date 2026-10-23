package log

import (
	"bytes"
	"fmt"
	stdlog "log"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type redactTestCfg struct{ Token string }

func (c redactTestCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("token", c.Token)
	return nil
}

// newRedactTestLogger 用与 InitDefaultLogger 相同的编码器接线，把输出收进内存
func newRedactTestLogger(t *testing.T) (*zap.Logger, *bytes.Buffer) {
	t.Helper()
	ResetRedactValues()
	t.Cleanup(ResetRedactValues)
	var out bytes.Buffer
	enc := &redactEncoder{Encoder: zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())}
	return zap.New(zapcore.NewCore(enc, zapcore.AddSync(&out), zapcore.DebugLevel)), &out
}

func TestRedactEncoder_登记值在任何位置都被掩码(t *testing.T) {
	const plain = "real-secret-9"
	t.Run("message、字段、嵌套对象、With 字段都被替换", func(t *testing.T) {
		logger, out := newRedactTestLogger(t)
		RegisterRedactValue(plain)
		logger.With(zap.String("with_pwd", plain)).Info("connect "+plain+" failed",
			zap.String("pwd", plain), zap.Object("cfg", redactTestCfg{Token: plain}))
		got := out.String()
		if strings.Contains(got, plain) {
			t.Fatalf("明文出现在日志中: %s", got)
		}
		if n := strings.Count(got, redactMask); n != 4 {
			t.Fatalf("期望 4 处掩码，实际 %d: %s", n, got)
		}
	})
	t.Run("含引号反斜杠换行的值按 JSON 转义形态也能替换", func(t *testing.T) {
		logger, out := newRedactTestLogger(t)
		v := "p@ss\"quo\\te\nline2"
		RegisterRedactValue(v)
		logger.Info("x", zap.String("pwd", v))
		got := out.String()
		if strings.Contains(got, v) || strings.Contains(got, jsonEscape(v)) {
			t.Fatalf("转义形态的明文出现在日志中: %s", got)
		}
		if !strings.Contains(got, redactMask) {
			t.Fatalf("期望出现掩码: %s", got)
		}
	})
	t.Run("一个值是另一个值的子串时两者都完整掩码", func(t *testing.T) {
		logger, out := newRedactTestLogger(t)
		RegisterRedactValue("abcdef")
		RegisterRedactValue("abcdefghij")
		logger.Info("k", zap.String("a", "abcdefghij"), zap.String("b", "abcdef"))
		if got := out.String(); strings.Contains(got, "ghij") || strings.Contains(got, "abcdef") {
			t.Fatalf("残留半截明文: %s", got)
		}
	})
	t.Run("短于阈值的值不登记", func(t *testing.T) {
		logger, out := newRedactTestLogger(t)
		RegisterRedactValue("1234")
		logger.Info("port 1234", zap.Int("port", 1234))
		if got := out.String(); !strings.Contains(got, "1234") || strings.Contains(got, redactMask) {
			t.Fatalf("短值不应被掩码: %s", got)
		}
	})
	t.Run("未登记任何值时输出原样", func(t *testing.T) {
		logger, out := newRedactTestLogger(t)
		logger.Info("plain message", zap.String("k", "plain-value"))
		if got := out.String(); !strings.Contains(got, "plain-value") || strings.Contains(got, redactMask) {
			t.Fatalf("无登记值时不应改动输出: %s", got)
		}
	})
}

func TestRedactString(t *testing.T) {
	ResetRedactValues()
	t.Cleanup(ResetRedactValues)
	RegisterRedactValue("real-secret-9")
	RegisterRedactValue("real-secret-9") // 重复登记幂等
	if got := RedactString("a real-secret-9 b real-secret-9"); got != "a *** b ***" {
		t.Fatalf("期望全部替换，实际 %q", got)
	}
	if got := RedactString("nothing here"); got != "nothing here" {
		t.Fatalf("未命中应原样返回，实际 %q", got)
	}
	redactor.mu.RLock()
	n := len(redactor.values)
	redactor.mu.RUnlock()
	if n != 1 {
		t.Fatalf("重复登记应去重，实际登记 %d 项", n)
	}
}

func TestRedactWriter_标准库log输出按值脱敏(t *testing.T) {
	ResetRedactValues()
	t.Cleanup(ResetRedactValues)
	const plain = "real-secret-9"
	RegisterRedactValue(plain)
	var out bytes.Buffer
	l := stdlog.New(redactWriter{w: &out}, "", 0)
	l.Printf("dsn user:%s@tcp(db)", plain)
	l.Print("no secret here")
	got := out.String()
	if strings.Contains(got, plain) || !strings.Contains(got, "user:"+redactMask+"@tcp(db)") || !strings.Contains(got, "no secret here") {
		t.Fatalf("标准库 log 输出脱敏不符: %q", got)
	}
}

// BenchmarkRedactEncoder 每条日志对全部登记值做一次扫描，量化登记数量对单条日志编码的额外开销
func BenchmarkRedactEncoder(b *testing.B) {
	for _, n := range []int{0, 10, 50} {
		b.Run(fmt.Sprintf("登记%d个值", n), func(b *testing.B) {
			ResetRedactValues()
			b.Cleanup(ResetRedactValues)
			for i := 0; i < n; i++ {
				RegisterRedactValue(fmt.Sprintf("secret-value-%04d-xxxxxxxx", i))
			}
			enc := &redactEncoder{Encoder: zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())}
			entry := zapcore.Entry{Message: "[Handler] 处理请求完成 user_id=10086 order_no=ORD202610080001 cost=12ms"}
			fields := []zapcore.Field{zap.Int64("user_id", 10086), zap.String("path", "/open/api/activity/list"), zap.Duration("cost", 12)}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				buf, _ := enc.EncodeEntry(entry, fields)
				buf.Free()
			}
		})
	}
}
