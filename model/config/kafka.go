// Package config
// @Author: yinwei
// @File: kafka
// @Version: 1.0.0
// @Date: 2024/7/23 11:25

package config

import (
	"github.com/coderyw/gomicro-util/util"
	"go.uber.org/zap/zapcore"
)

// kafka配置
//
// 消费topic代码内配置。
type KafkaCfg struct {
	// 默认的推送者，如果配置了多个，可以使用此参数指定一个默认的
	// 如果没有配置默认推送者，则使用第一个配置的推送者
	// 如果配置了，但是配置找不到则会返回错误
	DefaultProducer string `json:"default_producer" yaml:"default_producer"`
	// hosts 配置
	Hosts map[string]KafkaHost `json:"hosts" yaml:"hosts"`
	// 生产者相关配置。支持多个集群和多个有差异的生产者配置
	Producer []KafkaProducer `json:"producer" yaml:"producer"`
	//消费者相关配置。支持多个集群
	Consumer []KafkaConsumer `json:"consumer" yaml:"consumer"`

	// === 共享 librdkafka 全局连接调优；同时作用于生产者与消费者 ===

	// SocketKeepaliveEnable 启用 TCP keepalive，云/NAT/LB 场景防止中间设备切断长连接。默认 true；
	// 指针以区分"未配置(nil, 走默认)"与"显式 false"
	SocketKeepaliveEnable *bool `json:"socket_keepalive_enable" yaml:"socket_keepalive_enable"`
	// ConnectionsMaxIdleMs 客户端空闲连接关闭时间（ms）。默认 540000（9 分钟），
	// 应略低于 broker 端 connections.max.idle.ms（典型 600000），由客户端主动关闭避免 broker 先断开
	ConnectionsMaxIdleMs int `json:"connections_max_idle_ms" yaml:"connections_max_idle_ms"`
	// MetadataMaxAgeMs 元数据强制刷新周期（ms）。默认 180000（3 分钟），
	// 缩短可加快感知 broker/partition leader 变更（librdkafka 默认 900000 偏长）
	MetadataMaxAgeMs int `json:"metadata_max_age_ms" yaml:"metadata_max_age_ms"`
	// SocketTimeoutMs 单个 socket 请求超时（ms）。默认 30000，与 request.timeout.ms 对齐
	SocketTimeoutMs int `json:"socket_timeout_ms" yaml:"socket_timeout_ms"`
	// Debug 启用 librdkafka 内部调试，逗号分隔，如: generic,broker,topic,msg,protocol；使用 all 会非常冗长
	Debug string `json:"debug" yaml:"debug"`
}

// Init 集中管理共享默认值；调用方应在使用 cfg 前调用一次（confluent_manager 入口处统一调用）
func (c *KafkaCfg) Init() *KafkaCfg {
	if c.SocketKeepaliveEnable == nil {
		v := true
		c.SocketKeepaliveEnable = &v
	}
	if c.ConnectionsMaxIdleMs <= 0 {
		c.ConnectionsMaxIdleMs = 540000
	}
	if c.MetadataMaxAgeMs <= 0 {
		c.MetadataMaxAgeMs = 180000
	}
	if c.SocketTimeoutMs <= 0 {
		c.SocketTimeoutMs = 30000
	}
	return c
}

func (s KafkaCfg) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("default_producer", s.DefaultProducer)
	if s.Hosts != nil {
		enc.AddObject("hosts", zapcore.ObjectMarshalerFunc(func(encoder zapcore.ObjectEncoder) error {
			for k, v := range s.Hosts {
				v := v
				encoder.AddObject(k, v)
			}
			return nil
		}))
	}
	enc.AddArray("producer", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Producer {
			encoder.AppendObject(v)
		}
		return nil
	}))
	enc.AddArray("consumer", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Consumer {
			encoder.AppendObject(v)
		}
		return nil
	}))
	if s.SocketKeepaliveEnable != nil {
		enc.AddBool("socket_keepalive_enable", *s.SocketKeepaliveEnable)
	}
	enc.AddInt("connections_max_idle_ms", s.ConnectionsMaxIdleMs)
	enc.AddInt("metadata_max_age_ms", s.MetadataMaxAgeMs)
	enc.AddInt("socket_timeout_ms", s.SocketTimeoutMs)
	enc.AddString("debug", s.Debug)
	return nil
}

type KafkaHost struct {
	Host             []string `json:"host" yaml:"host"`
	Username         string   `json:"username" yaml:"username"`
	Pwd              string   `json:"pwd,nomarshal" yaml:"pwd"`
	SaslMechanisms   string   `json:"sasl_mechanisms" yaml:"sasl_mechanisms"`
	SecurityProtocol string   `json:"security_protocol" yaml:"security_protocol"`
}

func (s KafkaHost) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddArray("host", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Host {
			encoder.AppendString(v)
		}
		return nil
	}))
	enc.AddString("username", s.Username)
	enc.AddString("pwd", util.Mask(s.Pwd, 2, 3))
	return nil
}

type KafkaProducer struct {
	// 推送者地址，依赖相关标识名称
	Name string `json:"name" yaml:"name"`
	// 推送最大重试次数 默认 3 次
	MaxRetryTimes int `json:"max_retry_times" yaml:"max_retry_times"`
	// 是一个用于 Kafka 生产者请求的类型，表示生产者在收到响应前需要等待的副本确认数量。其定义如下
	// 0= 不等待任何响应，仅依赖 TCP ACK。
	// 1= 等待本地副本写入成功。
	// -1= 等待所有同步副本写入成功，最小同步副本数由 broker 配置 min.insync.replicas 决定。
	// 默认 -1
	RequiredAcks *int16 `json:"required_acks" yaml:"required_acks"`

	// aws msk生产可用性实践
	// 设置 delivery.timeout.ms 以指定从发送消息到从代理收到确认之间的总时间上限。这应该反映消息有效期的业务要求。
	// 将时间限制设置得足够高，以允许足够的重试来完成失效转移操作。对于大多数使用场景，我们建议将值设置为 60 秒或更高。
	DeliveryTimeoutMs int `json:"delivery_timeout_ms" yaml:"delivery_timeout_ms"`

	// 将 request.timeout.ms 设置为在尝试重新发送之前单个请求应等待的最长时间。对于大多数使用场景，我们建议将值设置为 10 秒或更高。
	RequestTimeoutMs int `json:"request_timeout_ms" yaml:"request_timeout_ms"`

	// 设置 retry.backoff.ms 以配置重试之间的延迟，以避免重试风暴和可用性影响。对于大多数使用场景，我们建议最小值设置为 200 毫秒
	RetryBackoffMs int `json:"retry_backoff_ms" yaml:"retry_backoff_ms"`

	// 设置 batch.size 以控制发送到集群的批次大小。我们建议将其值增加到 64 KB 或 128 KB 单位是byte。
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// 待发送队列（含未收到 ack 的重试消息）总字节上限，单位 KB，对应 librdkafka queue.buffering.max.kbytes。
	// 该内存在 C 侧、pprof 不可见；librdkafka 默认 1048576(1GB)，这里默认收敛为 65536(64MB)
	// （实际默认值见 kafka.defaultBufferMemoryKByte）。
	// 打满后 Produce 返回 Queue full 错误（快速失败）。
	BufferMemoryKByte int `json:"queue_buffering_max_kbytes" yaml:"queue_buffering_max_kbytes"`

	// LingerMs 生产者在发送批次前等待聚合的毫秒数，调大可提升吞吐。默认 5
	LingerMs int `json:"linger_ms" yaml:"linger_ms"`

	// CompressionType 压缩算法：none/gzip/snappy/lz4/zstd。默认 lz4（CPU 与压缩率较均衡）
	CompressionType string `json:"compression_type" yaml:"compression_type"`

	// MaxInFlightRequestsPerConnection 每个连接上未确认请求最大数。默认 5。
	// 注意：开启 enable.idempotence 时 librdkafka 要求 <= 5
	MaxInFlightRequestsPerConnection int `json:"max_in_flight_requests_per_connection" yaml:"max_in_flight_requests_per_connection"`

	// EnableIdempotence 是否开启幂等生产者；默认 true（nil 视为开启），与 acks=all 搭配可在重试场景下减少重复消息。
	// 为 false 时显式关闭（兼容旧环境或排障对比）。
	EnableIdempotence *bool `json:"enable_idempotence" yaml:"enable_idempotence"`

	// librdkafka 统计上报间隔（ms），对应 statistics.interval.ms，用于观测 C 侧队列内存（msg_cnt/msg_size）。
	// 默认 60000；配置为负数时关闭上报。
	StatisticsIntervalMs int `json:"statistics_interval_ms" yaml:"statistics_interval_ms"`
	// librdkafka 统计输出模式：-1 不打印不上报（不开启 statistics 回调），0 默认（等同1），1 仅日志打印，2 仅指标上报，3 日志和上报都开启。默认 1
	StatsMode int `json:"stats_mode" yaml:"stats_mode"`
}

func (s KafkaProducer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", s.Name)
	enc.AddInt("max_retry_times", s.MaxRetryTimes)
	if s.RequiredAcks == nil {
		s.RequiredAcks = new(int16)
	}
	enc.AddInt16("required_acks", *s.RequiredAcks)
	enc.AddInt("delivery_timeout_ms", s.DeliveryTimeoutMs)
	enc.AddInt("request_timeout_ms", s.RequestTimeoutMs)
	enc.AddInt("retry_backoff_ms", s.RetryBackoffMs)
	enc.AddInt("batch_size", s.BatchSize)
	enc.AddInt("queue_buffering_max_kbytes", s.BufferMemoryKByte)
	enc.AddInt("linger_ms", s.LingerMs)
	enc.AddString("compression_type", s.CompressionType)
	enc.AddInt("max_in_flight_requests_per_connection", s.MaxInFlightRequestsPerConnection)
	if s.EnableIdempotence != nil {
		enc.AddBool("enable_idempotence", *s.EnableIdempotence)
	}
	enc.AddInt("statistics_interval_ms", s.StatisticsIntervalMs)
	enc.AddInt("stats_mode", s.StatsMode)
	return nil
}

type KafkaConsumer struct {
	Name   string  `json:"name" yaml:"name"`
	Groups []Group `json:"groups" yaml:"groups"`
}

func (s KafkaConsumer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("name", s.Name)
	enc.AddArray("groups", zapcore.ArrayMarshalerFunc(func(encoder zapcore.ArrayEncoder) error {
		for _, v := range s.Groups {
			encoder.AppendObject(v)
		}
		return nil
	}))
	return nil
}

// 消费组 配置
type Group struct {
	GroupId string `json:"group_id" yaml:"group_id"`
	// 批量消费最大数量, 默认100
	MaxMulNum int `json:"max_mul_num" yaml:"max_mul_num"`
	// 批量消费等待数据超时时间，如果超时了还没有等待到max_mul_num这些数据，则直接去消费 默认1000ms
	MulTimeoutMs int `json:"max_mul_timeout_ms" yaml:"max_mul_timeout_ms"`
	// 批量消费超时时间类型：
	// 1. 在没有拉取到数据时候，等待剩余数据到来的时间；
	// 2. 不管有没有拉取到数据，直接等待超时，超时就返回消费数据
	// 默认 2
	MulTimeoutType uint8 `json:"mul_timeout_type" yaml:"mul_timeout_type"`
	// 当消费组没有提交过偏移量时，-1 从最新开始消费，-2从最旧开始消费，默认 -1
	OffsetInitial int64 `json:"offset_initial" yaml:"offset_initial"`
	// 配置项用于指定消费者处理每条消息的最大时间。
	//如果消息在该时间内未能被处理（即未能从 Messages 通道中读取），则认为处理超时，分区将暂停拉取新消息，直到处理恢复正常。
	//此机制通过 ticker 实现，实际超时时间为 MaxProcessingTime * ChannelBufferSize。默认值为 300000ms。
	//请注意，此值必须在代理配置中由 `group.min.session.timeout.ms` 和 `group.max.session.timeout.ms` 指定的允许范围内。
	// 华为云 kafka 配置的是 6000 ～ 1800000
	// 单位 ms 注意 `MaxProcessingTimeMill` must be >= `SessionTimeoutMs`
	MaxProcessingTimeMill int `json:"max_processing_time_mill" yaml:"max_processing_time_mill"`
	//此超时时间用于在使用 Kafka 的组管理功能时检测客户端故障。客户端会定期发送心跳包以向代理表明其存活状态。
	//如果在会话超时时间到期之前代理未收到任何心跳包，则会将此客户端从组中移除并启动重新平衡。
	//请注意，此值必须在代理配置中由 `group.min.session.timeout.ms` 和 `group.max.session.timeout.ms` 指定的允许范围内。
	//另请注意，当 `group.protocol` 设置为“consumer”时，不支持此客户端配置group.protocol。
	//在这种情况下，会话超时时间由代理配置控制group.consumer.session.timeout.ms。
	// default 45000 ms
	// 华为云 kafka 配置的是 6000 ～ 1800000
	SessionTimeoutMs int `json:"session_timeout_ms" yaml:"session_timeout_ms"`
	// 偏移量提交策略 0 = 消费后自动提交（提交消费后的数据），1= 拉取后自动提交（提交拉取到的数据，可能还没来得及消费），2= 手动提交（消费后手动提交每条消息）
	// 默认 0
	CommitType uint8 `json:"commit_type" yaml:"commit_type"`
	// 自动提交间隔时间（ms）默认 3000ms
	AutoCommitIntervalMs int `json:"auto_commit_interval_ms" yaml:"auto_commit_interval_ms"`
	// aws msk建议设置10秒
	HeartbeatIntervalMs int `json:"heartbeat_interval_ms" yaml:"heartbeat_interval_ms"`
	// 每个 topic 每个 partition 本地缓冲区大小。默认 1024.
	// 缓冲区越大，同 topic 每个分区消费互相影响会更小，具体请参考线程模型。
	// 缓冲区越大，如果消费速度跟不上可能会导致 rebalance
	TopicPartitionChannelSize int `json:"topic_partition_channel_size" yaml:"topic_partition_channel_size"`
	// 单分区 C 侧预取条数水位，对应 librdkafka queued.min.messages（按 topic+partition 生效）。
	// 本地队列消息数低于该值时 fetcher 持续向 broker 补齐；该内存在 C 侧、pprof 不可见。
	// librdkafka 默认 100000（约 50MB/分区），这里默认收敛为 2000（批量消费单批 max_mul_num 默认 100，2000 已充分流水线化）。
	QueuedMinMessages int `json:"queued_min_messages" yaml:"queued_min_messages"`
	// 单分区 C 侧预取字节上限，单位 KB，对应 librdkafka queued.max.messages.kbytes，与 queued_min_messages 先到者生效。
	// librdkafka 默认 65536(64MB)，这里默认收敛为 8192(8MB)，作为大消息场景的单分区内存硬上限。
	QueuedMaxMessagesKbytes int `json:"queued_max_messages_kbytes" yaml:"queued_max_messages_kbytes"`
	// 分区分配策略，对应 librdkafka partition.assignment.strategy（如 "cooperative-sticky" 或 "cooperative-sticky,range"）。
	// 默认空 = 沿用 librdkafka 默认（range,roundrobin 的 eager 协议）。
	// 注意：切到 cooperative-sticky 要求全组成员协议兼容，滚动发布期间需按两步升级
	// （先配 "cooperative-sticky,range" 全量发布一轮，再改 "cooperative-sticky" 发布第二轮）。
	PartitionAssignmentStrategy string `json:"partition_assignment_strategy" yaml:"partition_assignment_strategy"`
	// librdkafka 统计上报间隔（ms），对应 statistics.interval.ms，用于观测 C 侧预取队列内存（fetchq_cnt/fetchq_size）。
	// 默认 60000；配置为负数时关闭上报。
	StatisticsIntervalMs int `json:"statistics_interval_ms" yaml:"statistics_interval_ms"`
	// librdkafka 统计输出模式：-1 不打印不上报（不开启 statistics 回调），0 默认（等同1），1 仅日志打印，2 仅指标上报，3 日志和上报都开启。默认 1
	StatsMode int `json:"stats_mode" yaml:"stats_mode"`
	// 每分区 Go 侧消息通道容量（subscribe -> 分区 handler 的缓冲），满则反压暂停拉取。默认 256。
	MsgChanCap int `json:"msg_chan_cap" yaml:"msg_chan_cap"`
}

func (s Group) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("group_id", s.GroupId)
	enc.AddInt("max_mul_num", s.MaxMulNum)
	enc.AddInt("max_mul_timeout_ms", s.MulTimeoutMs)
	enc.AddUint8("mul_timeout_type", s.MulTimeoutType)
	enc.AddInt64("offset_initial", s.OffsetInitial)
	enc.AddInt("max_processing_time_mill", s.MaxProcessingTimeMill)
	enc.AddInt("session_timeout_ms", s.SessionTimeoutMs)
	enc.AddUint8("commit_type", s.CommitType)
	enc.AddInt("auto_commit_interval_ms", s.AutoCommitIntervalMs)
	enc.AddInt("queued_min_messages", s.QueuedMinMessages)
	enc.AddInt("queued_max_messages_kbytes", s.QueuedMaxMessagesKbytes)
	enc.AddString("partition_assignment_strategy", s.PartitionAssignmentStrategy)
	enc.AddInt("statistics_interval_ms", s.StatisticsIntervalMs)
	enc.AddInt("stats_mode", s.StatsMode)
	enc.AddInt("msg_chan_cap", s.MsgChanCap)
	return nil
}

func (g *Group) Init() *Group {
	if g.MaxMulNum <= 0 {
		g.MaxMulNum = 100
	}
	if g.MulTimeoutMs <= 0 {
		g.MulTimeoutMs = 100
	}
	if g.MulTimeoutType < 0 || g.MulTimeoutType > 2 {
		g.MulTimeoutType = 2
	}
	if g.OffsetInitial < -2 || g.OffsetInitial > -1 {
		g.OffsetInitial = -1
	}
	if g.MaxProcessingTimeMill <= 0 {
		g.MaxProcessingTimeMill = 30000
	}
	if g.SessionTimeoutMs <= 0 {
		g.SessionTimeoutMs = 30000
	}
	if g.CommitType < 0 || g.CommitType > 2 {
		g.CommitType = 0
	}
	if g.AutoCommitIntervalMs <= 0 {
		g.AutoCommitIntervalMs = 3000
	}
	if g.HeartbeatIntervalMs <= 0 {
		g.HeartbeatIntervalMs = 10000
	}
	if g.TopicPartitionChannelSize <= 0 {
		g.TopicPartitionChannelSize = 1024
	}
	if g.QueuedMinMessages <= 0 {
		g.QueuedMinMessages = 2000
	}
	if g.QueuedMaxMessagesKbytes <= 0 {
		g.QueuedMaxMessagesKbytes = 8192
	}
	if g.StatisticsIntervalMs == 0 {
		g.StatisticsIntervalMs = 60000
	} else if g.StatisticsIntervalMs < 0 {
		g.StatisticsIntervalMs = 0 // 负数表示关闭上报
	}
	if g.MsgChanCap <= 0 {
		g.MsgChanCap = 256
	}
	return g
}

const (
	// 消费后自动提交
	KAFKA_COMMITER_TYPE_AUTO_CONSUME = uint8(iota)
	// 拉取后自动提交
	KAFKA_COMMITER_TYPE_AUTO = uint8(iota)
	// 消费后手动提交
	KAFKA_COMMITER_TYPE_BY_HAND = uint8(iota)
)
