// Package go_cache 提供基于 github.com/patrickmn/go-cache 的进程内缓存实现。
//
// 该实现通过 context 中的 metadata.CONTEXT_NAME_SPACE 自动为 key 追加公共
// 命名空间前缀，调用方仍然使用业务原始 key 即可。它主要作为 internal/cache
// 下的具体后端实现，被上层缓存抽象组合使用。
package go_cache
