# 对象存储组件（library/component/storage）

统一的对象上传入口，支持 **AWS S3** 与 **华为云 OBS**。

核心设计：**业务代码只认「实例名」，不认云厂商**。同一个实例名挂在配置的哪张 provider map 下，就决定了它走哪朵云。这样 saas 线（S3）和 ns 线（华为 OBS）可以共用同一份业务代码，差异全部收敛在部署配置里。

---

## 一、30 秒接入

### 1. 配置实例（部署方填）

```yaml
storage:
  object_storage:
    default_instance: oss_core  # ← 默认实例，业务不指定实例名时用它
    aws_s3_config:            # saas 线：挂在这张 map 下 → 走 S3
      oss_core:               # ← 实例名
        region: ap-southeast-1
        use_default_credential_chain: true
        buckets:
          - is_default: true
            alias: core
            name: your-prod-bucket
            public_base_url: https://cdn.example.com
```

ns 线用**同一个实例名**，只是换一张 map：

```yaml
storage:
  object_storage:
    default_instance: oss_core
    huawei_obs_config:        # ns 线：挂在这张 map 下 → 走华为 OBS
      oss_core:               # ← 同名
        ak: xxx
        sk: xxx
        endpoint: obs.ap-southeast-3.myhuaweicloud.com
        buckets:
          - is_default: true
            alias: core
            name: your-ns-bucket
            default_acl: public-read
```

### 2. 业务侧不配实例名

配了 `default_instance` 之后，业务配置里**不要**再写一遍实例名，调用时把 `UploadRequest.Instance` 留空即可。少一处「跨配置段的字符串引用」，就少一类拼错名字、要等到上传那一刻才暴露的事故。

绝大多数服务只用一个实例，到此为止。真要在一个服务里用多个实例（不同用途落不同桶/不同云），在**调用处**显式传 `UploadRequest.Instance`。不要退回去加一个通用的「实例名」配置字段：它只能指一个实例，单实例场景下是 `default_instance` 的多余重复，多实例场景下又不够用，两头不靠。

> ⚠️ **默认实例只认显式配置**，不做「只有一个实例时自动成为默认」的隐式推导。否则部署方哪天加了第二个实例，既有业务的落桶位置就会被一个纯配置动作静默改掉。
>
> `default_instance` 填了但在两张 provider map 里找不到（或对应的那份配置是空的），**启动直接失败**——指向不存在实例的默认值，比没有默认值更危险。

### 3. 调用

```go
import (
    commonstorage "github.com/coderyw/gomicro-util/library/component/storage"
    storagemodel "github.com/coderyw/gomicro-util/model/storage"
)

ret, err := commonstorage.Upload(ctx, &storagemodel.UploadRequest{
    // Instance 留空 → 用 storage.object_storage.default_instance
    Key:            "excel/report_20260809.xlsx",  // 只写业务路径，租户前缀组件会加
    Body:           bytes.NewReader(data),
    ContentLength:  int64(len(data)),
    AllowOverwrite: true,
    Timeout:        30 * time.Second,
})
if err != nil {
    return err
}
url := ret.ObjectURL // 存库 / 返给前端，之后直接 GET 它就是下载，组件不提供 Download
```

初始化不用管——pmicro 启动时会自动调 `InitObjectStorage`。

---

## 二、实例名的三条约定

1. **不要带云厂商语义**。别叫 `s3_core` / `obs_core`，叫 `oss_core` / `object_storage` 这类中性名。名字里带 `s3`，ns 线就得配一个叫 "s3" 却指向华为云的实例，后来人必踩。
2. **两条线用同一个名字**。这样业务配置在两条线之间可以完全一致，只有 `storage.object_storage` 那一段不同。
3. **全局唯一**。同一个实例名不能同时出现在 `aws_s3_config` 和 `huawei_obs_config` 里，启动时会直接报 `duplicated object storage instance`。

---

## 三、配置字段

### 通用：桶配置

每个实例至少要有一个桶。`alias` 是业务侧引用的别名，`name` 是云上真实桶名。

| 字段 | 说明 |
|---|---|
| `is_default` | 是否默认桶。`UploadRequest.Bucket` 留空时用它；都没标则取列表第一个 |
| `alias` | 桶别名，业务通过 `UploadRequest.Bucket` 引用 |
| `name` | 云上真实桶名 |
| `public_base_url` | 公共访问域名（通常是 CDN），用于拼 `ObjectURL` |

### AWS S3 实例

| 字段 | 必填 | 说明 |
|---|---|---|
| `region` | ✅ | 如 `ap-southeast-1` |
| `use_default_credential_chain` | | **生产建议 true**，走 IRSA / Task Role，不落静态密钥 |
| `access_key_id` / `secret_access_key` | | `use_default_credential_chain=false` 时必填 |
| `endpoint` | | 自定义端点（MinIO / 私有网关），留空用 AWS 官方 |
| `use_path_style` | | MinIO 必须 `true` |
| `max_retries` | | 默认 3 |
| `dial_timeout` 等超时项 | | Go duration，如 `5s`；留空用内置默认 |

> ⚠️ **S3 侧没有对象 ACL 配置**。2023-04 后新建的桶默认 `Bucket owner enforced` + Block Public Access，ACL 已被禁用，带 ACL 的 PutObject 会返回 `AccessControlListNotSupported`。可见性一律由**桶策略**决定，公共地址由 `public_base_url` 拼出。

### 华为云 OBS 实例

| 字段 | 必填 | 说明 |
|---|---|---|
| `ak` / `sk` | ✅ | 为空时启动就失败（而不是等到上传报 403） |
| `endpoint` | ✅ | 如 `obs.ap-southeast-3.myhuaweicloud.com`。迁移旧配置时**逐字照抄**，SDK 不补 scheme |
| `connect_timeout` / `socket_timeout` | | **单位是秒的整数**，不是 duration。留空用 SDK 默认（60） |
| `max_retries` | | 留空用 SDK 默认（3）。⚠️ 不要为了"显式"填 0，SDK 只在 <0 时才回填默认，填 0 就是真的零重试 |
| 桶的 `default_acl` | | canned ACL，如 `public-read` |

### 两朵云的必配项（最容易配错的地方）

| | `public_base_url` | `default_acl` |
|---|---|---|
| **S3** | **必配**——不配 `ObjectURL` 返回空串且不报错 | **不存在此配置项** |
| **华为 OBS** | 可选，不配则回落 SDK 拼的地址 | **建议 `public-read`**，否则对象非公开可读 |

> 最典型的事故：看到华为实例配了 `default_acl: public-read`，顺手给 S3 也配上 → S3 侧根本没这个字段。反过来，S3 桶忘配 `public_base_url` → 拿到空 URL 写进业务表。

---

## 四、对象 Key 与租户隔离

**业务只写业务路径，租户前缀由组件统一追加，不要自己拼。**

```
业务传入:  excel/report.xlsx
实际落桶:  tenantok/excel/report.xlsx      # ctx 里租户为 ok
```

规则（`getPrefix` → `namespace.FmtByTypeOnCtx`）：

| 场景 | 结果 |
|---|---|
| ctx 携带租户标识 | 加 `tenant{租户}/` 前缀 |
| pod 设了环境变量 `TENANT_FALLBACK`（独享型部署，如 ns） | **不加前缀**，保持原始命名 |
| 未设 `TENANT_FALLBACK` 且 ctx 无租户 | 加 `tenantns/` 前缀 ⚠️ 见下 |

> ⚠️ **已知缺陷**：第三种情况会兜底成 `tenantns/`。这个默认值对 ns 线并不生效（ns 设了 `TENANT_FALLBACK`，前一条就短路了），真正命中的是**多租户 pod 丢失租户上下文**的调用——后台任务、异步 goroutine 等，结果是把对象写进 `tenantns/` 目录。
>
> **接入方注意**：调 `Upload` 时务必传递携带租户标识的 ctx。定时任务用 `namespace.GetTenantsInNamespace()` 拿到的 ctx，不要用裸 `context.Background()`。

---

## 五、API

### Upload

```go
func Upload(ctx context.Context, req *storage.UploadRequest) (*storage.UploadResult, error)
```

**UploadRequest 常用字段**

| 字段 | 说明 |
|---|---|
| `Instance` | **正常留空**，用 `default_instance`；只有一个服务用多个实例时才在调用处指定。都没有则返回 `ErrInvalidArgument` |
| `Key` | **必填**，业务路径，不含租户前缀 |
| `Body` | **必填**，`io.Reader` |
| `Bucket` | 桶别名，留空用实例默认桶。与 tenant 无关 |
| `ContentLength` | 已知长度时传入，便于 SDK 优化 |
| `ContentType` / `ContentDisposition` / `CacheControl` | 可选 |
| `AllowOverwrite` | **false 时会先 HeadObject 探测**，已存在则返回 `ErrObjectExists`（多一次请求） |
| `Timeout` | 请求级超时，**仅 S3 生效** |
| `Metadata` / `Tags` | `Tags` 仅 S3 生效 |
| `ACL` | **仅华为生效**，正常不该设置——可见性交给部署配置 |

**UploadResult**

| 字段 | 说明 |
|---|---|
| `ObjectURL` / `Location` | 访问地址。**S3 桶未配 `public_base_url` 时为空串**，务必判空 |
| `Bucket` | 实际使用的桶别名（默认桶已解析） |
| `Key` | 加过租户前缀的完整 key |
| `ETag` | 仅华为返回 |
| `VersionID` / `RequestID` | 排障用 |

调用方建议：

```go
if ret.ObjectURL == "" {
    return "", fmt.Errorf("upload succeeded but object url is empty, "+
        "check public_base_url of instance %q (bucket=%s key=%s)", instance, ret.Bucket, ret.Key)
}
```

### 下载 / 删除

**没有 Download 接口，也不需要**：拿 `UploadResult.ObjectURL` 直接 GET 就是下载。落库时存这个 URL，业务侧、前端、CDN 都按普通 HTTP 资源使用即可。

删除、预签名 URL、分片上传本组件不提供，有需求先直接用 SDK，但**务必自己补上 `tenant{租户}/` 前缀**（见第四节）——组件只在 `Upload` 里加前缀，裸调 SDK 是拿不到的。

### 启动期校验

**接入方什么都不用做**：`default_instance` 的存在性由 `InitObjectStorage` 强制校验，配错直接启动失败，不需要再挂校验钩子。

只有当服务自己引入了实例名配置项（多实例场景，按用途各配一个）时，那些字段才是无人约束的跨配置段引用——名字拼错要等到真正触发上传才报错，导出任务往往跑到最后一步才上传，白跑一整轮。这种情况下用下面的函数在 `BeforeStartFunc` 里自查（该钩子在 `InitObjectStorage` 之后执行）：

| 函数 | 返回 | 用途 |
|---|---|---|
| `HasInstance(name) bool` | 实例是否已注册 | 校验服务自定义的实例名配置项 |
| `Instances() []string` | 全部已注册实例名 | 校验失败时把可选值打进错误信息 |
| `DefaultInstance() string` | `default_instance` 解析结果，未配或未初始化为空串 | 区分「跟随默认」与「压根没接入对象存储」，例如还留着旧上传路径要做二选一路由时 |

> 三者都只用于启动阶段。不要在请求路径上做上传前探测——`Upload` 本身就会返回 `ErrInstanceNotFound`。
>
> 不需要区分是否接入的话，`DefaultInstance()` 也不用调：把 `Instance` 留空交给 `Upload` 自己回落即可。

---

## 六、两个 provider 的能力差异

| 能力 | S3 | 华为 OBS |
|---|---|---|
| `UploadRequest.Timeout` | ✅ | ❌ 忽略。SDK 只支持客户端级超时，用 `connect_timeout` / `socket_timeout` |
| `UploadRequest.Tags` | ✅ | ❌ 忽略。OBS 要走独立的 SetObjectTagging 接口 |
| `UploadRequest.ACL` | ❌ 忽略 | ✅ |
| 桶级 `default_acl` | ❌ 无此配置 | ✅ |
| 未配 `public_base_url` 时的 ObjectURL | 空串 | SDK 拼的地址 |
| 返回 `ETag` | ❌ | ✅ |

---

## 七、错误与排查

组件返回 `*storage.OpError`，格式：

```
storage op=put_object provider=s3 instance=oss_core bucket=core key=tenantok/excel/a.xlsx: <底层错误>
```

可用 `errors.Is` 判断的哨兵错误：

| 错误 | 含义 | 常见原因 |
|---|---|---|
| `ErrInstanceNotFound` | 实例未注册 | 实例名拼错；漏配 `storage.object_storage`；yaml 里名字带了尾空格 |
| `ErrBucketNotFound` | 桶别名不存在 | `UploadRequest.Bucket` 写的是真实桶名而不是 `alias` |
| `ErrObjectExists` | 对象已存在 | `AllowOverwrite=false` 且 key 重复 |
| `ErrInvalidArgument` | 入参非法 | `req`/`Body` 为 nil、`Key` 为空、`Instance` 为空且未配 `default_instance` |

其它典型症状：

| 症状 | 原因 |
|---|---|
| `ObjectURL` 为空但没报错 | S3 桶未配 `public_base_url` |
| 上传成功但下载 403 | 华为桶没配 `default_acl: public-read`；或 S3 桶策略未放开 |
| `AccessControlListNotSupported` (400) | 给 S3 传了 ACL——S3 侧不该有任何 ACL |
| key 变成 `tenantns/...` | ctx 丢了租户标识，见第四节 |
| key 出现双重前缀 `tenantok/tenantok/...` | 复用了同一个 `UploadRequest` 重试（当前版本已修，旧版本存在此问题） |

---

## 八、版本可用性 ⚠️

业务方拉的是**已发布的 tag**，与本仓库工作副本可能有差异。截至目前：

| 能力 | 已发布版本（如 `v1.1.0-rc.2`） | 当前未发布 |
|---|---|---|
| S3 provider | ✅ | ✅ |
| **华为 OBS provider** | ❌ **不存在** | ✅ |
| 桶级 `default_acl` | ❌ 不存在 | ✅（仅华为） |
| `HasInstance` / `Instances` | ❌ 不存在 | ✅ |
| `default_instance` / `DefaultInstance()` | ❌ 不存在（`Instance` 留空即报错） | ✅ |
| S3 忽略 `req.ACL` | ❌ 旧版会透传 | ✅ 已移除 |

**ns 线要走华为 OBS，必须等含 OBS provider 的版本发布并升级依赖**，在此之前 `huawei_obs_config` 配了也不会生效。接入前先确认自己 go.mod 里的版本。

---

## 九、已知限制

- **不支持配置热更新**。`InitObjectStorage` 由 pmicro 启动时调用一次；重复调用虽会原子替换并关闭旧 registry，但在途请求持有的 uploader 行为未定义。改对象存储配置请重启。
- **只有 Upload，这是有意的**。下载不需要本组件——`UploadResult.ObjectURL` 就是可直接 GET 的地址（S3 靠桶策略、华为靠 `default_acl` 放开可见性，见第三节）。删除、预签名 URL、分片上传目前没有真实用例，等有需求再加，需要的话先直接用 SDK。
  > ⚠️ 真要绕过组件直接调 SDK 时，**记得自己加租户前缀** `tenant{租户}/`——前缀只在本组件的 `Upload` 里追加（见第四节），裸调 SDK 拿不到它，删错/读错租户的对象就是这么来的。
- **provider 的 `Close()` 有意为空**。华为 SDK 的 `Close()` 会把内部 client 置 nil 且无同步，关停阶段与在途上传构成竞态；进程退出时释放空闲连接没有收益。
- `library/component/object-storage/`（带连字符）是另一个未完成的骨架包，**不要使用**，接入请认准本包 `library/component/storage/`。
