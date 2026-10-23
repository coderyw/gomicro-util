# local-cache

`local-cache` 提供了一个带命名空间隔离能力的进程内缓存封装。

默认实现基于 `github.com/patrickmn/go-cache`，并通过 `context.Context` 中的 `metadata.CONTEXT_NAME_SPACE` 自动为 key 添加命名空间前缀，适合做单进程热点数据、本地短时缓存、降级兜底缓存等场景。

## 特性

- 对外暴露统一的 `LocalCache` 接口，便于替换底层实现
- 默认支持过期时间和后台清理
- 自动根据上下文命名空间隔离 key，业务侧不需要手动拼接前缀
- 支持查看当前缓存条目，便于调试和排查问题

## 安装与引用

```go
import local_cache "github.com/coderyw/gomicro-util/library/db/cache/local-cache"
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"time"

	local_cache "github.com/coderyw/gomicro-util/library/db/cache/local-cache"
	"github.com/coderyw/gomicro-util/library/metadata"
)

func main() {
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "tenant-a")
	cache := local_cache.NewLocalCache()

	cache.Set(ctx, "user:1", "alice", time.Minute)

	value, ok := cache.Get(ctx, "user:1")
	fmt.Println(value, ok)

	cache.Delete(ctx, "user:1")
}
```

## 配置

默认构造：

```go
cache := local_cache.NewLocalCache()
```

自定义默认过期时间和清理间隔：

```go
cache := local_cache.NewLocalCache(
	local_cache.WithDefaultExpiration(time.Minute),
	local_cache.WithCleanupInterval(30*time.Second),
)

cache.Set(context.Background(), "token", "v1", local_cache.DefaultExpiration)
```

配置项说明：

- `WithDefaultExpiration`：设置缓存实例的默认过期时间
- `WithCleanupInterval`：设置过期键后台清理周期，`0` 表示不自动清理
- `WithLocalCache`：注入自定义底层缓存实现

## 接口说明

```go
type LocalCache interface {
	Set(ctx context.Context, k string, x interface{}, d time.Duration)
	Get(ctx context.Context, k string) (interface{}, bool)
	Delete(ctx context.Context, k string)
	Flush()
	Items(originKey ...bool) map[string]l_cache.Item
}
```

方法说明：

- `Set`：写入缓存
- `Get`：读取缓存，返回值中的 `bool` 表示是否命中
- `Delete`：删除指定 key
- `Flush`：清空全部缓存
- `Items`：返回当前缓存中的全部条目，默认返回去掉命名空间前缀后的原始 key

## 命名空间行为

该包会通过 `metadata.CONTEXT_NAME_SPACE` 自动格式化 key。

例如：

- 业务代码写入的 key 为 `user:1`
- `ctx` 中的 namespace 为 `demo`
- 实际存储到底层缓存的 key 会带上 namespace 前缀
- 调用 `Get/Delete` 时仍然使用原始 key `user:1`

如果调用 `Items()`：

- `Items()` 或 `Items(true)` 返回原始 key
- `Items(false)` 返回底层实际存储的 key

## 适用场景

- 单机热点数据缓存
- 本地短时幂等控制
- 远端缓存或数据库的前置缓存
- 对低延迟要求高、可接受进程重启丢失数据的场景

## 注意事项

- 这是本地内存缓存，不适用于多实例之间共享数据
- 进程重启后缓存会全部丢失
- 命名空间隔离依赖 `ctx` 中是否正确写入 `metadata.CONTEXT_NAME_SPACE`
- 当使用 `DefaultExpiration` 时，实际过期时间取决于 `WithDefaultExpiration` 的配置
