package kafka

import (
	"context"
	"fmt"
	"sync"

	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/mq/facade"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/model/cron"

	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
)

// confluentKafkaManager Confluent-Kafka-Go完整管理器
type confluentKafkaManager struct {
	consumerManager *ConfluentConsumerManager
	producerManager *ConfluentProducerManager
	mu              sync.RWMutex
	initialized     bool
}

// InitConfluentKafkaManager 创建Confluent Kafka管理器
func InitConfluentKafkaManager(serverName string, cfg config.KafkaCfg, handlers []model.Handler) (facade.KafkaManager, func(), []cron.CronJob, error) {
	k1 := &confluentKafkaManager{
		consumerManager: NewConfluentConsumerManager(),
		producerManager: NewConfluentProducerManager(),
	}
	jobs, err := k1.initManager(serverName, cfg, handlers)
	if err != nil {
		return nil, nil, nil, err
	}
	return k1, func() {
		k1.Stop()
	}, jobs, nil

}

// Init 初始化Kafka管理器
func (km *confluentKafkaManager) initManager(serverName string, cfg config.KafkaCfg, handlers []model.Handler) ([]cron.CronJob, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.initialized {
		return nil, fmt.Errorf("Kafka manager already initialized")
	}

	log.Info("[Kafka] Initializing Confluent Kafka manager", zap.String("server", serverName))

	// 填充 KafkaCfg 共享默认值（socket/keepalive/metadata 等），后续 producer/consumer 共用
	cfg.Init()

	// 初始化生产者
	if err := km.producerManager.InitProducer(cfg); err != nil {
		return nil, fmt.Errorf("failed to init producer: %w", err)
	}

	// 初始化消费者
	arr, err := km.consumerManager.InitKafkaConsumer(serverName, cfg, handlers)
	if err != nil {
		// 如果消费者初始化失败，停止生产者
		km.producerManager.Stop()
		return nil, fmt.Errorf("failed to init consumer: %w", err)
	}

	km.initialized = true
	log.Info("[Kafka] Confluent Kafka manager initialized successfully")
	return arr, nil
}

// formatTopic 按 ctx 里的租户为 topic 加前缀。
// 兜底租户模式（TENANT_FALLBACK 已配置）下由 FmtByType 保证返回原始 topic
func formatTopic(ctx context.Context, topic string) string {
	return namespace.FmtByType(namespace.NAMESPACE_TYPE_KAFKA, metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE), topic)
}

// Push 异步推送消息
func (km *confluentKafkaManager) Push(ctx context.Context, key, topic string, obj interface{}, instance ...string) error {
	topic = formatTopic(ctx, topic)
	if len(instance) > 0 {
		return km.producerManager.PushWithProducer(ctx, instance[0], key, topic, obj)
	}
	return km.producerManager.Push(ctx, key, topic, obj)
}

func (km *confluentKafkaManager) PushSync(ctx context.Context, key, topic string, obj interface{}, instance ...string) (err error) {
	topic = formatTopic(ctx, topic)
	if len(instance) > 0 {
		return km.producerManager.PushSyncWithProducer(ctx, instance[0], key, topic, obj)
	}
	return km.producerManager.PushSync(ctx, key, topic, obj)
}

func (km *confluentKafkaManager) GetProduceStats() []model.ProducerStats {
	km.checkInitialized()
	return km.producerManager.GetStats()
}

// GetConsumerCount 获取消费者数量
func (km *confluentKafkaManager) GetConsumerCount() int {
	km.checkInitialized()
	return km.consumerManager.GetConsumerCount()
}

// GetProducerCount 获取生产者数量
func (km *confluentKafkaManager) GetProducerCount() int {
	km.checkInitialized()
	return km.producerManager.GetProducerCount()
}

// GetDefaultProducer 获取默认生产者名称
func (km *confluentKafkaManager) GetDefaultProducer() string {
	km.checkInitialized()
	return km.producerManager.GetDefaultProducer()
}

// IsRunning 检查是否正在运行
func (km *confluentKafkaManager) IsRunning() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if !km.initialized {
		return false
	}

	return km.consumerManager.IsRunning() && km.producerManager.IsRunning()
}

// Flush 刷新所有生产者
func (km *confluentKafkaManager) Flush(timeoutMs int) {
	km.checkInitialized()
	km.producerManager.Flush(timeoutMs)
}

// GetProducerStats 获取生产者统计信息
func (km *confluentKafkaManager) GetProducerStats(producerName string) (map[string]interface{}, error) {
	km.checkInitialized()
	return km.producerManager.GetProducerStats(producerName)
}

// Stop 停止Kafka管理器
func (km *confluentKafkaManager) Stop() {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !km.initialized {
		return
	}

	log.Info("[Kafka] Stopping Confluent Kafka manager")

	// 停止消费者
	if km.consumerManager != nil {
		km.consumerManager.Stop()
	}

	// 停止生产者
	if km.producerManager != nil {
		km.producerManager.Stop()
	}

	km.initialized = false
	log.Info("[Kafka] Confluent Kafka manager stopped")
}

// checkInitialized 检查是否已初始化
func (km *confluentKafkaManager) checkInitialized() {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if !km.initialized {
		panic("Kafka manager not initialized. Call Init() first.")
	}
}

// GetConsumerManager 获取消费者管理器（用于高级操作）
func (km *confluentKafkaManager) GetConsumerManager() *ConfluentConsumerManager {
	km.checkInitialized()
	return km.consumerManager
}

// GetProducerManager 获取生产者管理器（用于高级操作）
func (km *confluentKafkaManager) GetProducerManager() *ConfluentProducerManager {
	km.checkInitialized()
	return km.producerManager
}

// IsInitialized 检查是否已初始化
func (km *confluentKafkaManager) IsInitialized() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.initialized
}

// HealthCheck 健康检查
func (km *confluentKafkaManager) HealthCheck() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	health := map[string]interface{}{
		"initialized": km.initialized,
		"running":     false,
		"consumers":   0,
		"producers":   0,
	}

	if km.initialized {
		health["running"] = km.consumerManager.IsRunning() && km.producerManager.IsRunning()
		health["consumers"] = km.consumerManager.GetConsumerCount()
		health["producers"] = km.producerManager.GetProducerCount()
		health["default_producer"] = km.producerManager.GetDefaultProducer()
	}

	return health
}

//// Restart 重启Kafka管理器
//func (km *confluentKafkaManager) Restart(serverName string, cfg config.KafkaCfg, handlers []mq.Handler) error {
//	log.Info("[Kafka] Restarting Confluent Kafka manager")
//
//	// 停止当前实例
//	km.Stop()
//
//	// 重新初始化
//	return km.Init(serverName, cfg, handlers)
//}

// BatchPush 批量推送消息
func (km *confluentKafkaManager) BatchPush(ctx context.Context, messages []BatchMessage) error {
	km.checkInitialized()

	for _, msg := range messages {
		if err := km.producerManager.Push(ctx, msg.Key, msg.Topic, msg.Data); err != nil {
			return fmt.Errorf("failed to push message %s: %w", msg.Key, err)
		}
	}

	return nil
}

// BatchPushSync 批量同步推送消息
func (km *confluentKafkaManager) BatchPushSync(ctx context.Context, messages []BatchMessage) error {
	km.checkInitialized()

	for _, msg := range messages {
		if err := km.producerManager.PushSync(ctx, msg.Key, msg.Topic, msg.Data); err != nil {
			return fmt.Errorf("failed to push sync message %s: %w", msg.Key, err)
		}
	}

	return nil
}

// BatchMessage 批量消息结构
type BatchMessage struct {
	Key   string
	Topic string
	Data  interface{}
}

// NewBatchMessage 创建批量消息
func NewBatchMessage(key, topic string, data interface{}) BatchMessage {
	return BatchMessage{
		Key:   key,
		Topic: topic,
		Data:  data,
	}
}

// Metrics 获取指标信息
func (km *confluentKafkaManager) Metrics() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	metrics := map[string]interface{}{
		"initialized": km.initialized,
		"consumers":   km.consumerManager.GetConsumerCount(),
		"producers":   km.producerManager.GetProducerCount(),
	}

	if km.initialized {
		metrics["consumer_running"] = km.consumerManager.IsRunning()
		metrics["producer_running"] = km.producerManager.IsRunning()
		metrics["default_producer"] = km.producerManager.GetDefaultProducer()
	}

	return metrics
}
