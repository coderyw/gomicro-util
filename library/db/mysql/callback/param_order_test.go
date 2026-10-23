package callback

import (
	"strings"
	"testing"

	"github.com/coderyw/gomicro-util/library/env"
)

// ==========================================================
// 占位符置换测试
//
// 背景：Restore 只保证语义等价，不保证占位符位置不变。
// 典型如 `LIMIT ? OFFSET ?` 会被 Restore 成 `LIMIT ?,?`（offset 在前），
// 若 args 不同步重排，limit 与 offset 的值就会互换——后台列表翻到第 3 页
// 开始重复展示第 2 页的数据（第 1 页无 OFFSET 子句、第 2 页两值相等，都看不出来）。
// ==========================================================

// applyPerm 模拟调用方：按置换重排 args，返回「最终发给 MySQL 的 SQL + 参数」
func applyPerm(t *testing.T, sql, prefix string, args []interface{}) (string, []interface{}) {
	t.Helper()
	newSQL, perm, ok := rewriteSQLWithParser(sql, prefix)
	if !ok {
		t.Fatalf("改写未通过一致性校验: %s", sql)
	}
	newArgs, ok := permuteVars(args, perm)
	if !ok {
		t.Fatalf("参数重排失败: sql=%s perm=%v args=%v", sql, perm, args)
	}
	return newSQL, newArgs
}

func TestParamOrder_LimitOffset(t *testing.T) {
	// 第 3 页：page=3 page_size=10 → GORM 产出 limit=10, offset=20
	sql := "SELECT * FROM `ca_base`.`b_loading_page_config` WHERE `is_delete`=? AND `page_type`=? ORDER BY `id` DESC LIMIT ? OFFSET ?"
	args := []interface{}{0, 1, 10, 20} // [is_delete, page_type, limit, offset]

	newSQL, newArgs := applyPerm(t, sql, "ok", args)
	t.Logf("in : %s | %v", sql, args)
	t.Logf("out: %s | %v", newSQL, newArgs)

	if !strings.Contains(newSQL, "`ok_ca_base`.`b_loading_page_config`") {
		t.Errorf("库名未改写: %s", newSQL)
	}
	if !strings.HasSuffix(newSQL, "LIMIT ?,?") {
		t.Fatalf("期望 Restore 成逗号形式，got: %s", newSQL)
	}
	// MySQL 语义：LIMIT a,b = 跳过 a 取 b。所以最后两个参数必须是 [offset, limit] = [20, 10]
	want := []interface{}{0, 1, 20, 10}
	assertArgs(t, newArgs, want)
}

func TestParamOrder_FirstAndSecondPageUnaffected(t *testing.T) {
	// 第 1 页：GORM 不发 OFFSET 子句
	sql1 := "SELECT * FROM `ca_base`.`t` WHERE `a`=? ORDER BY `id` DESC LIMIT ?"
	got1, args1 := applyPerm(t, sql1, "ok", []interface{}{0, 10})
	if !strings.HasSuffix(got1, "LIMIT ?") {
		t.Errorf("第 1 页分页子句不应变形: %s", got1)
	}
	assertArgs(t, args1, []interface{}{0, 10})

	// 第 2 页：offset == limit，重排前后等价，但必须仍然正确
	sql2 := "SELECT * FROM `ca_base`.`t` WHERE `a`=? ORDER BY `id` DESC LIMIT ? OFFSET ?"
	_, args2 := applyPerm(t, sql2, "ok", []interface{}{0, 10, 10})
	assertArgs(t, args2, []interface{}{0, 10, 10})
}

func TestParamOrder_LiteralLimitUnaffected(t *testing.T) {
	// 字面量分页：值被 Restore 一起搬走，语义不变，且没有占位符需要重排
	sql := "SELECT * FROM `ca_base`.`t` LIMIT 10 OFFSET 20"
	got, _, ok := rewriteSQLWithParser(sql, "ok")
	if !ok {
		t.Fatal("不应校验失败")
	}
	if !strings.HasSuffix(got, "LIMIT 20,10") {
		t.Errorf("字面量分页语义应保持「跳 20 取 10」，got: %s", got)
	}
}

func TestParamOrder_InClauseWithPagination(t *testing.T) {
	// IN 里的参数顺序必须原样保留，只有分页那两个换位置
	sql := "SELECT * FROM `ca_activity`.`b_bonus` WHERE `activity_id` IN (?,?,?) AND `is_del`=? ORDER BY `id` LIMIT ? OFFSET ?"
	args := []interface{}{101, 102, 103, 0, 10, 40}

	newSQL, newArgs := applyPerm(t, sql, "ok", args)
	t.Logf("out: %s | %v", newSQL, newArgs)
	assertArgs(t, newArgs, []interface{}{101, 102, 103, 0, 40, 10})
}

func TestParamOrder_NestedPagination(t *testing.T) {
	// 子查询与外层各有一个分页子句，两处都要各自换对
	sql := "SELECT * FROM (SELECT * FROM `ca_base`.`t` WHERE `a`=? LIMIT ? OFFSET ?) `x` WHERE `b`=? LIMIT ? OFFSET ?"
	args := []interface{}{1, 5, 15, 2, 10, 30}

	newSQL, newArgs := applyPerm(t, sql, "ok", args)
	t.Logf("out: %s | %v", newSQL, newArgs)
	assertArgs(t, newArgs, []interface{}{1, 15, 5, 2, 30, 10})
}

func TestParamOrder_UpdateAndDelete(t *testing.T) {
	// 写路径（走 rewrite 兜底）没有分页，参数顺序应原样不动
	sql := "UPDATE `ca_base`.`t` SET `name`=?,`status`=? WHERE `id`=?"
	newSQL, newArgs := applyPerm(t, sql, "ok", []interface{}{"a", 1, 99})
	if !strings.Contains(newSQL, "`ok_ca_base`.`t`") {
		t.Errorf("库名未改写: %s", newSQL)
	}
	assertArgs(t, newArgs, []interface{}{"a", 1, 99})
}

func TestParamOrder_ShortCircuits(t *testing.T) {
	cases := []struct {
		name   string
		sql    string
		prefix string
	}{
		{"prefix 为空", "SELECT * FROM `ca_base`.`t` LIMIT ? OFFSET ?", ""},
		{"SQL 无点号", "SELECT * FROM t LIMIT ? OFFSET ?", "ok"},
		{"解析失败", "SELECT * FROM `ca_base`.`t` WHERE (`a` = ? AND", "ok"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, perm, ok := rewriteSQLWithParser(tt.sql, tt.prefix)
			if !ok {
				t.Fatal("短路场景不应报校验失败")
			}
			if got != tt.sql {
				t.Errorf("短路场景不应改写 SQL: %s", got)
			}
			if perm != nil {
				t.Errorf("短路场景 perm 应为 nil: %v", perm)
			}
		})
	}
}

func TestPermuteVars(t *testing.T) {
	vars := []interface{}{"a", "b", "c"}

	if got, ok := permuteVars(vars, nil); !ok || len(got) != 3 || got[0] != "a" {
		t.Errorf("perm 为 nil 时应原样返回, got=%v ok=%v", got, ok)
	}
	if got, ok := permuteVars(vars, []int{0, 2, 1}); !ok {
		t.Error("合法置换应成功")
	} else {
		assertArgs(t, got, []interface{}{"a", "c", "b"})
	}
	if _, ok := permuteVars(vars, []int{0, 1}); ok {
		t.Error("长度不匹配应失败")
	}
	if _, ok := permuteVars(vars, []int{0, 1, 1}); ok {
		t.Error("重复下标（非双射）应失败")
	}
	if _, ok := permuteVars(vars, []int{0, 1, 9}); ok {
		t.Error("越界下标应失败")
	}
}

// ==========================================================
// 端到端：挂插件的 DryRun DB，断言最终 SQL + Vars 的组合语义
// ==========================================================

type pagingModel struct {
	ID int64 `gorm:"column:id;primaryKey"`
}

func (pagingModel) TableName() string { return "ca_base.b_loading_page_config" }

func TestPlugin_Pagination_EndToEnd(t *testing.T) {
	env.SetFallbackTenantName("")

	cases := []struct {
		name     string
		offset   int
		limit    int
		wantSQL  string
		wantTail []interface{}
	}{
		{"第 1 页", 0, 10, "LIMIT ?", []interface{}{10}},
		{"第 2 页", 10, 10, "LIMIT ?,?", []interface{}{10, 10}},
		{"第 3 页", 20, 10, "LIMIT ?,?", []interface{}{20, 10}},
		{"每页 20 第 3 页", 40, 20, "LIMIT ?,?", []interface{}{40, 20}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			db := newTestDB(t, "ok")

			var rows []pagingModel
			res := db.Model(&pagingModel{}).
				Where("is_delete = ?", 0).
				Order("id DESC").
				Offset(tt.offset).Limit(tt.limit).
				Find(&rows)

			sql := res.Statement.SQL.String()
			vars := res.Statement.Vars
			t.Logf("SQL : %s", sql)
			t.Logf("Vars: %v", vars)

			if !strings.Contains(sql, "`ok_ca_base`.`b_loading_page_config`") {
				t.Errorf("库名未改写: %s", sql)
			}
			if !strings.HasSuffix(sql, tt.wantSQL) {
				t.Fatalf("分页子句形态不符，want 以 %q 结尾, got: %s", tt.wantSQL, sql)
			}
			// 尾部参数必须与逗号形式的语义对齐：LIMIT offset,count
			assertArgs(t, vars[len(vars)-len(tt.wantTail):], tt.wantTail)
		})
	}
}

func assertArgs(t *testing.T, got []interface{}, want []interface{}) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("参数个数不符: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("参数不符: got %v want %v", got, want)
		}
	}
}

// TestPlugin_RawSQL_NotDoubleRewritten Raw SQL 会先后经过 rewrite（Before hook）
// 与 queryWithRewrite 两个回调，两次都会走一遍 parser 兜底。
// 必须确认第二遍不会把已经换过位的占位符再换一次（否则等于没修）。
func TestPlugin_RawSQL_NotDoubleRewritten(t *testing.T) {
	env.SetFallbackTenantName("")
	db := newTestDB(t, "ok")

	var rows []pagingModel
	res := db.Raw(
		"SELECT * FROM `ca_base`.`b_loading_page_config` WHERE `is_delete` = ? ORDER BY `id` DESC LIMIT ? OFFSET ?",
		0, 10, 20,
	).Scan(&rows)

	sql := res.Statement.SQL.String()
	t.Logf("SQL : %s", sql)
	t.Logf("Vars: %v", res.Statement.Vars)

	if !strings.Contains(sql, "`ok_ca_base`.`b_loading_page_config`") {
		t.Errorf("库名未改写: %s", sql)
	}
	if !strings.HasSuffix(sql, "LIMIT ?,?") {
		t.Fatalf("分页子句形态不符: %s", sql)
	}
	// 换一次 → [20 10]；若被换了两次会变回 [10 20]
	assertArgs(t, res.Statement.Vars[len(res.Statement.Vars)-2:], []interface{}{20, 10})
}

// TestPermutationFromOffsets_Direction 用 3-循环把置换方向钉死。
// 对合置换（如 limit/offset 两两对换）的逆等于自身，光靠分页用例无法区分
// perm[i]=rank[off] 与其逆写法 perm[rank[off]]=i；3-循环可以。
func TestPermutationFromOffsets_Direction(t *testing.T) {
	// 标记在新 SQL 中依次出现的原始偏移：30、10、20
	// → 原始顺序（偏移升序）为 10(0)、20(1)、30(2)
	// → 新位置 0 该取原下标 2，新位置 1 取 0，新位置 2 取 1
	got := permutationFromOffsets([]int{30, 10, 20})
	want := []int{2, 0, 1}
	if len(got) != len(want) {
		t.Fatalf("长度不符: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("置换方向错误: got %v want %v（逆置换是 [1 2 0]，不能与之混淆）", got, want)
		}
	}

	// 与 permuteVars 串起来验证语义：newVars[i] = vars[perm[i]]
	vars := []interface{}{"off10", "off20", "off30"}
	newVars, ok := permuteVars(vars, got)
	if !ok {
		t.Fatal("permuteVars 应成功")
	}
	assertArgs(t, newVars, []interface{}{"off30", "off10", "off20"})

	// 偏移重复 → 无法建立一一对应
	if permutationFromOffsets([]int{10, 10}) != nil {
		t.Error("重复偏移应返回 nil")
	}
}

// ==========================================================
// 降级触发条件的直接测试
//
// 此前 extractParamOrder 返回 ok=false 这条路径没有任何用例走到过，
// 意味着整条降级路径的触发器都没被测试挡着。以下三个用例分别覆盖
// 「标记重叠」「同一偏移出现两次」「标记少认一个」。
// ==========================================================

// 两个标记首尾相接时，导引符回扫会把 start 拉到上一个标记内部。
// 必须判为不可信，且绝不能因为 restored[prev:start] 而越界 panic。
func TestExtractParamOrder_OverlappingMarkers(t *testing.T) {
	restored := "'" + paramMarker(5) + "'" + paramMarker(7) + "'"
	origSQL := "?? "

	out, perm, ok := extractParamOrder(restored, []int{5, 7}, origSQL)
	if ok {
		t.Errorf("标记重叠应判不可信，got ok=true out=%s perm=%v", out, perm)
	}
	if out != origSQL {
		t.Errorf("不可信时应原样返回入参: %s", out)
	}
}

// 同一偏移的标记出现两次：可能是 Restore 复制了占位符，也可能是外部字面量
// 抢占了配额。此前会被当成「一个真标记 + 一个普通字面量」放过，
// 计数校验被骗过，标记原文残留进最终 SQL。
func TestExtractParamOrder_DuplicateMarker(t *testing.T) {
	restored := "SELECT '" + paramMarker(10) + "', '" + paramMarker(10) + "', '" + paramMarker(20) + "'"
	origSQL := "?  ?"

	out, _, ok := extractParamOrder(restored, []int{10, 20}, origSQL)
	if ok {
		t.Fatalf("同一偏移出现两次应判不可信，got: %s", out)
	}
	if out != origSQL {
		t.Errorf("不可信时应原样返回入参: %s", out)
	}
}

// 注入 3 个标记却只找回 2 个（Restore 吞掉了一个）→ 不可信
func TestExtractParamOrder_MissingMarker(t *testing.T) {
	restored := "SELECT '" + paramMarker(10) + "', '" + paramMarker(20) + "'"
	origSQL := "?  ?  ?"

	out, _, ok := extractParamOrder(restored, []int{10, 20, 30}, origSQL)
	if ok {
		t.Fatalf("少认一个标记应判不可信，got: %s", out)
	}
	if out != origSQL {
		t.Errorf("不可信时应原样返回入参: %s", out)
	}
}
