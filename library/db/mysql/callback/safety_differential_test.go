package callback

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// 随机生成"语法合法"的 SQL，覆盖常见子句组合
func genSQL(r *rand.Rand) string {
	tbl := []string{"`ca_base`.`t1`", "`ca_activity`.`t2`", "`ca_admin`.`u`", "`ca_game`.`g`"}[r.Intn(4)]
	col := func() string { return []string{"`id`", "`a`", "`b`", "`created_at`", "`status`"}[r.Intn(5)] }

	switch r.Intn(5) {
	case 0: // UPDATE
		return fmt.Sprintf("UPDATE %s SET %s=?,%s=? WHERE %s=?", tbl, col(), col(), col())
	case 1: // DELETE
		return fmt.Sprintf("DELETE FROM %s WHERE %s=? AND %s IN (?,?)", tbl, col(), col())
	case 2: // INSERT
		return fmt.Sprintf("INSERT INTO %s (%s,%s) VALUES (?,?),(?,?)", tbl, col(), col())
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	if r.Intn(4) == 0 {
		b.WriteString("COUNT(1)")
	} else {
		b.WriteString("*")
	}
	b.WriteString(" FROM " + tbl)
	if r.Intn(3) == 0 {
		b.WriteString(" LEFT JOIN `ca_game`.`g` ON `g`.`id`=" + col() + " AND `g`.`k`=?")
	}
	b.WriteString(" WHERE " + col() + "=?")
	switch r.Intn(5) {
	case 0:
		b.WriteString(" AND " + col() + " IN (?,?,?)")
	case 1:
		b.WriteString(" AND " + col() + " BETWEEN ? AND ?")
	case 2:
		b.WriteString(" AND " + col() + " LIKE ?")
	case 3:
		b.WriteString(" AND " + col() + " IN (SELECT `x` FROM `ca_activity`.`r` WHERE `y`=?)")
	}
	if r.Intn(3) == 0 {
		b.WriteString(" GROUP BY " + col())
		if r.Intn(2) == 0 {
			b.WriteString(" HAVING COUNT(1)>?")
		}
	}
	if r.Intn(2) == 0 {
		b.WriteString(" ORDER BY " + col() + " DESC")
	}
	switch r.Intn(4) {
	case 0:
		b.WriteString(" LIMIT ?")
	case 1:
		b.WriteString(" LIMIT ? OFFSET ?") // 会被重排的那类
	case 2:
		b.WriteString(" LIMIT 10 OFFSET 20")
	}
	if r.Intn(8) == 0 {
		b.WriteString(" FOR UPDATE")
	}
	return b.String()
}

func TestSafety_SQLTextNeverChanges(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	var total, textDiff, permIdentity, permChanged int
	seen := map[string]bool{}

	for i := 0; i < 30000; i++ {
		sql := genSQL(r)
		seen[sql] = true

		want := rewriteReference(sql, "ok") // 改动前的实现
		got, perm, ok := rewriteSQLWithParser(sql, "ok")
		if !ok {
			t.Fatalf("不应降级: %s", sql)
		}
		total++

		// ① SQL 文本必须与改动前逐字节一致
		if got != want {
			textDiff++
			if textDiff <= 3 {
				t.Errorf("SQL 文本变了！\n改动前: %s\n改动后: %s", want, got)
			}
		}

		// ② 置换是否恒等
		identity := true
		for k, p := range perm {
			if k != p {
				identity = false
				break
			}
		}
		if identity {
			permIdentity++
		} else {
			permChanged++
			// 非恒等的必须全部含 `LIMIT ? OFFSET ?`
			if !strings.Contains(sql, "LIMIT ? OFFSET ?") {
				t.Errorf("非分页语句也发生了参数重排: %s perm=%v", sql, perm)
			}
		}
	}

	fmt.Printf("随机语料 %d 条（去重后 %d 种形态）\n", total, len(seen))
	fmt.Printf("  ① SQL 文本与改动前不一致: %d 条\n", textDiff)
	fmt.Printf("  ② 参数顺序不变（恒等置换）: %d 条 —— 这些查询的行为与改动前完全相同\n", permIdentity)
	fmt.Printf("  ③ 参数顺序被修正: %d 条 —— 全部含 LIMIT ? OFFSET ?，即当前线上正在出错的那类\n", permChanged)
}
