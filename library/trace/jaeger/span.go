/**
 * @Author: dlan
 * @Description:
 * @File: span
 * @Version: 1.0.0
 * @Date: 2022/3/17 4:11 下午
 */
package jaeger

import (
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
)

// OpentracingSpan 对opentracing span进行包装
type OpentracingSpan interface {
	// Sets the end timestamp and finalizes Span state.
	//
	// With the exception of calls to Context() (which are always allowed),
	// Finish() must be the last call made to any span instance, and to do
	// otherwise leads to undefined behavior.
	Finish()
	// FinishWithOptions is like Finish() but with explicit control over
	// timestamps and log data.
	FinishWithOptions(opts opentracing.FinishOptions)

	// Context() yields the SpanContext for this Span. Note that the return
	// value of Context() is still valid after a call to Span.Finish(), as is
	// a call to Span.Context() after a call to Span.Finish().
	Context() opentracing.SpanContext

	// Sets or changes the operation name.
	//
	// Returns a reference to this Span for chaining.
	SetOperationName(operationName string) OpentracingSpan

	// Adds a tag to the span.
	//
	// If there is a pre-existing tag set for `key`, it is overwritten.
	//
	// Tag values can be numeric types, strings, or bools. The behavior of
	// other tag value types is undefined at the OpenTracing level. If a
	// tracing system does not know how to handle a particular value type, it
	// may ignore the tag, but shall not panic.
	//
	// Returns a reference to this Span for chaining.
	SetTag(key string, value interface{}) OpentracingSpan

	// LogFields is an efficient and type-checked way to record key:value
	// logging data about a Span, though the programming interface is a little
	// more verbose than LogKV(). Here's an example:
	//
	//    span.LogFields(
	//        log.String("event", "soft error-code"),
	//        log.String("type", "cache timeout"),
	//        log.Int("waited.millis", 1500))
	//
	// Also see Span.FinishWithOptions() and FinishOptions.BulkLogData.
	LogFields(fields ...log.Field)

	// LogKV is a concise, readable way to record key:value logging data about
	// a Span, though unfortunately this also makes it less efficient and less
	// type-safe than LogFields(). Here's an example:
	//
	//    span.LogKV(
	//        "event", "soft error-code",
	//        "type", "cache timeout",
	//        "waited.millis", 1500)
	//
	// For LogKV (as opposed to LogFields()), the parameters must appear as
	// key-value pairs, like
	//
	//    span.LogKV(key1, val1, key2, val2, key3, val3, ...)
	//
	// The keys must all be strings. The values may be strings, numeric types,
	// bools, Go error-code instances, or arbitrary structs.
	//
	// (Note to implementors: consider the log.InterleavedKVToFields() helper)
	LogKV(alternatingKeyValues ...interface{})

	// SetBaggageItem sets a key:value pair on this Span and its SpanContext
	// that also propagates to descendants of this Span.
	//
	// SetBaggageItem() enables powerful functionality given a full-stack
	// opentracing integration (e.g., arbitrary application data from a mobile
	// app can make it, transparently, all the way into the depths of a storage
	// system), and with it some powerful costs: use this feature with care.
	//
	// IMPORTANT NOTE #1: SetBaggageItem() will only propagate baggage items to
	// *future* causal descendants of the associated Span.
	//
	// IMPORTANT NOTE #2: Use this thoughtfully and with care. Every key and
	// value is copied into every local *and remote* child of the associated
	// Span, and that can add up to a lot of network and cpu overhead.
	//
	// Returns a reference to this Span for chaining.
	SetBaggageItem(restrictedKey, value string) OpentracingSpan

	// Gets the value for a baggage item given its key. Returns the empty string
	// if the value isn't found in this Span.
	BaggageItem(restrictedKey string) string

	// Provides access to the Tracer that created this Span.
	Tracer() opentracing.Tracer

	// Deprecated: use LogFields or LogKV
	LogEvent(event string)
	// Deprecated: use LogFields or LogKV
	LogEventWithPayload(event string, payload interface{})
	// Deprecated: use LogFields or LogKV
	Log(data opentracing.LogData)
}

const StoreTag = "store"

type span struct {
	span opentracing.Span
}

func newSpan(s opentracing.Span) OpentracingSpan {
	return &span{
		span: s,
	}
}

func (s *span) Finish() {
	if s.span != nil {
		s.span.SetTag(StoreTag, StoreTag)
		ui := uuid.NewString()
		s.span.SetTag("UUID", ui)
		//fmt.Println("SpanFinish UUID=", ui)
		s.span.Finish()
	}
}

func (s *span) FinishWithOptions(opts opentracing.FinishOptions) {
	if s.span != nil {
		s.span.FinishWithOptions(opts)
	}
}

func (s *span) Context() opentracing.SpanContext {
	if s.span != nil {
		s.span.Context()
	}
	return nil
}

func (s *span) SetOperationName(operationName string) OpentracingSpan {
	if s.span != nil {
		s.span = s.span.SetOperationName(operationName)
	}
	return s
}

func (s *span) SetTag(key string, value interface{}) OpentracingSpan {
	if s.span != nil {
		s.span = s.span.SetTag(key, value)
	}
	return s
}

func (s *span) LogFields(fields ...log.Field) {
	if s.span != nil {
		s.span.LogFields(fields...)
	}
}

func (s *span) LogKV(alternatingKeyValues ...interface{}) {
	if s.span != nil {
		s.span.LogKV(alternatingKeyValues...)
	}
}

func (s *span) SetBaggageItem(restrictedKey, value string) OpentracingSpan {
	if s.span != nil {
		s.span = s.span.SetBaggageItem(restrictedKey, value)
	}
	return s
}

func (s *span) BaggageItem(restrictedKey string) string {
	if s.span != nil {
		s.span.BaggageItem(restrictedKey)
	}
	return ""
}

func (s *span) Tracer() opentracing.Tracer {
	if s.span != nil {
		return s.span.Tracer()
	}
	return nil

}

// Deprecated: use LogFields or LogKV
func (s *span) LogEvent(event string) {
	if s.span != nil {
		s.span.LogEvent(event)
	}
}

func (s *span) LogEventWithPayload(event string, payload interface{}) {
	if s.span != nil {
		s.span.LogEventWithPayload(event, payload)
	}
}

func (s *span) Log(data opentracing.LogData) {
	if s.span != nil {
		s.span.Log(data)
	}
}
