# secret：凭据引用

> 返回 [README](../../README.md)。升级注意事项见 [upgrade/v3.1.0.md](../upgrade/v3.1.0.md)，相关环境变量见 [environment.md](../environment.md#凭据与启动)。

`library/component/secret` 让配置中的敏感值（数据库密码、Redis 密码、第三方密钥等）不再明文存放，而是写成**凭据引用**，启动时从云厂商凭据服务取回明文。

## 引用格式

```text
{scheme}://{凭据名}[/{JSON键名}]

csms://ns-ca-balance/mysql_password    华为云 CSMS（DEW 凭据管理）
awssm://ok-ca-balance/mysql_password   AWS Secrets Manager
```

- 不带键名：取整份凭据内容
- 带键名：凭据内容按 JSON 解析后取该键的值

## 在哪里可以用

pmicro 已自动注册 `csms`、`awssm` 两个 Provider，以下位置写引用即可，业务无需改代码：

| 位置 | 说明 |
|-|-|
| 服务 YAML 配置（本地文件、Nacos 配置） | 任意配置项的**值**，包括 `other` 自定义段；Nacos 热更新同样解析 |
| `site-var/nacos` 的 `GetNacosCfgData` / `ConfigListen` | 同上，`.yaml` 走 YAML、其余走 JSON |
| 环境变量 `NACOS_USERNAME` / `NACOS_PASSWORD` | 整个值是引用时解析，否则按明文使用 |

```yaml
storage:
  mysql:
    instance:
      core:
        host: 'db.internal'                       # 非秘密，直接写
        user: 'ca_balance'
        password: 'csms://ns-ca-balance/mysql_password'
```

业务代码也可以直接调用：

```go
pwd, err := secret.Resolve(ctx, "csms://ns-ca-balance/mysql_password")
err = secret.UnmarshalYAML(ctx, text, &cfg) // 解析文本中的引用后反序列化
```

## 规则与限制

- **引用必须是整个值**：拼在 DSN、URL 或文字中间（如 `root:csms://x/y@tcp(...)`）会被判为非法引用，配置解析失败
- **普通配置项不能含 `csms://`、`awssm://` 子串**：会被当作非法引用。启动期直接退出；热更新期只打 ERROR 并**保留旧配置**（表现为"改了配置没生效"）
- `http://`、`mysql://`、`redis://` 等常见 scheme 不会被当作引用；其他未注册的 scheme 原样保留，每个进程只告警一次
- 注释和映射的键名不参与解析
- **只把秘密写成引用**：库名、主机、端口等非秘密值不要放进凭据，取出的值同样会被日志脱敏

## 缓存、重试与超时

- 同一凭据在进程内只拉取一次并缓存，**轮换凭据需要重启服务**
- 单次拉取超时 5s；可重试错误按指数退避 + 全抖动重试，每个凭据名的整体 deadline 默认 20s（`SECRET_RETRY_DEADLINE_SECONDS`，0 为不重试）
- 凭据不存在、无权限等确定性错误不重试，直接失败
- 启动预算按「每个服务只引用一个凭据名」估算：引用多个凭据名时最坏耗时按个数叠加，此时**必须配置 startupProbe**，否则可能被 liveness 探针杀掉

## 日志脱敏

解析出的明文会登记到 `library/log` 的脱敏表，此后经 `library/log` 输出的日志（含 Nacos SDK、gorm 等适配日志与标准库 `log`）中出现该值时替换为 `***`，与按字段名的脱敏叠加生效。

- 键名为账号类（`username`、`user`、`account` 等，或以 `_user` 等结尾）的值不登记，避免账号名与库名、服务名相同时被大面积打码
- 长度小于 6 的值不登记
- 直接 `fmt.Print` 到标准输出、第三方库自建的 logger 不在覆盖范围内
- 业务也可以用 `log.RegisterRedactValue(v)` 登记自己的敏感值

## Provider 与云凭证

| scheme | 包 | 云凭证 | region |
|-|-|-|-|
| `csms` | `secret/provider/csms` | 华为云 SDK 默认链：环境变量 AK/SK → profile → ECS 元数据 → CCE Pod Identity，容器内无需落 AK/SK | `HUAWEICLOUD_SDK_REGION`，默认 `ap-southeast-3` |
| `awssm` | `secret/provider/awssm` | aws-sdk-go-v2 默认链（IRSA 自动生效） | `AWS_REGION` / `AWS_DEFAULT_REGION`，默认 `ap-southeast-1` |

自定义 Provider 实现 `secret.Provider` 接口（`Scheme()` + `Fetch(ctx, name)`）后调用 `secret.Register` 注册；确定性失败请用 `secret.NonRetryable(err)` 包装，避免无谓重试。

## Nacos 账密

`NACOS_USERNAME` / `NACOS_PASSWORD` 支持三种形态：

| 形态 | 行为 |
|-|-|
| 空 | 匿名登录 |
| 凭据引用 | 经 secret 解析，失败则启动失败 |
| 明文 | 原样使用；密码为明文时打 WARN（可在观测云按文案审计）。`NACOS_AUTH_ALLOW_PLAINTEXT=false` 时密码为明文直接启动失败 |

任何路径都不打印账密的值，只记录来源（未设置 / 引用 / 明文）。
