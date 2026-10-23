package callback

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ==========================================================
// Mock infrastructure
// ==========================================================

type fakeConnPool struct{}

func (f *fakeConnPool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, nil
}
func (f *fakeConnPool) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}
func (f *fakeConnPool) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return nil, nil
}
func (f *fakeConnPool) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

// ==========================================================
// Test model: 模拟 gorm-gen 生成的 model，TableName 带库名
// ==========================================================

type testSessionModel struct {
	ID        int64  `gorm:"column:id;primaryKey"`
	StartTime string `gorm:"column:start_time"`
	EndTime   string `gorm:"column:end_time"`
}

func (testSessionModel) TableName() string {
	return "ca_activity_redpacket.b_redpacket_activity_session"
}

// ==========================================================
// Test helper: 创建 DryRun + Plugin 的 gorm.DB
// ==========================================================

func newTestDB(t *testing.T, namespace string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      &fakeConnPool{},
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("newTestDB: %v", err)
	}

	plugin := NewPlugin(Config{
		NamespaceResolver: func(ctx context.Context) string {
			return namespace
		},
		EnableRawSQLRewrite: true,
	})
	if err := db.Use(plugin); err != nil {
		t.Fatalf("newTestDB: plugin register failed: %v", err)
	}

	return db
}

// ==========================================================
// 1. normalizeIdent
// ==========================================================

func TestNormalizeIdent(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"`db_name`", "db_name"},
		{"\"db_name\"", "db_name"},
		{" db_name ", "db_name"},
		{"`ca_activity_redpacket`", "ca_activity_redpacket"},
		{"db_name", "db_name"},
		{"", ""},
		{"  `quoted`  ", "quoted"},
		{"`a`.`b`", "a.b"},
	}
	for _, tt := range tests {
		if got := normalizeIdent(tt.input); got != tt.want {
			t.Errorf("normalizeIdent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ==========================================================
// 2. splitDBTable
// ==========================================================

func TestSplitDBTable(t *testing.T) {
	tests := []struct {
		input   string
		wantDB  string
		wantTbl string
		wantOK  bool
	}{
		{"ca_db.b_table", "ca_db", "b_table", true},
		{"`ca_db`.`b_table`", "ca_db", "b_table", true},
		{"b_table", "", "", false},
		{"", "", "", false},
		{"a.b.c", "", "", false},
		{".b_table", "", "", false},
		{"ca_db.", "", "", false},
		// normalizeIdent 只去反引号和首尾空格，不处理分隔符两侧内部空格
		{" ca_db.b_table ", "ca_db", "b_table", true},
	}
	for _, tt := range tests {
		db, tbl, ok := splitDBTable(tt.input)
		if ok != tt.wantOK || db != tt.wantDB || tbl != tt.wantTbl {
			t.Errorf("splitDBTable(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, db, tbl, ok, tt.wantDB, tt.wantTbl, tt.wantOK)
		}
	}
}

// ==========================================================
// 3. extractDBTableFromSQL
// ==========================================================

func TestExtractDBTableFromSQL(t *testing.T) {
	tests := []struct {
		input   string
		wantDB  string
		wantTbl string
		wantOK  bool
	}{
		{"`ca_db`.`b_table`", "ca_db", "b_table", true},
		{"ca_db.b_table", "ca_db", "b_table", true},
		{"`b_table`", "", "", false},
		{"", "", "", false},
		{"a.b.c", "", "", false},
	}
	for _, tt := range tests {
		db, tbl, ok := extractDBTableFromSQL(tt.input)
		if ok != tt.wantOK || db != tt.wantDB || tbl != tt.wantTbl {
			t.Errorf("extractDBTableFromSQL(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, db, tbl, ok, tt.wantDB, tt.wantTbl, tt.wantOK)
		}
	}
}

// ==========================================================
// 4. rewriteDBName
// ==========================================================

func TestRewriteDBName(t *testing.T) {
	tests := []struct {
		ns, dbName, want string
	}{
		// 正常改写
		{"ok", "ca_activity_redpacket", "ok_ca_activity_redpacket"},
		{"tenant1", "ca_db", "tenant1_ca_db"},
		// 幂等：已带前缀不再重复
		{"ok", "ok_ca_activity_redpacket", "ok_ca_activity_redpacket"},
		{"tenant1", "tenant1_ca_db", "tenant1_ca_db"},
		// 空 namespace → 不改写
		{"", "ca_db", "ca_db"},
		// 空 dbName → 返回空
		{"ok", "", ""},
		{"", "", ""},
		// 系统库 → 不改写
		{"ok", "information_schema", "information_schema"},
		{"ok", "mysql", "mysql"},
		{"ok", "performance_schema", "performance_schema"},
		{"ok", "sys", "sys"},
		// 带反引号 → 正常处理
		{"ok", "`ca_db`", "ok_ca_db"},
	}
	for _, tt := range tests {
		got := rewriteDBName(tt.ns, tt.dbName)
		if got != tt.want {
			t.Errorf("rewriteDBName(%q, %q) = %q, want %q", tt.ns, tt.dbName, got, tt.want)
		}
	}
}

// ==========================================================
// 5. shouldSkipSchema
// ==========================================================

func TestShouldSkipSchema(t *testing.T) {
	skip := []string{"information_schema", "INFORMATION_SCHEMA", "mysql", "MySQL", "performance_schema", "sys", "SYS"}
	for _, s := range skip {
		if !shouldSkipSchema(s) {
			t.Errorf("shouldSkipSchema(%q) should be true", s)
		}
	}
	noSkip := []string{"ca_db", "my_database", "custom_schema"}
	for _, s := range noSkip {
		if shouldSkipSchema(s) {
			t.Errorf("shouldSkipSchema(%q) should be false", s)
		}
	}
}

// ==========================================================
// 6. isDBNameRewritten
// ==========================================================

func TestIsDBNameRewritten(t *testing.T) {
	tests := []struct {
		ns, dbName string
		want       bool
	}{
		{"ok", "ok_ca_db", true},
		{"ok", "ca_db", false},
		{"ok", "ok_", true},
		{"tenant", "tenant_ca_db", true},
		{"tenant", "ca_db", false},
	}
	for _, tt := range tests {
		if got := isDBNameRewritten(tt.ns, tt.dbName); got != tt.want {
			t.Errorf("isDBNameRewritten(%q, %q) = %v, want %v", tt.ns, tt.dbName, got, tt.want)
		}
	}
}

// ==========================================================
// 7. rewriteQualifiedColumn（增强版）
// ==========================================================

func TestRewriteQualifiedColumn_Enhanced(t *testing.T) {
	tests := []struct {
		name      string
		ns        string
		col       clause.Column
		wantTable string
		wantName  string
	}{
		{
			name:      "db.table 格式 → 正常改写",
			ns:        "ok",
			col:       clause.Column{Table: "ca_activity_redpacket.b_redpacket_activity_session", Name: "start_time"},
			wantTable: "ok_ca_activity_redpacket.b_redpacket_activity_session",
			wantName:  "start_time",
		},
		{
			name:      "已带前缀 → 幂等",
			ns:        "ok",
			col:       clause.Column{Table: "ok_ca_activity_redpacket.b_redpacket_activity_session", Name: "start_time"},
			wantTable: "ok_ca_activity_redpacket.b_redpacket_activity_session",
			wantName:  "start_time",
		},
		{
			name:      "空 namespace → 不变",
			ns:        "",
			col:       clause.Column{Table: "ca_activity_redpacket.b_redpacket_activity_session", Name: "start_time"},
			wantTable: "ca_activity_redpacket.b_redpacket_activity_session",
			wantName:  "start_time",
		},
		{
			name:      "裸表名 → 不变（无法拆分出库名）",
			ns:        "ok",
			col:       clause.Column{Table: "b_redpacket_activity_session", Name: "id"},
			wantTable: "b_redpacket_activity_session",
			wantName:  "id",
		},
		{
			name:      "空 table → 不变",
			ns:        "ok",
			col:       clause.Column{Table: "", Name: "id"},
			wantTable: "",
			wantName:  "id",
		},
		{
			name:      "系统库表 → 不改写",
			ns:        "ok",
			col:       clause.Column{Table: "information_schema.tables", Name: "table_name"},
			wantTable: "information_schema.tables",
			wantName:  "table_name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteQualifiedColumn(tt.ns, tt.col)
			if got.Table != tt.wantTable {
				t.Errorf("Table: got %q, want %q", got.Table, tt.wantTable)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name: got %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

// ==========================================================
// 8. rewriteColumnInterface
// ==========================================================

func TestRewriteColumnInterface(t *testing.T) {
	// clause.Column 类型 → 正常改写
	col := clause.Column{Table: "ca_db.b_table", Name: "id"}
	got := rewriteColumnInterface("ok", col)
	c, ok := got.(clause.Column)
	if !ok {
		t.Fatal("expected clause.Column type")
	}
	if c.Table != "ok_ca_db.b_table" {
		t.Errorf("Table: got %q, want %q", c.Table, "ok_ca_db.b_table")
	}

	// 其他类型 → 原样返回
	s := "some_string"
	if rewriteColumnInterface("ok", s) != s {
		t.Error("non-Column type should be returned as is")
	}
}

// ==========================================================
// 9. rewriteSQLWithParser
// ==========================================================

func TestRewriteSQLWithParser_Comprehensive(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		prefix string
		check  func(t *testing.T, result string)
	}{
		{
			name:   "简单 SELECT 带库名前缀",
			sql:    "SELECT * FROM `ca_game`.`b_all_game` WHERE id = 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_game`") {
					t.Errorf("expected `ok_ca_game` in: %s", result)
				}
			},
		},
		{
			name:   "空 prefix → 返回原 SQL",
			sql:    "SELECT * FROM `ca_game`.`b_all_game`",
			prefix: "",
			check: func(t *testing.T, result string) {
				if result != "SELECT * FROM `ca_game`.`b_all_game`" {
					t.Errorf("empty prefix should return original, got: %s", result)
				}
			},
		},
		{
			name:   "SQL 中无点号 → 返回原 SQL",
			sql:    "SELECT * FROM b_all_game WHERE id = 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if result != "SELECT * FROM b_all_game WHERE id = 1" {
					t.Errorf("no dot should return original, got: %s", result)
				}
			},
		},
		{
			name:   "系统库不改写",
			sql:    "SELECT * FROM `information_schema`.`tables`",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if strings.Contains(result, "ok_information_schema") {
					t.Errorf("system schema should not be rewritten: %s", result)
				}
			},
		},
		{
			name:   "WHERE 中带 db.table.column 限定 → 列引用的库名也改写",
			sql:    "SELECT * FROM `ca_db`.`b_table` WHERE `ca_db`.`b_table`.`end_time` > '2026-01-01'",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_db`") {
					t.Errorf("expected ok_ca_db: %s", result)
				}
				// WHERE 列引用中的 ca_db 也应被改写
				cleaned := strings.ReplaceAll(result, "ok_ca_db", "")
				if strings.Contains(cleaned, "ca_db") {
					t.Errorf("WHERE column ref still has unrewritten ca_db: %s", result)
				}
			},
		},
		{
			name:   "多表 JOIN → FROM/ON 中所有库名都改写",
			sql:    "SELECT * FROM `ca_db`.`t1` JOIN `ca_db2`.`t2` ON `ca_db`.`t1`.`id` = `ca_db2`.`t2`.`t1_id`",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_db`") {
					t.Errorf("expected ok_ca_db: %s", result)
				}
				if !strings.Contains(result, "`ok_ca_db2`") {
					t.Errorf("expected ok_ca_db2: %s", result)
				}
				// ON 条件中的列引用库名也应被改写
				cleaned := strings.ReplaceAll(result, "ok_ca_db2", "")
				cleaned = strings.ReplaceAll(cleaned, "ok_ca_db", "")
				if strings.Contains(cleaned, "ca_db") {
					t.Errorf("ON clause still has unrewritten db ref: %s", result)
				}
			},
		},
		{
			name:   "幂等：已带前缀不双写",
			sql:    "SELECT * FROM `ok_ca_db`.`b_table` WHERE `ok_ca_db`.`b_table`.`id` = 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if strings.Contains(result, "ok_ok_ca_db") {
					t.Errorf("double prefix found: %s", result)
				}
			},
		},
		{
			name:   "复现 bug SQL（完整生产场景）",
			sql:    "SELECT * FROM `ca_activity_redpacket`.`b_redpacket_activity_session` WHERE `ca_activity_redpacket`.`b_redpacket_activity_session`.`end_time` > '2026-03-27 09:44:00.918' ORDER BY `ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time` ASC,`b_redpacket_activity_session`.`id` LIMIT 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_activity_redpacket`") {
					t.Errorf("expected ok_ca_activity_redpacket: %s", result)
				}
				// 不应存在未改写的 ca_activity_redpacket（排除已改写的）
				cleaned := strings.ReplaceAll(result, "ok_ca_activity_redpacket", "")
				if strings.Contains(cleaned, "ca_activity_redpacket") {
					t.Errorf("unrewritten ca_activity_redpacket found: %s", result)
				}
			},
		},
		{
			name:   "INSERT 语句",
			sql:    "INSERT INTO `ca_db`.`b_table` (`id`, `name`) VALUES (1, 'test')",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_db`") {
					t.Errorf("expected ok_ca_db in INSERT: %s", result)
				}
			},
		},
		{
			name:   "UPDATE 语句",
			sql:    "UPDATE `ca_db`.`b_table` SET `name` = 'new' WHERE `id` = 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_db`") {
					t.Errorf("expected ok_ca_db in UPDATE: %s", result)
				}
			},
		},
		{
			name:   "DELETE 语句",
			sql:    "DELETE FROM `ca_db`.`b_table` WHERE `id` = 1",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				if !strings.Contains(result, "`ok_ca_db`") {
					t.Errorf("expected ok_ca_db in DELETE: %s", result)
				}
			},
		},
		{
			name:   "子查询",
			sql:    "SELECT * FROM `ca_db`.`t1` WHERE `id` IN (SELECT `t1_id` FROM `ca_db`.`t2`)",
			prefix: "ok",
			check: func(t *testing.T, result string) {
				// 主查询和子查询中的库名都应改写
				count := strings.Count(result, "`ok_ca_db`")
				if count < 2 {
					t.Errorf("expected at least 2 occurrences of ok_ca_db, got %d: %s", count, result)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, ok := rewriteSQLWithParser(tt.sql, tt.prefix)
			if !ok {
				t.Fatalf("改写未通过一致性校验: %s", tt.sql)
			}
			t.Logf("input:  %s", tt.sql)
			t.Logf("output: %s", result)
			tt.check(t, result)
		})
	}
}

// ==========================================================
// 10. 端到端集成测试：Query + namespace
// ==========================================================

func TestPlugin_Query_WithNamespace(t *testing.T) {
	db := newTestDB(t, "ok")

	var results []testSessionModel
	result := db.
		Where("`ca_activity_redpacket`.`b_redpacket_activity_session`.`end_time` > ?", "2026-03-27 09:44:00.918").
		Order(clause.OrderByColumn{
			Column: clause.Column{Table: "ca_activity_redpacket.b_redpacket_activity_session", Name: "start_time"},
			Desc:   false,
		}).
		Limit(1).
		Find(&results)

	if result.Error != nil {
		t.Fatalf("query error: %v", result.Error)
	}

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	// FROM 子句应包含重写后的库名
	if !strings.Contains(sql, "`ok_ca_activity_redpacket`") {
		t.Errorf("FROM should contain rewritten db name, got: %s", sql)
	}
}

// ==========================================================
// 11. 端到端集成测试：Query + 空 namespace
// ==========================================================

func TestPlugin_Query_WithoutNamespace(t *testing.T) {
	db := newTestDB(t, "")

	var results []testSessionModel
	result := db.
		Where("end_time > ?", "2026-03-27 09:44:00.918").
		Order("start_time ASC").
		Limit(1).
		Find(&results)

	if result.Error != nil {
		t.Fatalf("query error: %v", result.Error)
	}

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	if strings.Contains(sql, "ok_ca_activity_redpacket") {
		t.Errorf("should not rewrite when namespace is empty: %s", sql)
	}
}

// ==========================================================
// 12. 端到端集成测试：db.Table("db.table") 方式
// ==========================================================

func TestPlugin_Query_TableMethod(t *testing.T) {
	db := newTestDB(t, "ok")

	var results []map[string]interface{}
	result := db.
		Table("ca_activity_redpacket.b_redpacket_activity_session").
		Where("end_time > ?", "2026-03-27 09:44:00.918").
		Order("start_time ASC, id").
		Limit(1).
		Find(&results)

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	if !strings.Contains(sql, "`ok_ca_activity_redpacket`") {
		t.Errorf("expected rewritten db name in Table() mode: %s", sql)
	}
}

// ==========================================================
// 13. 端到端集成测试：WHERE 用 clause.Gt 类型条件
// ==========================================================

func TestPlugin_Query_TypedWhereCondition(t *testing.T) {
	db := newTestDB(t, "ok")

	var results []testSessionModel
	result := db.
		Clauses(clause.Where{
			Exprs: []clause.Expression{
				clause.Gt{
					Column: clause.Column{
						Table: "ca_activity_redpacket.b_redpacket_activity_session",
						Name:  "end_time",
					},
					Value: "2026-03-27 09:44:00.918",
				},
			},
		}).
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Table: "ca_activity_redpacket.b_redpacket_activity_session",
				Name:  "start_time",
			},
		}).
		Limit(1).
		Find(&results)

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	// WHERE 中的 column 引用也应被改写
	if !strings.Contains(sql, "`ok_ca_activity_redpacket`") {
		t.Errorf("typed WHERE column should be rewritten: %s", sql)
	}
}

// ==========================================================
// 14. 端到端集成测试：ORDER BY 带 db.table 限定
// ==========================================================

func TestPlugin_Query_OrderByWithDBPrefix(t *testing.T) {
	db := newTestDB(t, "ok")

	var results []testSessionModel
	result := db.
		Where("end_time > ?", "2026-03-27 09:44:00.918").
		Order(clause.OrderByColumn{
			Column: clause.Column{
				Table: "ca_activity_redpacket.b_redpacket_activity_session",
				Name:  "start_time",
			},
		}).
		Limit(1).
		Find(&results)

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	if strings.Contains(sql, "`ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time`") &&
		!strings.Contains(sql, "`ok_ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time`") {
		t.Errorf("ORDER BY column db prefix should be rewritten: %s", sql)
	}
}

// ==========================================================
// 15. 回归测试：精确复现 cron job 的 bug 场景
// ==========================================================

func TestPlugin_CronJobBugReproduction(t *testing.T) {
	t.Run("空_namespace_定时任务场景", func(t *testing.T) {
		db := newTestDB(t, "")

		var results []testSessionModel
		result := db.
			Where("`ca_activity_redpacket`.`b_redpacket_activity_session`.`end_time` > ?", "2026-03-27 09:44:00.918").
			Order("`ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time` ASC,`b_redpacket_activity_session`.`id`").
			Limit(1).
			Find(&results)

		sql := result.Statement.SQL.String()
		t.Logf("SQL (empty ns): %s", sql)

		// 空 namespace → 不应有任何前缀改写
		if strings.Contains(sql, "ok_") {
			t.Errorf("empty namespace should not add any prefix: %s", sql)
		}
	})

	t.Run("有_namespace_正常场景", func(t *testing.T) {
		db := newTestDB(t, "ok")

		var results []testSessionModel
		result := db.
			Where("`ca_activity_redpacket`.`b_redpacket_activity_session`.`end_time` > ?", "2026-03-27 09:44:00.918").
			Order("`ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time` ASC,`b_redpacket_activity_session`.`id`").
			Limit(1).
			Find(&results)

		sql := result.Statement.SQL.String()
		t.Logf("SQL (ok ns): %s", sql)

		// FROM 子句应包含重写后的库名
		if !strings.Contains(sql, "`ok_ca_activity_redpacket`") {
			t.Errorf("expected ok_ca_activity_redpacket in SQL: %s", sql)
		}

		// 不应存在未改写的 ca_activity_redpacket（排除已改写的）
		cleaned := strings.ReplaceAll(sql, "ok_ca_activity_redpacket", "")
		if strings.Contains(cleaned, "ca_activity_redpacket") {
			t.Errorf("found unrewritten ca_activity_redpacket: %s", sql)
		}
	})
}

// ==========================================================
// 16. 多次改写幂等性测试
// ==========================================================

func TestPlugin_IdempotentRewrite(t *testing.T) {
	// 第一次改写
	sql1, _, _ := rewriteSQLWithParser(
		"SELECT * FROM `ca_db`.`b_table` WHERE `ca_db`.`b_table`.`id` = 1",
		"ok",
	)
	// 对改写结果再次改写
	sql2, _, _ := rewriteSQLWithParser(sql1, "ok")

	if sql1 != sql2 {
		t.Errorf("rewrite is not idempotent:\n  first:  %s\n  second: %s", sql1, sql2)
	}
}

// ==========================================================
// 17. SELECT 列带 db.table 限定
// ==========================================================

func TestPlugin_Query_SelectColumnsWithDBPrefix(t *testing.T) {
	db := newTestDB(t, "ok")

	var results []testSessionModel
	result := db.
		Select(
			clause.Column{Table: "ca_activity_redpacket.b_redpacket_activity_session", Name: "id"},
			clause.Column{Table: "ca_activity_redpacket.b_redpacket_activity_session", Name: "start_time"},
		).
		Where("end_time > ?", "2026-03-27 09:44:00.918").
		Limit(1).
		Find(&results)

	sql := result.Statement.SQL.String()
	t.Logf("Generated SQL: %s", sql)

	if strings.Contains(sql, "`ca_activity_redpacket`.`b_redpacket_activity_session`.`id`") &&
		!strings.Contains(sql, "`ok_ca_activity_redpacket`") {
		t.Errorf("SELECT columns should be rewritten: %s", sql)
	}
}

// ==========================================================
// 18. rewriteClauseTable 单元测试
// ==========================================================

func TestRewriteClauseTable(t *testing.T) {
	db := newTestDB(t, "ok")
	p := &Plugin{cfg: Config{
		NamespaceResolver: func(ctx context.Context) string { return "ok" },
	}}

	stmt := &gorm.Statement{DB: db}

	tests := []struct {
		name  string
		input clause.Table
		check func(t *testing.T, result clause.Table)
	}{
		{
			name:  "db.table 格式",
			input: clause.Table{Name: "ca_db.b_table"},
			check: func(t *testing.T, result clause.Table) {
				if !strings.Contains(result.Name, "ok_ca_db") {
					t.Errorf("expected ok_ca_db in Name: %q", result.Name)
				}
				if !result.Raw {
					t.Error("Raw should be true after rewrite")
				}
			},
		},
		{
			name:  "反引号格式",
			input: clause.Table{Name: "`ca_db`.`b_table`"},
			check: func(t *testing.T, result clause.Table) {
				if !strings.Contains(result.Name, "ok_ca_db") {
					t.Errorf("expected ok_ca_db: %q", result.Name)
				}
			},
		},
		{
			name:  "空 Name → 不变",
			input: clause.Table{Name: ""},
			check: func(t *testing.T, result clause.Table) {
				if result.Name != "" {
					t.Errorf("expected empty Name: %q", result.Name)
				}
			},
		},
		{
			name:  "保留 alias",
			input: clause.Table{Name: "ca_db.b_table", Alias: "t1"},
			check: func(t *testing.T, result clause.Table) {
				if result.Alias != "t1" {
					t.Errorf("Alias should be preserved: %q", result.Alias)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.rewriteClauseTable(stmt, tt.input, "ok")
			tt.check(t, result)
		})
	}
}

// ==========================================================
// 19. 端到端集成测试：配置了兜底租户（TENANT_FALLBACK）→ 不改写（数据保持原始库名）
// ==========================================================

func TestPlugin_FallbackTenant(t *testing.T) {
	// 模拟配置了兜底租户的 pod：设置 TENANT_FALLBACK，结束后还原，避免影响其它用例
	env.SetFallbackTenantName("ns")
	defer env.SetFallbackTenantName("")

	// Query 路径（queryWithRewrite）：即使 NamespaceResolver 返回 ok 也不应改写
	t.Run("Query", func(t *testing.T) {
		db := newTestDB(t, "ok")
		var results []testSessionModel
		result := db.
			Where("end_time > ?", "2026-03-27 09:44:00.918").
			Order("start_time ASC").
			Limit(1).
			Find(&results)
		if result.Error != nil {
			t.Fatalf("query error: %v", result.Error)
		}
		sql := result.Statement.SQL.String()
		t.Logf("Generated SQL: %s", sql)
		if strings.Contains(sql, "ok_ca_activity_redpacket") {
			t.Errorf("配置兜底租户后不应改写库名，got: %s", sql)
		}
	})

	// 写入路径（rewrite Before hook）：Create 也不应改写
	t.Run("Create", func(t *testing.T) {
		db := newTestDB(t, "ok")
		result := db.Create(&testSessionModel{ID: 1, StartTime: "2026-01-01", EndTime: "2026-01-02"})
		sql := result.Statement.SQL.String()
		t.Logf("Generated SQL: %s", sql)
		if strings.Contains(sql, "ok_ca_activity_redpacket") {
			t.Errorf("配置兜底租户后 Create 不应改写库名，got: %s", sql)
		}
	})

	// 更新路径（rewrite Before hook）：Update 也不应改写
	t.Run("Update", func(t *testing.T) {
		db := newTestDB(t, "ok")
		result := db.Model(&testSessionModel{}).Where("id = ?", 1).Update("end_time", "2026-02-01")
		sql := result.Statement.SQL.String()
		t.Logf("Generated SQL: %s", sql)
		if strings.Contains(sql, "ok_ca_activity_redpacket") {
			t.Errorf("配置兜底租户后 Update 不应改写库名，got: %s", sql)
		}
	})

	// 删除路径（rewrite Before hook）：Delete 也不应改写
	t.Run("Delete", func(t *testing.T) {
		db := newTestDB(t, "ok")
		result := db.Delete(&testSessionModel{}, 1)
		sql := result.Statement.SQL.String()
		t.Logf("Generated SQL: %s", sql)
		if strings.Contains(sql, "ok_ca_activity_redpacket") {
			t.Errorf("配置兜底租户后 Delete 不应改写库名，got: %s", sql)
		}
	})

	// Raw SQL 路径（EnableRawSQLRewrite）：原生 SQL 也不应改写
	t.Run("Raw", func(t *testing.T) {
		db := newTestDB(t, "ok")
		var results []testSessionModel
		result := db.Raw("SELECT * FROM ca_activity_redpacket.b_redpacket_activity_session WHERE id = ?", 1).Scan(&results)
		sql := result.Statement.SQL.String()
		t.Logf("Generated SQL: %s", sql)
		if strings.Contains(sql, "ok_ca_activity_redpacket") {
			t.Errorf("配置兜底租户后 Raw SQL 不应改写库名，got: %s", sql)
		}
	})
}

// ==========================================================
// Benchmark
// ==========================================================

func BenchmarkRewriteSQLWithParser(b *testing.B) {
	sql := "SELECT * FROM `ca_activity_redpacket`.`b_redpacket_activity_session` WHERE `ca_activity_redpacket`.`b_redpacket_activity_session`.`end_time` > '2026-03-27 09:44:00.918' ORDER BY `ca_activity_redpacket`.`b_redpacket_activity_session`.`start_time` ASC,`b_redpacket_activity_session`.`id` LIMIT 1"
	for i := 0; i < b.N; i++ {
		rewriteSQLWithParser(sql, "ok")
	}
}
