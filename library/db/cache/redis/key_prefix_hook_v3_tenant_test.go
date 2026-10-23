package redis

import (
	"context"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

// ==========================================================
// PrefixHookV3 多租户前缀单元测试（不依赖真实 Redis）
//
// 覆盖三类场景：
//  1. ctx 携带租户 → key 加租户前缀
//  2. 配置兜底租户（TENANT_FALLBACK）→ 一律不加前缀
//  3. 未配置兜底租户 → 按 ctx 是否携带租户决定
// ==========================================================

func tenantTestCtx(ns string) context.Context {
	return metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
}

// runProcessHook 构造命令走一遍 ProcessHook（next 不真正发请求），返回改写后的参数
func runProcessHook(t *testing.T, ctx context.Context, args ...interface{}) []interface{} {
	t.Helper()
	h := NewPrefixHookV3()
	cmd := goredis.NewCmd(ctx, args...)
	next := func(ctx context.Context, cmd goredis.Cmder) error { return nil }
	if err := h.ProcessHook(next)(ctx, cmd); err != nil {
		t.Fatalf("ProcessHook error: %v", err)
	}
	return cmd.Args()
}

// runPipelineHook 构造多条命令走一遍 ProcessPipelineHook，返回每条命令改写后的参数
func runPipelineHook(t *testing.T, ctx context.Context, cmdsArgs ...[]interface{}) [][]interface{} {
	t.Helper()
	h := NewPrefixHookV3()
	cmds := make([]goredis.Cmder, 0, len(cmdsArgs))
	for _, args := range cmdsArgs {
		cmds = append(cmds, goredis.NewCmd(ctx, args...))
	}
	next := func(ctx context.Context, cmds []goredis.Cmder) error { return nil }
	if err := h.ProcessPipelineHook(next)(ctx, cmds); err != nil {
		t.Fatalf("ProcessPipelineHook error: %v", err)
	}
	out := make([][]interface{}, 0, len(cmds))
	for _, cmd := range cmds {
		out = append(out, cmd.Args())
	}
	return out
}

// 正向：ctx 携带租户，key 应加租户前缀
func TestPrefixHookV3_WithTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")

	args := runProcessHook(t, tenantTestCtx("ok"), "get", "mykey")
	if args[1] != "tenantok:mykey" {
		t.Errorf("ctx 携带租户时 key 应改写为 tenantok:mykey, got %v", args[1])
	}

	// mset：隔一个改写一个（key value key value）
	args = runProcessHook(t, tenantTestCtx("ok"), "mset", "k1", "v1", "k2", "v2")
	if args[1] != "tenantok:k1" || args[3] != "tenantok:k2" {
		t.Errorf("mset 的 key 应全部加前缀, got %v", args)
	}
	if args[2] != "v1" || args[4] != "v2" {
		t.Errorf("mset 的 value 不应被改写, got %v", args)
	}
}

// 反向：未配置兜底租户且 ctx 不携带租户，key 保持原样
func TestPrefixHookV3_NoTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")

	args := runProcessHook(t, context.Background(), "get", "mykey")
	if args[1] != "mykey" {
		t.Errorf("ctx 无租户时 key 应保持原样 mykey, got %v", args[1])
	}
}

// 反向：不涉及 key 的命令不应被改写
func TestPrefixHookV3_NonKeyCommand(t *testing.T) {
	env.SetFallbackTenantName("")

	args := runProcessHook(t, tenantTestCtx("ok"), "info", "server")
	if args[1] != "server" {
		t.Errorf("非 key 命令参数不应被改写, got %v", args)
	}
}

// 兜底：配置了兜底租户后，即使 ctx 携带租户 key 也不加前缀
func TestPrefixHookV3_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	args := runProcessHook(t, tenantTestCtx("ok"), "get", "mykey")
	if args[1] != "mykey" {
		t.Errorf("兜底租户模式下 key 不应加前缀, got %v", args[1])
	}

	// ctx 无租户同样保持原样
	args = runProcessHook(t, context.Background(), "set", "mykey", "v")
	if args[1] != "mykey" {
		t.Errorf("兜底租户模式下（无租户 ctx）key 不应加前缀, got %v", args[1])
	}
}

// 正向：pipeline 中所有命令的 key 都应加租户前缀
func TestPrefixHookV3_Pipeline_WithTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")

	out := runPipelineHook(t, tenantTestCtx("ok"),
		[]interface{}{"get", "k1"},
		[]interface{}{"set", "k2", "v2"},
	)
	if out[0][1] != "tenantok:k1" || out[1][1] != "tenantok:k2" {
		t.Errorf("pipeline 中 key 应全部加前缀, got %v", out)
	}
}

// 兜底：pipeline 在兜底租户模式下同样不加前缀
func TestPrefixHookV3_Pipeline_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	out := runPipelineHook(t, tenantTestCtx("ok"),
		[]interface{}{"get", "k1"},
		[]interface{}{"set", "k2", "v2"},
	)
	if out[0][1] != "k1" || out[1][1] != "k2" {
		t.Errorf("兜底租户模式下 pipeline 的 key 不应加前缀, got %v", out)
	}
}

// 反向：pipeline 中 ctx 无租户时保持原样
func TestPrefixHookV3_Pipeline_NoTenantCtx(t *testing.T) {
	env.SetFallbackTenantName("")

	out := runPipelineHook(t, context.Background(),
		[]interface{}{"get", "k1"},
	)
	if out[0][1] != "k1" {
		t.Errorf("ctx 无租户时 pipeline 的 key 应保持原样, got %v", out)
	}
}

func TestPrefixHookV3_TenantCaseCompatibility(t *testing.T) {
	env.SetFallbackTenantName("")
	for _, ns := range []string{"ok", "OK", "Ok"} {
		ctx := tenantTestCtx(ns)
		args := runProcessHook(t, ctx, "eval", "return redis.call('get', KEYS[1])", 1, "UserID")
		if args[3] != "tenantok:UserID" {
			t.Fatal(args)
		}
		cmds := runPipelineHook(t, ctx, []interface{}{"set", "UserID", "VALUE"})
		if cmds[0][1] != "tenantok:UserID" || cmds[0][2] != "VALUE" {
			t.Fatal(cmds)
		}
	}
}
