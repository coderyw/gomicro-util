package namespace

import (
	"fmt"

	"github.com/coderyw/gomicro-util/library/metadata"
)

func ExampleGetTenantsInNamespace() {
	ts := GetTenantsInNamespace()
	for _, ctx := range ts {
		tenantName := metadata.Get[string](ctx, metadata.CONTEXT_NAME_SPACE)
		fmt.Println(tenantName)
	}
}
