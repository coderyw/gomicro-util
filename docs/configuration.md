# 服务 YAML 配置参考

> 返回 [README](../README.md)。环境变量见 [environment.md](environment.md)。

服务配置以 YAML 形式提供：本地运行（`ENVIRON=server-local` 且未配置配置中心地址）读取 `option.ConfigFile(...)` 指定的文件；非本地从 Nacos 读取 `NACOS_NAMESPACE_ID / DEFAULT_GROUP / <服务名>.yaml`。结构体定义在 `model/config`，**字段以代码为准**，本文是带注释的完整示例。

各段按需配置，未使用的组件可整段省略。示例中的地址、账号、密钥均为占位符。

**密码、密钥等敏感值建议写成凭据引用**（如 `password: 'csms://<凭据名>/<键名>'`），启动时自动从华为云 CSMS / AWS Secrets Manager 取回，详见 [secret.md](components/secret.md)。注意普通配置项的值不能包含 `csms://`、`awssm://` 子串。

## 完整示例

```yaml
# 服务名
server_name: {serviceName}

registry:
  # 支持 etcd nacos
  type: '{registryCenter}'
  hosts:
    - {registry host}
  account: '{注册中心账号}'
  pwd: '{密码}'
  # 如果是 nacos，这个是命名空间 id；如果是 istio，这个是 namespace 名称。
  # Nacos 模式下 NamespaceId 必须通过环境变量 NACOS_NAMESPACE_ID 指定，yaml 中的 name_id 会被环境变量覆盖；
  # 未设置 NACOS_NAMESPACE_ID 服务将启动失败
  name_id: ''
  # 是否开启istio模式。开启后只会进行服务注册到 type 的注册中心。服务发现调用基于 istio 代理
  enable_istio: true

# grpc服务相关配置
rpc_server:
  listen_on: ":9804"
  server:
    auths:
      #      // 认证方式 jwt | basic (basic标识明文密码认证)。认证方式为空时候，表示不开启认证
      - auth_type: 'basic'
        #        // 开启认证的服务端端点。为空时不会对任何端点认证。
        #        // 只支持精确匹配，以及以 * 结尾的前缀匹配（多个前缀命中时取最长的），不支持正则
        #        // 假定当前服务为 ca-delay-center
        #        //
        #        // ca-delay-center.DelayCenterService.PushDelayData 表示对 delayCenter.DelayCenterService.PushDelayData进行认证
        #        //
        #        // ca-delay-center.* 表示对 当前服务的多少有 service 的 method 都进行认证
        #        //
        #        // ca-delay-center.DelayCenterService.* 表示对 delayCenter.DelayCenterService的所有method进行认证
        auth_endpoint:
          - 'ca-delay-center.DelayCenterService.*'
        #        // 认证密钥 支持多个密钥
        auth_secret:
          - '<auth_secret>'
        #        // 验证版本 可能对应着不同类型认证方式版本，如jwt的版本号
        auth_version: '1.0.0'
        force_auth: true
      - auth_type: 'basic'
        auth_endpoint:
          - 'ca-delay-center.TestService.*'
        auth_secret:
          - '<auth_secret>'
        auth_version: '1.0.0'
        force_auth: false
  client:
    # 请求失败重试。总尝试次数（含首次调用），默认 3，-1 表示不开启重试。
    # 只对 gRPC Unavailable（连接不通，如上游发布期间）及错误信息含 "invalid connection" 的错误重试，业务错误不重试
    attempts: 3
    # 重试间隔，单位 ms，默认 500
    attempt_interval_ms: 500
    # 客户端连接池连接最大存活时间，单位 s。默认 3600（1 小时），-1 表示不过期（坏连接由状态检查剔除）
    pool_ttl_s: 3600
    auths:
      - auth_type: 'basic'
        auth_endpoint:
          - 'ca-delay-center.DelayCenterService.*'
        auth_secret: '<auth_secret>'
        auth_version: '1.0.0'

# http 配置
http:
  host: ":8080"
  # 普罗米修斯,不配置默认不开启性能监控
  enable_prometheus: false
  # 跨域 header 头
  cors_headers:
    - ''
  # 请求日志打印的 header 配置（打印在 HTTP Handle start 日志的 headers 字段）
  # 不配置则不打印 header；配置 * 打印所有 header；否则配置需要打印的 header 名，多个用 , 拼接
  # log_headers: 'X-Request-Id,User-Agent'
  log_headers: ''
  # jwt密钥
  jwt_secret: ''
  # jwt 过期时间，单位 s，默认 50400（14 小时）。必须写成数字，不能写 14*3600
  jwt_exp_time: 50400
  # 运维探活服务（独立端口，⚠️禁止暴露公网）。不配置 host 则不启动
  # 提供：GET /api/liveness 探活、GET /api/readiness 就绪查询、
  #      POST /api/set/unreadiness 摘流（下线前取消就绪）、POST /api/set/readiness 恢复就绪
  liveness:
    host: ':8085'
    # /api/set/* 两个写接口的密钥（请求需带 Authorization 头等于该值）；不配置则不鉴权
    secret: ''

log:
  # DebugLevel  - 1
  # InfoLevel 0
  # WarnLevel 1
  # ErrorLevel 2
  # DPanicLevel 3
  # PanicLevel  4
  # FatalLevel 5
  # 会打印Level及以上级别的日志
  level: 0
  # 注意命名与行为相反：disableStacktrace 为 true 时才「开启」自动堆栈，为 false（默认）时不打印自动堆栈
  disableStacktrace: false
  # 仅在 disableStacktrace: true 时生效：该级别及以上的日志附带堆栈，不配置时为 3（DPanic）
  stacktrace_level: 2

storage:
  mysql:
    default: 'core'
    # 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报（推到 pmicro/metrics，metrics 未开启则丢弃），3 日志和上报都开启。默认 1
    pool_stats_mode: 1
    # 连接池统计（打印/上报）间隔，单位 s。默认 60，-1 表示不开启
    pool_stats_interval: 60
    instance:
      core:
        # 连接地址
        host: 'mysql.example.com'
        # 连接端口
        port: '3306'
        # 用户名
        user: datamanager
        # 密码
        password: 'password'
        # 默认数据库名
        database: 'db_game_color'
        # 默认 false。SQL 日志始终会打印；debug: true 反而会把 GORM 自身日志降到只输出 Error
        debug: false
        # 数据库驱动，支持 mysql/postgresql/opengauss 默认 msyql
        driver: 'mysql'
        # 最大连接数，默认 200；-1 表示不限制（此时 idle 也会被压成 -1，即不保留空闲连接）
        open: 100
        # 最大空闲连接数，默认 20，且不超过 open
        idle: 20
        # 连接最大存活时间（单位 s，对应 SetConnMaxLifetime），默认 3600
        max_life_time: 3600
        # 连接最大空闲时间（单位 s，对应 SetConnMaxIdleTime），默认 300，且不超过 max_life_time
        max_idle_time: 300
        # ssl配置 
        ssl_config:
          enable: true
          #  证书位置 如果是本地运行，证书需要放在本地。如果是云上运行，推荐证书放在k8s 的K8s Secret
          #  kubectl create secret generic rds-ca-secret --from-file=global-bundle.pem //创建证书
          #  或者 yaml 方式
          #apiVersion: v1
          #kind: Secret
          #metadata:
          #  name: rds-ca-secret
          #type: Opaque
          #data:
          #  global-bundle.pem: <base64内容>
          # 然后 deployment 挂载证书
          #apiVersion: apps/v1
          #kind: Deployment
          #metadata:
          #  name: my-app
          #spec:
          #  template:
          #    spec:
          #      containers:
          #        - name: app
          #          image: my-app:latest
          #          volumeMounts:
          #            - name: rds-ca
          #              mountPath: /etc/ssl/rds
          #              readOnly: true
          #      volumes:
          #        - name: rds-ca
          #          secret:
          #            secretName: rds-ca-secret
          path: '/etc/ssl/global-bundle.pem'
        # 如果需要配置分表，则需要下面的配置
        sharding:
          - table_name: 'b_color_order' # 分表名
            sharding_key: 'sys_user_id' # 分表键
            sharding_num: 100 # 分表数量
        # 雪花id生产规则配置（当前是全局配置）
        snowflake_gen:
          # 是否开启默认数据中心（没有配置数据中心的表都使用它），配置了 snowflake_gen 时默认 true；整个进程只初始化一次默认中心
          enable_default: true
          # 雪花中心类型：0 以表名为中心；1 以库名为中心（取 model TableName() 返回的 `库.表` 中的库名，TableName 必须带库名，否则创建报错）；2 以服务名为中心
          snowflake_type: 0
          snowflakes:
            - worker_id_bit: 10
              data_center: 'ca_user'
        # 从库配置
        # ⚠️ 当前实现有缺陷：从库连接实际使用的是主库的连接信息（host/port/user/password/database/ssl_config/driver 均被忽略），
        # 且从库的 max_life_time / max_idle_time 被当作纳秒处理。修复前不要依赖读写分离
        replicas:
          - host: 'mysql-replica.example.com'
            port: '3306'
            user: datamanager1
            password: 'password'
            database: 'db_game_color'
            debug: true
            driver: 'mysql'
            open: 100
            idle: 20
            max_life_time: 3600
            ssl_config:
              enable: true
              path: '/etc/ssl/global-bundle.pem'
            sharding:
              - table_name: 'b_color_order' # 分表名
                sharding_key: 'sys_user_id' # 分表键
                sharding_num: 100 # 分表数量
      # 游戏库配置
      game:
        host: 'mysql.example.com'
        port: '3306'
        user: datamanager
        password: 'password'
        database: 'db_game_color'
        debug: true
        driver: 'mysql'
        open: 100
        idle: 20
        max_life_time: 3600
        ssl_config:
          enable: true
          path: '/etc/ssl/global-bundle.pem'
        sharding:
          - table_name: 'b_color_order' # 分表名
            sharding_key: 'sys_user_id' # 分表键
            sharding_num: 100 # 分表数量
        snowflake_gen:
          enable_default: true
          snowflake_type: 0
          snowflakes:
            - worker_id_bit: 10
              data_center: 'ca_user'
        replicas:
          - host: 'mysql-replica.example.com'
            port: '3306'
            user: datamanager1
            password: 'password'
            database: 'db_game_color'
            debug: true
            driver: 'mysql'
            open: 100
            idle: 20
            max_life_time: 3600
            ssl_config:
              enable: true
              path: '/etc/ssl/global-bundle.pem'
            sharding:
              - table_name: 'b_color_order' # 分表名
                sharding_key: 'sys_user_id' # 分表键
                sharding_num: 100 # 分表数量
  redis:
    # 是否开启 redis 配置
    enable: true
    # 默认 redis 实例
    default: 'core'
    # 连接池统计（打印/上报）间隔，单位 s。默认 15，-1 表示不开启；日志在状态无变化时不重复打印
    pool_stats_interval: 15
    # 连接池统计模式：-1 不打印不上报，0 默认（等同1），1 仅日志打印，2 仅指标上报（推到 pmicro/metrics，metrics 未开启则丢弃），3 日志和上报都开启。默认 1
    pool_stats_mode: 1
    # 实例列表
    instance:
      core: # core实例配置
        # 实例是否开启
        enable: true
        # core实例连接地址
        host:
          - 'redis.example.com:6379'
        # core实例密码
        password: ''
        # core实例数据库
        db: 0
        # core实例连接池大小
        pool_size: 100
        # core实例最大空闲连接数
        max_idle_conns: 20
        # 默认 false。true 时打印每条命令的日志；false 时只有用 WithRedisLog 标记过的 ctx 才打印
        debug: false
        # Redis 为集群模式且没有 proxy 时开启。开启后强制使用 TLS（InsecureSkipVerify），且 db 配置无效
        is_cluster: false
      activity: # activity实例配置
        # 实例是否开启
        enable: true
        # 实例连接地址
        host:
          - 'redis.example.com:6379'
        # 实例密码
        password: ''
        # core实例数据库
        db: 0
        # 实例连接池大小
        pool_size: 100
        # 实例最大空闲连接数
        max_idle_conns: 20
        # 默认 false。true 时打印每条命令的日志；false 时只有用 WithRedisLog 标记过的 ctx 才打印
        debug: false
        # Redis 为集群模式且没有 proxy 时开启。开启后强制使用 TLS（InsecureSkipVerify），且 db 配置无效
        is_cluster: false
  object_storage: # 对象存储配置
    aws_s3_config:
      s3_core:
        region: 'ap-southeast-1'
        endpoint: 'https://s3.ap-southeast-1.amazonaws.com'
        access_key_id: '<access_key_id>'
        secret_access_key: '<secret_access_key>'
        buckets:
          - alias: 'dev-ok-res'
            name: 'dev-ok-res'
            is_default: true
            public_base_url: 'https://s3.ap-southeast-1.amazonaws.com'

# kafka配置示例
kafka:
  # kafka集群配置
  hosts:
    core:
      host:
        - "localhost:9092"
      username: "user1"
      pwd: "password1"
    core1:
      host:
        - "localhost:9092"
      username: "user1"
      pwd: "password1"
  # 默认的推送者，如果配置了多个，可以使用此参数指定一个默认的
  # 如果没有配置默认推送者，则使用第一个配置的推送者
  # 如果配置了，但是配置找不到则会返回错误
  default_producer: "core"
  # 生产者相关配置。支持多个集群和多个有差异的生产者配置
  producer:
    - name: "core"
      # 推送最大重试次数 默认 3 次
      max_retry_times: 3
      # 是一个用于 Kafka 生产者请求的类型，表示生产者在收到响应前需要等待的副本确认数量。其定义如下
      # 0= 不等待任何响应，仅依赖 TCP ACK。
      # 1= 等待本地副本写入成功。
      # -1= 等待所有同步副本写入成功，最小同步副本数由 broker 配置 min.insync.replicas 决定。
      # 默认 -1
      required_acks: -1
      # 待发送队列（含未收到 ack 的重试消息）总字节上限，单位 KB，对应 librdkafka queue.buffering.max.kbytes。
      # 该内存在 C 侧、pprof 不可见；librdkafka 默认 1048576(1GB)，本库默认收敛为 65536(64MB)。
      # 打满后 Push 返回 Queue full 错误（快速失败），业务侧需有错误处理/告警
      queue_buffering_max_kbytes: 65536
      # librdkafka 统计间隔（ms），对应 statistics.interval.ms。
      # 按间隔采集 C 侧发送队列内存（producer_queue_msg_cnt/msg_size），输出方式由 stats_mode 控制。
      # 默认 60000；配置为负数时关闭
      statistics_interval_ms: 60000
      # librdkafka 统计输出模式：-1 不打印不上报（不开启 statistics 回调），0 默认（等同1），1 仅日志打印，2 仅指标上报（推到 pmicro/metrics，metrics 未开启则丢弃），3 日志和上报都开启。默认 1
      stats_mode: 1
  #消费者相关配置。支持多个集群
  consumer:
    - name: "core1"
      groups:
        - group_id: "group1"
          # 批量消费最大数量, 默认100
          max_mul_num: 100
          # 批量消费超时时间，默认 100ms
          max_mul_timeout_ms: 100
          # 批量消费超时类型（不满 max_mul_num 时何时触发消费）：
          # 0 / 2 = 固定周期：每隔 max_mul_timeout_ms 把已攒到的消息交给 handler
          # 1 = 空闲超时：每收到一条消息重新计时，距上一条消息 max_mul_timeout_ms 内没有新消息才触发
          # 不配置时为 0，行为与 2 相同
          mul_timeout_type: 2
          # 当消费组没有提交过偏移量时，-1 从最新开始消费，-2从最旧开始消费，默认 -1
          offset_initial: -1
          # 对应 librdkafka max.poll.interval.ms：两次拉取之间允许的最长处理时间，超过则消费者被踢出消费组并触发再均衡。
          # 默认 30000
          # 请注意，此值必须在代理配置中由 `group.min.session.timeout.ms` 和 `group.max.session.timeout.ms` 指定的允许范围内。
          # 华为云 kafka 配置的是 6000 ～ 1800000
          # 单位 ms 注意 `MaxProcessingTimeMill` must be >= `SessionTimeoutMs`
          max_processing_time_mill: 6000
          # 此超时时间用于在使用 Kafka 的组管理功能时检测客户端故障。客户端会定期发送心跳包以向代理表明其存活状态。
          # 如果在会话超时时间到期之前代理未收到任何心跳包，则会将此客户端从组中移除并启动重新平衡。
          # 请注意，此值必须在代理配置中由 `group.min.session.timeout.ms` 和 `group.max.session.timeout.ms` 指定的允许范围内。
          # 另请注意，当 `group.protocol` 设置为“consumer”时，不支持此客户端配置group.protocol。
          # 在这种情况下，会话超时时间由代理配置控制group.consumer.session.timeout.ms。
          # 默认 30000（框架总会下发该值）
          # 华为云 kafka 配置的是 6000 ～ 1800000
          session_timeout_ms: 6000
          # 偏移量提交策略，默认 0：
          # 0 = 消费完成后记录偏移量，由定时任务按 auto_commit_interval_ms 批量提交
          # 1 = 交给 librdkafka 自动提交。⚠️ 当前实现未存储 offset（enable.auto.offset.store=false），实际不会提交，请勿使用
          # 2 = 每条（批量时为每批最后一条）消费完成后同步提交
          commit_type: 0
          # 自动提交间隔时间（ms）默认 3000ms
          auto_commit_interval_ms: 3000
          # ---- C 侧（librdkafka）内存与再均衡相关配置 ----
          # 单分区 C 侧预取条数水位，对应 librdkafka queued.min.messages（按 topic+partition 生效）。
          # 本地队列消息数低于该值时 fetcher 持续向 broker 补齐；该内存在 C 侧、pprof 不可见。
          # librdkafka 默认 100000（约 50MB/分区），本库默认收敛为 2000（单批 max_mul_num 默认 100，2000 已充分流水线化）。
          # 消费积压时该水位直接决定客户端内存占用，堆积本身存放在 broker、不占客户端内存
          queued_min_messages: 2000
          # C 侧预取队列字节上限，单位 KB，对应 librdkafka queued.max.messages.kbytes，与 queued_min_messages 先到者生效。
          # librdkafka 默认 65536(64MB)，本库默认收敛为 8192(8MB)，作为大消息场景的内存硬上限
          queued_max_messages_kbytes: 8192
          # 分区分配策略，对应 librdkafka partition.assignment.strategy。
          # 默认空 = 沿用 librdkafka 默认（range,roundrobin 的 eager 协议：任一成员变动全组重新分配）。
          # 大消费组（成员数多）建议切 cooperative-sticky（增量式再均衡），但要求全组成员协议兼容，
          # 滚动发布需按两步升级：先配 "cooperative-sticky,range" 全量发布一轮，再改 "cooperative-sticky" 发布第二轮。
          # eager 与 cooperative 策略不可混配
          partition_assignment_strategy: ""
          # librdkafka 统计间隔（ms），对应 statistics.interval.ms。
          # 按间隔采集 C 侧预取队列内存（consumer_fetchq_cnt/fetchq_size），输出方式由 stats_mode 控制。
          # 默认 60000；配置为负数时关闭
          statistics_interval_ms: 60000
          # librdkafka 统计输出模式：-1 不打印不上报（不开启 statistics 回调），0 默认（等同1），1 仅日志打印，2 仅指标上报（推到 pmicro/metrics，metrics 未开启则丢弃），3 日志和上报都开启。默认 1
          stats_mode: 1
          # 每分区 Go 侧消息通道容量（拉取协程 -> 分区消费协程的缓冲），满则反压暂停拉取。
          # 实际取 topic_partition_channel_size（默认 1024）；只有它 <=0 时才用 msg_chan_cap，而它总会被初始化为 1024，
          # 因此当前 msg_chan_cap 实际不生效，要调整容量请配置 topic_partition_channel_size
          topic_partition_channel_size: 1024

# lark通知相关配置
lark_cfg:
  # 群组必须写在 groups 下，key 即 SendLarkMessage 的 groupId
  groups:
    group_1:
      webhook: https://open.feishu.cn/open-apis/bot/v2/hook/<token>
      sign: <sign_secret>
      # 可选，false 时该群组的消息静默不发送
      enabled: true
    group_2:
      webhook: https://open.feishu.cn/open-apis/bot/v2/hook/<token>
      sign: <sign_secret>

# 监控指标上报配置（可选，不配则不上报）
metrics:
  # 上报总开关，默认 false。开启后框架定时把指标推送到 Truewatch DataWay
  enable: true
  # 关闭业务自定义打点（metrics.Client）。默认 false 即开启采集；业务侧高基数误用时的运维急停开关
  disable_business: false
  # 上报间隔，单位 s，默认 15
  interval_s: 15
  truewatch:
    # DataWay 地址
    dataway_url: https://xxx.dataway.truewatch.com
    # 工作空间 token
    workspace_token: tkn_xxx

# 自定义配置
other:

```

## 分段说明

| 配置段 | 结构体 | 详细说明 |
|-|-|-|
| `registry` | `config.RegistryCfg` | [pmicro](components/pmicro.md) |
| `rpc_server` | `config.RpcServerCfg` | [pmicro · RPC 客户端行为](components/pmicro.md#rpc-客户端行为) |
| `http` | `config.HttpCfg` | [pmicro · 探活与就绪](components/pmicro.md#探活与就绪liveness-server) |
| `log` | `config.LogCfg` | [observability · log](components/observability.md#log) |
| `storage.mysql` | `config.DbConfig` | [mysql](components/mysql.md) |
| `storage.redis` | `config.RedisConfig` | [redis](components/redis.md) |
| `storage.object_storage` | `config.ObjectStorage` | [component · 对象存储](components/component.md#对象存储) |
| `kafka` | `config.KafkaCfg` | [mq · Kafka](components/mq.md#kafka) |
| `lark_cfg` | `config.LarkCfg` | [component · lark-message](components/component.md#lark-message) |
| `metrics` | `config.MetricsCfg` | [observability · metrics](components/observability.md#metrics监控指标采集与上报) |
| `other` | 业务自定义 | [config-center · customer-config](components/config-center.md#customer-config) |

## 统计输出模式（`pool_stats_mode` / `stats_mode`）

MySQL、Redis 连接池与 Kafka librdkafka 统计使用同一套取值：

| 值 | 行为 |
|-|-|
| -1 | 不统计：不打印日志也不上报指标 |
| 0 | 默认值，等同 1 |
| 1 | 仅日志打印（默认） |
| 2 | 仅指标上报，推到 `pmicro/metrics`；metrics 未开启时直接丢弃 |
| 3 | 日志打印和指标上报都开启 |
