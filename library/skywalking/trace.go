// Package skywalking
// @Author: yinwei
// @File: trace
// @Version: 1.0.0
// @Date: 2024/7/15 16:10

package skywalking

import (
	"github.com/apache/skywalking-go/toolkit/trace"
	"github.com/coderyw/gomicro-util/library/log"
	"go.uber.org/zap"
	"runtime"
	"strings"
)

// 所有支持的 golang 组件类型
const (
	UNKNOWN = 0

	// DUBBO
	DUBBO = 3

	// RocketMQ
	ROCKETMQ = 25

	// Kafka
	KAFKA = 27

	// RocketMQ Producer
	ROCKETMQ_PRODUCER = 38

	// RocketMQ Consumer
	ROCKETMQ_CONSUMER = 39

	// Kafka Producer
	KAFKA_PRODUCER = 40

	// Kafka Consumer
	KAFKA_CONSUMER = 41

	// Elasticsearch
	ELASTICSEARCH = 47

	// Pulsar
	PULSAR = 72

	// Pulsar Producer
	PULSAR_PRODUCER = 73

	// Pulsar Consumer
	PULSAR_CONSUMER = 74

	// AMQP
	AMQP = 143

	// AMQP Producer
	AMQP_PRODUCER = 144

	// AMQP Consumer
	AMQP_CONSUMER = 145

	// ServiceComb Mesher
	SERVICECOMB_MESHER = 5001

	// ServiceComb ServiceCenter
	SERVICECOMB_SERVICECENTER = 5002

	// MOSN
	MOSN = 5003

	// Go HTTP Server
	GO_HTTP_SERVER = 5004

	// Go HTTP UniversalClient
	GO_HTTP_CLIENT = 5005

	// Gin
	GIN = 5006

	// Gear
	GEAR = 5007

	// Go Micro UniversalClient
	GO_MICRO_CLIENT = 5008

	// Go Micro Server
	GO_MICRO_SERVER = 5009

	// Kratos
	KRATOS = 5010

	// Go MySQL
	GO_MYSQL = 5012

	// OpenFunction
	OPEN_FUNCTION = 5013

	// Go Redis
	GO_REDIS = 5014

	// Echo
	ECHO = 5015

	// Lock
	LOCK = 5016

	// Go Mux Server
	GO_MUX_SERVER = 5017

	// Iris
	IRIS = 5018

	// Go Fast HTTP UniversalClient
	GO_FAST_HTTP_CLIENT = 5019

	// Go Fast HTTP Server
	GO_FAST_HTTP_SERVER = 5020

	// Fiber
	FIBER = 5021

	// GoFrame
	GO_FRAME = 5022

	// Go-Zero
	GO_ZERO = 5023
)

// PrepareAsync/AsyncFinish 与 CaptureContext/ContinueContext 用法对比
// | 对比项            |  PrepareAsync/AsyncFinish    | CaptureContext/ContinueContext |
// +---------------+-------------------------------+--------------------------------+
// | 是否创建新 Span   | 否                            | 是                             |
// | 是否可用于异步    | 是                            | 是                              |
// |是否适用于跨组件传递| 否                            | 是                             |
// | 原始 Span 是否结束|          不结束，等你手动 finish|         结束，继续链路需要新的 span|
// | 适合场景｜同一个 Span 跨 goroutine                | 不同 span 跨 goroutine/跨组件    |

// CaptureContext 从当前上下文 context.Context 中 提取并快照当前 Trace 信息，方便在别的地方（如 goroutine、异步回调、channel）里继续使用。
//
//   - 将当前 Trace 上下文传递给 goroutine
//   - 在异步任务中继续当前 Trace
//   - 保持链路追踪完整，不断链
//
// 示例：
//
//	func doAsync(ctx context.Context) {
//	   // 捕获当前 Trace 的快照
//	   snapshot := agent.CaptureContext(ctx)
//
//	   go func() {
//	       // 在 goroutine 中继续 Trace
//	       newCtx := agent.ContinueContext(context.Background(), snapshot)
//
//	       // 开启一个新的 Span
//	       span := agent.CreateLocalSpan(newCtx)
//	       defer agent.EndSpan(newCtx)
//
//	       // 执行异步逻辑
//	       doWork()
//
//	       span.Tag("key", "value") // 可选：添加一些 tags
//	   }()
//	}
//
// 传递给其他协程需要使用下面方法 ContinueContext
//
// Returns:
//   - trace.ContextSnapshotRef
func CaptureContext() trace.ContextSnapshotRef {
	return trace.CaptureContext()
}

// ContinueContext 在新的 goroutine 或异步逻辑中，通过传入 snapshot 创建一个新的 context.Context，并继续原始链路追踪。
//
// 使用场景：继续使用另一个协程的上下文，使得当前协程加入另一个协程的链路
//
// Parameters:
//   - ctx: trace.ContextSnapshotRef
func ContinueContext(ctx trace.ContextSnapshotRef) {
	trace.ContinueContext(ctx)
}

// PrepareAsync 当你需要让一个 已有的 span 延续到另一个 goroutine 去结束时用。
//
// 示例：
//
// span := agent.CreateLocalSpan(ctx)
//
// PrepareAsync(span)
//
// agent.StopSpan(ctx) // 在当前 goroutine detach 掉 span，但不关闭
//
//	go func(span agent.SpanRef) {
//	   // 在其他 goroutine 做事
//	   doSomething()
//	   AsyncFinish(span) // 手动关闭 span
//	}(span)
//
// Parameters:
//   - ref: trace.SpanRef span 信息
func PrepareAsync(ref *trace.SpanRef) {
	ref.PrepareAsync()
}

func AsyncFinish(ref *trace.SpanRef) {
	ref.AsyncFinish()
}

// Start
//
//	@Description: 开启本地方法span
//	@Author yw 2024-07-16 13:28:41
func Start() (*trace.SpanRef, error) {
	ref, err := trace.CreateLocalSpan(getFuncName())
	trace.SetComponent(OPEN_FUNCTION)
	return ref, err
}

// StartRedis
//
//	@Description: 开启本地 goredis span
//	@Author yw 2024-08-01 09:49:36
func StartRedis() (*trace.SpanRef, error) {
	ref, err := trace.CreateLocalSpan(getFuncName())
	trace.SetComponent(GO_REDIS)
	return ref, err
}

// StartMysql
//
//	@Description: 开启本地mysql  span
//	@Author yw 2024-08-01 09:49:50
func StartMysql() (*trace.SpanRef, error) {
	ref, err := trace.CreateLocalSpan(getFuncName())
	trace.SetComponent(GO_MYSQL)
	return ref, err
}

// SetTag
//
//	@Description: 为span设置tag
//	@Author yw 2024-07-25 15:26:07
//	@param key
//	@param value
func SetTag(key string, value string) {
	trace.SetTag(key, value)
}

func StopSpan() {
	trace.StopSpan()
}

// StartOperationName
//
//	@Description: 开启operationName为name的本地方法span
//	@Author yw 2024-07-16 13:28:56
//	@param name
func StartOperationName(name string) (*trace.SpanRef, error) {
	ref, err := trace.CreateLocalSpan(name)
	if err != nil {
		log.Error("Create Entry Span Error", zap.Error(err))
	}
	return ref, err
}

//func getFuncName() string {
//	pc, _, _, ok := runtime.Caller(2)
//	if ok {
//		name := runtime.FuncForPC(pc).Name()
//		index := strings.LastIndex(name, ".")
//		name = name[index+1:]
//		return name
//	}
//	return "FuncName"
//}

// 获取调用栈中第 skip 层的函数名（兼容泛型）
func getFuncName() string {
	pc, _, _, _ := runtime.Caller(2)
	return formatGenericName(runtime.FuncForPC(pc).Name())
}

// 格式化泛型函数名（示例实现，需根据实际编译结果调整）
func formatGenericName(name string) string {
	i := strings.LastIndex(name, "/")
	if i > 0 {
		return strings.TrimSuffix(name[i+1:], "[...]")
	}
	return name
}
