package go_cache

import (
	"context"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/coderyw/gomicro-util/library/db/cache/local-cache/local-cache-entity"
	"github.com/coderyw/gomicro-util/library/namespace"
)

func NewLocalGoCache(defaultExpiration, cleanupInterval time.Duration) *LocalGoCache {
	return &LocalGoCache{
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
}

type LocalGoCache struct {
	cache *cache.Cache
}

func (l *LocalGoCache) Set(ctx context.Context, k string, x interface{}, d time.Duration) {
	l.cache.Set(namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_COMMON, k), x, d)
}

func (l *LocalGoCache) Get(ctx context.Context, k string) (interface{}, bool) {
	return l.cache.Get(namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_COMMON, k))
}

func (l *LocalGoCache) Delete(ctx context.Context, k string) {
	l.cache.Delete(namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_COMMON, k))
}

func (l *LocalGoCache) Flush() {
	l.cache.Flush()
}

func (l *LocalGoCache) Items(originKey ...bool) map[string]local_cache_entity.Item {
	bo := true
	if len(originKey) > 0 {
		bo = originKey[0]
	}
	it := l.cache
	mp := make(map[string]local_cache_entity.Item)
	for k, v := range it.Items() {
		if bo {
			// 对 k 进行还原
			k = namespace.GetOriginKey(namespace.NAMESPACE_TYPE_COMMON, k)
		}
		mp[k] = local_cache_entity.Item{
			Expiration: v.Expiration,
			Object:     v.Object,
		}
	}
	return mp
}

// ItemsOnCtx 返回 ctx 所属租户的缓存条目，key 已剥掉命名空间前缀。
//
// 与 Items 不同：这里按 ctx 的租户前缀过滤，不会把其它租户的条目混进来。
// ctx 无租户标识（NS 独享部署 / 兜底租户）时，前缀为空串，此时只认同样没有
// 任何租户前缀的裸 key —— 既不会误吞其它租户的条目，也不会让未开多租户的
// 部署因为"一律拒绝"而导致批量枚举/清空静默失效。
func (l *LocalGoCache) ItemsOnCtx(ctx context.Context) map[string]local_cache_entity.Item {
	prefix := namespace.FmtByTypeOnCtx(ctx, namespace.NAMESPACE_TYPE_COMMON, "")
	mp := make(map[string]local_cache_entity.Item)
	for k, v := range l.cache.Items() {
		if prefix == "" {
			if namespace.GetNameSpaceFromKey(namespace.NAMESPACE_TYPE_COMMON, k) != "" {
				continue
			}
			mp[k] = local_cache_entity.Item{Expiration: v.Expiration, Object: v.Object}
			continue
		}
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		mp[strings.TrimPrefix(k, prefix)] = local_cache_entity.Item{Expiration: v.Expiration, Object: v.Object}
	}
	return mp
}

// FlushOnCtx 只清空 ctx 所属租户的缓存条目。
//
// Flush() 是全局清空，多租户下按租户失效要用这个。
func (l *LocalGoCache) FlushOnCtx(ctx context.Context) {
	for k := range l.ItemsOnCtx(ctx) {
		l.Delete(ctx, k)
	}
}
