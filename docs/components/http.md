# HTTP：gin 封装、HTTP 客户端与上下文元数据

> 返回 [README](../../README.md)。HTTP server 本身由 pmicro 启动，见 [pmicro.md](pmicro.md)。

## gin

`library/gin` 提供参数绑定校验与统一响应。使用统一响应需在注册路由时开启默认响应：`option.RegisterHttpHandler(f, true)`。

### 参数校验

```go
func HttpReqRouter(ginCtx *gin.Context) {
	req := new(HttpReqObject)
	// 绑定参数并校验
	if err := gin.BindAndCheck(ginCtx, req); err != nil {
		gin.SetError(ginCtx, err)
		return
	}
	// 继续业务逻辑
}
```

请求结构体实现 `Validator` / `CValidator` 接口后，`BindAndCheck` 绑定时会自动调用其校验方法。

### 默认响应结构

```go
type NsHttpResp struct {
	Code   uint32      `json:"code"`   // 错误码，0 表示正常
	Msg    string      `json:"msg"`    // 错误信息
	Detail string      `json:"detail"` // 目前与 Msg 相同，不会暴露内部错误详情
	Data   interface{} `json:"data"`   // 响应数据
}
```

### 设置响应

```go
// 错误响应：后续中间件会解析出错误码并返回
gin.SetError(ginCtx, error_code.INVALID_PARAM.New(err))

// 正常响应：数据会被包在 NsHttpResp 的 data 中
gin.SetResponse(ginCtx, gin.H{"user_id": 0, "name": "小名"})

// 设置响应但不打印响应体日志（大 body / 敏感数据）
gin.SetResponseWithNoResponseLog(ginCtx, data)

// 不使用默认响应结构，由业务自行写响应体
gin.SetNoDefaultResponse(ginCtx)
ginCtx.JSON(http.StatusOK, body)

// ⚠️ SetRespHeader 当前不可用：中间件在响应体写出之后才回写 header，设置不会生效；
// 传入非 *gin.Context 时也读不到。需要自定义响应头请直接用 ginCtx.Header(k, v)
```

## HTTP 客户端

`library/http` 有两个子包：

- `http_req`：**推荐入口**，泛型 `Request[K]` 方法式调用，默认 20s 超时，带 skywalking 埋点与日志
- `net_http`：函数式旧版，`net_http.Post[T](ctx, url, reqInfo)`，复用同一个 `Request` 参数结构；**T 必须是指针类型**，否则响应反序列化失败

注意事项：

- 只有 `Timeout` 为 0 或 20s 时复用全局 client，其他值每次请求新建 client
- ⚠️ `http_req` 的 `Get` 在设置了 `Timeout` 时会直接修改全局 client 的超时（存在并发竞争），影响之后所有使用默认 client 的请求
- 两个包创建请求时都没有绑定 ctx，**ctx 的取消和超时不会作用到 HTTP 请求上**，超时只能靠 `Timeout`
- 发出的请求不会自动携带 traceId

```go
type ApiPostResp struct {
	Id int `json:"id"`
}

req := http_req.NewRequest[*ApiPostResp](reqBody)
req.Header.Set("X-Foo", "bar")
resp, httpCode, err := req.Post(ctx, "https://example.com/api/post")
```

`Request` 还支持 `Query`、`Timeout`、`ContentType`，以及 `ReqBodyReMake` / `RespBodyMake` 两个钩子用于请求加密、响应解密。

## metadata：上下文元数据

`library/metadata` 负责在 ctx 中携带并跨服务透传用户 id、渠道、请求 ip、租户等元数据，支持 `gin.Context` 与 go-micro metadata。支持的 key 见包注释（`go doc ./library/metadata`）。

```go
// 设置
ctx = metadata.SetCtxData(ctx, metadata.CONTEXT_USER_ID, int64(123))

// 获取：取不到返回零值；⚠️ 值的类型与 T 不一致时会 panic
uid := metadata.Get[int64](ctx, metadata.CONTEXT_USER_ID)

// 获取：只有 ctx 中完全没有元数据时才 panic；key 不存在或类型不一致时返回零值
uid = metadata.MustGet[int64](ctx, metadata.CONTEXT_USER_ID)
```

只能设置包注释中列出的标准 key 或以 `Custom_` 为前缀的 key，**其他 key 会被直接丢弃**，连本进程内都读不回来。

> `pmicro/session` 的 `UserId(ctx)` 等方法读取的是另一套 key，本库内没有代码写入，不能代替 `metadata`。
