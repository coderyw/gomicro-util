package env

import (
	"fmt"
	"os"
	"strings"

	"github.com/coderyw/gomicro-util/model/cst"
)

// Read once at startup, as with other env configuration. Keep invalid values
// for checkEnv/SyncEnv to report instead of panicking during package init.
var tenantNameCase = normalizeTenantNameCase(os.Getenv(cst.ENV_TENANT_NAME_CASE))

func normalizeTenantNameCase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "lower"
	}
	return value
}

func parseTenantNameCase(value string) (string, error) {
	value = normalizeTenantNameCase(value)
	switch value {
	case "lower", "upper", "original":
		return value, nil
	default:
		return "", fmt.Errorf("%s must be lower, upper or original, got %q", cst.ENV_TENANT_NAME_CASE, value)
	}
}

// GetTenantNameCase returns the middleware tenant naming mode.
// Applications must call SyncEnv before using middleware to validate startup configuration.
func GetTenantNameCase() string { return tenantNameCase }

// SetTenantNameCase validates startup/test configuration. Do not change the mode
// while requests are running: resource names must remain stable within a process.
func SetTenantNameCase(value string) error {
	mode, err := parseTenantNameCase(value)
	if err != nil {
		return err
	}
	tenantNameCase = mode
	return nil
}
