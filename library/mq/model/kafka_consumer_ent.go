package model

import (
	"context"
	"time"
)

type KafkaConsumerMessage struct {
	// 上下文，如果是批量消费时，请使用这个ctx，这里会传入这原始单条消息的traceid和namespace；
	// 如果是单条消费，这的ctx的和方法传递的是一致的。
	Ctx            context.Context
	Headers        map[string][]byte
	Timestamp      time.Time
	BlockTimestamp time.Time
	Key, Value     []byte
	Topic          string
	Partition      int32
	Offset         int64
}
