package kafka

import (
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/model/cst"
	"go.uber.org/zap"
)

// batchHandler 批量消息处理器
func (cm *ConsumerGroup) batchHandler(ch chan *kafka.Message, hd model.Handler, partition int32) {
	defer cm.wg.Done()
	// 执行 before stop
	defer func() {
		if hd.BeforeStop != nil {
			hd.BeforeStop()
		}
	}()

	mulMaxWait := time.Duration(cm.MulTimeoutMs) * time.Millisecond
	ticker := time.NewTicker(mulMaxWait)
	defer ticker.Stop()
	// 批量消费最大数量
	data := make([]*kafka.Message, 0, cm.MaxMulNum)

	for {
		select {
		case msg, more := <-ch:
			if !more {
				log.Info("[Kafka] batch consumer stopped", zap.Any("key", cm.key), zap.Any("partition", partition))
				return
			}

			data = append(data, msg)

			if len(data) >= cm.MaxMulNum {
				cm.processBatchMessages(data, partition)
				data = data[:0]
			} else if cm.MulTimeoutType == 1 {
				ticker.Reset(mulMaxWait)
			}

		case <-ticker.C:
			if len(data) > 0 {
				cm.processBatchMessages(data, partition)
				data = data[:0]
			}
		}
	}
}

// processBatchMessages 处理批量消息
func (cm *ConsumerGroup) processBatchMessages(data []*kafka.Message, partition int32) {

	if cm.stopFlag.Load() { //如果已经停止了，不去处理下面数据
		return
	}
	// 批量处理
	cm.processBatch(partition, data)
	// 提交偏移量
	cm.commit(data[len(data)-1])
}

// processBatch 处理批量消息
func (cm *ConsumerGroup) processBatch(partition int32, msgs []*kafka.Message) {
	if len(msgs) == 0 {
		return
	}
	ctx := cm.handleHeader(msgs[0])
	start := time.Now()
	var err error
	defer func() {
		if pinfo := recover(); pinfo != nil {
			// panic 会打印异常，不会进行处理。这里要业务去处理 panic。
			log.ErrorCtx(ctx, "[Kafka] panic", zap.Any("panic", pinfo))
			log.PrintPanic(pinfo)
			err = errors.New("panic")
		} else if err != nil {
			log.ErrorCtx(ctx, "[Kafka] consumption error", zap.String("topic", cm.handler.Topics), zap.Error(err))
		}

		du := time.Since(start)
		log.InfoCtx(ctx, "[Kafka] consumption completed",
			zap.String("GroupId", cm.GroupId),
			zap.String("Topic", cm.handler.Topics),
			zap.Int32("Partition", partition),
			zap.Int("Count", len(msgs)),
			zap.Duration("Duration", du),
			zap.String(log.DUR_LOG_KEY, cst.GetKafkaConsumerHandleLevel(du)))
	}()

	// 转换为Sarama格式的消息
	saramaMsgs := make([]model.KafkaConsumerMessage, len(msgs))
	for i, msg := range msgs {
		saramaMsgs[i] = model.KafkaConsumerMessage{
			Ctx:       cm.handleHeader(msg),
			Topic:     *msg.TopicPartition.Topic,
			Partition: msg.TopicPartition.Partition,
			Offset:    int64(msg.TopicPartition.Offset),
			Key:       msg.Key,
			Value:     msg.Value,
			Headers:   cm.convertToSaramaHeaders(msg.Headers),
			Timestamp: msg.Timestamp,
		}
	}
	log.InfoCtx(ctx, "[Kafka] batch consumption start",
		zap.String("groupId", cm.GroupId),
		zap.String("topic", cm.handler.Topics),
		zap.Int32("partition", partition),
		zap.Int("count", len(msgs)))
	_, err = cm.handler.HandlerFuncs(ctx, saramaMsgs)
}
