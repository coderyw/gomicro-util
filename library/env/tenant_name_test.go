package env

import (
	"github.com/coderyw/gomicro-util/model/cst"
	"strings"
	"testing"
)

func TestTenantNameCaseConfig(t *testing.T) {
	old := tenantNameCase
	t.Cleanup(func() { tenantNameCase = old })
	for _, tc := range []struct{ value, want string }{
		{"", "lower"}, {"lower", "lower"}, {"LOWER", "lower"}, {" Lower \n", "lower"},
		{" UPPER\t", "upper"}, {"Original", "original"}, {" \n", "lower"},
	} {
		if err := SetTenantNameCase(tc.value); err != nil {
			t.Fatal(err)
		}
		if got := GetTenantNameCase(); got != tc.want {
			t.Fatalf("%q: got %q, want %q", tc.value, got, tc.want)
		}
	}
	if err := SetTenantNameCase("invalid"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if GetTenantNameCase() != "lower" {
		t.Fatal("invalid setter changed active mode")
	}
	// Simulate an invalid startup environment value: validate via the normal error path.
	tenantNameCase = "invalid"
	if err := SyncEnv(); err == nil || !strings.Contains(err.Error(), cst.ENV_TENANT_NAME_CASE) {
		t.Fatalf("startup should return configuration error: %v", err)
	}
}
