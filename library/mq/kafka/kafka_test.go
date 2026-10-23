// Package kafka
// @Author: yinwei
// @File: kafka_test
// @Version: 1.0.0
// @Date: 2024/11/4 17:29

package kafka

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/coderyw/gomicro-util/library/cron"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/single"
	"github.com/coderyw/gomicro-util/library/trace"
	"github.com/coderyw/gomicro-util/model/config"
	cron2 "github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

var wg = sync.WaitGroup{}

func createTopic() {
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": "b-2-public.pockafka.w4c0u7.c3.kafka.ap-southeast-1.amazonaws.com:9196",
	})
	if err != nil {
		panic(err)
	}
	defer admin.Close()

	topic := "tenant_0-test-topic"

	spec := kafka.TopicSpecification{
		Topic:             topic,
		NumPartitions:     3,
		ReplicationFactor: 2,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	results, err := admin.CreateTopics(
		ctx,
		[]kafka.TopicSpecification{spec},
	)

	if err != nil {
		panic(err)
	}

	for _, r := range results {
		fmt.Println(r.Topic, r.Error)
	}
}

func TestCreateTopic(t *testing.T) {
	createTopic()
}

func TestKafkaSingle(t *testing.T) {
	wg.Add(1)
	//    - '192.168.3.33:9092'
	//    - '192.168.44.199:9092'
	//    - '192.168.55.82:9092'
	acks := int16(1)
	//ctx := metadata.SetCtxData(context.TODO(), metadata.CONTEXT_NAME_SPACE, "123")
	mgr, c, cj, err := InitConfluentKafkaManager("teste1", config.KafkaCfg{
		DefaultProducer: "ca-msk-demo",
		Hosts: map[string]config.KafkaHost{
			"ca-msk-demo": {
				// 私网
				Host: []string{
					"b-1.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
					"b-2.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
					"b-3.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
				},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "ca-msk-demo",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
		},
		Consumer: []config.KafkaConsumer{
			{
				Name: "ca-msk-demo",
				Groups: []config.Group{
					{
						GroupId:               "test-21111",
						MaxMulNum:             50,
						MulTimeoutMs:          1000,
						MulTimeoutType:        2,
						MaxProcessingTimeMill: 30000,
						SessionTimeoutMs:      30000,
						OffsetInitial:         -2,
					},
				},
			},
		},
	}, []model.Handler{
		{
			InstanceName: "ca-msk-demo",
			GroupId:      "test-21111",
			HandlerFunc: func(ctx context.Context, msg model.KafkaConsumerMessage) error {
				log.InfoCtx(ctx, "consumer", zap.Any("data", msg))
				return nil
			},
			HandlerFuncs: nil,
			Topics:       "user-bet-sts-topic",
			BeforeStop:   nil,
		},
	})
	if err != nil {
		t.Error(err)
	} else {
		cron.Init()
		for _, v := range cj {
			cron.AddCron(v)
		}
		cron.AddCron(cron2.CronJob{
			Name:    "test-cron",
			CronSpc: "*/1 * * * * ?",
			DoJob: func() {
				return
				k := rand.Int63n(2)
				k2 := rand.Int63n(10)
				ctx := metadata.SetCtxData(context.TODO(), metadata.CONTEXT_NAME_SPACE, fmt.Sprint(k))
				mgr.Push(trace.GetTraceCtx(ctx), fmt.Sprintf("test-topic:%v", k2), "test-topic",
					"{\"balance_record_id\":119863906031005701,\"create_time\":1754538645,\"user_id\":\"74569708752400390\",\"transfer_amount\":10000,\"bet_amount\":0,\"trace_id\":\"E4A4743D3D4D3D28\",\"money_type\":2,\"update_type\":11,\"game_id\":0,\"company_id\":0,\"detail\":\"\",\"is_count_round\":0,\"bet_status\":0,\"balance\":10083}",
					"ca-msk-demo")
			},
		})
		<-single.OsKillSingle()
		cron.CloseCron()
		c()
		time.Sleep(3 * time.Second)
	}
}

func TestKafkaBatch(t *testing.T) {
	wg.Add(1)
	//    - '192.168.3.33:9092'
	//    - '192.168.44.199:9092'
	//    - '192.168.55.82:9092'
	acks := int16(1)
	if _, c, _, err := InitConfluentKafkaManager("teste1", config.KafkaCfg{
		DefaultProducer: "ca-msk-demo",
		Hosts: map[string]config.KafkaHost{
			"ca-msk-demo": {
				// 私网
				Host: []string{
					"b-1.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
					"b-2.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
					"b-3.devkafkacluster.dzcict.c4.kafka.ap-southeast-1.amazonaws.com:9092",
				},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "ca-msk-demo",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
		},
		Consumer: []config.KafkaConsumer{
			{
				Name: "ca-msk-demo",
				Groups: []config.Group{
					{
						GroupId:               "test-221",
						MaxMulNum:             50,
						MulTimeoutMs:          1000,
						MulTimeoutType:        2,
						MaxProcessingTimeMill: 30000,
						SessionTimeoutMs:      30000,
						OffsetInitial:         -2,
					},
				},
			},
		},
	}, []model.Handler{
		{
			InstanceName: "ca-msk-demo",
			GroupId:      "test-221",
			HandlerFuncs: func(ctx context.Context, msgs []model.KafkaConsumerMessage) (int, error) {
				log.InfoCtx(ctx, "consumer", zap.Any("data", msgs))
				return 0, nil
			},
			Topics:     "user-bet-sts-topic",
			BeforeStop: nil,
		},
	}); err != nil {
		t.Error(err)
	} else {
		<-single.OsKillSingle()
		c()
		time.Sleep(3 * time.Second)
	}
}

func TestPush(t *testing.T) {
	acks := int16(1)
	if mgr, c, _, err := InitConfluentKafkaManager("teste1", config.KafkaCfg{
		DefaultProducer: "core",
		Hosts: map[string]config.KafkaHost{
			"core": {
				Host: []string{"192.168.3.33:9092", "192.168.44.199:9092", "192.168.55.82:9092"},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "core",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
		},
	}, nil); err != nil {
		t.Error(err)
	} else {
		t := time.NewTicker(time.Millisecond * 3000)
		sg := single.OsKillSingle()
	forBk:
		for {
			select {
			case <-t.C:
				mgr.Push(context.Background(), "key", "topic-test", "{\"balance_record_id\":119863906031005701,\"create_time\":1754538645,\"user_id\":\"74569708752400390\",\"transfer_amount\":10000,\"bet_amount\":0,\"trace_id\":\"E4A4743D3D4D3D28\",\"money_type\":2,\"update_type\":11,\"game_id\":0,\"company_id\":0,\"detail\":\"\",\"is_count_round\":0,\"bet_status\":0,\"balance\":10083}")
			case <-sg:
				break forBk
			}
		}

		c()
		time.Sleep(3 * time.Second)
	}
}

func TestPushSync(t *testing.T) {
	acks := int16(-1)
	if mgr, c, _, err := InitConfluentKafkaManager("teste1", config.KafkaCfg{
		DefaultProducer: "core",
		Hosts: map[string]config.KafkaHost{
			"core": {
				Host: []string{"192.168.3.33:9092", "192.168.44.199:9092", "192.168.55.82:9092"},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "core",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
		},
	}, nil); err != nil {
		t.Error(err)
	} else {
		t := time.NewTicker(time.Millisecond * 200)
		sg := single.OsKillSingle()
	forBk:
		for {
			select {
			case <-t.C:
				mgr.PushSync(context.Background(), "key", "ca_balance_balance_stats_topic", "{\"balance_record_id\":119863906031005701,\"create_time\":1754538645,\"user_id\":\"74569708752400390\",\"transfer_amount\":10000,\"bet_amount\":0,\"trace_id\":\"E4A4743D3D4D3D28\",\"money_type\":2,\"update_type\":11,\"game_id\":0,\"company_id\":0,\"detail\":\"\",\"is_count_round\":0,\"bet_status\":0,\"balance\":10083}")
			case <-sg:
				break forBk
			}
		}

		c()
		time.Sleep(3 * time.Second)
	}
}

func TestPushSyncInstance(t *testing.T) {

	acks := int16(1)
	if mgr, c, _, err := InitConfluentKafkaManager("teste1", config.KafkaCfg{
		DefaultProducer: "core",
		Hosts: map[string]config.KafkaHost{
			"core": {
				Host: []string{"192.168.3.33:9092", "192.168.44.199:9092", "192.168.55.82:9092"},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "core",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
			{
				Name:          "bt",
				MaxRetryTimes: 3,
				RequiredAcks:  &acks,
			},
		},
	}, nil); err != nil {
		t.Error(err)
	} else {
		t := time.NewTicker(time.Millisecond * 200)
		t2 := time.NewTicker(time.Millisecond * 200)
		sg := single.OsKillSingle()
	forBk:
		for {
			select {
			case <-t.C:
				mgr.PushSync(context.Background(),
					"key",
					"ca_balance_balance_stats_topic", "{\"balance_record_id\":119863906031005701,\"create_time\":1754538645,\"user_id\":\"74569708752400390\",\"transfer_amount\":10000,\"bet_amount\":0,\"trace_id\":\"E4A4743D3D4D3D28\",\"money_type\":2,\"update_type\":11,\"game_id\":0,\"company_id\":0,\"detail\":\"\",\"is_count_round\":0,\"bet_status\":0,\"balance\":10083}",
					"core",
				)
				mgr.PushSync(context.Background(),
					"key",
					"ca_balance_balance_stats_topic", "{\"balance_record_id\":119863906031005701,\"create_time\":1754538645,\"user_id\":\"74569708752400390\",\"transfer_amount\":10000,\"bet_amount\":0,\"trace_id\":\"E4A4743D3D4D3D28\",\"money_type\":2,\"update_type\":11,\"game_id\":0,\"company_id\":0,\"detail\":\"\",\"is_count_round\":0,\"bet_status\":0,\"balance\":10083}",
					"bt",
				)
			case <-t2.C:
				log.Info("stats", zap.Any("s", mgr.GetProduceStats()))
			case <-sg:
				break forBk
			}
		}

		c()
		time.Sleep(3 * time.Second)
	}
}
