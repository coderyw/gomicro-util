package callback

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
)

// ==========================================================
// 1) 差分测试：证明「加了标记替换」不改变改写后的 SQL 文本
//
// 参考实现 = 改动前的老逻辑（只加库名前缀，不替换占位符）。
// 新实现在 Restore 前把 ? 换成标记、Restore 后再换回 ?，
// 输出必须与参考实现逐字节一致——否则说明影响到了 LIMIT 之外的东西。
// ==========================================================

// referenceVisitor 复刻改动前的 visitor：只改库名，Leave 不做任何事
type referenceVisitor struct {
	ast.Node
	prefix string
}

func (v *referenceVisitor) Enter(in ast.Node) (ast.Node, bool) {
	switch n := in.(type) {
	case *ast.TableName:
		if n.Schema.O != "" && !shouldSkipSchema(n.Schema.O) && !isDBNameRewritten(v.prefix, n.Schema.O) {
			n.Schema.O = namespaceFmt(v.prefix, n.Schema.O)
			n.Schema.L = strings.ToLower(n.Schema.O)
		}
	case *ast.ColumnName:
		if n.Schema.O != "" && !shouldSkipSchema(n.Schema.O) && !isDBNameRewritten(v.prefix, n.Schema.O) {
			n.Schema.O = namespaceFmt(v.prefix, n.Schema.O)
			n.Schema.L = strings.ToLower(n.Schema.O)
		}
	}
	return in, false
}

func (v *referenceVisitor) Leave(in ast.Node) (ast.Node, bool) { return in, true }

// rewriteReference 改动前的 rewriteSQLWithParser
func rewriteReference(sql, prefix string) string {
	if prefix == "" || strings.TrimSpace(sql) == "" || !strings.Contains(sql, ".") {
		return sql
	}
	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	if err != nil || len(stmts) == 0 {
		return sql
	}
	v := &referenceVisitor{prefix: prefix}
	var out bytes.Buffer
	for i, stmt := range stmts {
		if i > 0 {
			out.WriteByte(';')
		}
		stmt.Accept(v)
		if err := stmt.Restore(format.NewRestoreCtx(format.DefaultRestoreFlags, &out)); err != nil {
			return sql
		}
	}
	return out.String()
}

// corpus 覆盖仓库里常见的 SQL 形态
var corpus = []struct {
	name string
	sql  string
}{
	{"分页列表", "SELECT * FROM `ca_base`.`b_loading_page_config` WHERE `is_delete`=? AND `page_type`=? ORDER BY `id` DESC LIMIT ? OFFSET ?"},
	{"count", "SELECT COUNT(1) FROM `ca_base`.`b_loading_page_config` WHERE `is_delete`=? AND `page_type`=?"},
	{"IN 列表", "SELECT `id`,`name` FROM `ca_admin`.`admin_users` WHERE `id` IN (?,?,?,?)"},
	{"JOIN + ON 参数", "SELECT a.* FROM `ca_base`.`t1` a LEFT JOIN `ca_game`.`t2` b ON a.`id`=b.`aid` AND b.`type`=? WHERE a.`st`=? LIMIT ? OFFSET ?"},
	{"UPDATE", "UPDATE `ca_base`.`t` SET `name`=?,`status`=?,`updated_at`=? WHERE `id`=?"},
	{"INSERT", "INSERT INTO `ca_base`.`t` (`a`,`b`,`c`) VALUES (?,?,?)"},
	{"INSERT 多行", "INSERT INTO `ca_base`.`t` (`a`,`b`) VALUES (?,?),(?,?)"},
	{"DELETE", "DELETE FROM `ca_base`.`t` WHERE `id`=? AND `st`=?"},
	{"WHERE 子查询", "SELECT * FROM `ca_base`.`t` WHERE `id` IN (SELECT `tid` FROM `ca_activity`.`r` WHERE `uid`=?) AND `st`=?"},
	{"BETWEEN", "SELECT * FROM `ca_base`.`t` WHERE `created_at` BETWEEN ? AND ? ORDER BY `id` LIMIT ?"},
	{"LIKE + 分页", "SELECT * FROM `ca_base`.`t` WHERE `name` LIKE ? ORDER BY `id` DESC LIMIT ? OFFSET ?"},
	{"GROUP BY HAVING", "SELECT `uid`,SUM(`amt`) FROM `ca_balance`.`t` WHERE `st`=? GROUP BY `uid` HAVING SUM(`amt`)>? LIMIT ? OFFSET ?"},
	{"CASE WHEN", "SELECT CASE WHEN `a`=? THEN ? ELSE ? END FROM `ca_base`.`t` WHERE `id`=?"},
	{"FOR UPDATE", "SELECT * FROM `ca_admin`.`report_export_task` WHERE `biz_type`=? AND `task_status`=? ORDER BY `id` LIMIT ? FOR UPDATE"},
	{"函数参数", "SELECT DATE_FORMAT(`created_at`,?) FROM `ca_base`.`t` WHERE `id`>? ORDER BY `id` LIMIT ? OFFSET ?"},
	{"UNION", "SELECT `id` FROM `ca_base`.`t1` WHERE `a`=? UNION ALL SELECT `id` FROM `ca_base`.`t2` WHERE `b`=?"},
	{"嵌套分页", "SELECT * FROM (SELECT * FROM `ca_base`.`t` WHERE `a`=? LIMIT ? OFFSET ?) `x` WHERE `b`=? LIMIT ? OFFSET ?"},
	{"ORDER BY 多列", "SELECT * FROM `ca_activity`.`t` WHERE `is_del`=? ORDER BY `sort` DESC,`id` DESC LIMIT ? OFFSET ?"},
	{"无占位符", "SELECT * FROM `ca_game_base`.`b_game_type` ORDER BY `sort` DESC"},
	{"字面量分页", "SELECT * FROM `ca_base`.`t` ORDER BY `id` LIMIT 10 OFFSET 20"},
	{"字符串字面量", "SELECT * FROM `ca_base`.`t` WHERE `name`='abc' AND `a`=? LIMIT ? OFFSET ?"},
	{"字面量撞车", "SELECT * FROM `ca_base`.`t` WHERE `name`='" + paramMarker(9999) + "' AND `a`=? LIMIT ? OFFSET ?"},
	{"EXISTS 子查询", "SELECT * FROM `ca_base`.`t` WHERE EXISTS (SELECT 1 FROM `ca_activity`.`r` WHERE `r`.`tid`=`t`.`id` AND `r`.`st`=?) AND `a`=? LIMIT ? OFFSET ?"},
	{"CTE", "WITH `c` AS (SELECT `id` FROM `ca_base`.`t` WHERE `a`=?) SELECT * FROM `c` LIMIT ? OFFSET ?"},
	{"窗口函数", "SELECT `id`,ROW_NUMBER() OVER (PARTITION BY `uid` ORDER BY `id` DESC) FROM `ca_base`.`t` WHERE `a`=? LIMIT ? OFFSET ?"},
	{"INSERT SELECT", "INSERT INTO `ca_base`.`t1` (`a`,`b`) SELECT `a`,`b` FROM `ca_base`.`t2` WHERE `c`=? LIMIT ?"},
	{"ON DUPLICATE", "INSERT INTO `ca_base`.`t` (`a`,`b`) VALUES (?,?) ON DUPLICATE KEY UPDATE `b`=VALUES(`b`)"},
	{"optimizer hint", "SELECT /*+ MAX_EXECUTION_TIME(1000) */ * FROM `ca_base`.`t` WHERE `id`=? LIMIT ? OFFSET ?"},
	{"FORCE INDEX", "SELECT * FROM `ca_base`.`t` FORCE INDEX (`idx_a`) WHERE `a`=? LIMIT ? OFFSET ?"},
	{"FOR UPDATE SKIP LOCKED", "SELECT * FROM `ca_admin`.`report_export_task` WHERE `st`=? ORDER BY `id` LIMIT ? FOR UPDATE SKIP LOCKED"},
	{"多语句", "UPDATE `ca_base`.`t` SET `a`=? WHERE `id`=?;DELETE FROM `ca_base`.`t2` WHERE `b`=?"},
	{"IS NULL / NOT IN", "SELECT * FROM `ca_base`.`t` WHERE `d` IS NULL AND `e` NOT IN (?,?) ORDER BY `id` LIMIT ? OFFSET ?"},
}

func TestRewrite_NoTextualRegression(t *testing.T) {
	for _, c := range corpus {
		t.Run(c.name, func(t *testing.T) {
			want := rewriteReference(c.sql, "ok")
			got, _, ok := rewriteSQLWithParser(c.sql, "ok")
			if !ok {
				t.Fatalf("改写被拒绝: %s", c.sql)
			}
			if got != want {
				t.Errorf("改写后的 SQL 文本与改动前不一致\n改动前: %s\n改动后: %s", want, got)
			}
		})
	}
}

// ==========================================================
// 2) 重排面调查：除了 LIMIT，还有哪些语法会让占位符换位置
// ==========================================================

func TestSurvey_WhichSyntaxReordersParams(t *testing.T) {
	var reordered []string
	for _, c := range corpus {
		_, perm, ok := rewriteSQLWithParser(c.sql, "ok")
		if !ok {
			t.Fatalf("改写被拒绝: %s", c.sql)
		}
		if !isIdentityPerm(perm) {
			reordered = append(reordered, fmt.Sprintf("%-12s perm=%v", c.name, perm))
		}
	}
	fmt.Println("发生占位符重排的语句：")
	if len(reordered) == 0 {
		fmt.Println("  （无）")
	}
	for _, r := range reordered {
		fmt.Println("  " + r)
	}
}

func isIdentityPerm(perm []int) bool {
	for i, p := range perm {
		if i != p {
			return false
		}
	}
	return true
}

// ==========================================================
// 3) 撞车：SQL 里本来就含有与标记同形的字符串字面量
//    标记带进程随机 salt，外部无从伪造；这里用真实标记构造最坏情况。
// ==========================================================

func TestMarkerCollision_LiteralPreserved(t *testing.T) {
	// 偏移 9999 不可能对应真实占位符 → 该字面量必须原样保留
	fake := paramMarker(9999)
	sql := "SELECT * FROM `ca_base`.`t` WHERE `name`='" + fake + "' AND `a`=? ORDER BY `id` LIMIT ? OFFSET ?"

	got, perm, ok := rewriteSQLWithParser(sql, "ok")
	t.Logf("ok=%v perm=%v\nsql=%s", ok, perm, got)

	if !ok {
		t.Fatalf("不应因字面量撞车而降级: %s", got)
	}
	if !strings.Contains(got, "`ok_ca_base`.`t`") {
		t.Errorf("库名未改写: %s", got)
	}
	if !strings.Contains(got, fake) {
		t.Errorf("原有字面量必须原样保留: %s", got)
	}
	if n := strings.Count(got, "?"); n != 3 {
		t.Errorf("占位符数量应为 3，实际 %d: %s", n, got)
	}
	if len(perm) != 3 || perm[0] != 0 || perm[1] != 2 || perm[2] != 1 {
		t.Errorf("置换不符，期望 [0 2 1]，实际 %v", perm)
	}
}

// TestMarkerCollision_RealOffset 字面量恰好复用了某个真实占位符的偏移（最坏构造）。
// 每个偏移至多被消费一次，先到先得；真实占位符在 SQL 中位于该字面量之前，
// 因而先消费掉配额，字面量随后被当作普通文本放过——结果仍然正确。
//
// 残余理论风险：若伪造字面量排在真实占位符之前，会顶掉配额。防线是标记带
// 进程内随机 salt（crypto/rand），外部无从得知，也就无法构造出同形字面量。
func TestMarkerCollision_RealOffset(t *testing.T) {
	// 第一个 ? 出现在字面量之前，其偏移与字面量长度无关，可以直接算出来
	head := "SELECT * FROM `ca_base`.`t` WHERE `a`="
	off := len(head)
	sql := head + "? AND `n`='" + paramMarker(off) + "' LIMIT ? OFFSET ?"
	if sql[off] != '?' {
		t.Fatalf("构造前提不成立：偏移 %d 处是 %q", off, sql[off])
	}

	got, perm, ok := rewriteSQLWithParser(sql, "ok")
	t.Logf("ok=%v perm=%v sql=%s", ok, perm, got)

	if !strings.Contains(got, "`ok_ca_base`.`t`") {
		t.Errorf("必须保留租户前缀: %s", got)
	}
	if n := strings.Count(got, "?"); n != 3 {
		t.Errorf("占位符数量应为 3，实际 %d: %s", n, got)
	}
	if !strings.Contains(got, paramMarker(off)) {
		t.Errorf("同形字面量必须原样保留: %s", got)
	}
	if ok && (len(perm) != 3 || perm[1] != 2 || perm[2] != 1) {
		t.Errorf("分页仍应正确换位，实际 %v", perm)
	}
}

// TestNextParamMarker 手写扫描器的边界
func TestNextParamMarker(t *testing.T) {
	m := paramMarker(75)
	cases := []struct {
		name     string
		in       string
		wantOK   bool
		wantOff  int
		wantSpan string // 被识别为标记的完整片段
	}{
		{"裸引号形式", "LIMIT '" + m + "',1", true, 75, "'" + m + "'"},
		{"带字符集导引符", "a=_UTF8MB4'" + m + "'", true, 75, "_UTF8MB4'" + m + "'"},
		{"没有引号包裹", "a=" + m, false, 0, ""},
		{"缺少开头引号", "a=" + paramMarkerPrefix + "75__'", false, 0, ""},
		{"缺少结尾引号", "a='" + paramMarkerPrefix + "75__X", false, 0, ""},
		{"缺少结尾下划线", "a='" + paramMarkerPrefix + "75'", false, 0, ""},
		{"数字部分为空", "a='" + paramMarkerPrefix + "__'", false, 0, ""},
		{"无标记", "SELECT 1 FROM t", false, 0, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end, off, ok := nextParamMarker(c.in, 0)
			if ok != c.wantOK {
				t.Fatalf("ok: got %v want %v", ok, c.wantOK)
			}
			if !ok {
				return
			}
			if off != c.wantOff {
				t.Errorf("offset: got %d want %d", off, c.wantOff)
			}
			if span := c.in[start:end]; span != c.wantSpan {
				t.Errorf("片段: got %q want %q", span, c.wantSpan)
			}
		})
	}
}

// ==========================================================
// 4) 降级契约：一致性校验失败时走的 rewriteSQLPlain，
//    必须与改动前的改写实现逐字节一致——保证最坏情况不超出原有错误范围
// ==========================================================

func TestDegradePath_EqualsPreChangeBehaviour(t *testing.T) {
	for _, c := range corpus {
		t.Run(c.name, func(t *testing.T) {
			want := rewriteReference(c.sql, "ok")
			got := rewriteSQLPlain(c.sql, "ok")
			if got != want {
				t.Errorf("降级结果与改动前不一致\n改动前: %s\n降级后: %s", want, got)
			}
		})
	}
}

// TestDegradePath_KeepsTenantPrefix 降级时仍然要加租户前缀，
// 只是不重排参数——绝不能退化成「原样不改、裸库名执行」。
func TestDegradePath_KeepsTenantPrefix(t *testing.T) {
	sql := "SELECT * FROM `ca_base`.`b_loading_page_config` WHERE `a`=? LIMIT ? OFFSET ?"
	got := rewriteSQLPlain(sql, "ok")
	if !strings.Contains(got, "`ok_ca_base`.`b_loading_page_config`") {
		t.Fatalf("降级路径必须保留租户前缀: %s", got)
	}
	if strings.Count(got, "?") != 3 {
		t.Errorf("占位符数量应保持 3 个: %s", got)
	}
	t.Logf("降级输出: %s", got)
}
