package kafka

import (
	"context"
	json2 "encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/pmicro/wrapper/validator"
	"github.com/coderyw/gomicro-util/library/skywalking"
	"github.com/coderyw/gomicro-util/library/trace"
	"github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cst"
	"github.com/coderyw/gomicro-util/util"
	"github.com/coderyw/gomicro-util/util/json"
	"go.uber.org/zap"
)

var cpm *ConfluentProducerManager

type pushStats struct {
	PushTime time.Duration
	Instance string
	Topic    string
}

type producerStats struct {
	Instance    string
	PushCount   int64
	AvaPushTime time.Duration
	Topic       string
}

// ConfluentProducerManager Confluent-Kafka-Go生产者管理器
type ConfluentProducerManager struct {
	producers       map[string]*kafka.Producer
	defaultProducer string
	mu              sync.RWMutex
	stopChan        chan struct{}
	stopFlag        bool
	statsMu         sync.RWMutex
	pushData        map[string]*producerStats
	pushtimeStats   chan pushStats
}

// NewConfluentProducerManager 创建Confluent生产者管理器
func NewConfluentProducerManager() *ConfluentProducerManager {
	cpm = &ConfluentProducerManager{
		producers:     make(map[string]*kafka.Producer),
		stopChan:      make(chan struct{}),
		pushtimeStats: make(chan pushStats, 1024),
		pushData:      make(map[string]*producerStats),
	}
	return cpm
}

// InitProducer 初始化生产者
func (pm *ConfluentProducerManager) InitProducer(cfg config.KafkaCfg) error {
	if cfg.Producer == nil || len(cfg.Producer) == 0 {
		return nil
	}

	log.Info("初始化Confluent Kafka producer", zap.Any("host", cfg))

	// 是否获取到默认producer
	gotDefault := false

	// 循环初始化
	for _, v := range cfg.Producer {
		if cfg.DefaultProducer != "" && v.Name == cfg.DefaultProducer {
			gotDefault = true
		}
		// 配置编辑
		configMap, err := pm.buildProducerConfig(cfg, v)
		if err != nil {
			return fmt.Errorf("failed to build config for producer %s: %w", v.Name, err)
		}

		producer, err := kafka.NewProducer(configMap)
		if err != nil {
			return fmt.Errorf("failed to create producer %s: %w", v.Name, err)
		}

		pm.mu.Lock()
		pm.producers[v.Name] = producer
		pm.mu.Unlock()

		// 启动错误处理协程
		go pm.handleDeliveryReports(producer, v.Name,
			config.PoolStatsLogEnabled(v.StatsMode), config.PoolStatsReportEnabled(v.StatsMode))
	}

	// 验证默认生产者配置
	if cfg.DefaultProducer != "" && !gotDefault {
		return fmt.Errorf("default producer [%s] not found in config", cfg.DefaultProducer)
	}

	// 设置默认生产者
	if cfg.DefaultProducer == "" {
		log.Warn("[Kafka] 默认推送者没配置", zap.String("使用第一个配置", cfg.Producer[0].Name))
		pm.defaultProducer = cfg.Producer[0].Name
	} else {
		pm.defaultProducer = cfg.DefaultProducer
	}
	go cpm.stats()
	return nil
}

// 推送统计
func (pm *ConfluentProducerManager) stats() {
	for {
		select {
		case <-pm.stopChan:
			log.Info("[Kafka] Producer 停止 stats")
			return
		case stats := <-pm.pushtimeStats:
			pm.statsMu.Lock()
			key := fmt.Sprintf("%v:%v", stats.Instance, stats.Topic)
			if val, ok := pm.pushData[key]; ok {
				val.AvaPushTime = (val.AvaPushTime*time.Duration(val.PushCount) + stats.PushTime) / time.Duration(val.PushCount+1)
				val.PushCount++
			} else {
				pm.pushData[key] = &producerStats{
					Instance:    stats.Instance,
					AvaPushTime: stats.PushTime,
					PushCount:   1,
					Topic:       stats.Topic,
				}
			}
			pm.statsMu.Unlock()
		}
	}
}

// GetStats GetStats 获取推送统计信息
//
// Returns：
//   - []ProducerStats 返回 instance 下的 topic 推送统计
func (pm *ConfluentProducerManager) GetStats() []model.ProducerStats {
	pm.statsMu.RLock()
	defer pm.statsMu.RUnlock()
	tr := make([]model.ProducerStats, 0, len(pm.pushData))
	for _, v := range pm.pushData {
		tr = append(tr, model.ProducerStats{
			Instance:    v.Instance,
			PushCount:   v.PushCount,
			AvaPushTime: v.AvaPushTime.String(),
			Topic:       v.Topic,
		})
	}
	return tr
}

// 生产者配置默认值；当 producerCfg 中对应字段未配置（零值/nil）时采用
const (
	defaultAcks                             = "-1"
	defaultRetries                          = 3
	defaultLingerMs                         = 5
	defaultCompressionType                  = "lz4"
	defaultMaxInFlightRequestsPerConnection = 5
	defaultRequestTimeoutMs                 = 30000
	defaultDeliveryTimeoutMs                = 120000
	// aws msk 建议：重试延迟至少 200ms，避免重试风暴
	defaultRetryBackoffMs = 200
	// aws msk 建议：批次 64~128KB
	defaultBatchSize = 131072 // 128KB
	// aws msk 建议：缓冲区 ~64MB
	defaultBufferMemoryKByte = 65536 // ≈ 64MB
	defaultEnableIdempotence = true
)

// buildProducerConfig 构建生产者配置：所有可调参数均走"配置优先，未配置则取默认"。
// 共享连接级参数从 cfg（KafkaCfg）读取，调用前应已通过 cfg.Init() 填充默认值。
func (pm *ConfluentProducerManager) buildProducerConfig(cfg config.KafkaCfg, producerCfg config.KafkaProducer) (*kafka.ConfigMap, error) {
	h, ok := cfg.Hosts[producerCfg.Name]
	if !ok {
		return nil, fmt.Errorf("[Kafka] host配置不存在: %v", producerCfg.Name)
	}

	acks := defaultAcks
	if producerCfg.RequiredAcks != nil {
		switch *producerCfg.RequiredAcks {
		case 0:
			acks = "0"
		case 1:
			acks = "1"
		case -1:
			acks = "-1"
		}
	}

	retries := defaultRetries
	if producerCfg.MaxRetryTimes > 0 {
		retries = producerCfg.MaxRetryTimes
	}

	lingerMs := defaultLingerMs
	if producerCfg.LingerMs > 0 {
		lingerMs = producerCfg.LingerMs
	}

	compressionType := defaultCompressionType
	if producerCfg.CompressionType != "" {
		compressionType = producerCfg.CompressionType
	}

	maxInFlight := defaultMaxInFlightRequestsPerConnection
	if producerCfg.MaxInFlightRequestsPerConnection > 0 {
		maxInFlight = producerCfg.MaxInFlightRequestsPerConnection
	}

	requestTimeoutMs := defaultRequestTimeoutMs
	if producerCfg.RequestTimeoutMs > 0 {
		requestTimeoutMs = producerCfg.RequestTimeoutMs
	}

	deliveryTimeoutMs := defaultDeliveryTimeoutMs
	if producerCfg.DeliveryTimeoutMs > 0 {
		deliveryTimeoutMs = producerCfg.DeliveryTimeoutMs
	}

	retryBackoffMs := defaultRetryBackoffMs
	if producerCfg.RetryBackoffMs > 0 {
		retryBackoffMs = producerCfg.RetryBackoffMs
	}

	batchSize := defaultBatchSize
	if producerCfg.BatchSize > 0 {
		batchSize = producerCfg.BatchSize
	}

	bufferKBytes := defaultBufferMemoryKByte
	if producerCfg.BufferMemoryKByte > 0 {
		bufferKBytes = producerCfg.BufferMemoryKByte
	}

	enableIdempotence := defaultEnableIdempotence
	if producerCfg.EnableIdempotence != nil {
		enableIdempotence = *producerCfg.EnableIdempotence
	}

	// 共享连接级参数：cfg 已经过 Init()，直接信任非零值
	socketKeepalive := true
	if cfg.SocketKeepaliveEnable != nil {
		socketKeepalive = *cfg.SocketKeepaliveEnable
	}

	configMap := &kafka.ConfigMap{
		"bootstrap.servers":                     strings.Join(h.Host, ","),
		"client.id":                             fmt.Sprintf("confluent-producer-%s", producerCfg.Name),
		"acks":                                  acks,
		"retries":                               retries,
		"linger.ms":                             lingerMs,
		"compression.type":                      compressionType,
		"max.in.flight.requests.per.connection": maxInFlight,
		"request.timeout.ms":                    requestTimeoutMs,
		"delivery.timeout.ms":                   deliveryTimeoutMs,
		"retry.backoff.ms":                      retryBackoffMs,
		"batch.size":                            batchSize,
		"queue.buffering.max.kbytes":            bufferKBytes,
		"enable.idempotence":                    enableIdempotence,

		// 长连接 / 拓扑刷新相关（共享自 KafkaCfg）
		"socket.keepalive.enable": socketKeepalive,
		"connections.max.idle.ms": cfg.ConnectionsMaxIdleMs,
		"metadata.max.age.ms":     cfg.MetadataMaxAgeMs,
		"socket.timeout.ms":       cfg.SocketTimeoutMs,
	}

	// librdkafka 调试；见 https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md
	if debug := strings.TrimSpace(cfg.Debug); debug != "" {
		_ = configMap.SetKey("debug", debug)
		_ = configMap.SetKey("log.thread.name", true)
		log.Info("[Kafka] Confluent producer librdkafka debug enabled",
			zap.String("producer", producerCfg.Name),
			zap.String("debug", debug))
	}

	// C 侧队列统计（msg_cnt/msg_size），默认 60s，间隔配负数或 stats_mode=-1 时关闭回调
	statsIntervalMs := producerCfg.StatisticsIntervalMs
	if statsIntervalMs == 0 {
		statsIntervalMs = 60000
	}
	if statsIntervalMs > 0 && producerCfg.StatsMode != config.PoolStatsModeOff {
		configMap.SetKey("statistics.interval.ms", statsIntervalMs)
	}

	// SASL配置
	if len(h.Username) > 0 {
		if h.SecurityProtocol == "" || h.SaslMechanisms == "" {
			return nil, fmt.Errorf("security.protocol and sasl.mechanisms is required when sasl is enabled")
		}
		configMap.SetKey("security.protocol", h.SecurityProtocol)
		configMap.SetKey("sasl.mechanisms", h.SaslMechanisms)
		configMap.SetKey("sasl.username", h.Username)
		configMap.SetKey("sasl.password", h.Pwd)
	}

	return configMap, nil
}

// handleDeliveryReports 处理投递报告
// statsLogEnabled/statsReportEnabled 控制 librdkafka 统计的日志打印与指标上报（来自 stats_mode 配置）
func (pm *ConfluentProducerManager) handleDeliveryReports(producer *kafka.Producer, producerName string, statsLogEnabled, statsReportEnabled bool) {
	for {
		select {
		case <-pm.stopChan:
			return
		case evt := <-producer.Events():
			switch e := evt.(type) {
			case *kafka.Message:
				if e.TopicPartition.Error != nil {
					log.Error("[Kafka] Producer delivery failed",
						zap.String("producer", producerName),
						zap.String("topic", *e.TopicPartition.Topic),
						zap.Int32("partition", e.TopicPartition.Partition),
						zap.Int64("offset", int64(e.TopicPartition.Offset)),
						zap.Error(e.TopicPartition.Error))
				} else {
					log.Debug("[Kafka] Producer delivery succeeded",
						zap.String("producer", producerName),
						zap.String("topic", *e.TopicPartition.Topic),
						zap.Int32("partition", e.TopicPartition.Partition),
						zap.Int64("offset", int64(e.TopicPartition.Offset)))
				}
			case kafka.Error:
				log.Error("[Kafka] Producer error",
					zap.String("producer", producerName),
					zap.Error(e))
			case *kafka.Stats:
				// statistics.interval.ms 触发的 librdkafka 统计，观测 C 侧发送队列内存
				reportLibrdkafkaStats("producer", producerName, "", e.String(), statsLogEnabled, statsReportEnabled)
			}
		}
	}
}

// Push 异步推送消息
func (pm *ConfluentProducerManager) Push(ctx context.Context, key, topic string, obj interface{}) error {

	return pm.pushMessageWithProducer(ctx, pm.defaultProducer, key, topic, obj, false)
}

// PushSync 同步推送消息
func (pm *ConfluentProducerManager) PushSync(ctx context.Context, key, topic string, obj interface{}) error {
	return pm.pushMessageWithProducer(ctx, pm.defaultProducer, key, topic, obj, true)
}

// getProducer 获取生产者实例
func (pm *ConfluentProducerManager) getProducer(producerName string) (*kafka.Producer, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if producerName == "" {
		producerName = pm.defaultProducer
	}

	producer, exists := pm.producers[producerName]
	if !exists {
		return nil, fmt.Errorf("producer %s not found", producerName)
	}

	return producer, nil
}

// GetProducerCount 获取生产者数量
func (pm *ConfluentProducerManager) GetProducerCount() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.producers)
}

// GetDefaultProducer 获取默认生产者名称
func (pm *ConfluentProducerManager) GetDefaultProducer() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.defaultProducer
}

// Flush 刷新所有生产者
func (pm *ConfluentProducerManager) Flush(timeoutMs int) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for name, producer := range pm.producers {
		remaining := producer.Flush(timeoutMs)
		if remaining > 0 {
			log.Warn("[Kafka] Producer flush incomplete",
				zap.String("producer", name),
				zap.Int("remaining", remaining))
		}
	}
}

// Stop 停止所有生产者
func (pm *ConfluentProducerManager) Stop() {
	log.Info("[Kafka] Stopping Confluent producer manager")

	pm.mu.Lock()
	if pm.stopFlag {
		pm.mu.Unlock()
		return
	}
	pm.stopFlag = true
	close(pm.stopChan)
	pm.mu.Unlock()

	pm.mu.RLock()
	for name, producer := range pm.producers {
		log.Info("[Kafka] Closing producer", zap.String("name", name))
		producer.Close()
	}
	pm.mu.RUnlock()

	log.Info("[Kafka] Confluent producer manager stopped")
}

// IsRunning 检查是否正在运行
func (pm *ConfluentProducerManager) IsRunning() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return !pm.stopFlag
}

// GetProducerStats 获取生产者统计信息
func (pm *ConfluentProducerManager) GetProducerStats(producerName string) (map[string]interface{}, error) {
	producer, err := pm.getProducer(producerName)
	if err != nil {
		return nil, err
	}

	stats := producer.GetFatalError()
	if stats != nil {
		return map[string]interface{}{
			"error": stats.Error(),
		}, nil
	}

	return map[string]interface{}{
		"status": "healthy",
	}, nil
}

// PushWithProducer 使用指定生产者推送消息
func (pm *ConfluentProducerManager) PushWithProducer(ctx context.Context, producerName, key, topic string, obj interface{}) error {
	return pm.pushMessageWithProducer(ctx, producerName, key, topic, obj, false)
}

// PushSyncWithProducer 使用指定生产者同步推送消息
func (pm *ConfluentProducerManager) PushSyncWithProducer(ctx context.Context, producerName, key, topic string, obj interface{}) error {
	return pm.pushMessageWithProducer(ctx, producerName, key, topic, obj, true)
}

// pushMessageWithProducer 使用指定生产者推送消息的核心方法
func (pm *ConfluentProducerManager) pushMessageWithProducer(ctx context.Context, producerName, key, topic string, obj interface{}, sync bool) (err error) {
	skywalking.Start()

	start := time.Now()
	if len(producerName) == 0 {
		return errors.New("producer name is empty")
	}
	defer func() {
		duration := time.Since(start)
		log.InfoCtx(ctx, "[Kafka] PushWithProducer",
			zap.String("producer", producerName),
			zap.String("topic", topic),
			zap.Any("data", obj),
			zap.Duration("Duration", duration),
			zap.String(log.DUR_LOG_KEY, cst.GetHttpHandleLevel(duration)),
			zap.Bool("sync", sync))
		skywalking.StopSpan()
		// 推送统计消息
		select {
		case pm.pushtimeStats <- pushStats{Instance: producerName, Topic: topic, PushTime: duration}:
		default:
		}
	}()
	// 获取指定生产者
	producer, err := pm.getProducer(producerName)
	if err != nil {
		return err
	}
	message, err := getKafkaMsg(ctx, key, topic, obj)
	if err != nil {
		return err
	}
	if sync {
		// 同步发送
		deliveryChan := make(chan kafka.Event, 1)
		err = producer.Produce(message, deliveryChan)
		if err != nil {
			return fmt.Errorf("failed to produce message: %w", err)
		}

		// 等待投递结果
		select {
		case evt := <-deliveryChan:
			if msg, ok := evt.(*kafka.Message); ok {
				if msg.TopicPartition.Error != nil {
					return fmt.Errorf("delivery failed: %w", msg.TopicPartition.Error)
				}
			}
		case <-ctx.Done():
			return fmt.Errorf("delivery canceled: %w", ctx.Err())
		case <-time.After(30 * time.Second):
			return errors.New("delivery timeout")
		}
	} else {
		// 异步发送
		err = producer.Produce(message, nil)
		if err != nil {
			return fmt.Errorf("failed to produce message: %w", err)
		}
	}

	return nil
}

// 整合数据
func getKafkaMsg(ctx context.Context, key, topic string, obj interface{}) (message *kafka.Message, err error) {
	// 参数验证
	if obj == nil {
		return nil, errors.New("message is empty")
	}
	if len(topic) == 0 {
		return nil, errors.New("topic is empty")
	}
	// 参数校验
	if val, ok := obj.(validator.Validator); ok {
		if err := val.Validate(); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}
	}
	// 兼容各种类型数据（这里对 string 做特殊处理）
	var msg []byte
	switch obj.(type) {
	case string: //字符串类型
		msg = util.Str2Bytes(obj.(string))
	case []byte: //字节数组类型
		msg = obj.([]byte)
	case json2.RawMessage:
		msg = obj.(json2.RawMessage)
	default:
		// 序列化消息
		msg, err = json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal message: %w", err)
		}
	}

	// 环境处理 对本地运行的特殊处理，防止消费了开发环境数据
	if env.IsLocal() && os.Getenv("kafka_local_dev") == "true" {
		topic += "_dev"
	}
	// 创建消息
	message = &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:       []byte(key),
		Value:     msg,
		Timestamp: time.Now(),
		Headers:   getPushHeader(ctx),
	}
	return
}

func getPushHeader(ctx context.Context) []kafka.Header {
	hd := make([]kafka.Header, 0)
	// 获取 traceId；无追踪上下文时生成一个 uuid，保证生产的消息始终带 traceId（NS 原有兜底，合并后补回）
	traceId, _ := trace.GetTraceId(ctx)
	if traceId == "" {
		traceId = uuid.NewString()
	}
	hd = append(hd, kafka.Header{Key: model.MQ_HEADER_TRACE_ID, Value: []byte(traceId)})
	hd = append(hd, kafka.Header{Key: model.MQ_HEADER_NAME_SPACE, Value: util.Str2Bytes(metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE))})
	return hd
}

func msgMarshal(obj interface{}) (string, error) {
	if val, ok := obj.(string); ok {
		return val, nil
	}
	val, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	return Bytes2Str(val), nil
}
