// Package local_cache 提供了一个带命名空间隔离能力的进程内缓存封装。
//
// 默认实现基于 github.com/patrickmn/go-cache，并通过 context 中的
// metadata.CONTEXT_NAME_SPACE 自动为 key 添加租户命名空间前缀。
// 调用方在业务代码中仍然使用原始 key 即可，不需要手动拼接前缀。
//
// 典型使用方式：
//
//	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "tenant-a")
//	cache := local_cache.NewLocalCache()
//	cache.Set(ctx, "user:1", "alice", time.Minute)
//
//	v, ok := cache.Get(ctx, "user:1")
//	if ok {
//		fmt.Println(v)
//	}
//
// 如需替换底层实现，可通过 WithLocalCache 注入自定义的 LocalCache 构造函数。
package local_cache
