// Package cache 提供项目内部使用的缓存实现集合。
//
// 该目录用于承载具体缓存后端的实现，供上层缓存抽象按需组装或注入。
// 当前内置实现为基于 github.com/patrickmn/go-cache 的进程内缓存，
// 位于子包 go_cache。
//
// 这些实现主要关注底层存取能力本身；面向业务侧的统一封装、命名空间
// 处理和构造入口位于 library/db/cache/local-cache。
package cache
