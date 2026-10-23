package kafka

import (
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/pmicro/metrics"
	"github.com/coderyw/gomicro-util/util/json"
	"go.uber.org/zap"
)

// librdkafkaStats 只解析内存观测相关字段。
// 完整字段见 librdkafka STATISTICS.md（statistics.interval.ms 触发的统计 JSON）。
type librdkafkaStats struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// producer 待发送队列当前消息条数/字节数（对应 queue.buffering.max.* 封顶的那块内存）
	MsgCnt  int64 `json:"msg_cnt"`
	MsgSize int64 `json:"msg_size"`
	// consumer 每分区 C 侧预取队列（对应 queued.min.messages / queued.max.messages.kbytes 封顶的那块内存）
	Topics map[string]struct {
		Partitions map[string]struct {
			FetchqCnt  int64 `json:"fetchq_cnt"`
			FetchqSize int64 `json:"fetchq_size"`
		} `json:"partitions"`
	} `json:"topics"`
}

// reportLibrdkafkaStats 汇总输出 librdkafka C 侧队列的内存观测数据。
// 该部分内存在 Go heap profile 中不可见，此统计是其唯一观测手段。
// logEnabled 时打印日志；reportEnabled 时推送到 pmicro/metrics（metrics 未开启则由其内部丢弃）。
func reportLibrdkafkaStats(role, instance, group, statsJson string, logEnabled, reportEnabled bool) {
	if !logEnabled && !reportEnabled {
		return
	}
	var st librdkafkaStats
	if err := json.Unmarshal([]byte(statsJson), &st); err != nil {
		log.Warn("[Kafka] parse librdkafka stats failed",
			zap.String("role", role),
			zap.String("instance", instance),
			zap.Error(err))
		return
	}

	var fetchqCnt, fetchqSize int64
	for _, t := range st.Topics {
		for _, p := range t.Partitions {
			fetchqCnt += p.FetchqCnt
			fetchqSize += p.FetchqSize
		}
	}

	if logEnabled {
		log.Info("[Kafka] librdkafka stats",
			zap.String("role", role),
			zap.String("instance", instance),
			zap.String("group", group),
			zap.String("client", st.Name),
			zap.Int64("producer_queue_msg_cnt", st.MsgCnt),
			zap.Int64("producer_queue_msg_size", st.MsgSize),
			zap.Int64("consumer_fetchq_cnt", fetchqCnt),
			zap.Int64("consumer_fetchq_size", fetchqSize))
	}
	if reportEnabled {
		// 按角色只上报对应的队列指标，避免另一侧恒为 0 的无效序列
		if role == "producer" {
			tags := metrics.Tags{"instance": instance}
			metrics.SetGauge("kafka_producer_queue_msg_cnt", tags, float64(st.MsgCnt))
			metrics.SetGauge("kafka_producer_queue_msg_size", tags, float64(st.MsgSize))
		} else {
			tags := metrics.Tags{"instance": instance, "group": group}
			metrics.SetGauge("kafka_consumer_fetchq_cnt", tags, float64(fetchqCnt))
			metrics.SetGauge("kafka_consumer_fetchq_size", tags, float64(fetchqSize))
		}
	}
}
