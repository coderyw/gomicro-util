package kafka

import (
	"context"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/library/trace"
	"github.com/coderyw/gomicro-util/model/cst"
	"go.uber.org/zap"
)

// singleHandler 单条消息处理器
func (cm *ConsumerGroup) singleHandler(ch chan *kafka.Message, hd model.Handler, partition int32) {
	// 单个消费停止时候，需要清理 waitgroup
	defer cm.wg.Done()
	// 执行 before stop
	defer func() {
		// 捕获一下异常，防止BeforeStop运行时出现了 panic
		defer func() {
			if err := recover(); err != nil {
				log.Error("[Kafka] panic", zap.Any("panic", err))
				log.PrintPanic(err)
			}
		}()
		if hd.BeforeStop != nil {
			hd.BeforeStop()
		}
	}()
	// 循环消费
	for msg := range ch {
		if err := cm.processSingleMessage(hd.HandlerFunc, msg, cm.GroupId); err != nil {
			log.Error("[Kafka] single message processing error", zap.Error(err))
			// 这里值记录错误信息
		}
		cm.commit(msg)
	}

	log.Info("[Kafka] single consumer stopped", zap.Any("key", cm.key), zap.Any("partition", partition))
}

// 从mq获取tracedid
func (cm *ConsumerGroup) handleHeader(msg *kafka.Message) context.Context {
	ctx := context.Background()
	// 解析 namespace 从 topic
	if msg.TopicPartition.Topic != nil {
		ctx = namespace.GetNameSpaceCtxFromKey(ctx, namespace.NAMESPACE_TYPE_KAFKA, *msg.TopicPartition.Topic)
	} else {
		log.Warn("[Kafka] topic is nil")
	}
	gotTrace := false
	for _, v := range msg.Headers {
		switch v.Key {
		// traceId
		case model.MQ_HEADER_TRACE_ID:

			if len(v.Value) != 0 {
				gotTrace = true
				ctx = trace.SetTraceId(ctx, Bytes2Str(v.Value))
			}
		}
	}
	if gotTrace {
		return ctx
	}
	return trace.SetTraceId(ctx, uuid.NewString())
}

// processSingleMessage 处理单条消息
func (cm *ConsumerGroup) processSingleMessage(handler model.HandlerFunc, msg *kafka.Message, groupId string) error {
	// 停掉 sky 的上报
	//trace2.CreateEntrySpan("Kafka/ConsumerTopic/"+*msg.TopicPartition.Topic, func(headerKey string) (string, error) {
	//	return cm.extractSw8Header(msg), nil
	//})
	//defer trace2.StopSpan()
	// 生成 traceId
	ctx := cm.handleHeader(msg)

	// 防止出现 panic
	defer func() {
		if err := recover(); err != nil {
			log.ErrorCtx(ctx, "[Kafka] panic", zap.Any("panic", err))
			log.PrintPanic(err)
		}
	}()

	// 20260618移除消费开始打印
	//log.InfoCtx(ctx, "[Kafka] consumption",
	//	zap.String("groupId", groupId),
	//	zap.Time("time", msg.Timestamp),
	//	zap.String("topic", *msg.TopicPartition.Topic),
	//	zap.Int32("partition", msg.TopicPartition.Partition),
	//	zap.Int64("offset", int64(msg.TopicPartition.Offset)),
	//	zap.String("data", Bytes2Str(msg.Value)))

	start := time.Now()

	// 转换为包装的格式的消息
	saramaMsg := model.KafkaConsumerMessage{
		Topic:     *msg.TopicPartition.Topic,
		Partition: msg.TopicPartition.Partition,
		Offset:    int64(msg.TopicPartition.Offset),
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   cm.convertToSaramaHeaders(msg.Headers),
		Ctx:       ctx,
		Timestamp: msg.Timestamp,
	}

	if err := handler(ctx, saramaMsg); err != nil {
		return err
	}

	du := time.Since(start)
	log.InfoCtx(ctx, "[Kafka] consumption completed",
		zap.String("groupId", groupId),
		zap.Time("time", msg.Timestamp),
		zap.String("topic", *msg.TopicPartition.Topic),
		zap.Int32("partition", msg.TopicPartition.Partition),
		zap.Int64("offset", int64(msg.TopicPartition.Offset)),
		zap.Duration("Duration", du),
		zap.String(log.DUR_LOG_KEY, cst.GetKafkaConsumerHandleLevel(du)))

	return nil
}

// convertToSaramaHeaders 转换消息头为Sarama格式
func (cm *ConsumerGroup) convertToSaramaHeaders(headers []kafka.Header) map[string][]byte {
	result := make(map[string][]byte, len(headers))

	for _, header := range headers {
		result[header.Key] = header.Value
	}

	return result
}
