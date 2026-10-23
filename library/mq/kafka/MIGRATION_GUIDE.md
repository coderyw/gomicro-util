# 从Sarama迁移到Confluent-Kafka-Go指南

## 概述

本指南将帮助您从Sarama Kafka客户端迁移到Confluent-Kafka-Go。Confluent-Kafka-Go是基于librdkafka的Go客户端，提供了更好的性能和稳定性。本指南涵盖了消费者和生产者的完整迁移方案。

## 主要差异

### 1. 依赖库
```go
// 原Sarama依赖
import (
    "github.com/IBM/sarama"
    cluster "github.com/coderyw/gomicro-util/library/mq/kafka/sarama-cluster"
)

// 新的Confluent依赖
import (
    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)
```

### 2. 消费者管理器
```go
// 原Sarama消费者管理器
manager := NewConsumerManager()

// 新的Confluent消费者管理器
manager := NewConfluentConsumerManager()
```

### 3. 生产者管理器
```go
// 原Sarama生产者（全局函数）
Push(ctx, key, topic, obj)
PushSync(ctx, key, topic, obj)

// 新的Confluent生产者管理器
manager := NewConfluentProducerManager()
manager.Push(ctx, key, topic, obj)
manager.PushSync(ctx, key, topic, obj)
```

### 4. 完整管理器
```go
// 新的Confluent完整管理器（推荐）
manager := NewConfluentKafkaManager()
manager.Init(serverName, cfg, handlers)
manager.Push(ctx, key, topic, obj)
manager.Stop()
```

### 5. 配置方式
```go
// 原Sarama配置
clusterCfg := cluster.NewConfig()
clusterCfg.Net.SASL.Enable = true
clusterCfg.Net.SASL.User = username
clusterCfg.Net.SASL.Password = password

// 新的Confluent配置
configMap := &kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":         "my-group",
    "security.protocol": "SASL_PLAINTEXT",
    "sasl.mechanisms":   "PLAIN",
    "sasl.username":     username,
    "sasl.password":     password,
}
```

## 迁移步骤

### 步骤1: 更新依赖

在`go.mod`文件中添加Confluent-Kafka-Go依赖：

```go
require (
    github.com/confluentinc/confluent-kafka-go/v2 v2.3.0
)
```

### 步骤2: 更新导入

将代码中的Sarama导入替换为Confluent：

```go
// 替换前
import (
    "github.com/IBM/sarama"
    cluster "github.com/coderyw/gomicro-util/library/mq/kafka/sarama-cluster"
)

// 替换后
import (
    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
)
```

### 步骤3: 更新消费者管理器

```go
// 替换前
manager := NewConsumerManager()

// 替换后
manager := NewConfluentConsumerManager()
```

### 步骤4: 更新生产者管理器

```go
// 替换前（全局函数）
Push(ctx, key, topic, obj)
PushSync(ctx, key, topic, obj)

// 替换后（推荐使用完整管理器）
manager := NewConfluentKafkaManager()
manager.Init(serverName, cfg, handlers)
manager.Push(ctx, key, topic, obj)
manager.PushSync(ctx, key, topic, obj)
```

### 步骤5: 更新配置

配置结构保持不变，但内部实现会使用Confluent的配置方式：

```go
cfg := config2.KafkaCfg{
    DefaultConsumer: "default",
    Consumer: []config2.KafkaConsumer{
        {
            KafkaHost: config2.KafkaHost{
                Name: "default",
                Host: []string{"localhost:9092"},
                // SASL配置保持不变
                Username: "user",
                Pwd:      "pass",
            },
            Groups: []config2.Group{
                {
                    GroupName: "test-group",
                    MaxMulNum: 100,
                },
            },
        },
    },
}
```

### 步骤6: 更新消息处理器

消息处理器接口保持不变，内部会自动进行消息格式转换：

```go
handlers := []mq.Handler{
    {
        GroupId: "test-group",
        Topics:  "test-topic",
        HandlerFunc: func(ctx context.Context, msg sarama.ConsumerMessage) error {
            // 处理逻辑保持不变
            log.Printf("Processing message: %s", string(msg.Value))
            return nil
        },
    },
}
```

## 性能优化

### 1. 生产者优化

Confluent-Kafka-Go提供了更好的生产者性能：

```go
// 批量推送优化
messages := []BatchMessage{
    NewBatchMessage("key1", "topic1", data1),
    NewBatchMessage("key2", "topic2", data2),
    NewBatchMessage("key3", "topic3", data3),
}

// 批量异步推送
err := manager.BatchPush(ctx, messages)

// 批量同步推送
err := manager.BatchPushSync(ctx, messages)
```

### 2. 消费者批量处理优化

Confluent-Kafka-Go提供了更好的批量处理性能：

```go
handlers := []mq.Handler{
    {
        GroupId: "batch-group",
        Topics:  "batch-topic",
        HandlerFuncs: func(ctx context.Context, msgs []sarama.ConsumerMessage) (int, error) {
            // 批量处理逻辑
            for _, msg := range msgs {
                // 处理消息
            }
            return len(msgs), nil
        },
    },
}
```

### 3. 配置优化

```go
// 优化配置示例
cfg := config2.KafkaCfg{
    DefaultProducer: "primary",
    Producer: []config2.KafkaProducer{
        {
            KafkaHost: config2.KafkaHost{
                Host: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
            },
            MaxRetryTimes: 3,
            RequiredAcks:  int16Ptr(-1), // WaitForAll
        },
    },
    Consumer: []config2.KafkaConsumer{
        {
            KafkaHost: config2.KafkaHost{
                Host: []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
            },
            Groups: []config2.Group{
                {
                    GroupName:              "optimized-group",
                    MaxMulNum:             200,        // 增加批量大小
                    MulTimeoutMs:          500,        // 减少超时时间
                    MaxProcessingTimeMill: 5000,       // 增加处理时间
                },
            },
        },
    },
}
```

## 错误处理

### 1. 连接错误处理

```go
// Confluent提供了更详细的错误信息
if err := manager.Init("server", cfg, handlers); err != nil {
    if kafkaErr, ok := err.(kafka.Error); ok {
        switch kafkaErr.Code() {
        case kafka.ErrTransport:
            log.Error("Transport error:", kafkaErr)
        case kafka.ErrAllBrokersDown:
            log.Error("All brokers down:", kafkaErr)
        default:
            log.Error("Kafka error:", kafkaErr)
        }
    }
    return err
}
```

### 2. 生产者错误处理

```go
// 同步推送错误处理
if err := manager.PushSync(ctx, key, topic, obj); err != nil {
    log.Error("Sync push failed:", err)
    // 处理错误
}

// 异步推送错误处理（通过事件监听）
// 错误会在handleDeliveryReports中处理
```

### 3. 消费者消息处理错误

```go
HandlerFunc: func(ctx context.Context, msg sarama.ConsumerMessage) error {
    // 处理消息
    if err := processMessage(msg); err != nil {
        // 记录错误但不中断消费
        log.Error("Message processing error:", err)
        return err
    }
    return nil
},
```

## 监控和调试

### 1. 状态监控

```go
// 监控完整状态
health := manager.HealthCheck()
log.Printf("Health: %+v", health)

// 监控消费者状态
log.Printf("Consumer count: %d", manager.GetConsumerCount())
log.Printf("Producer count: %d", manager.GetProducerCount())
log.Printf("Is running: %t", manager.IsRunning())

// 获取生产者统计信息
stats, err := manager.GetProducerStats("default")
if err != nil {
    log.Error("Failed to get producer stats:", err)
} else {
    log.Printf("Producer stats: %+v", stats)
}
```

### 2. 指标监控

```go
// 获取指标信息
metrics := manager.Metrics()
log.Printf("Metrics: %+v", metrics)
```

### 3. 日志配置

```go
// 启用详细日志
configMap := &kafka.ConfigMap{
    "bootstrap.servers": "localhost:9092",
    "group.id":         "my-group",
    "debug":            "all",  // 启用所有调试信息
    "log.thread.name":  true,
    "log.connection.close": true,
}
```

## 常见问题

### 1. 连接问题

**问题**: 无法连接到Kafka集群
**解决方案**: 检查网络连接和SASL配置

```go
// 确保SASL配置正确
configMap := &kafka.ConfigMap{
    "bootstrap.servers": "kafka:9092",
    "security.protocol": "SASL_PLAINTEXT",
    "sasl.mechanisms":   "PLAIN",
    "sasl.username":     "username",
    "sasl.password":     "password",
}
```

### 2. 生产者性能问题

**问题**: 生产者性能不如预期
**解决方案**: 调整生产者配置

```go
Producer: []config2.KafkaProducer{
    {
        KafkaHost: config2.KafkaHost{
            Host: []string{"kafka1:9092", "kafka2:9092"},
        },
        MaxRetryTimes: 3,
        RequiredAcks:  int16Ptr(1), // 减少确认等待
    },
},
```

### 3. 消费者性能问题

**问题**: 消费性能不如预期
**解决方案**: 调整批量处理参数

```go
Groups: []config2.Group{
    {
        GroupName:              "performance-group",
        MaxMulNum:             500,        // 增加批量大小
        MulTimeoutMs:          100,        // 减少等待时间
        MaxProcessingTimeMill: 10000,      // 增加处理时间
    },
},
```

### 4. 内存问题

**问题**: 内存使用过高
**解决方案**: 调整缓冲区大小

```go
// 消费者配置
configMap := &kafka.ConfigMap{
    "bootstrap.servers":        "localhost:9092",
    "group.id":                "my-group",
    "fetch.min.bytes":          1,
    "fetch.wait.max.ms":        500,
    "max.partition.fetch.bytes": 1048576,  // 1MB
    "queued.max.messages.kbytes": 1024,    // 1MB队列
}

// 生产者配置
configMap := &kafka.ConfigMap{
    "bootstrap.servers":        "localhost:9092",
    "batch.size":              16384,      // 16KB
    "buffer.memory":           33554432,   // 32MB
    "linger.ms":               1,
    "compression.type":        "snappy",
}
```

### 5. 消息丢失问题

**问题**: 消息可能丢失
**解决方案**: 使用同步推送或调整确认机制

```go
// 使用同步推送确保消息不丢失
err := manager.PushSync(ctx, key, topic, obj)

// 或者调整生产者配置
Producer: []config2.KafkaProducer{
    {
        RequiredAcks: int16Ptr(-1), // WaitForAll
        MaxRetryTimes: 5,
    },
},
```

## 测试

### 1. 单元测试

```go
func TestConfluentKafkaManager(t *testing.T) {
    manager := NewConfluentKafkaManager()
    defer manager.Stop()
    
    // 测试配置构建
    cfg := config2.KafkaCfg{...}
    handlers := []mq.Handler{...}
    
    // 初始化管理器
    err := manager.Init("test-server", cfg, handlers)
    if err != nil {
        t.Logf("Initialization failed (expected in test environment): %v", err)
    } else {
        assert.True(t, manager.IsInitialized())
        assert.Equal(t, 1, manager.GetConsumerCount())
        assert.Equal(t, 1, manager.GetProducerCount())
    }
}
```

### 2. 集成测试

```go
func TestConfluentIntegration(t *testing.T) {
    manager := NewConfluentKafkaManager()
    defer manager.Stop()
    
    // 配置测试环境
    cfg := config2.KafkaCfg{...}
    handlers := []mq.Handler{...}
    
    // 初始化管理器
    err := manager.Init("test-server", cfg, handlers)
    if err != nil {
        t.Logf("Integration test failed (expected in test environment): %v", err)
        return
    }
    
    // 验证状态
    assert.True(t, manager.IsInitialized())
    assert.Equal(t, 1, manager.GetConsumerCount())
    assert.Equal(t, 1, manager.GetProducerCount())
    assert.True(t, manager.IsRunning())
    
    // 测试消息推送
    ctx := context.Background()
    message := map[string]interface{}{"test": "data"}
    err = manager.Push(ctx, "test-key", "test-topic", message)
    if err != nil {
        t.Logf("Push failed (expected in test environment): %v", err)
    }
}
```

### 3. 性能测试

```go
func BenchmarkConfluentKafkaManager(b *testing.B) {
    manager := NewConfluentKafkaManager()
    defer manager.Stop()
    
    // 初始化管理器
    cfg := config2.KafkaCfg{...}
    handlers := []mq.Handler{...}
    err := manager.Init("benchmark-server", cfg, handlers)
    if err != nil {
        b.Skipf("Benchmark skipped: %v", err)
    }
    
    ctx := context.Background()
    message := map[string]interface{}{"benchmark": "data"}
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = manager.Push(ctx, fmt.Sprintf("key-%d", i), "benchmark-topic", message)
    }
}
```

## 总结

迁移到Confluent-Kafka-Go的主要优势：

1. **更好的性能**: 基于librdkafka，性能更优
2. **更稳定的连接**: 更好的错误处理和重连机制
3. **更丰富的配置**: 支持更多Kafka配置选项
4. **更好的监控**: 提供更详细的监控信息
5. **统一管理**: 消费者和生产者统一管理，简化代码
6. **批量操作**: 支持批量消息推送和处理
7. **健康检查**: 提供完整的健康检查和指标监控
8. **向后兼容**: 保持原有的接口和配置结构

### 推荐使用方式

```go
// 推荐使用完整管理器
manager := NewConfluentKafkaManager()
defer manager.Stop()

// 初始化
err := manager.Init(serverName, cfg, handlers)
if err != nil {
    log.Fatal(err)
}

// 使用
err = manager.Push(ctx, key, topic, data)
if err != nil {
    log.Error(err)
}

// 监控
health := manager.HealthCheck()
log.Printf("Health: %+v", health)
```

通过遵循本指南，您可以顺利地从Sarama迁移到Confluent-Kafka-Go，同时保持代码的兼容性和稳定性，并获得更好的性能和可维护性。 