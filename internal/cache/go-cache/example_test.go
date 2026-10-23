package go_cache

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/coderyw/gomicro-util/library/metadata"
)

func exampleContext() context.Context {
	return metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, "demo")
}

func ExampleNewLocalGoCache() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "profile", "alice", time.Minute)

	value, ok := cache.Get(ctx, "profile")
	fmt.Println(ok)
	fmt.Println(value)

	// Output:
	// true
	// alice
}

func ExampleLocalGoCache_Set() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "token", "v1", time.Minute)

	_, ok := cache.Get(ctx, "token")
	fmt.Println(ok)

	// Output:
	// true
}

func ExampleLocalGoCache_Get() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "user:1", "alice", time.Minute)

	value, ok := cache.Get(ctx, "user:1")
	fmt.Println(ok)
	fmt.Println(value)

	// Output:
	// true
	// alice
}

func ExampleLocalGoCache_Delete() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "session", "active", time.Minute)
	cache.Delete(ctx, "session")

	_, ok := cache.Get(ctx, "session")
	fmt.Println(ok)

	// Output:
	// false
}

func ExampleLocalGoCache_Flush() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "k1", "v1", time.Minute)
	cache.Set(ctx, "k2", "v2", time.Minute)
	cache.Flush()

	fmt.Println(len(cache.Items()))

	// Output:
	// 0
}

func ExampleLocalGoCache_Items() {
	ctx := exampleContext()
	cache := NewLocalGoCache(time.Minute, time.Minute)

	cache.Set(ctx, "user:1", "alice", time.Minute)
	cache.Set(ctx, "user:2", "bob", time.Minute)

	keys := make([]string, 0, len(cache.Items()))
	for key := range cache.Items() {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Println(keys)

	rawKeys := make([]string, 0, len(cache.Items(false)))
	for key := range cache.Items(false) {
		rawKeys = append(rawKeys, key)
	}
	sort.Strings(rawKeys)
	fmt.Println(rawKeys)

	// Output:
	// [user:1 user:2]
	// [tenantdemo:user:1 tenantdemo:user:2]
}
