package log

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/coderyw/gomicro-util/library/trace"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Tiny struct {
	A string `json:"a"`
}

func TestFieldString(t *testing.T) {
	te := Tiny{
		A: "fwe",
	}
	Info("test", zap.Any("te", te))
}

type User struct {
	Name  string `json:"name"`
	Pwd   string `json:"pwd"`
	Phone string `json:"phone"`
	T     Tiny   `json:"t"`
}

func (u Tiny) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("a", u.A)
	return nil
}

func (u User) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("phone", Mask(u.Phone, 1, 1))
	enc.AddString("password", u.Pwd)
	enc.AddString("token", u.Name)
	return enc.AddObject("t", u.T)
}

func TestUserLog(t *testing.T) {
	u := User{
		Name:  "234",
		Pwd:   "1123",
		Phone: "935345345",
		T:     Tiny{A: "234"},
	}
	Info("user", zap.Any("user", u))
}

func Mask(s string, start, end int) string {
	n := len(s)
	if start < 0 || end < 0 || n <= start+end {
		return "***"
	}

	// 预估容量：start + "***" + end
	buf := make([]byte, 0, start+3+end)
	buf = append(buf, s[:start]...)
	buf = append(buf, "***"...)
	buf = append(buf, s[n-end:]...)
	return string(buf)
}

func TestMask(t *testing.T) {
	phone := "909934433"
	fmt.Println(Mask(phone, 1, 10))
}

func TestError(t *testing.T) {
	InitDefaultLogger(srv_config.LogCfg{}, "test", "")

	Warn("test")

	Error("test")

	fmt.Printf("%c\n", 0x1B)

	type A struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	b := new(A)
	// data, _ := json.Marshal(b)
	jsonString := `{"name": "John", "age": 30}`
	Info("3", zap.Any("s", b), zap.Reflect("s", jsonString), zap.Any("s", b))
}

func Error1() error {
	err := Error2()
	if err != nil {
		Error("打印错误", zap.String("错误key", "info"))
	}
	return nil
}

func Error2() error {
	return Error3()
}

func Error3() error {
	err := fmt.Errorf("cuowu3")
	Error("打印错误", zap.Error(err))
	Warn("nocuowu")
	return err
}

func TestErrorS(t *testing.T) {
	defaultLogger.Info("test", zap.Any("ge", "gegw"))
	defaultLogger.Debug("test")
	defaultLogger.Error("test")
}

func TestErrors(t *testing.T) {
	e := errors.New("test")
	Error("test", zap.Error(e))

	e1 := fmt.Errorf("test1")

	Error("test1", zap.Error(errors.Wrap(e1, "wrap1")))
	Error("test1", zap.Error(errors.WithStack(e1)))
}

func TestLogNamespace(t *testing.T) {
	defaultLogger.Info("test", zap.Namespace("odo"), zap.Any("hh", "fwef"))
	defaultLogger.Error("oye", zap.Namespace("这是一个错误"), zap.Error(fmt.Errorf("hh")))
}

func TestTraceId(t *testing.T) {
	traceId := uuid.NewString()
	ctx := context.WithValue(context.Background(), trace.CtxTraceIdKey, traceId)
	InfoCtx(ctx, "info", zap.Any("detail", "detail"))
	WarnCtx(ctx, "warn", zap.Any("detail", "detail"))
	DebugCtx(ctx, "debug", zap.Any("detail", "detail"))
	ErrorCtx(ctx, "error", zap.Any("detail", "detail"), zap.Error(fmt.Errorf("this is error")))
}

func TestGinLog(t *testing.T) {
	ginLog.Info("test")
}
