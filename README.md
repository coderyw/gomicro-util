# gomicro-util

> Casino 业务线的 Golang 公共基础库与微服务框架。基于 [go-micro v4](https://go-micro.dev/) 封装了一套开箱即用的应用脚手架（`pmicro`），并沉淀了数据库、缓存、消息队列、限流、加解密、链路追踪、多租户等通用组件。一个 `pmicro.NewProgram(...)` 即可拉起 gRPC + HTTP + Kafka 消费 + 定时任务的完整服务。

> gomicro-util 由 ca-common-util 复制而来，**不依赖 ca-common-model**：错误码（`model/error-code`）与 gRPC 公共结果枚举（`model/common-result`）已内置。内置错误码数值与 ca-common-model 一致，可与存量服务互通；业务错误码在业务项目内用 `error_code.NewErrorCode(...)` 定义，无需修改本库。

## 文档导航

| 我想… | 看这里 |
|-|-|
| 从零启动一个服务 | [docs/getting-started.md](docs/getting-started.md) |
| 查 YAML 配置项 | [docs/configuration.md](docs/configuration.md) |
| 查环境变量及对应公共方法 | [docs/environment.md](docs/environment.md) |
| 配置里的密码改用凭据引用 | [docs/components/secret.md](docs/components/secret.md) |
| 了解多租户隔离规则 | [docs/multi-tenant.md](docs/multi-tenant.md) |
| 按目录查某个包是做什么的 | [docs/packages.md](docs/packages.md) |
| 升级框架版本 | [docs/upgrade/](#升级与变更) · [CHANGELOG.md](CHANGELOG.md) |
| 查某个函数的签名与说明 | `go doc ./library/<包>`，或在 IDE 中悬停 |

## 快速开始

需要 **Go 1.25+**，并配置私有模块：

```bash
go env -w GOPRIVATE=gitlab.jkluck.vip
go get gitlab.jkluck.vip/casino/golang/gomicro-util
```

最小服务（HTTP）：

```go
package main

import (
	"github.com/gin-gonic/gin"
	"gitlab.jkluck.vip/casino/golang/gomicro-util/library/log"
	"gitlab.jkluck.vip/casino/golang/gomicro-util/library/pmicro"
	"gitlab.jkluck.vip/casino/golang/gomicro-util/library/pmicro/option"
)

func main() {
	prog := pmicro.NewProgram(
		option.ServiceName("my-service"),
		option.ConfigFile("etc/cfg.yaml"), // 本地运行读此文件
		option.RegisterHttpHandler(func(e *gin.Engine) {
			e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
		}, true),
	)
	if err := prog.Run(); err != nil {
		log.Fatal(err.Error())
	}
}
```

本地运行设置 `ENVIRON=server-local` 即可。gRPC、Kafka、定时任务等完整示例见 [getting-started.md](docs/getting-started.md)。

运行期依赖按所用组件而定：MySQL/PostgreSQL、Redis、Kafka、Nacos 或 etcd。

## 组件总览

| 分类 | 包路径 | 作用 | 文档 |
|-|-|-|-|
| **应用框架** | `library/pmicro` | 配置加载、gRPC/HTTP 服务、Kafka 消费、Cron、优雅停机 | [pmicro](docs/components/pmicro.md) |
| **环境与租户** | `library/env`、`library/namespace` | 环境变量、多租户列表与资源命名 | [environment](docs/environment.md) · [multi-tenant](docs/multi-tenant.md) |
| **数据库** | `library/db/mysql` | GORM 多实例（MySQL/PG/OpenGauss）、分表、雪花主键 | [mysql](docs/components/mysql.md) |
| **缓存** | `library/db/cache/redis`、`local-cache` | Redis 多实例、分布式锁、缓存读穿、定时同步；进程内缓存 | [redis](docs/components/redis.md) |
| **消息队列** | `library/mq`、`library/beanstalk` | Kafka（单条/批量消费）、RabbitMQ、beanstalk 延迟队列 | [mq](docs/components/mq.md) |
| **HTTP** | `library/gin`、`library/http`、`library/metadata` | 参数校验与统一响应、HTTP 客户端、ctx 元数据透传 | [http](docs/components/http.md) |
| **定时任务** | `library/cron`、`library/consistent_hash` | 定时任务、多租户定时任务、一致性哈希任务分片 | [scheduling](docs/components/scheduling.md) |
| **可观测** | `library/log`、`library/trace`、`library/skywalking`、`library/pmicro/metrics` | zap 日志、链路追踪、指标采集与 TrueWatch 上报 | [observability](docs/components/observability.md) |
| **配置中心** | `library/site-var`、`library/customer-config` | etcd/nacos 读写监听、自定义配置泛型读取 | [config-center](docs/components/config-center.md) |
| **凭据引用** | `library/component/secret` | 配置中写 `csms://` / `awssm://` 引用，启动时从云凭据服务取回并自动日志脱敏 | [secret](docs/components/secret.md) |
| **业务组件** | `library/component/*`、`util/ratelimit` | jwt、飞书机器人、短信、限流、对象存储 | [component](docs/components/component.md) |
| **测试** | `library/mock`、`mocks/` | mock 开关与 mockery 生成的 mock | [testing](docs/components/testing.md) |
| **工具** | `util/*` | 加解密、编码、数值、日期、雪花位运算等纯工具包 | [packages · util](docs/packages.md#util) |

## 项目结构

```text
gomicro-util/
├── init.go       # 库级初始化（decimal JSON 序列化为数字格式）
├── library/      # 框架与基础组件（见上表）
├── internal/     # 仅供本仓库内部使用的实现（多租户、本地缓存、存储 provider、APM）
├── model/        # 配置结构体、常量、facade 接口
├── util/         # 无框架依赖的纯工具包
├── mocks/        # interface 的 mock，层级与源码一致
├── docs/         # 使用文档（本 README 为入口）
└── test/         # 基准测试与示例
```

## 升级与变更

| 版本 | 说明 |
|-|-|
| v3.1.0 | [凭据引用组件（secret）带来的依赖与行为变化](docs/upgrade/v3.1.0.md) |
| v3.0.0.0 | [租户命名大小写与自动续约锁](docs/upgrade/v3.0.0.0.md) |
| v1.x | [gorm / confluent-kafka 切换、`NACOS_NAMESPACE_ID` 必填](docs/upgrade/v1.x.md) |

逐版本变更见 [CHANGELOG.md](CHANGELOG.md)，未正式发布的变更见 [CHANGELOG-UNSTABLE.md](CHANGELOG-UNSTABLE.md)。

## 文档维护约定

每类信息只维护一处，修改代码时同步更新对应位置：

| 改动 | 更新位置 |
|-|-|
| 新增/修改环境变量 | [docs/environment.md](docs/environment.md) |
| 新增/修改 YAML 配置项 | `model/config` 字段注释 + [docs/configuration.md](docs/configuration.md) |
| 新增/修改公共方法 | 方法注释（`go doc` 可见） |
| 组件用法变化 | `docs/components/` 下对应文档 |
| 新增包 | 包内 `doc.go` + [docs/packages.md](docs/packages.md)，必要时补充本页组件总览 |
| 破坏性变更 | `docs/upgrade/<版本>.md` + 本页「升级与变更」 |

README 只做导航，不再直接写组件用法。
