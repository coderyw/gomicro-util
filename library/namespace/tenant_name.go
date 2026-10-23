package namespace

import (
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/model/cst"
	"strings"
)

// EnvTenantNameCase is the deployment configuration key; env owns its parsing.
const EnvTenantNameCase = cst.ENV_TENANT_NAME_CASE

// NormalizeTenantName formats a tenant name without changing context metadata,
// resource suffixes, delimiters, or fallback behavior. Empty names stay empty.
func NormalizeTenantName(name string) string {
	switch env.GetTenantNameCase() {
	case "upper":
		return strings.ToUpper(name)
	case "original":
		return name
	default:
		return strings.ToLower(name)
	}
}
