# Kafka Architecture

本文档描述 `library/mq/kafka` 在当前仓库中的运行结构，基于 `confluent-kafka-go` 实现。

## Overall

```mermaid
flowchart LR
    A[业务服务 / Handler 注册] --> B[pmicro.Program.Start]
    B --> C[mq.InitKafka]
    C --> D[InitConfluentKafkaManager]

    subgraph KM[confluentKafkaManager]
        D --> PM[ConfluentProducerManager]
        D --> CM[ConfluentConsumerManager]
    end

    PM --> P1[多个 kafka.Producer]
    P1 --> K[(Kafka Cluster)]
    K --> C1[多个 kafka.Consumer]
    C1 --> CM

    CM --> G1[ConsumerGroup<br/>instance + groupId + topic]
    G1 --> H1[单条消费 HandlerFunc]
    G1 --> H2[批量消费 HandlerFuncs]

    B --> J[cron jobs]
    CM --> J
```

## Producer Flow

```mermaid
flowchart TD
    A[业务代码调用 mq.GetKafkaManager.Push / PushSync] --> B[confluentKafkaManager.Push]
    B --> C[namespace.FmtByType<br/>补齐租户 topic]
    C --> D[ConfluentProducerManager.pushMessageWithProducer]
    D --> E[getKafkaMsg]
    E --> E1[校验 Validator]
    E --> E2[序列化 string / bytes / JSON]
    E --> E3[注入 headers<br/>mq_trace_id + namespace]
    E --> F[kafka.Producer.Produce]
    F --> G[(Kafka Topic)]
    F --> H[delivery reports 协程]
    D --> I[推送耗时统计 pushData]
```

## Consumer Flow

```mermaid
flowchart TD
    A[配置 KafkaCfg.Consumer + MQKafkaHandler] --> B[ConfluentConsumerManager.InitKafkaConsumer]
    B --> C[buildGroupConfigs]
    C --> D[buildConsumers]
    D --> E[startConsumers]
    E --> F[kafka.NewConsumer + SubscribeTopics]
    F --> G[ConsumerGroup.subscribe]
    G --> H[ReadMessage 轮询]
    H --> I[按 partition 分发到 msgChans]

    I --> J1[singleHandler]
    I --> J2[batchHandler]

    J1 --> K1[handleHeader<br/>恢复 traceId / namespace]
    K1 --> L1[包装为 KafkaConsumerMessage]
    L1 --> M1[HandlerFunc]

    J2 --> K2[按 MaxMulNum / MulTimeoutMs 聚合]
    K2 --> L2[包装为Array KafkaConsumerMessage]
    L2 --> M2[HandlerFuncs]

    M1 --> N[commit]
    M2 --> N
    N --> O1[CommitType=0<br/>消费后缓存 offset]
    N --> O2[CommitType=1<br/>Kafka 自动提交]
    N --> O3[CommitType=2<br/>逐条 CommitMessage]
    O1 --> P[cron doCommit 周期提交]
```

## Consumer Concurrency Model

这里的“线程模型”更准确地说是 Go 协程模型。当前实现里，一个 `ConsumerGroup` 绑定一个 `kafka.Consumer` 和一个 `subscribe()` 读协程；但由于订阅使用正则，单个 `ConsumerGroup` 可能同时消费多个实际 topic，所以内部已经变成 `topic -> partition -> worker` 的两级展开模型。

```mermaid
flowchart TD
    A[ConfluentConsumerManager.startConsumers] --> B[创建 1 个 ConsumerGroup]
    B --> C[创建 1 个 kafka.Consumer]
    C --> D[启动 1 个 subscribe 协程]

    subgraph G[ConsumerGroup 内部]
        D --> E[ReadMessage 轮询]
        E --> F{topic 是否已存在}
        F -- 否 --> G1[创建 msgChans topic]
        F -- 是 --> H1[复用该 topic 的分区表]
        G1 --> I1{partition 是否已存在对应 channel}
        H1 --> I1
        I1 -- 否 --> J1[创建 msgChans topic partition]
        J1 --> J2{消费模式}
        I1 -- 是 --> K[复用已有 partition channel]
        J2 -- 单条 --> L[singleHandler 协程]
        J2 -- 批量 --> M[batchHandler 协程]
        N --> L
        N --> M
        K --> N[消息写入该 topic partition channel]
        J1 --> N
        L --> O[handleHeader -> HandlerFunc -> commit]
        M --> P[聚合消息 -> HandlerFuncs -> commit]
        O --> Q[waitCommit 或 CommitMessage]
        P --> Q
    end
```

单个 `ConsumerGroup` 的并发规则：

- 固定 1 个读协程：`subscribe()` 负责轮询 `ReadMessage`，它自己不执行业务处理。
- 先按 topic 分层：`msgChans` 现在是 `map[topic]map[partition]chan`，用于承接正则订阅命中的多个实际 topic。
- 再按分区惰性扩容：某个 `topic/partition` 首次收到消息时，才创建该分区专属的 `channel + handler goroutine`。
- 同一 `topic/partition` 内严格串行：消息进入同一个 channel，由同一个 worker 顺序处理。
- 不同 `topic`、不同 `partition` 之间并行：只要 channel 不同，就有独立 worker。
- 并发上限约等于“当前活跃的 topic-partition 数”，而不是消息数，也不是 handler 数。
- channel 满时不会丢消息：当前实现会先打 `channel is full` 告警，然后阻塞写入，背压会直接传回读协程。

停机收敛模型：

```mermaid
sequenceDiagram
    participant M as ConsumerManager
    participant S as subscribe goroutine
    participant C as topic-partition channels
    participant H as handler goroutines
    participant K as kafka.Consumer

    M->>S: 关闭 stopChan
    S-->>S: 跳出 ReadMessage 循环
    S->>C: 关闭所有 msgChans
    C-->>H: channel close
    H-->>H: 处理剩余消息并退出
    H->>S: wg.Done
    S->>S: doCommit()
    S->>K: consumer.Close()
```

停机阶段的顺序是：

- 先停读，不再从 Kafka 拉新消息。
- 再关闭所有 `topic/partition channel`，让已有 handler 把 channel 中剩余消息处理完。
- 等所有 handler 协程退出后，补做一次 `doCommit()`。
- 最后关闭底层 `kafka.Consumer`。

## Key Design Points

- 启动入口在 `library/pmicro/program.go`，Kafka 不是独立进程，而是作为服务启动流程的一部分初始化。
- 管理器拆分为 `ProducerManager` 和 `ConsumerManager`，由 `confluentKafkaManager` 统一对外暴露。
- 消费者实例的粒度是 `instanceName + groupId + topic`，每个实例内部再按 `partition` 开 goroutine 处理，保证分区内串行。
- `HandlerFunc` 用于单条消费，`HandlerFuncs` 用于批量消费；两者最终都走同一套 trace、日志和 offset 提交流程。
- topic 会经过 `namespace.FmtByType(...)` 处理，消费侧则通过 topic 和 header 反推 namespace 到上下文。
- 默认提交策略是 `CommitType=0`，即消费完成后先缓存 offset，再通过 cron 周期性提交。
