// Package pmicro 基于 go-micro v4 的微服务应用框架。
//
// 业务服务通过 NewProgram 传入 option 包中的选项，再调用 Run 启动。框架负责：
// 加载配置（本地 YAML / Nacos）、初始化日志、Redis、MySQL、gRPC、Kafka、指标上报、
// HTTP 与探活服务、定时任务，并在收到 SIGINT/SIGTERM 后按序优雅停机。
//
// 使用方式：
//
//	prog := pmicro.NewProgram(
//		option.ServiceName("my-service"),
//		option.ConfigFile("etc/cfg.yaml"),
//		option.RegisterHttpHandler(func(e *gin.Engine) {
//			e.GET("/ping", func(c *gin.Context) { c.String(200, "pong") })
//		}, true),
//	)
//	if err := prog.Run(); err != nil {
//		log.Fatal(err.Error())
//	}
//
// 启动前必须设置环境变量 ENVIRON，本地为 server-local。
//
// 文档：docs/getting-started.md、docs/components/pmicro.md、docs/configuration.md、docs/environment.md
package pmicro
