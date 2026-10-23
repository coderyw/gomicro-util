# Casino Golang common util 非稳定版本

Golang公共包，以及框架库。包含微服务gomicro、数据库mysql，redis、链路追踪jaeger，skywalking、gin、轻量级延迟队列beanstalk、

## 版本变动

### 待发布：优化-v3.0.0.0

- 新增 `TENANT_NAME_CASE`：`lower`（默认）、`upper`、`original`。配置值忽略大小写和首尾空白；由 `library/env` 启动时读取，非法值在 `SyncEnv()` 校验时报错，不在包初始化时 panic。运行中不切换命名模式。
- 统一 MySQL、Redis（含锁、扫描及同步任务）、Kafka、S3/OBS storage 组件、本地缓存及定时任务资源名称中的租户段。既有小写租户默认生成的名称不变，业务名称、分隔符、原始 ctx 和 `TENANT_FALLBACK` 行为不变；保留 Cron 原有锁名结构。
- RabbitMQ 及旧 `object-storage` 接口仍透传调用方提供的资源名，没有新增自动租户前缀；调用方需自行保证租户隔离。
- 自动续约锁不再随请求 ctx 取消而停止续约：在业务 `do()` 返回或 panic 后停止续约并校验持有者解锁。业务仍使用原 ctx，应自行处理取消并确保最终退出；没有新增强制续约上限。
- 解锁使用保留租户信息的独立 ctx。保留原有 Redis 全局超时策略，不开启 `ContextTimeoutEnabled`；清理 ctx 的 2 秒 deadline 不代表整个解锁网络操作严格在 2 秒内结束，I/O 仍受客户端读写超时约束。


### v1.1.0

> 已转正式版，详见 CHANGELOG.md 的 v1.1.0 条目。

feat(namespace): 添加租户列表变更监听及动态消费组订阅调整

- 新增租户列表变更监听器注册与通知机制，仅在租户列表内容变化时触发
- 监听器回调被保护，避免panic影响配置监听流程
- 修改命名空间租户加载逻辑，内容变更时通知所有监听器
- Kafka消费者管理器支持监听租户变更，动态重订阅消费组主题
- 优化消费组主题生成，改为按租户列表精确订阅，避免误订其他命名空间租户topic
- 新增对重订阅请求的通道管理，支持重试和并发安全投递
- 新增多租户topic生成规则单元测试，验证去重排序与兜底租户逻辑
- 新增租户变更监听相关测试，覆盖触发时机和监听器panic隔离
- 废弃原有通过正则表达式匹配topic的订阅方式，提升多租户隔离度和订阅准确性

feat(tenant): 重构租户管理及云厂商识别功能

- 引入 tenant 包，定义 Tenant 结构体及云厂商枚举类型和解析方法
- mul-tenant 模块支持活跃状态租户过滤及兜底租户配置，优化租户信息管理
- nacos_tenant.go 新增返回全部租户信息接口及上下文切片转换，增强租户数据访问
- env 包新增云厂商自动探测逻辑，优先读取环境变量，依次探测 DMI 与 IMDS 元数据服务
- 实现多云厂商（华为、AWS、Azure、Google、腾讯、阿里云）识别支持
- 新增详尽云厂商探测和租户配置解析单元测试，提升代码覆盖率和稳定性
- 修正本地环境租户加载逻辑，支持空租户兜底及多租户正确初始化

feat(env,pmicro)!: Nacos NamespaceId 改由 NACOS_NAMESPACE_ID 环境变量指定（BREAKING CHANGE）

- 新增常量 `cst.ENV_NACOS_NAMESPACE_ID`、包级变量 `nacosNamespaceId` 与访问器 `env.GetNacosNamespaceId()`，`init()` 中读入
- `checkEnv()` 非本地环境走 nacos 分支时校验该变量，为空直接返回错误；启动日志同时打印 NACOS_URLS 与 NACOS_NAMESPACE_ID
- `InitCfg` 的 `Registry.NameId`、`initFromNacos` 的 `nameId` 均由 `ENVIRON` 改取 `GetNacosNamespaceId()`，后者为空时返回错误不再初始化
- yaml 中的 `registry.name_id` 会被该环境变量覆盖；README 标注 `ENVIRON` 为[废弃]，建议命名 `[tenant_id]-[service]`
- 与多租户元数据所在的 `TENANT_CFG_NACOS_NAMESPACE`（默认 tenant）职责不同，互不影响
- 升级前须先在部署侧注入 NACOS_NAMESPACE_ID，否则非 local 环境启动失败

chore(deps): 升级 go-redis 及相关间接依赖

- `github.com/redis/go-redis/v9` v9.14.0 → v9.20.1
- `github.com/klauspost/cpuid/v2` v2.2.9 → v2.2.10
- 移除间接依赖 `github.com/dgryski/go-rendezvous`

feat(pmicro/http): 请求日志支持按配置打印请求 header

- `HttpCfg` 新增 `log_headers` 配置：不配置不打印（保持原行为）；配置 `*` 打印所有 header；否则按 `,` 拼接需要打印的 header 名（如 `X-Request-Id,User-Agent`，大小写不敏感）
- 命中的 header 以 `headers` 字段输出在 `HTTP Handle start` 日志中，支持同名多值
- `middleware.CustomLogger` 签名改为可变参数 `CustomLogger(logHeaders ...string)`，原无参调用方式兼容

feat(secret): 新增凭据引用解析组件，Nacos/本地配置中的敏感项可写成 `csms://凭据名/键` 或 `awssm://凭据名/键` 引用，配置加载时自动替换为凭据明文

- 新增 `library/component/secret`：引用扫描/替换（字符串值 JSON 转义加引号、数字布尔保类型）、按 scheme 路由 Provider、进程内缓存（一凭据一拉取，轮换靠重启）、按错误分类重试（指数退避 + 全抖动，整体 deadline 收口，详见下方 feat(secret) 启动期重试条目；Provider 客户端初始化失败同样重试，覆盖 sidecar 未就绪时凭据链出站超时；凭据不存在、region 非法等确定性错误不重试）；引用须为整段标量值，yaml 注释内的引用不解析；错误与日志不携带凭据值
- 新增 Provider：`provider/csms`（华为云 CSMS，凭证走 SDK 默认链，原生支持 CCE Pod Identity，region 取 `HUAWEICLOUD_SDK_REGION` 缺省 ap-southeast-3）、`provider/awssm`（AWS Secrets Manager，默认凭证链/IRSA，region 取 `AWS_REGION` 缺省 ap-southeast-1）；由 pmicro 配置加载时自动注册
- 挂点：`site-var/nacos` 的 `GetNacosCfgData`/`ConfigListen` 与 pmicro 的 `initFromCfgFile` 在反序列化前统一解析引用；启动阶段解析失败直接报错退出，热更新阶段解析失败只告警并保留旧配置
- 无引用的明文配置原样透传，对存量服务零影响；未注册的 `xxx://` scheme 只告警不替换
- 新增依赖 `github.com/huaweicloud/huaweicloud-sdk-go-v3`、`github.com/aws/aws-sdk-go-v2/service/secretsmanager`
- 引用替换改走 yaml.v3 / encoding-json 的节点树（`secret.UnmarshalYAML` / `UnmarshalJSON`）：只替换值位置上整体为引用的标量，注释、映射键不参与，块标量 / 引号风格 / 锚点别名 / 流式集合由解析器处理；拼在 DSN/URL/文字中间的引用判 ErrInvalidRef；反序列化错误经 `SanitizeDecodeError` 脱去凭据值（已登记明文按值掩码 + yaml 反引号原文抹为 `***`）。`ResolveText` 文本级替换与自写的注释/标量边界扫描一并移除
- 同一凭据的并发首次拉取用 `golang.org/x/sync/singleflight` 合并为一次远程调用（x/sync 已是直接依赖）；拉取失败不写缓存，下次重新拉取
- 引用拼接判定改为"值内任意位置含已注册 scheme 的 `xxx://` 即非法"（按子串而非单词切分），`124csms://x`、`abccsms://x` 这类紧贴前缀的写法不再静默变成字面量密码
- 日志脱敏按值兜底：解析出的字符串明文登记到 `library/log` 脱敏表（`RegisterRedactValue`），`InitDefaultLogger` 的编码器外包一层 `redactEncoder`，条目编码后整行把登记值替换为 `***`，覆盖 message / 任意字段 / MarshalLogObject 嵌套对象；与既有按字段名的脱敏叠加而非替代。短于 6 字符的值不登记，值的 JSON 转义形态一并登记

feat(pmicro): 启动打印构建信息，运行侧可从日志确认镜像对应的代码提交与框架版本

- `Program.Init` 日志初始化后打一条 INFO `[Init] 构建信息`：go 版本、服务自身 `vcs.revision` / `vcs.time` / `vcs.modified`（构建上下文带 .git 时 Go 自动内嵌）、`common_util` 依赖版本（来自 go.mod / vendor/modules.txt，replace 时另打目标）；只挑白名单字段，不整体打印 BuildInfo，-ldflags / GOFLAGS 等构建参数不落日志

fix(pmicro): 热更新回调对未注册自定义配置的服务增加守卫（`reloadCfg` 跳过 other 段解析），并补齐 `updateFromJSON` 空入参、启动日志敏感字段脱敏的回归测试

fix(redis-sync/consistent_hash): 修复同步分区静默无主问题（生产事故：metadata_provider 分区 68 无人消费）

- nacos 一致性哈希视图源统一只取 healthy && enable 实例：`OnNow` 由 `SelectAllInstances` 改为 `SelectInstances{HealthyOnly}`，订阅回调同步过滤——非优雅退出的幽灵实例不再占据哈希环导致其名下分区无人执行；metadata id 为空的实例跳过并告警；空实例列表视为注册中心抖动不下发，保留当前环
- consistent_hash 周期性强制重下发分区分配（约 1 分钟一次，无成员变化也重发），通知丢失/消费方状态漂移可在一分钟内自愈；job 通知改为非阻塞发送（缓冲满则丢弃并由重下发补偿），避免消费方停摆冻结整个 change 循环；每约 5 分钟打印环成员与本实例分配快照日志便于对账
- redis-sync 新增与哈希环解耦的无主分区兜底：分区每轮扫描完成后写心跳 key（TTL=Ts*5）；每轮同步结束检查全部分区心跳，缺失即 SETNX 抢占接管代扫一轮，并打 ERROR `partition orphan detected`（用于日志告警）；冷启动有 Ts*5 静默期防误报；被接管分区的兜底扫描周期约为 Ts*5（正常归属恢复后回到 Ts）
- `canSyncPt` 为空的实例不再跳过同步轮次，仍参与无主分区检查

feat(pmicro)!: 探活重构：新增独立探活服务与 readiness 探针（BREAKING CHANGE）

- 移除 `HttpCfg.EnableLiveApi` 字段：代码中引用该字段的调用方需删除对应赋值；yaml 中 `enable_live_api` 配置不再生效
- 主 HTTP 服务无条件注册 `/api/liveness`（兼容原默认开启探活的行为）；`http.host` 为空时不再以默认 `:8080` 启动主 HTTP 服务
- 新增 `http.liveness.host` 配置：配置后启动独立探活服务，提供 `/api/liveness` 与 `/api/readiness`；不配置则不启动
- readiness 语义：`Start()` 完成后置为就绪，`Stop()` 开始时立即置为未就绪，供滚动发布提前摘流
- HTTP/探活服务监听失败改为同步返回错误，不再在 goroutine 内 `log.Fatal`

fix(mysql): 连接池状态巡检周期由 10s 调整为 30s

feat(pmicro): 增加服务就绪和取消就绪接口及相关控制

- 新增HTTP接口支持服务就绪(readiness)和取消就绪(unreadiness)状态管理
- 增加基于Secret的API访问权限控制以保护就绪接口
- 服务停止流程中新增多处日志记录，增强停止过程可观测性
- 定时任务停止时添加日志提示
- Redis连接池统计功能支持配置统计间隔和减少重复打印
- 配置项新增Redis连接池统计间隔参数和探活服务密钥配置
- 探活服务支持可选的密钥认证机制

feat(mysql): 优化连接池状态统计与日志记录

- 新增连接池统计打印间隔配置项 PoolStatsInterval，支持自定义间隔时间
- 改进定时任务调度逻辑，仅在开启统计时创建任务
- 添加连接池状态变更检测，避免重复日志输出
- 引入 mulConnStats 同步映射缓存各连接池状态，提升日志准确性和性能
- 规范连接池状态日志字段及格式，增强可读性

feat(secret): 启动期凭据重试 deadline 化、单次拉取超时，Nacos 账密支持凭据引用，启动前探测 sidecar 就绪

- 重试：base 1s / cap 5s / 全抖动，整体 deadline 默认 20s（`SECRET_RETRY_DEADLINE_SECONDS`，0 = 只尝试一次）；每次尝试另有单次超时 5s 且不超过 deadline 剩余，单次超时返回可重试的 `secret.ErrAttemptTimeout`，一次解析的总耗时以 deadline 为上界。deadline 按凭据名计算：一个服务只引用一个凭据名时，同名凭据第二次起命中缓存；引用多个凭据名时最坏耗时按个数叠加
- csms：客户端 HTTP 超时由 SDK 默认 120s 改为 5s；SDK 调用不接 ctx，`Fetch` 在 ctx 结束时立即返回。awssm：关掉 SDK 自带的 3 次重试（重试统一由组件收口），`AccessDeniedException`、`DecryptionFailure` 归为不可重试；429/限流仍可重试
- Nacos 账密：`NACOS_USERNAME` / `NACOS_PASSWORD` 可写成凭据引用（如 `csms://ns-test/nacos_password`），启动时经组件取明文；空值匿名、其余按明文。密码为明文时打 WARN「Nacos 账密来源=明文」，`NACOS_AUTH_ALLOW_PLAINTEXT=false` 时直接启动失败；账号名明文只记来源
- sidecar 就绪探测：加载配置前轮询 `127.0.0.1:15021/healthz/ready`，上限默认 10s（`STARTUP_PROXY_WAIT_SECONDS`，0 关闭，LOCAL 跳过）；连续 3s 连不上判无 sidecar 放行。**无 istio 注入的 Pod（CronJob、未上 istio 的产品线）每次启动固定多等约 3s，部署模板请设 `STARTUP_PROXY_WAIT_SECONDS=0`**
- 启动耗时：最坏约为 sidecar 等待 10s + 凭据解析 20s（单凭据名）+ Nacos 登录与初始化，贴近未配 startupProbe 时约 40s 的 liveness 红线；引用多个凭据名或调大 deadline 的服务**须配置 startupProbe**
- 日志脱敏：账号类键名（`username` / `user` / `account` 及以 `_` `-` `.` 连接的后缀形态，如 `nacos_username`、`mysql_user`）取出的值不再登记按值脱敏，避免账号名与库名、服务名同串时被大面积打码；其余键与整份凭据照常登记。约定只把秘密写成引用
- 日志脱敏覆盖面补齐：标准库 `log` 包输出改经按值脱敏写到标准错误；Nacos SDK 日志适配器 `Info` 中直接打印到标准输出的两行同样先脱敏。直接 `fmt.Print`、第三方库自建 logger 仍不在覆盖范围
- `UnmarshalJSON`：文本不含已注册 scheme 时直接 `json.Unmarshal`（与改造前行为、开销一致）；含引用时顶层值之后还有内容会报错（对齐 `json.Unmarshal`）；JSON 里 `\/` 转义写法的引用同样识别
- 反序列化错误：没有替换引用时原样返回原错误（保留原文与错误类型，如 `*yaml.TypeError`、`*json.UnmarshalTypeError`）；替换过引用时仍经 `SanitizeDecodeError` 脱敏为新错误
- 行为提醒：值里任意位置出现已注册 scheme 的 `csms://` / `awssm://` 子串即视为引用，普通配置项（文案、链接）含这类子串会让整份配置解析失败——启动期直接退出，热更新期只打 ERROR 并保留旧配置
- 新增常量 `cst.ENV_SECRET_RETRY_DEADLINE_SECONDS`、`cst.ENV_STARTUP_PROXY_WAIT_SECONDS`、`cst.ENV_NACOS_AUTH_ALLOW_PLAINTEXT`；`secret.ErrAttemptTimeout`；`github.com/aws/smithy-go` 由间接依赖转为直接依赖（版本不变）

### v1.2.0-rc.1

fix(aes): 修复AES解密边界检查与填充校验

- 新增decryptCBC函数，避免外部输入触发cipher.CryptBlocks panic
- 校验IV长度等于块大小，密文长度为块大小整数倍
- 完善pkcs7Unpadding填充合法性校验，防止越界和篡改
- AESEncryptCBC改用pkcs7Unpadding替代过时pkcs5UnPadding方法
- 加强加密库安全性，降低非法输入导致panic风险

fix(rsa): 提升RSA密钥安全位数及新增单元测试

- 将RSA密钥默认位数从512提升至2048，增强安全性
- 新增生成密钥对及加解密的完整单元测试覆盖
- 校验非法密钥输入返回错误避免panic

fix(pmicro): 修正微服务重试逻辑中的时延和返回值

- gRPC不可用错误重试前增加100ms等待，避免快速重试压垮下游
- 超时错误处理增加重试前等待时间，提高重试效果
- 调整错误处理逻辑返回值，确保重试机制准确响应

fix(config): 清理配置示例中的敏感信息及无效引用

- 重新设置AliSmsCfg示例字段为空，避免暴露密钥
- 清空MySQL配置的密码示例值，防止泄露密码
- 清理微信配置中app_id及secret示例值，保障安全
- updateFromJSON避免操作nil配置实例，提升稳定性

style(documentation): 完善README文件项目结构与组件介绍

- 补充项目模块目录及说明，方便新用户理解项目布局
- 增加快速开始示例代码，降低使用门槛
- 详细描述限流组件差异及应用场景，指导合理选型
- 新增对象存储接口说明，便于多云存储无缝切换
- 补充customer-config泛型配置用法及注意事项
- 完善数据库和工具包列表，提高文档完整性

fix(log): 移除开发时输出，完善敏感字段脱敏处理

- 删除log/core.go中无用打印，避免输出敏感字段内容
- 对password和secret字段替换为“****”，加强日志安全

chore(gitignore): 添加codegraph相关忽略文件配置

- 新增.codegraph目录和相关规则忽略，保持代码仓库整洁

fix(token): 优化导入顺序与变量声明，提高代码规范

- 调整引入包顺序，分组本地及第三方库
- 添加常用Token相关常量注释说明，加强代码可读性

### v1.1.0-rc.8
feat(redis): 新增待同步队列批量投递接口 PutWaitSyncKeys（含 release/v1.0.0 合并，为 rc.7 超集；该版本号曾短暂发布过仅含本特性的构建，已于同日以本内容覆盖重发，重发前拉取过 rc.8 的环境需清模块缓存并更新 go.sum）

- 注意：`RedisInterface` 接口新增 `PutWaitSyncKeys` 方法，自行实现该接口（而非使用本库实现/mock）的调用方需补齐实现

### v1.1.0-rc.8
feat(redis): 新增待同步队列批量投递接口 PutWaitSyncKeys（含 release/v1.0.0 合并，为 rc.7 超集；该版本号曾短暂发布过仅含本特性的构建，已于同日以本内容覆盖重发，重发前拉取过 rc.8 的环境需清模块缓存并更新 go.sum）

### v1.1.0-rc.7
refactor(util): 分布式令牌桶限流工具重构为接口，方便调用方进行 mock

### v1.1.0-rc.6
feat(util): 添加分布式令牌桶限流工具


### v1.1.0-rc.5

fix(mq): 优化Kafka消费者日志打印

- 注释移除消费开始时的日志输出，减少日志冗余
- 统一消费完成日志的字段命名格式为小写
- 消费完成日志中新增时间戳和消息偏移量信息
- 保留并优化消费耗时及日志级别字段显示

### v1.1.0-rc.4

chore(deps): 更新依赖版本并优化模块引用

- github.com/alicebob/miniredis/v2版本添加并统一引用
- 升级github.com/klauspost/cpuid/v2至v2.2.10版本
- 升级github.com/redis/go-redis/v9至v9.20.1版本
- 升级go.uber.org/atomic至v1.11.0版本
- 移除重复和冗余依赖声明，清理模块文件
- 新增github.com/zeebo/xxh3依赖及对应版本记录

fix(consistent_hash): 说明Init函数无需外部调用

- 注释更新明确Init初始化函数为底层自动初始化
- 移除用户调用该函数的必要性说明

### v1.1.0-rc.3

refactor(mq): 优化Kafka消费处理中的上下文传递和panic日志

- 修改HandlerFuncs接口，简化返回值，去掉无用int返回
- 在processBatch中使用trace.GetTraceCtx获取trace上下文
- 增加panic捕获时的日志记录，明确业务需处理panic
- 修正调用HandlerFuncs时只接收error返回值
- 新增GetCtxFromKafkaHeader函数，从Kafka消息头提取traceId构建上下文
- 增加NewOrAddTraceIdCtx函数，合并多个traceId到上下文中

### v1.1.0-rc.2

feat(redis): 添加Redis连接池状态监控定时任务

- 修改InitRedis函数，新增返回cron.CronJob对象
- 定时任务每15秒记录各实例连接池状态统计信息
- 适配调用方，支持获取定时任务以便注册执行
- 更新相关测试代码，调用InitRedis时接收新增返回值
- 程序启动时将该定时任务纳入调度管理

### v1.1.0-rc.1

@Francis/release/v1.0.0

feat(redis): 优化同步任务写入和版本管理逻辑

- 提取并实现高效的 FNV-1a 哈希函数，避免重复分配提升性能
- 优化 putWaitSync 接口，区分正常写入和重试写入模式
- 扩展 sync_add.lua 脚本支持重试逻辑，使用 GT 保证版本推迟生效
- 引入 versionKeyOf 方法确保 zset 与版本 hash 同 slot，支持集群一致性
- 修正 parseSyncItems 方法，严格类型断言避免异常数据影响解析
- 调整日志及错误处理，避免同步任务注册出错导致死锁
- 增加单元测试覆盖 putWaitSync 不同情景下的行为和边界条件
- 改进同步循环逻辑，重试失败的数据使用 putWaitSyncRetry 推迟执行
- 更新代码注释与流程图，增强代码可读性和维护性

### v1.0.0-rc20260504

**fix(redis/sync): 修复 SyncTaskRun 并发执行问题，分区同步走协程池**

  - cron 任务改为按 syncKey 绑定闭包，新增 runOne 替代全局 SyncTaskRun，
    避免多个 cron 条目同时触发全量同步、SkipIfStillRunning 跨条目去重失效的问题
  - RedisSyncJob 持有 ants.Pool 引用，sync 内按分区 Submit 到协程池并发执行；
    池满或 pool 为 nil 时降级当前 goroutine 同步执行，避免漏分区
  - 抽出 syncPartition 单独承载单分区主循环，去掉 ptFor 标号
  - 补 6 个单测：runOne 三条分支、分区并发耗时断言、池关闭降级、pool=nil 兜底

### v1.0.0-rc20260503

refactor(redis/sync): 使用 Lua + 版本号重构同步流程，加入 cron 抖动

- 新增 sync_add / sync_range / sync_rem 三个 Lua 脚本，将 ZSET 与版本号操作原子化
- range 时通过 HINCRBY 写入版本号，rem 时 HGET 校验版本号；put 时 HDEL 让旧版本失效，
  避免取/删窗口内业务写入被误删
- AddJob 入口对 Ts、QueueNum 做默认值兜底（5s / 10），避免零值导致 cron 非法或 hashMod 除零 panic
- cron 周期增加 ±clamp(Ts/5, 1, 30)s 抖动，避免多副本同秒齐发
- 单分区批量从 20 提升到 50；同步/删除失败后用 now+1 重入，避免本轮被立即再次扫到

### v1.0.0-rc20260502

refactor(redis/sync)!: RedisSyncFunc 增加 error 返回值

- RedisSyncFunc 由 func(ctx, string) 改为 func(ctx, string) error，让业务可显式返回失败而非仅依赖 panic
- safeCall 直接透传 doSync 返回的 error，外层带标签 break 可统一识别业务失败与 panic
- 同步更新单测 mock 实现以匹配新签名

BREAKING CHANGE: 调用方需为 RedisSyncOption.DoSync 增加 error 返回值

### v1.0.0-rc20260501

**fix(redis/sync): 修复同步任务死循环、数据丢失与并发安全问题**

- 单条 doSync 后立即 ZRem，替换原"批量删除"避免并发写入数据丢失
- safeCall 包裹 doSync 拦截 panic，并通过返回 err 触发外层带标签 break，消除毒数据死循环
- syncPool.Submit 失败时补 wg.Done，避免 WaitGroup 永久阻塞
- WaitGroup 改为 SyncTaskRun 局部变量，规避跨周期复用 panic 风险
- AddJob 使用 LoadOrStore 做 syncKey 幂等校验，防止重复注册导致的 goroutine 泄漏
- ServerChange 对未注册 syncKey 跳过处理，避免写入残缺 Job 触发 nil panic
- 增加单分区 maxLoops 兜底与耗时/总数日志，便于线上观测

### v1.0.0-rc20260401

feat(mq): 添加 RabbitMQ 支持并实现消息追踪功能

- 引入 RabbitMQ 客户端依赖包 github.com/rabbitmq/amqp091-go
- 新增 RabbitMqManager 接口定义，包含 Publish 和 PublishWithConfirm 方法
- 实现 Kafka 消息头中添加 trace_id 以支持链路追踪
- 修改 Kafka 生产者在发送消息时注入 trace_id 到消息头部
- 添加 RabbitMQ 配置结构体和相关参数设置
- 更新 Kafka 消费者从消息头中读取 trace_id
- 添加 AWS SDK 依赖用于后续 S3 服务集成
- 优化 Nacos 注册日志输出，增加注册结果反馈

refactor(redis): 移除已弃用的EvalShaOrScript方法

- 从Redis接口中删除EvalShaOrScript方法定义
- 移除Redis实现中的EvalShaOrScript方法及其相关逻辑
- 删除SHA1哈希计算和脚本缓存相关代码
- 清理mocks中对应的模拟方法实现
- 简化Redis facade接口定义

feat(library/gin): 添加自定义参数校验证器接口

- 引入 CValidator 接口用于自定义参数校验
- 在 Validate 函数中集成 CValidator 验证逻辑
- 实现 DoValidate 方法支持自定义校验规则
- 保持原有 Validator 接口兼容性
- 提供 INVALID_PARAM 错误码统一处理机制

feat(redis): 重构Redis日志钩子实现并改进日志控制机制

- 添加debug字段到logHook结构体以支持调试模式控制
- 修改logHook方法接收器为指针类型以支持内部状态访问
- 将WithoutRedisLog函数重命名为WithRedisLog并修改逻辑实现
- 使用结构体类型作为context键值替代字符串类型
- 更新isLogEnabled方法为logHook的方法并优化日志启用判断逻辑
- 调整Redis客户端初始化中的日志钩子配置方式
- 修改测试配置中的Debug默认值为false
- 添加TestRedis_withLog测试函数验证日志功能

refactor(metadata): 重构元数据管理功能

- 移除 CONTEXT_PHONE 相关字段和处理逻辑
- 为常量添加 string 类型声明
- 添加 Other 字段支持自定义元数据存储
- 实现 ToMicroMetadata 和 FromMicroMetadata 方法支持微服务元数据转换
- 添加自定义元数据前缀 CONTEXT_CUSTOM_PREFIX 支持
- 优化 SetCtxData 函数避免不必要的 context 重建
- 更新文档注释和完善函数说明
- 移除 parsePhone 相关逻辑和依赖

**feat(http): 添加HTTP服务器配置和优化中间件**

- 添加HttpServerCfg配置结构体，支持读写超时、空闲超时和请求头大小限制
- 在生产环境自动设置Gin为Release模式
- 将IP请求头配置提取为全局变量，统一管理CloudFlare相关头信息
- 移除日志中间件中的响应体捕获功能以提升性能
- 注释掉IP获取失败时的日志警告，避免不必要的日志输出

### v1.0.0-202601-6

**feat(kafka): 扩展生产者消息类型兼容性**

- 支持字符串、字节数组和 JSON 原始消息类型的直接传递
- 使用类型断言替代简单的字符串检查
- 添加了对 json.RawMessage 类型的原生支持
- 保持了对其他类型的 JSON 序列化回退机制
- 优化了消息转换的性能和类型安全性

**feat(tracer): 支持多种trace id头部字段**

- 添加了对 X-Trace-Id、Trace-Id 和 Traceid 头部字段的支持
- 实现了从请求头部获取现有trace id的逻辑
- 当请求中没有trace id时生成新的uuid作为trace id
- 将获取到的trace id设置到上下文中供后续使用
- 统一了trace id的管理和传递机制

### v1.0.0-202601-5

**feat(auth): 添加 JWT 认证支持**

- 新增 grpc_jwt.go 文件实现 JWT token 解析功能
- 添加 AuthGrpcClaims 结构体用于存储认证信息
- 实现 ParseGrpcJwtToken 函数解析和验证 JWT token
- 在 auth_server.go 中添加 JWT 认证类型处理逻辑
- 实现 jwtCheck 函数支持多密钥轮询验证
- 修改 client wrapper 支持 JWT 认证类型
- 添加单元测试验证 JWT 解析功能

### v1.0.0-202601-4

**refactor(kafka): 优化Confluent Kafka消息推送逻辑与测试代码**

- 替换旧InitKafka调用为InitConfluentKafkaManager以统一管理
- pushMessageWithProducer方法新增参数校验和简化逻辑
- 抽离消息构造到getKafkaMsg方法，支持string类型特殊处理
- 处理本地环境topic自动添加后缀避免误消费
- 新增msgMarshal函数及单元测试覆盖多种数据类型序列化
- 测试代码中替换Push调用为mgr.Push以使用管理器实例
- 日志统计改为调用mgr获取生产统计数据接口

**refactor(gorm): 替换旧版gorm依赖为官方gorm模块**

- 统一替换所有代码中的 github.com/coderyw/gorm 为 gorm.io/gorm
- 替换相关驱动和插件依赖地址为 gorm.io/driver/mysql、gorm.io/driver/postgres 及 gorm.io/plugin/dbresolver
- 更新 go.mod 和 go.sum，移除旧版gorm相关依赖，添加官方gorm最新版本依赖
- 修正相关包引用导入路径，确保代码兼容官方gorm库
- 注释和删除部分旧版gorm相关代码实现及调用
- 新增和完善部分测试代码，使用官方gorm接口进行数据库操作验证
- 修正部分日志和错误处理代码中gorm包的导入和使用

**perf(log): 移除HTTP中间件中的请求响应体日志记录**

- 注释掉最大body长度限制配置
- 移除请求body读取和打印功能
- 移除响应body读取和打印功能
- 减少日志输出大小提升性能
- 避免敏感数据在日志中暴露

**feat(log): 添加不记录响应日志的功能并优化日志中间件**

- 新增 SetResponseWithNoResponseLog 函数用于设置响应值且不打印响应日志
- 优化 Tiny 结构体的 MarshalLogObject 方法，正确输出字段值
- 修改日志中间件移除原始查询参数记录
- 为 NsHttpResp 结构体实现 MarshalLogObject 接口支持结构化日志输出

### v1.0.0-202601-3

refactor(log): 统一日志字段类型提升日志性能

- 将所有zap.Any字段替换为具体类型如zap.String、zap.Int、zap.Int64等
- 修复consistent_hash模块中nodeIdx和current字段的日志类型
- 修复cron模块中name字段的日志类型
- 修复redis模块中多个字段的日志类型包括syncKey、key、values等
- 修复mysql模块中连接池统计信息的日志类型
- 修复http模块中code、body、Duration等字段的日志类型
- 修复gorm日志中Duration和SQL相关字段的日志类型
- 优化日志序列化性能，避免不必要的类型转换

refactor(redis): convert Redis methods to pointer receivers

- 将 Redis 结构体的方法接收者统一改为指针类型以统一操作和避免拷贝
- 修正cache测试中GetRedis的类型断言为指针类型
- 优化锁相关方法，改为指针接收者，提升性能与一致性
- 统一redis.go中所有Redis方法的接收者为指针类型，改进资源管理与调用效率
- 保持原有功能不变，提升代码可维护性和扩展性

### v1.0.0-202601-2

merge from main

- 新增限流方法
- 增加对cron自动初始化运行

### v1.0.0-202601-1

feat(log): 添加用户对象日志脱敏功能和元数据脱敏处理

- 实现User结构体的MarshalLogObject方法，对手机号、密码等敏感信息进行脱敏
- 添加Mask工具函数，支持自定义字符串脱敏规则
- 在日志测试中验证用户信息脱敏功能
- 增加RPC_METADATA_X_AUTH_SECRET_MICRO常量用于认证密钥处理
- 实现metadata脱敏功能，对X-Auth-Secret字段进行脱敏处理
- 创建util.Mask工具函数提供通用字符串脱敏能力

### v1.0.0-202512171017

feat(config): 支持自定义配置变更回调函数

- 新增 changeHandler 字段用于处理配置变更
- 修改 initCustomerConfig 函数签名，支持传入变更回调
- 在配置更新时调用变更回调函数
- 更新 server_config 和 options 模块以支持新功能
- 修改 CHANGELOG 版本号并更新相关描述


### v1.0.0-202512161435

fix(config): 修复客户配置默认值存储问题

- 移除了初始化时对 defaultValue 的重复赋值
- 在统一存储非指针值逻辑后正确设置 defaultValue
- 添加了针对指针和非指针默认值的测试用例
- 增加了 GetCustomerConfig 的单元测试覆盖
- 引入 reflect 包以支持深度比较测试断言

### v1.0.0-202512151812

feat(config): 新增自定义配置管理功能

- 新增 customer_config 包用于统一管理自定义配置
- 实现泛型方式获取自定义配置的方法 GetCustomerConfig[T]()
- 在 pmicro 配置模块中集成自定义配置初始化和更新逻辑
- 支持通过 JSON 格式动态更新自定义配置
- 添加相关单元测试验证配置的正确性
- 修改服务配置选项，将自定义配置解析替换为默认配置注入
- 移除原有的自定义配置反序列化函数参数，改为内部统一处理

### v1.0.0-202512121612

refactor(config): 优化自定义配置解析逻辑并更新选项命名

- 移除 OtherCfg 和 OtherUnmarshal 选项，统一使用 CustomCfgUnmarshalFunc
- 修改 nacosOtherConfigUnmarshal 方法签名，仅保留自定义解析函数参数
- 更新 listenNacosConfig 和 reloadCfg 方法以适配新的配置解析方式
- 重命名 BeforeFunc 为 BeforeStartFunc 并更新其调用时机
- 删除已废弃的 InItFunc 和 MetaData 选项及相关代码
- 清理冗余的配置初始化注释代码块
- 更新 CHANGELOG 中的版本号格式
- 在 README 中新增完整的服务启动示例代码

### v1.0.0-202512121358

feat(config): 新增服务配置管理模块

- 实现了从etcd和nacos配置中心加载服务配置的功能
- 支持配置文件方式加载服务配置
- 添加了配置中心配置变更监听机制
- 实现了RPC服务端和客户端认证信息匹配逻辑
- 提供了配置获取接口，支持获取自定义配置和其他配置信息
- 增加了配置校验机制和默认值设置
- 实现了认证密钥掩码显示功能
- 优化了配置初始化流程，支持多环境配置管理

