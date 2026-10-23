# db/mysql：MySQL / PostgreSQL

> 返回 [README](../../README.md)。

`library/db/mysql` 基于 GORM 封装多实例数据库，支持 `mysql` / `postgresql` / `opengauss` 驱动，内置分表、雪花主键与连接池统计。

> ⚠️ **从库（`replicas`）当前不可用**：从库连接实际使用的是主库的连接信息，从库的 host/port/user 等配置都被忽略，`max_life_time` / `max_idle_time` 还被当作纳秒处理。见 `library/db/mysql/mysql.go` 的 `initReplica`，修复前不要依赖读写分离。pmicro 启动时按 `storage.mysql` 配置自动初始化。

## 配置

配置结构为 `config.DbConfig`：`default` 指定默认实例，`instance` 下按实例名配置连接、分表、雪花 ID 与从库。完整带注释示例见 [configuration.md](../configuration.md) 的 `storage.mysql` 段。

```yaml
storage:
  mysql:
    default: 'core'
    pool_stats_mode: 1
    pool_stats_interval: 60
    instance:
      core:
        host: '<host>'
        port: '3306'
        user: '<user>'
        password: '<password>'
        database: '<database>'
        driver: 'mysql'
        open: 100
        idle: 20
        max_life_time: 3600
```

## 使用

```go
u := new(User)

// 默认实例
mysql.DB(ctx).Model(User{}).Select("id").Where("id = ?", 123).First(u)

// 指定实例
mysql.DB(ctx, "game").Model(User{}).Select("id").Where("id = ?", 123).First(u)

// 事务
err := mysql.DB(ctx).Transaction(func(tx *gorm.DB) error {
	return tx.Model(User{}).Where("id = ?", 123).First(u).Error
})
```

`DB(ctx)` 会根据 ctx 中的租户信息自动切换租户库名（规则见 [multi-tenant.md](../multi-tenant.md)），所以**务必传入请求链路上的 ctx**，不要用 `context.Background()` 代替。

## 雪花主键

在实例的 `snowflake_gen` 中配置后，创建记录时由 `snowflake:before_create` 回调自动填充主键。雪花 worker 注册依赖 Redis，Redis key 前缀受环境变量 `DISABLE_EONID_HASHTAG` 控制（NS 部署必须为 `1`），见 [environment.md](../environment.md#可选功能)。

## 连接池统计

`InitConnect` 返回的定时任务按 `pool_stats_interval` 周期采集各实例的连接池状态，输出方式由 `pool_stats_mode` 控制，取值见 [configuration.md · 统计输出模式](../configuration.md#统计输出模式pool_stats_mode--stats_mode)。

上报指标（gauge，tag 为 `instance=<实例名>`）：`mysql_pool_max_open_connections` / `mysql_pool_in_use` / `mysql_pool_idle` / `mysql_pool_open_connections` / `mysql_pool_wait_count` / `mysql_pool_wait_duration_ms` / `mysql_pool_max_idle_closed` / `mysql_pool_max_idle_time_closed` / `mysql_pool_max_lifetime_closed`。

## 子包

| 子包 | 职责 |
|-|-|
| `db/mysql/callback` | GORM callback 扩展：链路、表名、租户库名处理 |
| `db/mysql/sharding` | 分表路由：规则解析、SQL 改写、主键生成，详见 [sharding/README.md](../../library/db/mysql/sharding/README.md) |
| `db/mysql/sharding/sqlparse` | 分表场景使用的 SQL 解析器，详见 [sqlparse/README.md](../../library/db/mysql/sharding/sqlparse/README.md) |
