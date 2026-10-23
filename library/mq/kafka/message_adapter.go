package kafka

import (
	"context"
	"time"

	"github.com/IBM/sarama"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

// SaramaMessage 定义Sarama消息结构，用于保持接口兼容性
type SaramaMessage struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []SaramaHeader
	Timestamp time.Time
}

// SaramaHeader 定义Sarama消息头结构
type SaramaHeader struct {
	Key   []byte
	Value []byte
}

// ConfluentToSaramaMessage 将Confluent消息转换为Sarama消息
func ConfluentToSaramaMessage(msg *kafka.Message) SaramaMessage {
	headers := make([]SaramaHeader, len(msg.Headers))
	for i, header := range msg.Headers {
		headers[i] = SaramaHeader{
			Key:   []byte(header.Key),
			Value: header.Value,
		}
	}

	return SaramaMessage{
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Timestamp: msg.Timestamp,
	}
}

// ConfluentToSaramaMessages 将Confluent消息切片转换为Sarama消息切片
func ConfluentToSaramaMessages(msgs []*kafka.Message) []SaramaMessage {
	result := make([]SaramaMessage, len(msgs))
	for i, msg := range msgs {
		result[i] = ConfluentToSaramaMessage(msg)
	}
	return result
}

// SaramaToConfluentMessage 将Sarama消息转换为Confluent消息
func SaramaToConfluentMessage(msg *sarama.ConsumerMessage) *kafka.Message {
	headers := make([]kafka.Header, len(msg.Headers))
	for i, header := range msg.Headers {
		headers[i] = kafka.Header{
			Key:   string(header.Key),
			Value: header.Value,
		}
	}

	topic := msg.Topic
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: msg.Partition,
			Offset:    kafka.Offset(msg.Offset),
		},
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Timestamp: msg.Timestamp,
	}
}

// SaramaToConfluentMessages 将Sarama消息切片转换为Confluent消息切片
func SaramaToConfluentMessages(msgs []sarama.ConsumerMessage) []*kafka.Message {
	result := make([]*kafka.Message, len(msgs))
	for i := range msgs {
		result[i] = SaramaToConfluentMessage(&msgs[i])
	}
	return result
}

// MessageAdapter 消息适配器接口
type MessageAdapter interface {
	// 单条消息处理适配器
	AdaptSingleMessage(msg *kafka.Message) SaramaMessage
	// 批量消息处理适配器
	AdaptBatchMessages(msgs []*kafka.Message) []SaramaMessage
	// 提取消息头
	ExtractHeader(msg *kafka.Message, key string) string
}

// DefaultMessageAdapter 默认消息适配器
type DefaultMessageAdapter struct{}

// NewMessageAdapter 创建消息适配器
func NewMessageAdapter() MessageAdapter {
	return &DefaultMessageAdapter{}
}

// AdaptSingleMessage 适配单条消息
func (a *DefaultMessageAdapter) AdaptSingleMessage(msg *kafka.Message) SaramaMessage {
	return ConfluentToSaramaMessage(msg)
}

// AdaptBatchMessages 适配批量消息
func (a *DefaultMessageAdapter) AdaptBatchMessages(msgs []*kafka.Message) []SaramaMessage {
	return ConfluentToSaramaMessages(msgs)
}

// ExtractHeader 提取消息头
func (a *DefaultMessageAdapter) ExtractHeader(msg *kafka.Message, key string) string {
	for _, header := range msg.Headers {
		if string(header.Key) == key {
			return string(header.Value)
		}
	}
	return ""
}

// HandlerAdapter 处理器适配器，用于适配不同的消息格式
type HandlerAdapter struct {
	adapter MessageAdapter
}

// NewHandlerAdapter 创建处理器适配器
func NewHandlerAdapter() *HandlerAdapter {
	return &HandlerAdapter{
		adapter: NewMessageAdapter(),
	}
}

// AdaptSingleHandler 适配单条消息处理器
func (ha *HandlerAdapter) AdaptSingleHandler(handler func(ctx context.Context, msg SaramaMessage) error) func(ctx context.Context, msg *kafka.Message) error {
	return func(ctx context.Context, msg *kafka.Message) error {
		saramaMsg := ha.adapter.AdaptSingleMessage(msg)
		return handler(ctx, saramaMsg)
	}
}

// AdaptBatchHandler 适配批量消息处理器
func (ha *HandlerAdapter) AdaptBatchHandler(handler func(ctx context.Context, msgs []SaramaMessage) (int, error)) func(ctx context.Context, msgs []*kafka.Message) (int, error) {
	return func(ctx context.Context, msgs []*kafka.Message) (int, error) {
		saramaMsgs := ha.adapter.AdaptBatchMessages(msgs)
		return handler(ctx, saramaMsgs)
	}
}
