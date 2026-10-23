package callback

import (
	"github.com/coderyw/gomicro-util/library/env"
	"strings"
	"testing"
)

func TestTenantCaseSQLCompatibility(t *testing.T) {
	env.SetFallbackTenantName("")
	for _, ns := range []string{"ok", "OK", "Ok"} {
		db := newTestDB(t, ns)
		var rows []testSessionModel
		for _, query := range []string{"SELECT * FROM ca_activity_redpacket.b_redpacket_activity_session WHERE start_time = ?", "SELECT * FROM ok_ca_activity_redpacket.b_redpacket_activity_session WHERE start_time = ?"} {
			result := db.Raw(query, "UPPER_VALUE").Find(&rows)
			sql := result.Statement.SQL.String()
			if result.Error != nil || !strings.Contains(sql, "`ok_ca_activity_redpacket`") || strings.Contains(sql, "ok_ok_") {
				t.Fatalf("%s: %s, %v", ns, sql, result.Error)
			}
			if len(result.Statement.Vars) != 1 || result.Statement.Vars[0] != "UPPER_VALUE" {
				t.Fatalf("vars changed: %v", result.Statement.Vars)
			}
		}
		result := db.Find(&rows)
		if sql := result.Statement.SQL.String(); !strings.Contains(sql, "`ok_ca_activity_redpacket`") || strings.Contains(sql, "ok_ok_") {
			t.Fatal(sql)
		}
	}
}
