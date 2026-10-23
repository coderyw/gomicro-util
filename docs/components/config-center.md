# 配置中心与自定义配置

> 返回 [README](../../README.md)。服务主配置的加载流程见 [configuration.md](../configuration.md)。

## site-var

etcd 和 nacos 配置读写与监听。

- **etcd**：`SyncETCDConfig()` 初始化（地址取环境变量 `ETCD_URLS`）；`GetEtcdCfgData(key, &data)` 读配置：etcd 中的值必须是 `{"config_name", "desc", "server", "data": {...}}` 结构（`config.EtcdConfig`），只把 `data` 字段反序列化到 `data`；`Put`/`Get`/`GetList`/`Delete` 读写、`KeepAlive` 租约续期、`Lock`/`LockAlways` 分布式锁都是 `*Etcd` 的方法，需先通过 `etcd.GetDefaultEtcdService()` 获取实例。key 约定统一为 `/{env}/config/...`（见 `etcd/key.go` 的 `GetMysqlConfigKey` 等生成函数）。
- **nacos**：`SyncNacos(cfg)` 初始化；`GetNacosCfgData(dataId, group, &data)` 拉取配置（dataId 以 `.yaml` 结尾走 YAML 解析，否则 JSON）；`ConfigListen[T](dataId, group, onChange)` 泛型监听配置变更（**T 必须是指针类型**，否则 panic）：

```go
_ = nacos.ConfigListen[*MyConf]("app.yaml", "DEFAULT_GROUP", func(c *MyConf) {
  atomicConf.Store(c)
})
```

## customer-config

以泛型方式读取服务自定义配置段（YAML 中的 `other` 块）。必须先通过 `option.CustomDefaultCfg(cfg, changeFunc)` 注册默认值，否则调用 `GetCustomerConfig` 会 panic。

- `other` 段会先转成 JSON 再反序列化到你的结构体，所以**字段必须写 `json` tag**，只写 `yaml` tag 时带下划线的字段读不到
- `changeFunc` 在**启动加载时也会调用一次**，之后仅在 Nacos 配置变更时调用；回调参数是结构体值（非指针）
- 热更新只在 Nacos 配置源下生效

- 适用场景：
  - 业务服务有自己独立的配置结构体，但仍想复用 common util 的统一启动流程
  - 需要在公共组件初始化后，以强类型方式读取业务侧配置

- 使用方式：

```go
type CustomerCfg struct {
  FeatureSwitch bool `json:"feature_switch"`
}

cfg := customer_config.GetCustomerConfig[CustomerCfg]()
```
