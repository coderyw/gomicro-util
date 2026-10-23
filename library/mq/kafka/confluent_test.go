package kafka

import (
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	config2 "github.com/coderyw/gomicro-util/model/config"
)

func TestNewConfluentConsumerManager(t *testing.T) {
	manager := NewConfluentConsumerManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.GetConsumerCount())
	assert.True(t, manager.IsRunning())
}

func TestConfluentConsumerManager_Stop(t *testing.T) {
	manager := NewConfluentConsumerManager()

	// 启动一个goroutine来模拟消费者
	manager.wg.Add(1)
	go func() {
		defer manager.wg.Done()
		select {
		case <-manager.stopChan:
			return
		case <-time.After(100 * time.Millisecond):
			return
		}
	}()

	manager.Stop()
	assert.False(t, manager.IsRunning())
}

func TestConfluentBuildGroupConfigs(t *testing.T) {
	manager := NewConfluentConsumerManager()

	cfg := config2.KafkaCfg{
		Hosts: map[string]config2.KafkaHost{"default": config2.KafkaHost{Host: []string{"localhost:9092"}}},
		Consumer: []config2.KafkaConsumer{
			{
				Name: "default",
				Groups: []config2.Group{
					{
						GroupId:   "test-group",
						MaxMulNum: 50,
					},
				},
			},
		},
	}

	groupCfg, err := manager.buildGroupConfigs(cfg)
	require.NoError(t, err)
	assert.Len(t, groupCfg, 1)
	assert.Equal(t, "test-group", cfg.Consumer[0].Groups[0].GroupId)
}

func TestConfluentBuildGroupConfigs_DefaultConsumerNotFound(t *testing.T) {
	manager := NewConfluentConsumerManager()

	cfg := config2.KafkaCfg{
		Hosts: map[string]config2.KafkaHost{"default": config2.KafkaHost{Host: []string{"localhost:9092"}}},

		Consumer: []config2.KafkaConsumer{
			{
				Name: "default",
				Groups: []config2.Group{
					{
						GroupId: "test-group",
					},
				},
			},
		},
	}

	_, err := manager.buildGroupConfigs(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default consumer not found")
}

func TestMessageAdapter(t *testing.T) {
	adapter := NewMessageAdapter()

	topic := "test-topic"
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: 0,
			Offset:    123,
		},
		Key:   []byte("key"),
		Value: []byte("value"),
		Headers: []kafka.Header{
			{
				Key:   "header-key",
				Value: []byte("header-value"),
			},
		},
		Timestamp: time.Now(),
	}

	saramaMsg := adapter.AdaptSingleMessage(msg)
	assert.Equal(t, "test-topic", saramaMsg.Topic)
	assert.Equal(t, int32(0), saramaMsg.Partition)
	assert.Equal(t, int64(123), saramaMsg.Offset)
	assert.Equal(t, "key", string(saramaMsg.Key))
	assert.Equal(t, "value", string(saramaMsg.Value))
	assert.Len(t, saramaMsg.Headers, 1)
	assert.Equal(t, "header-key", string(saramaMsg.Headers[0].Key))
	assert.Equal(t, "header-value", string(saramaMsg.Headers[0].Value))
}

func TestMessageAdapter_ExtractHeader(t *testing.T) {
	adapter := NewMessageAdapter()

	msg := &kafka.Message{
		Headers: []kafka.Header{
			{
				Key:   "sw8",
				Value: []byte("test-sw8-value"),
			},
			{
				Key:   "other",
				Value: []byte("other-value"),
			},
		},
	}

	sw8Header := adapter.ExtractHeader(msg, "sw8")
	assert.Equal(t, "test-sw8-value", sw8Header)

	otherHeader := adapter.ExtractHeader(msg, "other")
	assert.Equal(t, "other-value", otherHeader)

	notFoundHeader := adapter.ExtractHeader(msg, "not-found")
	assert.Equal(t, "", notFoundHeader)
}
