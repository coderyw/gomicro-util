# 从零启动一个 pmicro 服务

> 返回 [README](../README.md)。

## 1. 准备

```bash
go get github.com/coderyw/gomicro-util
```

需要 Go 1.25+，所有依赖都是公开模块。雪花 ID 依赖 `github.com/coderyw/goeonid/v2`。

## 2. 准备配置

- 本地：在项目中放一份 YAML（例如 `etc/cfg.yaml`），通过 `option.ConfigFile(...)` 指定。字段参考 [configuration.md](configuration.md)，只需写用到的组件。
- 非本地：配置放在 Nacos `NACOS_NAMESPACE_ID / DEFAULT_GROUP / <服务名>.yaml`。

## 3. 设置环境变量

本地最少只需：

```bash
export ENVIRON=server-local
```

非本地需要 `NACOS_URLS`、`NACOS_NAMESPACE_ID`，多租户部署还需要 `NAMESPACE`。完整清单见 [environment.md](environment.md)。

## 4. 编写 main

```go
package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/coderyw/gomicro-util/library/log"
	"github.com/coderyw/gomicro-util/library/mq/model"
	"github.com/coderyw/gomicro-util/library/pmicro"
	"github.com/coderyw/gomicro-util/library/pmicro/option"
	"github.com/coderyw/gomicro-util/model/cron"
	"go-micro.dev/v4"
)

func main() {
	prog := pmicro.NewProgram(
		// 服务名：注册中心、Nacos dataId（<服务名>.yaml）都基于它
		option.ServiceName("my-service"),
		// 本地运行读取的配置文件
		option.ConfigFile("etc/cfg.yaml"),
		// 注册 gRPC handler
		option.RegisterHandlerFunc(func(service micro.Service) error {
			return pb.RegisterMyServiceHandler(service.Server(), &handler{})
		}),
		// HTTP 路由
		option.RegisterHttpHandler(func(e *gin.Engine) {
			e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
		}, true),
		// Kafka 单条消费：consumer 实例名对应 kafka.consumer[].name；groupId 省略时默认为服务名
		option.MQHandler("core1", "topic-1", func(ctx context.Context, msg model.KafkaConsumerMessage) error {
			return nil
		}, "consumer-group-1"),
		// Kafka 批量消费
		option.MQMulHandler("core1", "topic-2", func(ctx context.Context, msgs []model.KafkaConsumerMessage) (int, error) {
			return len(msgs), nil
		}, "consumer-group-2"),
		// 定时任务（可变参数，可传多个）
		option.Cron(cron.CronJob{
			Name:    "demo",
			DoJob:   func() { /* do job */ },
			CronSpc: "0 */5 * * * ?",
		}),
		// 业务自定义配置（YAML 的 other 段，结构体字段须写 json tag）；启动时和 Nacos 变更时回调
		option.CustomDefaultCfg(&MyCfg{}, func(cfg any) {}),
		// 生命周期钩子
		option.BeforeStartFunc(func() error { return nil }), // 存储组件初始化后、gRPC 启动前
		option.AfterFunc(func() error { return nil }),       // gRPC、Kafka 启动后，HTTP 与定时任务启动前
		option.BeforeStopFunc(func() error { return nil }),  // 停服时
	)
	if err := prog.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
```

`Run()` 的启动与停机顺序见 [pmicro · 启动与停机生命周期](components/pmicro.md#启动与停机生命周期)。

## 5. 下一步

| 想做的事 | 文档 |
|-|-|
| 读写 MySQL | [components/mysql.md](components/mysql.md) |
| 用 Redis、分布式锁、缓存读穿 | [components/redis.md](components/redis.md) |
| 生产/消费 Kafka | [components/mq.md](components/mq.md) |
| HTTP 参数校验与统一响应 | [components/http.md](components/http.md) |
| 定时任务、多租户定时任务、任务分片 | [components/scheduling.md](components/scheduling.md) |
| 多租户资源隔离 | [multi-tenant.md](multi-tenant.md) |
| 写单测与 mock | [components/testing.md](components/testing.md) |
