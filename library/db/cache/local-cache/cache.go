package local_cache

import (
	"time"

	"github.com/coderyw/gomicro-util/internal/cache/go-cache"
)

const (
	// NoExpiration 表示缓存项永不过期。
	NoExpiration time.Duration = -1
	// DefaultExpiration 表示写入时使用缓存实例的默认过期时间。
	DefaultExpiration time.Duration = 0
)

// LocalCacheOptions 定义 NewLocalCache 的可选配置。
type LocalCacheOptions struct {
	// localCacheFunc 用于创建底层缓存实现。
	localCacheFunc func(defaultExpiration, cleanupInterval time.Duration) LocalCache
	// defaultExpiration 是缓存实例的默认过期时间，默认值为 NoExpiration。
	defaultExpiration time.Duration
	// cleanupInterval 是过期键的后台清理间隔；为 0 时表示不自动清理。
	cleanupInterval time.Duration
}

// WithLocalCache 注入自定义的缓存实现工厂。
func WithLocalCache(localCache func(defaultExpiration, cleanupInterval time.Duration) LocalCache) func(options *LocalCacheOptions) {
	return func(options *LocalCacheOptions) {
		options.localCacheFunc = localCache
	}
}

// WithDefaultExpiration 设置缓存实例的默认过期时间。
//
// 当调用 Set 时传入 DefaultExpiration，会回退到这里配置的时长。
func WithDefaultExpiration(defaultExpiration time.Duration) func(options *LocalCacheOptions) {
	return func(options *LocalCacheOptions) {
		options.defaultExpiration = defaultExpiration
	}
}

// WithCleanupInterval 设置过期键的后台清理间隔。
func WithCleanupInterval(cleanupInterval time.Duration) func(options *LocalCacheOptions) {
	return func(options *LocalCacheOptions) {
		options.cleanupInterval = cleanupInterval
	}
}

// NewLocalCache 创建本地缓存实例。
//
// 默认使用 go-cache 作为底层实现，默认过期时间为 NoExpiration。
func NewLocalCache(opts ...func(options *LocalCacheOptions)) LocalCache {
	options := LocalCacheOptions{
		localCacheFunc:    newLocalGoCache,
		defaultExpiration: NoExpiration,
	}
	for _, o := range opts {
		o(&options)
	}
	return options.localCacheFunc(options.defaultExpiration, options.cleanupInterval)
}

func newLocalGoCache(defaultExpiration, cleanupInterval time.Duration) LocalCache {
	return go_cache.NewLocalGoCache(defaultExpiration, cleanupInterval)
}
