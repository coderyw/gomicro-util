package callback

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
	mysqldriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ==========================================================
// Raw / Exec 路径：与改动前的行为逐场景对照
//
// Raw SQL 与 GORM 构建的语句有本质区别：SQL 是人写的，可能本来就写成
// `LIMIT ?,?`（offset 在前）、字符串常量里可能含 ?、参数个数可能与占位符对不上，
// 而且 Raw 会先后经过 rewrite 与 queryWithRewrite 两个回调各改写一次。
//
// 契约：
//   1. 发给 MySQL 的 SQL 文本必须与改动前逐字节一致
//   2. 参数只在「改动前算错的」语句上变，其余一个字不动
// ==========================================================

func newPlainDBForRaw(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysqldriver.New(mysqldriver.Config{
		DSN: "u:p@tcp(127.0.0.1:3306)/x", SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRawSQL_SameAsBeforeExceptFixedPagination(t *testing.T) {
	env.SetFallbackTenantName("")

	cases := []struct {
		name     string
		sql      string
		args     []interface{}
		exec     bool
		wantVars string // 期望的最终参数；与「构建后的参数」不同即表示被修正
	}{
		{
			name: "LIMIT ? OFFSET ? —— 修正",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `a`=? ORDER BY `id` LIMIT ? OFFSET ?",
			args: []interface{}{1, 10, 20}, wantVars: "[1 20 10]",
		},
		{
			// 最关键的一条：raw SQL 本来就写成逗号形式，参数顺序本就是 [offset, limit]，
			// 再换一次就错了。必须保持不变。
			name: "已写成 LIMIT ?,? —— 不得二次交换",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `a`=? LIMIT ?,?",
			args: []interface{}{1, 20, 10}, wantVars: "[1 20 10]",
		},
		{
			name: "IN (?) 切片展开 + 分页 —— 只动末尾两个",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `id` IN (?) AND `a`=? LIMIT ? OFFSET ?",
			args: []interface{}{[]int{7, 8, 9}, 1, 10, 20}, wantVars: "[7 8 9 1 20 10]",
		},
		{
			// GORM 的 clause.Expr.Build 逐字节扫 ?，不识别字符串边界；
			// 但发给 MySQL 时它在引号内，parser 也不把它算作占位符，两边口径一致。
			name: "字符串常量里含 ?",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `n` LIKE '%?%' AND `a`=? LIMIT ? OFFSET ?",
			args: []interface{}{1, 10, 20}, wantVars: "[1 20 10]",
		},
		{
			name: "注释里含 ?",
			sql:  "SELECT * FROM `ca_base`.`t` /* ? */ WHERE `a`=? LIMIT ? OFFSET ?",
			args: []interface{}{1, 10, 20}, wantVars: "[1 20 10]",
		},
		{
			name: "命名参数 @ —— 到回调时已是标准 ?",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `a`=@a LIMIT @l OFFSET @o",
			args: []interface{}{map[string]interface{}{"a": 1, "l": 10, "o": 20}}, wantVars: "[1 20 10]",
		},
		{
			name: "Exec UPDATE —— 无分页，参数不动",
			sql:  "UPDATE `ca_base`.`t` SET `a`=? WHERE `id`=?",
			args: []interface{}{1, 2}, exec: true, wantVars: "[1 2]",
		},
		{
			name: "多语句 —— 参数不动",
			sql:  "UPDATE `ca_base`.`t` SET `a`=? WHERE `id`=?;DELETE FROM `ca_base`.`t2` WHERE `b`=?",
			args: []interface{}{1, 2, 3}, exec: true, wantVars: "[1 2 3]",
		},
		{
			// 调用方 bug（MySQL 也会拒绝）。数量校验不过 → 降级 → 参数原样，与改动前一致
			name: "参数少于占位符 —— 降级",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `a`=? AND `b`=? LIMIT ? OFFSET ?",
			args: []interface{}{1}, wantVars: "[1]",
		},
		{
			name: "参数多于占位符",
			sql:  "SELECT * FROM `ca_base`.`t` WHERE `a`=?",
			args: []interface{}{1, 2, 3}, wantVars: "[1 2 3]",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// ① 不挂插件，拿 GORM 构建后的 SQL
			var sink []struct{}
			plain := newPlainDBForRaw(t)
			var built *gorm.DB
			if c.exec {
				built = plain.Exec(c.sql, c.args...)
			} else {
				built = plain.Raw(c.sql, c.args...).Scan(&sink)
			}
			builtSQL := built.Statement.SQL.String()

			// ② 改动前的行为：只改库名，参数原样
			wantSQL := rewriteReference(builtSQL, "ok")

			// ③ 改动后：走真实插件
			var sink2 []struct{}
			tenant := newTestDB(t, "ok")
			var got *gorm.DB
			if c.exec {
				got = tenant.Exec(c.sql, c.args...)
			} else {
				got = tenant.Raw(c.sql, c.args...).Scan(&sink2)
			}
			gotSQL := got.Statement.SQL.String()
			gotVars := fmt.Sprint(got.Statement.Vars)

			if gotSQL != wantSQL {
				t.Errorf("SQL 文本与改动前不一致\n改动前: %s\n改动后: %s", wantSQL, gotSQL)
			}
			if gotVars != c.wantVars {
				t.Errorf("参数不符\n got: %s\nwant: %s\n sql: %s", gotVars, c.wantVars, gotSQL)
			}
		})
	}
}

// TestApplySQLRewrite_DegradeKeepsPrefix 端到端驱动 applySQLRewrite 走降级分支。
//
// 契约：一致性校验不通过时，SQL 仍然必须带租户前缀（只是参数不重排）。
// 若这里退化成「原样不改」，SQL 会带裸库名执行，可能读到别的租户——
// 那是改动前不存在的新故障，绝不允许。
func TestApplySQLRewrite_DegradeKeepsPrefix(t *testing.T) {
	env.SetFallbackTenantName("")

	// 构造一条能强制降级的 SQL：在真实占位符**之前**放一个与其偏移同号的伪造标记，
	// 伪造标记会先抢占配额，真标记随后成为重复项 → 判不可信 → 降级。
	// 标记长度随偏移位数变化，迭代到不动点。
	head := "SELECT * FROM `ca_base`.`t` WHERE `n`='"
	off := len(head)
	var sql string
	for i := 0; i < 5; i++ {
		sql = head + paramMarker(off) + "' AND `a`=? LIMIT ? OFFSET ?"
		idx := -1
		for k := len(head); k < len(sql); k++ {
			if sql[k] == '?' {
				idx = k
				break
			}
		}
		if idx == off {
			break
		}
		off = idx
	}
	if sql[off] != '?' {
		t.Fatalf("构造失败：偏移 %d 处是 %q", off, sql[off])
	}

	_, _, ok := rewriteSQLWithParser(sql, "ok")
	if ok {
		t.Skipf("该构造未触发降级（识别已足够精确），跳过：%s", sql)
	}

	// 走 applySQLRewrite
	db := newTestDB(t, "ok")
	stmt := db.Statement
	stmt.SQL.Reset()
	stmt.SQL.WriteString(sql)
	stmt.Vars = []interface{}{1, 10, 20}

	applySQLRewrite(db, stmt, "ok")

	gotSQL := stmt.SQL.String()
	t.Logf("降级后 SQL: %s", gotSQL)

	if !strings.Contains(gotSQL, "`ok_ca_base`.`t`") {
		t.Fatalf("降级时丢失租户前缀，会读到错误的库！got: %s", gotSQL)
	}
	if fmt.Sprint(stmt.Vars) != "[1 10 20]" {
		t.Errorf("降级时参数应原样沿用，got: %v", stmt.Vars)
	}
}
