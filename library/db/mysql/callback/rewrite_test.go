package callback

import (
	"context"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// ==========================================================
// rewrite（Before hook）单元测试
//
// 覆盖 database.go rewrite 的全部分支：
//  1. 正向：Create/Update/Delete/Raw/Exec 携带租户 → 库名加前缀
//  2. 反向：ns 为空 / resolver 未配置 / 系统库 → 不改写
//  3. 边界：nil 入参、AutoMigrate（Schema.Table 为空）、EnableRawSQLRewrite 关闭
//
// 兜底租户（TENANT_FALLBACK）分支由 TestPlugin_FallbackTenant 覆盖。
// ==========================================================

// newRewriteTestDB 与 newTestDB 相同，但允许自定义 Config
func newRewriteTestDB(t *testing.T, cfg Config) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      &fakeConnPool{},
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("newRewriteTestDB: %v", err)
	}
	if err := db.Use(NewPlugin(cfg)); err != nil {
		t.Fatalf("newRewriteTestDB: plugin register failed: %v", err)
	}
	return db
}

// ----------------------------------------------------------
// 正向：携带租户时各写入路径的库名都应加前缀
// ----------------------------------------------------------

func TestRewrite_Create_WithNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	result := db.Create(&testSessionModel{ID: 1, StartTime: "2026-01-01", EndTime: "2026-01-02"})
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("Create 应改写库名为 ok_ca_activity_redpacket，got: %s", sql)
	}
}

func TestRewrite_Update_WithNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	result := db.Model(&testSessionModel{}).Where("id = ?", 1).Update("end_time", "2026-02-01")
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("Update 应改写库名为 ok_ca_activity_redpacket，got: %s", sql)
	}
}

func TestRewrite_Delete_WithNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	result := db.Delete(&testSessionModel{}, 1)
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("Delete 应改写库名为 ok_ca_activity_redpacket，got: %s", sql)
	}
}

func TestRewrite_Raw_WithNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	var results []testSessionModel
	result := db.Raw("SELECT * FROM ca_activity_redpacket.b_redpacket_activity_session WHERE id = ?", 1).Scan(&results)
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("Raw SQL 应改写库名为 ok_ca_activity_redpacket，got: %s", sql)
	}
}

func TestRewrite_Exec_WithNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	result := db.Exec("UPDATE ca_activity_redpacket.b_redpacket_activity_session SET end_time = ? WHERE id = ?", "2026-02-01", 1)
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("Exec SQL 应改写库名为 ok_ca_activity_redpacket，got: %s", sql)
	}
}

// Update 的 Where 里带子查询：应通过 rewriteValue 递归改写子查询的表
func TestRewrite_Update_WithSubQuery(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	sub := db.Table("ca_activity_redpacket.b_other_table").Select("id")
	result := db.Model(&testSessionModel{}).
		Where("id IN (?)", sub).
		Update("end_time", "2026-02-01")
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("子查询的库名也应改写为 ok_ca_activity_redpacket，got: %s", sql)
	}
	if strings.Contains(strings.ReplaceAll(sql, "ok_ca_activity_redpacket", ""), "ca_activity_redpacket") {
		t.Errorf("SQL 中不应残留未改写的库名，got: %s", sql)
	}
}

// ----------------------------------------------------------
// 反向：不应改写的场景
// ----------------------------------------------------------

// resolver 返回空租户：保持原始库名
func TestRewrite_EmptyNamespace(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "")

	result := db.Create(&testSessionModel{ID: 1, StartTime: "2026-01-01", EndTime: "2026-01-02"})
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if !strings.Contains(sql, "ca_activity_redpacket") || strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("空租户时应保持原始库名 ca_activity_redpacket，got: %s", sql)
	}
}

// 未配置 NamespaceResolver：保持原始库名
func TestRewrite_NilResolver(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newRewriteTestDB(t, Config{EnableRawSQLRewrite: true})

	result := db.Create(&testSessionModel{ID: 1, StartTime: "2026-01-01", EndTime: "2026-01-02"})
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if strings.Contains(sql, "_ca_activity_redpacket") {
		t.Errorf("未配置 resolver 时不应改写库名，got: %s", sql)
	}
}

// 系统库（mysql/information_schema 等）：即使携带租户也不改写
func TestRewrite_SystemSchemaSkipped(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	result := db.Table("mysql.user").Create(map[string]interface{}{"id": 1})
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if strings.Contains(sql, "ok_mysql") {
		t.Errorf("系统库 mysql 不应加租户前缀，got: %s", sql)
	}
}

// 关闭 EnableRawSQLRewrite：Raw SQL 不做兜底改写
func TestRewrite_RawSQLRewriteDisabled(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newRewriteTestDB(t, Config{
		NamespaceResolver:   func(ctx context.Context) string { return "ok" },
		EnableRawSQLRewrite: false,
	})

	result := db.Exec("UPDATE ca_activity_redpacket.b_redpacket_activity_session SET end_time = ? WHERE id = ?", "2026-02-01", 1)
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("关闭 EnableRawSQLRewrite 后 Raw SQL 不应被改写，got: %s", sql)
	}
}

// 兜底：配置了兜底租户（TENANT_FALLBACK）后，rewrite 直接短路：
// 不改写库名，且不触发 NamespaceResolver（守卫在 resolver 之前）
func TestRewrite_FallbackTenant(t *testing.T) {
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	resolverCalled := false
	db := newRewriteTestDB(t, Config{
		NamespaceResolver: func(ctx context.Context) string {
			resolverCalled = true
			return "ok"
		},
		EnableRawSQLRewrite: true,
	})

	// 结构化改写路径：Create 不应改写
	result := db.Create(&testSessionModel{ID: 1, StartTime: "2026-01-01", EndTime: "2026-01-02"})
	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("兜底租户模式下 Create 不应改写库名，got: %s", sql)
	}

	// parser 兜底路径：即使开启 EnableRawSQLRewrite，Exec 也不应改写
	result = db.Exec("UPDATE ca_activity_redpacket.b_redpacket_activity_session SET end_time = ? WHERE id = ?", "2026-02-01", 1)
	sql = result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)
	if strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("兜底租户模式下 Exec 不应改写库名，got: %s", sql)
	}

	if resolverCalled {
		t.Error("兜底租户模式下不应触发 NamespaceResolver")
	}
}

// ----------------------------------------------------------
// 边界：nil 入参与 AutoMigrate 场景
// ----------------------------------------------------------

// nil / 不完整入参不应 panic
func TestRewrite_NilSafety(t *testing.T) {
	env.SetFallbackTenantName("")
	p := NewPlugin(Config{
		NamespaceResolver: func(ctx context.Context) string { return "ok" },
	})

	p.rewrite(nil)
	p.rewrite(&gorm.DB{})
	p.rewrite(&gorm.DB{Statement: &gorm.Statement{}}) // Context 为 nil
}

// AutoMigrate 场景（Schema 非空但 Schema.Table 为空）：直接跳过，不触发 resolver
func TestRewrite_SkipAutoMigrate(t *testing.T) {
	env.SetFallbackTenantName("")

	resolverCalled := false
	p := NewPlugin(Config{
		NamespaceResolver: func(ctx context.Context) string {
			resolverCalled = true
			return "ok"
		},
	})

	tx := &gorm.DB{Statement: &gorm.Statement{
		Context: context.Background(),
		Schema:  &schema.Schema{},
	}}
	p.rewrite(tx)

	if resolverCalled {
		t.Error("AutoMigrate 场景（Schema.Table 为空）不应触发 NamespaceResolver")
	}
}
