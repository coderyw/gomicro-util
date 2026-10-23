// Package mq
// @Author: yinwei
// @File: mq
// @Version: 1.0.0
// @Date: 2024/7/2 14:22

package mq

import (
	"context"
	"testing"

	"github.com/IBM/sarama"
	"github.com/coderyw/gomicro-util/library/mock"
	"github.com/coderyw/gomicro-util/library/mq/facade"
	kafka2 "github.com/coderyw/gomicro-util/library/mq/kafka"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/trace"
	facade2 "github.com/coderyw/gomicro-util/mocks/library/mq/facade"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"github.com/coderyw/gomicro-util/util"
)

var kafkaManager facade.KafkaManager

func GetKafkaManager(ctx context.Context) facade.KafkaManager {
	// 判断是否开启Mock模式（用于测试）
	if t := mock.CheckAndGetMockClient[facade.KafkaManager](ctx, func(t *testing.T) facade.KafkaManager {
		var mc any = facade2.NewMockKafkaManager(t)
		if val, ok := mc.(facade.KafkaManager); ok {
			return val
		}
		return nil
	}); t != nil {
		return t
	}
	// 检查Redis连接是否已初始化
	if kafkaManager == nil {
		panic("获取redis连接为空")
	}

	return kafkaManager
}

// 初始化kafka
func InitKafka(serverName string, cfg config.KafkaCfg, handler []model.Handler) (closeFunc func(), arr []cron.CronJob, err error) {

	km, cf, arr, err := kafka2.InitConfluentKafkaManager(serverName, cfg, handler)
	kafkaManager = km
	return cf, arr, err

}

// todo 实现 rabbitmq
func InitRabbitMq(serverName string, cfg config.RabbitMqCfgs) (closeFunc func(), arr []cron.CronJob, err error) {

	return nil, nil, nil
}

// GetCtxFromKafkaHeader 从 kafka header获取上下文（traceId）
func GetCtxFromKafkaHeader(headers []*sarama.RecordHeader) context.Context {
	var tid string
	for _, v := range headers {
		if util.Bytes2Str(v.Key) == model.MQ_HEADER_TRACE_ID {
			tid = string(v.Value)
			break
		}
	}
	return trace.GetTraceCtx(context.Background(), tid)
}
