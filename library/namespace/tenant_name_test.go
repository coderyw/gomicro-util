package namespace

import (
	"context"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/metadata"
)

func TestTenantNameCaseCompatibility(t *testing.T) {
	old := env.GetTenantNameCase()
	t.Cleanup(func() { _ = env.SetTenantNameCase(old) })
	env.SetFallbackTenantName("")
	// Fixed expected names deliberately capture the existing naming contract.
	for _, tc := range []struct {
		kind   NamespaceFmt
		before string
	}{
		{NAMESPACE_TYPE_MYSQL, "ok_Order/ID"},
		{NAMESPACE_TYPE_REDIS, "tenantok:Order/ID"},
		{NAMESPACE_TYPE_KAFKA, "tenantok_Order/ID"},
		{NAMESPACE_TYPE_COMMON, "tenantok:Order/ID"},
		{NAMESPACE_TYPE_OSS, "tenantok/Order/ID"},
	} {
		for _, mode := range []string{"lower", "upper", "original"} {
			if err := env.SetTenantNameCase(mode); err != nil {
				t.Fatal(err)
			}
			name, expectedTenant := "Ok", "ok"
			if mode == "upper" {
				expectedTenant = "OK"
			}
			if mode == "original" {
				expectedTenant = "Ok"
			}
			expected := tc.before
			// Only substitute the known tenant segment of the legacy expected name.
			if tc.kind == NAMESPACE_TYPE_MYSQL {
				expected = expectedTenant + "_Order/ID"
			} else {
				expected = "tenant" + expectedTenant + tc.before[len("tenantok"):]
			}
			ctx := metadata.SetCtxData(context.Background(), metadata.CONTEXT_NAME_SPACE, name)
			if got := FmtByTypeOnCtx(ctx, tc.kind, "Order/ID"); got != expected {
				t.Fatalf("%s/%s: %q != %q", tc.kind, mode, got, expected)
			}
			if got := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE); got != name {
				t.Fatal("context metadata changed")
			}
			if got := FmtByType(tc.kind, "", "Order/ID"); got != "Order/ID" {
				t.Fatal(got)
			}
			env.SetFallbackTenantName("ns")
			got := FmtByTypeOnCtx(ctx, tc.kind, "Order/ID")
			env.SetFallbackTenantName("")
			if got != "Order/ID" {
				t.Fatal(got)
			}
		}
		if err := env.SetTenantNameCase("lower"); err != nil {
			t.Fatal(err)
		}
		if got := FmtByType(tc.kind, "ok", "Order/ID"); got != tc.before {
			t.Fatalf("legacy name changed: %q != %q", got, tc.before)
		}
	}
	if err := env.SetTenantNameCase("lower"); err != nil {
		t.Fatal(err)
	}
	if got := FmtByTypeOnCtx(context.Background(), NAMESPACE_TYPE_OSS, "Image.PNG", "NS"); got != "tenantns/Image.PNG" {
		t.Fatal(got)
	}
}
