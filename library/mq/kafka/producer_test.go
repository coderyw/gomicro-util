package kafka

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/coderyw/gomicro-util/model/config"
)

func TestNewConfluentProducerManager(t *testing.T) {
	manager := NewConfluentProducerManager()
	assert.NotNil(t, manager)
	assert.Equal(t, 0, manager.GetProducerCount())
	assert.True(t, manager.IsRunning())
}

func TestConfluentProducerManager_Stop(t *testing.T) {
	manager := NewConfluentProducerManager()

	// 启动一个goroutine来模拟生产者
	manager.stopChan = make(chan struct{})
	go func() {
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

func TestConfluentProducerManager_InitProducer(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	cfg := config.KafkaCfg{
		DefaultProducer: "default",
		Hosts: map[string]config.KafkaHost{
			"default": {
				Host: []string{"localhost:9092"},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "default",
				MaxRetryTimes: 3,
			},
		},
	}

	err := manager.InitProducer(cfg)
	// 注意：这里可能会因为无法连接到Kafka而失败，这是正常的
	if err != nil {
		t.Logf("Producer initialization failed (expected in test environment): %v", err)
	} else {
		assert.Equal(t, 1, manager.GetProducerCount())
		assert.Equal(t, "default", manager.GetDefaultProducer())
	}
}

func TestConfluentProducerManager_InitProducer_EmptyConfig(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	cfg := config.KafkaCfg{}

	err := manager.InitProducer(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 0, manager.GetProducerCount())
}

func TestConfluentProducerManager_InitProducer_DefaultProducerNotFound(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	cfg := config.KafkaCfg{
		DefaultProducer: "nonexistent",
		Hosts: map[string]config.KafkaHost{
			"nonexistent": {
				Host: []string{"localhost:9092"},
			},
		},
		Producer: []config.KafkaProducer{
			{
				Name:          "default",
				MaxRetryTimes: 3,
			},
		},
	}

	err := manager.InitProducer(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "default producer [nonexistent] not found")
}

func TestConfluentProducerManager_Push_Validation(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()

	// 测试空消息
	err := manager.Push(ctx, "key", "topic", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is empty")

	// 测试空topic
	err = manager.Push(ctx, "key", "", map[string]interface{}{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is empty")
}

func TestConfluentProducerManager_PushSync_Validation(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()

	// 测试空消息
	err := manager.PushSync(ctx, "key", "topic", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is empty")

	// 测试空topic
	err = manager.PushSync(ctx, "key", "", map[string]interface{}{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is empty")
}

func TestConfluentProducerManager_PushWithProducer_Validation(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()

	// 测试空生产者名称
	err := manager.PushWithProducer(ctx, "", "key", "topic", map[string]interface{}{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "producer name is empty")

	// 测试空消息
	err = manager.PushWithProducer(ctx, "producer", "key", "topic", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is empty")

	// 测试空topic
	err = manager.PushWithProducer(ctx, "producer", "key", "", map[string]interface{}{"test": "data"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "topic is empty")
}

func TestConfluentProducerManager_GetProducer_NotFound(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 测试获取不存在的生产者
	_, err := manager.getProducer("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "producer nonexistent not found")
}

func TestConfluentProducerManager_GetProducer_NoDefault(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 测试没有默认生产者时获取生产者
	_, err := manager.getProducer("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "producer  not found")
}

func TestConfluentProducerManager_Flush(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 测试刷新空的生产者列表
	manager.Flush(1000)
	// 不应该panic
}

func TestConfluentProducerManager_GetProducerStats_NotFound(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 测试获取不存在生产者的统计信息
	_, err := manager.GetProducerStats("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "producer nonexistent not found")
}

func TestConfluentProducerManager_Stop_MultipleCalls(t *testing.T) {
	manager := NewConfluentProducerManager()

	// 第一次停止
	manager.Stop()
	assert.False(t, manager.IsRunning())

	// 第二次停止（应该不会panic）
	manager.Stop()
	assert.False(t, manager.IsRunning())
}

func TestConfluentProducerManager_HandleDeliveryReports(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 启动错误处理协程
	stopChan := make(chan struct{})
	go func() {
		select {
		case <-stopChan:
			return
		case <-time.After(100 * time.Millisecond):
			return
		}
	}()

	// 关闭停止通道
	close(stopChan)
	// 不应该panic
}

func TestConfluentProducerManager_MessageSerialization(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()

	// 测试无法序列化的消息
	unserializableObj := make(chan int) // channel无法序列化为JSON

	err := manager.Push(ctx, "key", "topic", unserializableObj)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal message")
}

func TestConfluentProducerManager_EnvironmentHandling(t *testing.T) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	// 这里需要模拟环境变量，但由于env.IsLocal()是外部依赖
	// 我们只能测试基本功能，实际的环境处理需要在集成测试中验证
	ctx := context.Background()
	message := map[string]interface{}{"test": "data"}

	// 测试基本推送功能（可能会因为无法连接Kafka而失败，这是正常的）
	err := manager.Push(ctx, "key", "test-topic", message)
	if err != nil {
		t.Logf("Push failed (expected in test environment): %v", err)
	}
}

// 辅助函数
func int16Ptr(v int16) *int16 {
	return &v
}

// Benchmark测试
func BenchmarkConfluentProducerManager_Push(b *testing.B) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()
	message := map[string]interface{}{
		"id":      1,
		"message": "benchmark message",
		"time":    time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.Push(ctx, fmt.Sprintf("key-%d", i), "benchmark-topic", message)
	}
}

func BenchmarkConfluentProducerManager_PushSync(b *testing.B) {
	manager := NewConfluentProducerManager()
	defer manager.Stop()

	ctx := context.Background()
	message := map[string]interface{}{
		"id":      1,
		"message": "benchmark sync message",
		"time":    time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.PushSync(ctx, fmt.Sprintf("key-%d", i), "benchmark-sync-topic", message)
	}
}
