package kafka

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/namespace"
	"github.com/coderyw/gomicro-util/model/cron"
	tenant2 "github.com/coderyw/gomicro-util/model/tenant"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/coderyw/gomicro-util/library/log"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
)

// ConfluentConsumerManager 使用Confluent-Kafka-Go的消费者管理器
type ConfluentConsumerManager struct {
	consumers map[string]*ConsumerGroup
	stopChan  chan struct{}
	stopFlag  atomic.Bool
	wg        *sync.WaitGroup
	mu        sync.RWMutex
	// kafkaCfg 持有共享 librdkafka 全局参数（已 Init），供 buildConsumerConfig 下发给消费者
	kafkaCfg config2.KafkaCfg
}

// NewConfluentConsumerManager 创建Confluent消费者管理器
func NewConfluentConsumerManager() *ConfluentConsumerManager {
	kafkaManager := &ConfluentConsumerManager{
		consumers: make(map[string]*ConsumerGroup),
		stopChan:  make(chan struct{}),
		wg:        &sync.WaitGroup{},
	}
	return kafkaManager
}

// topicHandler 主题处理器
type confluentTopicHandler struct {
	topics      []string
	HandlerFunc map[string]model.Handler
	confluentGroupConfig
}

type topicGroupHandler struct {
	info kafka.TopicPartition
	// groupId|topic|partition
	key         string
	handlerFunc model.Handler
	hostCfg     config2.KafkaHost
	group       config2.Group
}

// groupConfig 消费组配置
type confluentGroupConfig struct {
	Name string
	config2.KafkaHost
	config2.Group
}

// defaultGroupConfig 默认消费组配置
func defaultConfluentGroupConfig(hostCfg config2.KafkaHost) confluentGroupConfig {
	return confluentGroupConfig{
		KafkaHost: hostCfg,
		Group: config2.Group{
			MaxMulNum:               100,
			MulTimeoutMs:            1000,
			MulTimeoutType:          2,
			OffsetInitial:           -1,
			MaxProcessingTimeMill:   30000,
			CommitType:              0,
			AutoCommitIntervalMs:    3000,
			QueuedMinMessages:       2000,
			QueuedMaxMessagesKbytes: 8192,
			StatisticsIntervalMs:    60000,
			MsgChanCap:              256,
		},
	}
}

// InitKafkaConsumer 初始化Kafka消费者
func (cm *ConfluentConsumerManager) InitKafkaConsumer(serverName string, cfg config2.KafkaCfg, handlers []model.Handler) ([]cron.CronJob, error) {
	if len(handlers) == 0 || len(cfg.Consumer) == 0 {
		return nil, nil
	}

	log.Info("[Kafka] begin to init consumer, it may take a few seconds...")

	// 持有 kafka 共享配置，供后续 buildConsumerConfig 下发 socket/metadata/debug 等共享参数
	cm.kafkaCfg = cfg

	// 构建消费组配置
	groupCfg, err := cm.buildGroupConfigs(cfg)
	if err != nil {
		return nil, err
	}

	// 构建主题处理器
	err = cm.buildConsumers(serverName, handlers, groupCfg)
	if err != nil {
		return nil, err
	}

	// 创建消费者
	err = cm.startConsumers(serverName)
	if err != nil {
		return nil, err
	}

	// 监听租户变更，动态调整各消费组的订阅列表
	cm.watchTenantChange()
	return cm.cronJob(), nil
}

// buildGroupConfigs 构建消费组配置
func (cm *ConfluentConsumerManager) buildGroupConfigs(cfg config2.KafkaCfg) (map[string]confluentGroupConfig, error) {
	groupCfg := make(map[string]confluentGroupConfig)

	// 遍历配置，初始化所有group的对应配置
	for _, v := range cfg.Consumer {
		// 获取 host
		h, ok := cfg.Hosts[v.Name]
		if !ok {
			return nil, fmt.Errorf("host %s not found", v.Name)
		}
		for _, k := range v.Groups {
			k.Init() // 初始化默认值
			cgg := confluentGroupConfig{
				Name: v.Name,
				KafkaHost: config2.KafkaHost{
					Host:             h.Host,
					Username:         h.Username,
					Pwd:              h.Pwd,
					SaslMechanisms:   h.SaslMechanisms,
					SecurityProtocol: h.SecurityProtocol,
				},
				Group: k,
			}
			// 每个实例区分一下
			groupCfg[fmt.Sprintf("%v|%v", v.Name, k.GroupId)] = cgg
		}
	}
	return groupCfg, nil
}

func (cm *ConfluentConsumerManager) cronJob() []cron.CronJob {
	var arr []cron.CronJob
	for _, v := range cm.consumers {
		cr := v.GetCronJob()
		if cr != nil {
			arr = append(arr, cr...)
		}
	}
	return arr
}

// buildConsumers 构造消费者
func (cm *ConfluentConsumerManager) buildConsumers(serverName string, handlers []model.Handler, groupCfg map[string]confluentGroupConfig) error {
	// 过滤同一个 group 下是否存在 topic 重复问题
	topicFilter := make(map[string]map[string]struct{})
	cm.consumers = make(map[string]*ConsumerGroup)
	for i, handler := range handlers {
		// 校验 handler 参数
		if handler.HandlerFuncs == nil && handler.HandlerFunc == nil {
			return fmt.Errorf("handler %s is invalid", handler.InstanceName)
		}
		if handler.GroupId == "" {
			handler.GroupId = serverName
		}
		handlers[i] = handler
		groupKey := fmt.Sprintf("%v|%v", handler.InstanceName, handler.GroupId)
		// 检查topic是否已存在
		filter, ok := topicFilter[groupKey]
		if !ok {
			filter = make(map[string]struct{})
			topicFilter[groupKey] = filter
		}
		if _, exists := filter[handler.Topics]; !exists {
			filter[handler.Topics] = struct{}{}
		} else {
			continue //过滤掉相同 group·下重复的 topic 消费
		}

		// key
		key := fmt.Sprintf("%v|%v|%v", handler.InstanceName, handler.GroupId, handler.Topics)

		cs, ok := cm.consumers[key]
		if !ok {
			cs = &ConsumerGroup{
				mgrWg:         cm.wg,
				alreadyCommit: make(map[int32]int64),
				key:           key,
				resubChan:     make(chan []string, 1),
			}
			cm.consumers[key] = cs
		}

		cm.consumers[key].handler = handler

		// 获取配置
		if cfg, ok := groupCfg[groupKey]; ok {
			cm.consumers[key].confluentGroupConfig = cfg
		} else {
			return fmt.Errorf("group %s not found", key)
		}

	}

	return nil
}

// startConsumers 启动消费者
func (cm *ConfluentConsumerManager) startConsumers(serverName string) error {
	// 遍历 所有 group
	for insGroupId, th := range cm.consumers {
		// 定制配置
		configMap, err := cm.buildConsumerConfig(serverName, th.confluentGroupConfig)
		if err != nil {
			return fmt.Errorf("invalid config for group %s: %w", insGroupId, err)
		}
		// 根据配置初始化一个 consumer 客户端
		consumer, err := kafka.NewConsumer(configMap)
		if err != nil {
			return fmt.Errorf("failed to create consumer for group %s: %w", insGroupId, err)
		}

		// 订阅主题，注册 rebalance 回调：分区被调走时清理对应的 channel 与 handler goroutine
		topics := cm.makeTopics(th.handler.Topics)
		th.lastTopics = topics
		err = consumer.SubscribeTopics(topics, th.rebalanceCb)
		if err != nil {
			consumer.Close() //
			return fmt.Errorf("failed to subscribe topics for group %s: %w", insGroupId, err)
		}
		log.Info("[Kafka] init consumer", zap.Any("key", th.key))
		th.consumer = consumer

		// 等待计数+1
		// 开启协程处理
		cm.wg.Add(1)
		go th.subscribe(cm.stopChan)

	}

	return nil
}

// makeTopics 按当前租户列表生成精确订阅的 topic 列表
//
// 此前多租户模式用正则 ^.*topic$ 后缀匹配，会误订同集群上其他命名空间租户的 topic；
// 现改为只订阅本命名空间已加载租户的精确 topic，租户变化时经 watchTenantChange 重订阅。
func (cm *ConfluentConsumerManager) makeTopics(topic string) []string {
	return makeTopicsForTenants(topic, namespace.GetAllTenantsInfo())
}

// makeTopicsForTenants 按租户列表生成精确 topic 列表（去重、升序，便于变更比对）
//
// 每个租户对应 FmtByType 命名（tenant<租户名>_<topic>）；租户名为空（兜底租户）
// 或兜底租户模式下即原始 topic。租户列表为空（未开启租户监听 / etcd 注册中心）时
// 兜底返回原始 topic，保持单租户兼容。
func makeTopicsForTenants(topic string, ts []tenant2.Tenant) []string {
	if len(ts) == 0 {
		return []string{topic}
	}
	seen := make(map[string]struct{}, len(ts))
	arr := make([]string, 0, len(ts))
	for _, t := range ts {
		name := namespace.FmtByType(namespace.NAMESPACE_TYPE_KAFKA, t.TenantName, topic)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		arr = append(arr, name)
	}
	sort.Strings(arr)
	return arr
}

// watchTenantChange 注册租户变更监听：租户上线/下线时对各消费组重新计算订阅列表，
// 有变化的经 resubChan 投递给对应 Poll goroutine 执行重订阅（新增租户分区 assign 后
// 按需建 channel；下线租户分区经 rebalance revoke 走既有的 rebalanceCb 清理路径）。
// 注册后立即对账一次，补上 startConsumers 与注册之间可能漏掉的变更窗口。
// manager 随进程生命周期存在：Stop 后监听器经 stopFlag 变为 no-op，但不注销。
//
// ⚠️ 两条隐式契约，改动前务必先读：
//
//  1. reconcile 必须忽略回调入参、重新读取全局租户列表。nacos 回调可能并发触发，
//     parseTenantCfg 在锁外通知导致通知顺序可能与写入顺序不一致；正因为 reconcile
//     每次都在 cm.mu 内重读全局（而非信任入参），任何交错最终都收敛到最新状态。
//     把 OnTenantsChange 的入参"顺手"用起来会立刻引入订阅漂移。
//  2. pushResub 必须保持永不阻塞（依赖 resubChan 的 cap≥1 + 丢弃旧目标）。
//     本函数持 cm.mu 写锁调用它，而 Stop() 不持该锁——正是靠"发送不阻塞"才没有锁序问题。
func (cm *ConfluentConsumerManager) watchTenantChange() {
	reconcile := func() {
		// 写锁：注册后的立即对账（init goroutine）与 nacos 变更回调可能并发执行，
		// 必须互斥，否则 lastTopics 读写与 pushResub 投递存在竞争
		cm.mu.Lock()
		defer cm.mu.Unlock()
		if cm.stopFlag.Load() {
			return
		}
		for _, th := range cm.consumers {
			topics := cm.makeTopics(th.handler.Topics)
			if slices.Equal(topics, th.lastTopics) {
				continue
			}
			th.lastTopics = topics
			th.pushResub(topics)
		}
	}
	namespace.OnTenantsChange(func(_ []tenant2.Tenant) {
		reconcile()
	})
	reconcile()
}

// buildConsumerConfig 构建消费者配置
func (cm *ConfluentConsumerManager) buildConsumerConfig(serverName string, cfg confluentGroupConfig) (*kafka.ConfigMap, error) {
	configMap := &kafka.ConfigMap{
		"bootstrap.servers":        strings.Join(cfg.Host, ","), // 使用第一个服务器
		"group.id":                 cfg.GroupId,
		"client.id":                serverName,
		"auto.offset.reset":        "latest",
		"enable.auto.commit":       false,
		"auto.commit.interval.ms":  3000, // 与 Group.Init() 默认一致
		"enable.auto.offset.store": false,
		"max.poll.interval.ms":     300000,
		"fetch.min.bytes":          1,
		"fetch.wait.max.ms":        500,
		"isolation.level":          "read_committed",

		// =========== aws的建议 ===========
		// 设置 heartbeat.interval.ms 以微调组协调器如何将消费者视为正常。对于大多数使用场景，我们建议将值设置为 10 秒。
		"heartbeat.interval.ms": 10000,
	}

	// SASL配置
	if len(cfg.Username) > 0 {
		if cfg.SecurityProtocol == "" || cfg.SaslMechanisms == "" {
			return nil, fmt.Errorf("security.protocol and sasl.mechanisms is required when sasl is enabled")
		}
		configMap.SetKey("security.protocol", cfg.SecurityProtocol)
		configMap.SetKey("sasl.mechanisms", cfg.SaslMechanisms)
		configMap.SetKey("sasl.username", cfg.Username)
		configMap.SetKey("sasl.password", cfg.Pwd)
	}

	if cfg.HeartbeatIntervalMs > 0 {
		configMap.SetKey("heartbeat.interval.ms", cfg.HeartbeatIntervalMs)
	}

	// 偏移量配置
	switch cfg.OffsetInitial {
	case -1:
		configMap.SetKey("auto.offset.reset", "latest")
	case -2:
		configMap.SetKey("auto.offset.reset", "earliest")
	default:
		configMap.SetKey("auto.offset.reset", "latest")
	}
	// `max.poll.interval.ms`must be >= `session.timeout.ms`

	// 处理时间配置
	if cfg.MaxProcessingTimeMill > 0 {
		configMap.SetKey("max.poll.interval.ms", cfg.MaxProcessingTimeMill)
	}
	// 默认 session.timout.ms配置是 45000
	if cfg.SessionTimeoutMs > 0 { //当小于默认值时，配置session.timout.ms为max.poll.interval.ms
		configMap.SetKey("session.timeout.ms", cfg.SessionTimeoutMs)
	}

	// 拉取时自动提交
	if cfg.CommitType == config2.KAFKA_COMMITER_TYPE_AUTO {
		configMap.SetKey("enable.auto.commit", true)
	}
	// 自动提交时间间隔
	if cfg.AutoCommitIntervalMs > 0 {
		configMap.SetKey("auto.commit.interval.ms", cfg.AutoCommitIntervalMs)
	}

	// === 共享 librdkafka 全局连接参数（来自 KafkaCfg，已 Init） ===
	socketKeepalive := true
	if cm.kafkaCfg.SocketKeepaliveEnable != nil {
		socketKeepalive = *cm.kafkaCfg.SocketKeepaliveEnable
	}
	configMap.SetKey("socket.keepalive.enable", socketKeepalive)
	configMap.SetKey("connections.max.idle.ms", cm.kafkaCfg.ConnectionsMaxIdleMs)
	configMap.SetKey("metadata.max.age.ms", cm.kafkaCfg.MetadataMaxAgeMs)
	configMap.SetKey("socket.timeout.ms", cm.kafkaCfg.SocketTimeoutMs)
	if debug := strings.TrimSpace(cm.kafkaCfg.Debug); debug != "" {
		configMap.SetKey("debug", debug)
		configMap.SetKey("log.thread.name", true)
		log.Info("[Kafka] Confluent consumer librdkafka debug enabled",
			zap.String("group", cfg.GroupId),
			zap.String("debug", debug))
	}

	// C 侧预取队列封顶（按 topic+partition 生效，先到者为准）。
	// librdkafka 默认 100000 条 / 64MB 每分区，该内存 pprof 不可见，是消费积压时 RSS 升高的主因
	if cfg.QueuedMinMessages > 0 {
		configMap.SetKey("queued.min.messages", cfg.QueuedMinMessages)
	}
	if cfg.QueuedMaxMessagesKbytes > 0 {
		configMap.SetKey("queued.max.messages.kbytes", cfg.QueuedMaxMessagesKbytes)
	}
	// 分区分配策略。切 cooperative-sticky 需全组成员协议兼容，按配置显式开启
	if cfg.PartitionAssignmentStrategy != "" {
		configMap.SetKey("partition.assignment.strategy", cfg.PartitionAssignmentStrategy)
	}
	// C 侧队列统计（fetchq_cnt/fetchq_size），负数配置在 Init 中已归零表示关闭；stats_mode=-1 同样关闭回调
	if cfg.StatisticsIntervalMs > 0 && cfg.StatsMode != config2.PoolStatsModeOff {
		configMap.SetKey("statistics.interval.ms", cfg.StatisticsIntervalMs)
	}

	return configMap, nil
}

// cleanupChannels 清理通道
func (cm *ConfluentConsumerManager) cleanupChannels(msgChans map[string]chan *kafka.Message) {
	for _, ch := range msgChans {
		close(ch)
	}
}

// runBeforeStop 运行停止前回调
func (cm *ConfluentConsumerManager) runBeforeStop(beforeStop func()) {
	if beforeStop != nil {
		beforeStop()
	}
}

// Stop 停止消费者
func (cm *ConfluentConsumerManager) Stop() {
	log.Info("[Kafka] stop consumer begin")

	close(cm.stopChan)
	cm.stopFlag.Store(true)
	cm.wg.Wait()
	log.Info("[Kafka] stop consumer end")
}

// GetConsumerCount 获取消费者数量
func (cm *ConfluentConsumerManager) GetConsumerCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.consumers)
}

// IsRunning 检查是否正在运行
func (cm *ConfluentConsumerManager) IsRunning() bool {
	return !cm.stopFlag.Load()
}

func Bytes2Str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
