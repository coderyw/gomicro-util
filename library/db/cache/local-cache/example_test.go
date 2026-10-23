package local_cache

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/coderyw/gomicro-util/library/metadata"
)

func ExampleNewLocalCache() {
	ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "demo")
	cache := NewLocalCache()

	cache.Set(ctx, "profile", "alice", time.Minute)

	value, ok := cache.Get(ctx, "profile")
	fmt.Println(ok)
	fmt.Println(value)

	keys := make([]string, 0, len(cache.Items()))
	for key := range cache.Items() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println(keys)

	// Output:
	// true
	// alice
	// [profile]
}

func ExampleNewLocalCache_withOptions() {
	ctx := context.Background()
	cache := NewLocalCache(
		WithDefaultExpiration(time.Minute),
		WithCleanupInterval(30*time.Second),
	)

	cache.Set(ctx, "token", "v1", DefaultExpiration)

	_, ok := cache.Get(ctx, "token")
	fmt.Println(ok)

	// Output:
	// true
}
