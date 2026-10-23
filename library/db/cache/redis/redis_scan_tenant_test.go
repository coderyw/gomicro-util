package redis

import (
	"context"
	"sort"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/env"
)

// ==========================================================
// Redis.Scan 多租户前缀单元测试（miniredis，不依赖真实 Redis）
//
// SCAN 不在 PrefixHookV3 的 cmdModes 里 —— 它没有 key 参数，要加前缀的是 MATCH
// 后面那个可选且位置浮动的 pattern，位置表表达不了。所以前缀由 Redis.Scan 自己
// 进出各处理一次，本文件锁定这个契约：
//
//	入参 match 是原始 pattern，返回值也是原始 key。
//
// 覆盖三种部署形态：ctx 带租户 / ctx 无租户（NS）/ 配了 TENANT_FALLBACK（独享型）。
// ==========================================================

// newScanTestRedis 起一个 miniredis，并按生产接线（redis.go 里 client.AddHook）挂上
// PrefixHookV3，返回封装后的 *Redis 和底层裸客户端（用于旁路写入种子数据）。
func newScanTestRedis(t *testing.T) (*Redis, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	client.AddHook(NewPrefixHookV3())
	t.Cleanup(func() { _ = client.Close() })
	return &Redis{UniversalClient: client}, mr
}

// seedKeys 直接往 miniredis 写，绕过 hook，这样种子数据的 key 就是「Redis 里真实的样子」
func seedKeys(t *testing.T, mr *miniredis.Miniredis, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if err := mr.Set(k, "v"); err != nil {
			t.Fatalf("seed %s: %v", k, err)
		}
	}
}

func scanAll(t *testing.T, r *Redis, ctx context.Context, match string) []string {
	t.Helper()
	var (
		cursor uint64
		got    []string
	)
	for {
		batch, next, err := r.Scan(ctx, cursor, match, 100)
		if err != nil {
			t.Fatalf("Scan error: %v", err)
		}
		got = append(got, batch...)
		if next == 0 {
			break
		}
		cursor = next
	}
	sort.Strings(got)
	return got
}

func assertKeys(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("key 数量不符：got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("key 不符：got %v, want %v", got, want)
		}
	}
}

// ctx 携带租户：pattern 应被加上租户前缀，只扫到本租户的 key，且返回原始 key
func TestRedisScan_WithTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)

	seedKeys(t, mr,
		"tenantok:cashback:cfg:1",
		"tenantok:cashback:cfg:2",
		"tenantxx:cashback:cfg:9", // 别的租户，不该被扫到
		"cashback:cfg:8",          // 无前缀（NS 那份数据），不该被扫到
	)

	got := scanAll(t, r, tenantTestCtx("ok"), "cashback:cfg:*")
	assertKeys(t, got, []string{"cashback:cfg:1", "cashback:cfg:2"})
}

// 回环：Scan 返回的 key 直接交给 UNLINK 必须删对
//
// 这是本次修复的核心契约 —— Scan 剥前缀、UNLINK（modeAllFrom1）再加回去。
// 少了任何一边都会变成 tenantok:tenantok:… 删了个空且不报错。
func TestRedisScan_RoundTripWithUnlink(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)
	ctx := tenantTestCtx("ok")

	seedKeys(t, mr,
		"tenantok:cashback:cfg:1",
		"tenantok:cashback:cfg:2",
		"tenantxx:cashback:cfg:9",
		"cashback:cfg:8",
	)

	keys := scanAll(t, r, ctx, "cashback:cfg:*")
	if _, err := r.UniversalClient.Unlink(ctx, keys...).Result(); err != nil {
		t.Fatalf("Unlink error: %v", err)
	}

	for _, k := range []string{"tenantok:cashback:cfg:1", "tenantok:cashback:cfg:2"} {
		if mr.Exists(k) {
			t.Fatalf("本租户 key 应被删除，仍存在：%s", k)
		}
	}
	for _, k := range []string{"tenantxx:cashback:cfg:9", "cashback:cfg:8"} {
		if !mr.Exists(k) {
			t.Fatalf("不该被误删：%s", k)
		}
	}
}

// ctx 无租户（NS 线）：pattern 与返回值都保持原样
func TestRedisScan_WithoutTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)

	seedKeys(t, mr,
		"cashback:cfg:1",
		"cashback:cfg:2",
		"tenantok:cashback:cfg:9", // 有前缀的不该被无租户 ctx 扫到
	)

	got := scanAll(t, r, context.Background(), "cashback:cfg:*")
	assertKeys(t, got, []string{"cashback:cfg:1", "cashback:cfg:2"})
}

// 配了 TENANT_FALLBACK 的独享型部署：即便 ctx 带租户也一律不加前缀
func TestRedisScan_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	t.Cleanup(func() { env.SetFallbackTenantName("") })

	r, mr := newScanTestRedis(t)
	seedKeys(t, mr,
		"cashback:cfg:1",
		"cashback:cfg:2",
		"tenantok:cashback:cfg:9",
	)

	got := scanAll(t, r, tenantTestCtx("ok"), "cashback:cfg:*")
	assertKeys(t, got, []string{"cashback:cfg:1", "cashback:cfg:2"})
}

// match 为空（原语义是遍历全库）不能被前缀拼成 "tenantok:" 这种没有通配符的死 pattern，
// 且多租户下要收敛成「遍历本租户」而不是遍历全库
func TestRedisScan_EmptyMatch(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)
	seedKeys(t, mr,
		"tenantok:a",
		"tenantok:b",
		"tenantxx:c", // 别的租户
		"d",          // 无前缀
	)

	got := scanAll(t, r, tenantTestCtx("ok"), "")
	assertKeys(t, got, []string{"a", "b"})
}

// 业务 key 本身就以 "tenant" 开头时不能被误剥
//
// 剥前缀只能剥「本次自己加的那个」。若图省事用 GetOriginKey（判据是 HasPrefix(key,
// "tenant") 再按 ":" 切），NS 线上一个名为 tenant_stats:… 的裸 key 会被剥成 stats:…，
// 调用方拿去 DEL 就删了个不存在的 key。
func TestRedisScan_KeyLooksLikeTenantPrefix(t *testing.T) {
	env.SetFallbackTenantName("")
	r, mr := newScanTestRedis(t)

	// NS 线：ctx 无租户，key 原样返回，哪怕它长得像租户前缀
	seedKeys(t, mr, "tenant_stats:daily", "tenant_stats:weekly")
	got := scanAll(t, r, context.Background(), "tenant_stats:*")
	assertKeys(t, got, []string{"tenant_stats:daily", "tenant_stats:weekly"})
}

// 兜底租户下 match 为空仍是遍历全库（NS 的原有行为不能被改掉）
func TestRedisScan_EmptyMatchFallback(t *testing.T) {
	env.SetFallbackTenantName("ns")
	t.Cleanup(func() { env.SetFallbackTenantName("") })

	r, mr := newScanTestRedis(t)
	seedKeys(t, mr, "a", "b", "c")

	got := scanAll(t, r, tenantTestCtx("ok"), "")
	assertKeys(t, got, []string{"a", "b", "c"})
}
