/**
 * @Author: dlan
 * @Description:
 * @File: jaeger_test
 * @Version: 1.0.0
 * @Date: 2022/3/17 4:53 下午
 */
package jaeger

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"
)

func TestNotSpan(t *testing.T) {
	value := NotTraceCtx.Value(NotTraceKey{})
	fmt.Printf("%v\n", value)
	fmt.Println(value == nil)
}

var Closer io.Closer

func InitJaeger(serverName string) {
	tracerHost = "0.0.0.0:6831"
	sTracer, _, err := NewTracer(serverName, "0.0.0.0:6831")
	if err != nil {
		panic(err)
	} else {
		opentracing.SetGlobalTracer(sTracer)
	}
}

func TestWrite(t *testing.T) {
	InitJaeger("test")
	defer Closer.Close()
	WriterJaeger()
}

func WriterJaeger() {
	sp, sCtx := StartRootSpanWithoutTracer(TraceCtx, "WriterJaeger", "diidiwksmegfi2i32")
	defer sp.Finish()
	sp.SetTag("WriterJaeger", "WriterJaeger")
	sleep()

	Func2(sCtx)
}

func Func2(ctx context.Context) {
	sp, sCtx := StartChildSpanFromContextWithoutName(ctx)
	sp.SetTag("tag2", "tag2")
	defer sp.Finish()
	sleep()
	Func3(sCtx)
	Func6(sCtx)
}

func Func3(ctx context.Context) {
	sp, sCtx := StartChildSpanFromContextWithoutName(ctx)
	sp.SetTag("tag3", "tag3")
	defer sp.Finish()
	sleep()
	Func4(sCtx)
	Func5(sCtx)
}

func Func4(ctx context.Context) {
	sp, _ := StartChildSpanFromContextWithoutName(ctx)
	sleep()
	defer sp.Finish()
}

func Func5(ctx context.Context) {
	sp, _ := StartChildSpanFromContextWithoutName(ctx)
	defer sp.Finish()
	sleep()
}

func Func6(ctx context.Context) {
	sp, _ := StartChildSpanFromContextWithoutName(ctx)
	defer sp.Finish()
	sleep()
}

func sleep() {
	rand.Seed(time.Now().UnixNano())
	time.Sleep(time.Duration(rand.Intn(100)+1) * time.Millisecond)
}
