package kafka

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mq/model"
	config2 "github.com/coderyw/gomicro-util/model/config"
	"github.com/coderyw/gomicro-util/model/cron"
	"go.uber.org/zap"
)

type ConsumerGroup struct {
	key string
	// 消费客户端
	consumer *kafka.Consumer
	// handler
	handler model.Handler
	confluentGroupConfig
	// 等待
	mgrWg *sync.WaitGroup

	wg sync.WaitGroup

	msgChans map[string]map[int32]chan *kafka.Message

	// 待提交
	waitCommit sync.Map
	// 已经提交的偏移量
	alreadyCommit map[int32]int64

	stopFlag atomic.Bool

	// resubChan 租户变更触发的重订阅目标 topic 列表；容量 1，只保留最新目标，
	// 由 subscribe 的 Poll goroutine 消费并执行 SubscribeTopics
	resubChan chan []string
	// lastTopics 当前订阅的 topic 列表（升序），仅由 watchTenantChange 的对账逻辑
	// 在 manager 写锁内读写
	lastTopics []string
}

// pushResub 投递最新的重订阅目标；channel 已满时丢弃旧目标、循环重试直到投递成功，
// 对并发发送安全（对账逻辑与订阅失败重投可能同时调用）
//
// ⚠️ 永不阻塞是硬约束：调用方 watchTenantChange 的 reconcile 持 manager 写锁调用本函数，
// 一旦阻塞就会锁死整个 manager。这依赖 resubChan 的 cap≥1 + 丢弃旧目标语义，
// 改成无缓冲 channel 会立即变成死锁。
func (cm *ConsumerGroup) pushResub(topics []string) {
	// 防御：nil channel 上发送与接收都恒走 default，下面的循环将永不退出，
	// 且是持 manager 写锁空转。构造 ConsumerGroup 时漏初始化 resubChan 即会触发，
	// 编译与测试都照不到，这里直接丢弃投递
	if cm.resubChan == nil {
		log.Error("[Kafka] resubChan 未初始化，丢弃重订阅投递", zap.String("key", cm.key), zap.Strings("topics", topics))
		return
	}
	for {
		select {
		case cm.resubChan <- topics:
			return
		default:
			// 满则丢弃一个旧目标后重试
			select {
			case <-cm.resubChan:
			default:
			}
		}
	}
}

func (c *ConsumerGroup) GetCronJob() []cron.CronJob {
	if c.CommitType == config2.KAFKA_COMMITER_TYPE_AUTO_CONSUME {
		it := c.AutoCommitIntervalMs
		if it <= 1000 { //小于 1s 的基于 1s
			it = 1000
		}

		// 消费后提交，才需要定时去提交
		return []cron.CronJob{
			cron.CronJob{
				Name:            fmt.Sprintf("%v %v %v 偏移量提交", c.Name, c.GroupId, c.handler.Topics),
				DoJob:           c.doCommit,
				CronSpc:         fmt.Sprintf("@every %vs", it/1000),
				DoBeforeDone:    false,
				DelayBeforeDone: false,
			},
			//{
			//	Name:            "TopicPartitionWorker check",
			//	DoJob:           c.partitionCheck,
			//	CronSpc:         fmt.Sprintf("@every %vs", 5*60),
			//	DoBeforeDone:    false,
			//	DelayBeforeDone: false,
			//},
		}
	}
	return nil
}

func (cg *ConsumerGroup) doCommit() {
	var arr []kafka.TopicPartition
	cg.waitCommit.Range(func(key, value any) bool {
		// 分区
		//pt := key.(int32)
		msg := value.(kafka.TopicPartition)
		lastOffset := cg.alreadyCommit[msg.Partition]
		if lastOffset != int64(msg.Offset) { //只有与上一次提交的偏移量不一致时候，才去提交
			cg.alreadyCommit[msg.Partition] = int64(msg.Offset)
			// 这里为什么+1.因为 confluent 这个消息的 offset 指的不是当前消息的偏移量。而是当前消费到的位置，所以需要+1
			msg.Offset++
			arr = append(arr, msg)
		}
		return true
	})
	if len(arr) == 0 {
		return
	}
	_, err := cg.consumer.CommitOffsets(arr)
	if err != nil {
		log.Error("提交 kafka 偏移量错误", zap.String("name", cg.Name), zap.String("groupId", cg.GroupId), zap.Any("pt", arr), zap.Error(err))
	} else {
		log.Debug("提交偏移量成功", zap.String("name", cg.Name), zap.String("groupId", cg.GroupId), zap.Any("pt", arr))
	}
}

func (cm *ConsumerGroup) close() {

	cm.stopFlag.Store(true)

	// 关闭所有分区消费通道。
	// 必须同时从 map 删除：下面的 cm.consumer.Close() 内部会循环 Poll 直到退组完成，
	// 而 Poll 会触发 rebalanceCb 收到 RevokedPartitions，若 map 里还留着已关闭的 channel，
	// rebalanceCb 会对同一个 channel 二次 close 而 panic（该 goroutine 无 recover）
	for topic, v := range cm.msgChans {
		for pt, k := range v {
			log.Info("[Kafka] close consumer", zap.String("topic", topic), zap.Any("partition", pt))
			close(k)
			delete(v, pt)
		}
	}
	cm.wg.Wait() // 等待所有消费完成
	// 提交消费者一次
	cm.doCommit()
	cm.consumer.Close()
	cm.mgrWg.Done()
}

// subscribe 订阅消息
func (cm *ConsumerGroup) subscribe(stopChan <-chan struct{}) {
	defer func() {
		cm.close()
	}()

	cm.msgChans = make(map[string]map[int32]chan *kafka.Message)
	log.Info("[Kafka] Go start consumer read", zap.String("key", cm.key))
	for {
		select {
		case <-stopChan:
			log.Info("[Kafka] stop consumer read", zap.String("Event", "stop"))
			return
		case topics := <-cm.resubChan:
			// 租户变更触发的重订阅：在 Poll goroutine 内执行，与 rebalanceCb/msgChans 同 goroutine，无需加锁。
			// 下线租户的分区经 rebalance revoke 走 rebalanceCb 清理；新增租户的分区 assign 后按需建 channel
			log.Info("[Kafka] resubscribe topics on tenant change", zap.String("key", cm.key), zap.Strings("topics", topics))
			if err := cm.consumer.SubscribeTopics(topics, cm.rebalanceCb); err != nil {
				log.Error("[Kafka] resubscribe topics error", zap.String("key", cm.key), zap.Strings("topics", topics), zap.Error(err))
				// 重新入队自愈：lastTopics 已前进，若不重试，实际订阅将漂移到下一次租户变更才被纠正。
				// 自己是 resubChan 唯一消费者，下轮循环重试；消费者已停止的场景由 stopChan 分支先退出
				cm.pushResub(topics)
			}
		default:
			// 轮询事件（消息、错误、统计；rebalance 回调也在本 goroutine 内被触发）
			ev := cm.consumer.Poll(1000)
			if ev == nil {
				continue
			}
			switch e := ev.(type) {
			case *kafka.Message:
				cm.handleMessage(e)
			case kafka.Error:
				if e.Code() != kafka.ErrTimedOut {
					log.Error("[Kafka] consumer error", zap.Error(e))
				}
			case *kafka.Stats:
				// statistics.interval.ms 触发的 librdkafka 统计，观测 C 侧预取队列内存
				reportLibrdkafkaStats("consumer", cm.Name, cm.GroupId, e.String(),
					config2.PoolStatsLogEnabled(cm.StatsMode), config2.PoolStatsReportEnabled(cm.StatsMode))
			}
		}
	}
}

// handleMessage 处理单条消息
func (cm *ConsumerGroup) handleMessage(msg *kafka.Message) {
	if msg.TopicPartition.Topic == nil {
		log.Warn("[Kafka] consumer Msg pull topic is nil", zap.Any("msg", msg))
		return
	}
	// 取出对应的channel
	// 这里做 topic 进行分层，是因为如果消费使用的 topic 是通配符的，那么这个一条 consumer 就会消费多个 topic 的消息
	mp, ok := cm.msgChans[*msg.TopicPartition.Topic]
	if !ok {
		mp = make(map[int32]chan *kafka.Message)
		cm.msgChans[*msg.TopicPartition.Topic] = mp
	}
	ch, ok := mp[msg.TopicPartition.Partition]
	if !ok {
		// 初始化 channel，容量走配置（优先 topic_partition_channel_size，其次 msg_chan_cap），满则反压
		chanCap := cm.TopicPartitionChannelSize
		if chanCap <= 0 {
			chanCap = cm.MsgChanCap
		}
		if chanCap <= 0 {
			chanCap = 1024
		}
		ch = make(chan *kafka.Message, chanCap)
		mp[msg.TopicPartition.Partition] = ch
		cm.wg.Add(1)
		// 多功能单个和批量消费协程
		if cm.handler.HandlerFuncs != nil {
			log.Info("[Kafka] Go 批量消费", zap.Any("topic", msg.TopicPartition.Topic), zap.Any("partition", msg.TopicPartition.Partition))
			go cm.batchHandler(ch, cm.handler, msg.TopicPartition.Partition)
		} else {
			log.Info("[Kafka] Go 单个消费", zap.Any("topic", msg.TopicPartition.Topic), zap.Any("partition", msg.TopicPartition.Partition))
			go cm.singleHandler(ch, cm.handler, msg.TopicPartition.Partition)
		}
	}
	// 推送消息到每个分区
	select {
	case ch <- msg:
		return
	default:
		log.Warn("[Kafka] consumer Msg push to channel error: channel is full", zap.Any("ChannelSize", len(ch)), zap.Any("topic", msg.TopicPartition))
		ch <- msg
	}

}

// rebalanceCb 处理分区再均衡事件。
// confluent-kafka-go 在 Poll 所在 goroutine 内触发本回调，与 handleMessage/close 属同一 goroutine，
// 因此对 msgChans 的读写无需加锁；回调返回后由客户端自动执行默认的 assign/unassign（兼容 eager 与 cooperative 协议）。
// 修复问题：此前分区被调走后对应的 channel 与 handler goroutine 永不回收，
// 且残留的待提交偏移量可能覆盖新 owner 的提交。
func (cm *ConsumerGroup) rebalanceCb(_ *kafka.Consumer, event kafka.Event) error {
	switch e := event.(type) {
	case kafka.AssignedPartitions:
		log.Info("[Kafka] partitions assigned",
			zap.String("name", cm.Name),
			zap.String("groupId", cm.GroupId),
			zap.Any("partitions", e.Partitions))
	case kafka.RevokedPartitions:
		log.Info("[Kafka] partitions revoked",
			zap.String("name", cm.Name),
			zap.String("groupId", cm.GroupId),
			zap.Any("partitions", e.Partitions))
		// 停机路径上 close() 已经关闭并清空了所有 channel，而 consumer.Close() 内部的 Poll
		// 仍会把退组的 RevokedPartitions 投递到这里；此时不能再碰 channel，否则二次 close 会 panic。
		// close() 里已先置 stopFlag，与本回调同 goroutine，顺序可见性有保证。
		if cm.stopFlag.Load() {
			return nil
		}
		for _, tp := range e.Partitions {
			if tp.Topic == nil {
				continue
			}
			if mp, ok := cm.msgChans[*tp.Topic]; ok {
				if ch, ok := mp[tp.Partition]; ok {
					// 关闭后 handler 消费完 channel 内剩余消息自行退出；
					// 从 map 删除，避免 close() 二次关闭，分区回流时会重新创建
					close(ch)
					delete(mp, tp.Partition)
				}
			}
			// 丢弃该分区的待提交偏移量，避免 cron 提交覆盖新 owner 的进度（key 与 commit() 保持一致）
			cm.waitCommit.Delete(fmt.Sprintf("%v%v", *tp.Topic, tp.Partition))
		}
	}
	return nil
}

func (cm *ConsumerGroup) commit(msg *kafka.Message) {
	// 手动提交偏移量
	if cm.CommitType == config2.KAFKA_COMMITER_TYPE_BY_HAND {
		// 手动提交，这里会等到 kafka 提交成功后才会返回
		if _, err := cm.consumer.CommitMessage(msg); err != nil {
			log.Error("[Kafka] 拉取消息后自动提交错误", zap.Error(err))
		}
	} else if cm.CommitType == config2.KAFKA_COMMITER_TYPE_AUTO {
		// 拉取后自动提交，这里不需要处理，只需要改配置即可
	} else {
		key := fmt.Sprintf("%v%v", *msg.TopicPartition.Topic, msg.TopicPartition.Partition)
		// 这一步需要判断提交的偏移量是否正确
		pt, ok := cm.waitCommit.Load(key)
		if ok && pt.(kafka.TopicPartition).Offset >= msg.TopicPartition.Offset {
			// 如果待提交的偏移量已经存在，并且是大于当前的值，则不需要处理
			return
		}
		cm.waitCommit.Store(key, msg.TopicPartition)
	}
}
