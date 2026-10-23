package sync

import (
	"context"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
	"github.com/coderyw/gomicro-util/library/namespace"
)

// ==========================================================
// versionKeyOf 单测：验证 key 与 versionKey 在经过
// key_prefix_hook_v3.go 那套"无脑整串前置拼接、不识别 {}"的多租户前缀
// 注入之后，解析出的 hash tag 是否一致（即是否会落到 Redis Cluster 同一 slot）。
//
// 这里不直接依赖 hook 实现（避免 sync ← redis 的 import 循环），而是用
// hook 内部同一套前缀计算函数 namespace.FmtByType 手工模拟它的行为：
// 对整串做一次无脑前置拼接，不识别 {}。
// ==========================================================

// simulateHookPrefix 模拟 key_prefix_hook_v3.go 的 applyPrefixFast：
// 对整串做前缀拼接，完全不识别/不处理 {}。
func simulateHookPrefix(nsKey, raw string) string {
	return namespace.FmtByType(namespace.NAMESPACE_TYPE_REDIS, nsKey, raw)
}

// extractHashTag 模拟 Redis Cluster 的 hash tag 提取规则：
// 找第一个 '{'，再找它之后第一个 '}'，取中间内容；找不到则整串即 tag。
func extractHashTag(s string) string {
	l := strings.IndexByte(s, '{')
	if l == -1 {
		return s
	}
	r := strings.IndexByte(s[l+1:], '}')
	if r == -1 {
		return s
	}
	return s[l+1 : l+1+r]
}

func versionKeyTestCtx(ns string) context.Context {
	return metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, ns)
}

// 有租户：versionKeyOf 预先把 hook 会加的前缀塞进 {} 内部，
// 经 hook 整串再拼一次前缀后，key 与 versionKey 的 hash tag 必须一致。
func TestVersionKeyOf_WithTenant_SameHashTagAsKey(t *testing.T) {
	env.SetFallbackTenantName("")

	key := "ca-activity:bet_amount:wait:sync:43"
	ctx := versionKeyTestCtx("ok")

	versionKey := versionKeyOf(ctx, key)

	// 模拟 hook 对 key 和 versionKey 分别做一次整串前置拼接
	finalKey := simulateHookPrefix("ok", key)
	finalVersionKey := simulateHookPrefix("ok", versionKey)

	tagOfKey := extractHashTag(finalKey)
	tagOfVersionKey := extractHashTag(finalVersionKey)

	if tagOfKey != tagOfVersionKey {
		t.Fatalf("key 与 versionKey 加前缀后 hash tag 应一致（同 slot），got key tag=%q versionKey tag=%q\nfinalKey=%q\nfinalVersionKey=%q",
			tagOfKey, tagOfVersionKey, finalKey, finalVersionKey)
	}

	// finalKey 和 finalVersionKey 都应以租户前缀开头（数据隔离要求：前缀必须在 key 最前面）
	if !strings.HasPrefix(finalKey, "tenantok:") {
		t.Errorf("finalKey 应以 tenantok: 开头, got %q", finalKey)
	}
	if !strings.HasPrefix(finalVersionKey, "tenantok:") {
		t.Errorf("finalVersionKey 应以 tenantok: 开头（数据隔离要求）, got %q", finalVersionKey)
	}

	// key 本身完全不受 versionKeyOf 影响，兼容线上已有数据
	if finalKey != "tenantok:"+key {
		t.Errorf("key 的加前缀结果不应受 versionKeyOf 改动影响, got %q", finalKey)
	}
}

// 无租户（或兜底租户模式）：FmtByTypeOnCtx 原样返回 key，
// versionKeyOf 行为应与改动前完全一致（回归保护）。
func TestVersionKeyOf_NoTenant_Unchanged(t *testing.T) {
	env.SetFallbackTenantName("")

	key := "ca-activity:bet_amount:wait:sync:43"
	versionKey := versionKeyOf(context.Background(), key)

	want := "{" + key + "}:version"
	if versionKey != want {
		t.Errorf("无租户时 versionKeyOf 应保持原样, got %q want %q", versionKey, want)
	}
}

// 兜底租户模式：即使 ctx 携带租户，也不应加前缀，versionKeyOf 行为与无租户一致。
func TestVersionKeyOf_FallbackTenant_Unchanged(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	key := "ca-activity:bet_amount:wait:sync:43"
	versionKey := versionKeyOf(versionKeyTestCtx("ok"), key)

	want := "{" + key + "}:version"
	if versionKey != want {
		t.Errorf("兜底租户模式下 versionKeyOf 应保持原样, got %q want %q", versionKey, want)
	}
}
