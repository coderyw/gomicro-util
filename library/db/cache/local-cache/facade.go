package local_cache

import (
	"context"
	"time"

	"github.com/coderyw/gomicro-util/library/db/cache/local-cache/local-cache-entity"
)

// LocalCache 定义了本地缓存的基础操作。
type LocalCache interface {
	// Set 写入缓存，并为 key 绑定指定的过期时间。
	//
	// key 会根据 ctx 中的 metadata.CONTEXT_NAME_SPACE 自动附加命名空间前缀。
	Set(ctx context.Context, k string, x interface{}, d time.Duration)
	// Get 按 key 读取缓存。
	//
	// 返回值中的 bool 表示缓存项是否存在。
	Get(ctx context.Context, k string) (interface{}, bool)
	// Delete 删除指定 key 的缓存项。
	Delete(ctx context.Context, k string)
	// Flush 清空所有缓存项。
	Flush()

	// Items 返回当前缓存中的所有条目。
	//
	// originKey 为 true 时返回去掉命名空间前缀后的原始 key；默认值为 true。
	//
	// ⚠️ 多租户场景下会跨租户合并返回。业务代码要按当前 ctx 的租户枚举缓存条目，
	// 用 ItemsOnCtx，不要用 Items。
	Items(originKey ...bool) map[string]local_cache_entity.Item

	// ItemsOnCtx 返回 ctx 所属租户的缓存条目，key 已剥掉命名空间前缀。
	//
	// 与 Items 不同：只返回属于当前租户的条目，不会跨租户合并。
	// ctx 无租户标识（NS 独享部署 / 兜底租户）时，只返回没有任何租户前缀的裸 key 条目。
	ItemsOnCtx(ctx context.Context) map[string]local_cache_entity.Item

	// FlushOnCtx 只清空 ctx 所属租户的缓存条目。
	//
	// ⚠️ Flush() 会清空所有租户的缓存，多租户场景下的按租户失效场景要用 FlushOnCtx。
	FlushOnCtx(ctx context.Context)
}
