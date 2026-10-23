package callback

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// ---- 复杂嵌套 SQL 生成器：最多 3 层，子查询可出现在 FROM / WHERE / SELECT，各层都可能自带分页 ----

func page(r *rand.Rand) string {
	switch r.Intn(4) {
	case 0:
		return " LIMIT ?"
	case 1:
		return " LIMIT ? OFFSET ?"
	case 2:
		return " LIMIT ?,?"
	}
	return ""
}

func sub(r *rand.Rand, depth int) string {
	tbl := []string{"`ca_base`.`t1`", "`ca_activity`.`t2`", "`ca_admin`.`u`"}[r.Intn(3)]
	var b strings.Builder
	b.WriteString("SELECT `id` FROM ")
	if depth > 0 && r.Intn(2) == 0 {
		b.WriteString("(" + sub(r, depth-1) + ") `d" + strconv.Itoa(depth) + "`") // FROM 派生表
	} else {
		b.WriteString(tbl)
	}
	b.WriteString(" WHERE `a`=?")
	if depth > 0 {
		switch r.Intn(4) {
		case 0:
			b.WriteString(" AND `b` IN (" + sub(r, depth-1) + ")") // WHERE 子查询
		case 1:
			b.WriteString(" AND EXISTS (" + sub(r, depth-1) + ")") // 相关子查询
		case 2:
			b.WriteString(" AND `c` > (SELECT MAX(`x`) FROM `ca_game`.`g` WHERE `k`=?)") // 标量子查询
		}
	}
	if r.Intn(3) == 0 {
		b.WriteString(" ORDER BY `id` DESC")
	}
	b.WriteString(page(r))
	return b.String()
}

func genNested(r *rand.Rand) string {
	switch r.Intn(6) {
	case 0: // UNION 各分支各自分页
		return "(" + sub(r, 1) + ") UNION ALL (" + sub(r, 1) + ")"
	case 1: // JOIN 派生表
		return "SELECT * FROM (" + sub(r, 2) + ") `x` LEFT JOIN (" + sub(r, 1) +
			") `y` ON `x`.`id`=`y`.`id` AND `y`.`z`=? WHERE `x`.`w`=?" + page(r)
	case 2: // SELECT 列表里的标量子查询
		return "SELECT `id`,(SELECT COUNT(1) FROM `ca_game`.`g` WHERE `gid`=`t`.`id` AND `s`=?) FROM `ca_base`.`t1` `t` WHERE `a`=?" + page(r)
	case 3: // UPDATE + 子查询
		return "UPDATE `ca_base`.`t1` SET `a`=? WHERE `id` IN (" + sub(r, 1) + ")"
	case 4: // DELETE + 子查询
		return "DELETE FROM `ca_base`.`t1` WHERE `id` IN (SELECT `id` FROM (" + sub(r, 1) + ") `z`)"
	}
	return "SELECT * FROM (" + sub(r, 2) + ") `outer` WHERE `q`=?" + page(r)
}

// ---- 独立预言机：把第 k 个 ? 换成唯一字面量，看它们最终落在什么顺序 ----
func oraclePerm(sql, prefix string) ([]int, bool) {
	var b strings.Builder
	n := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '?' {
			b.WriteString(strconv.Itoa(900000 + n))
			n++
			continue
		}
		b.WriteByte(sql[i])
	}
	if n == 0 {
		return nil, true
	}
	out := rewriteSQLPlain(b.String(), prefix) // 不注入标记的改写
	type hit struct{ pos, idx int }
	var hits []hit
	for k := 0; k < n; k++ {
		lit := strconv.Itoa(900000 + k)
		p := strings.Index(out, lit)
		if p < 0 {
			return nil, false
		}
		hits = append(hits, hit{p, k})
	}
	for i := 1; i < len(hits); i++ {
		for j := i; j > 0 && hits[j].pos < hits[j-1].pos; j-- {
			hits[j], hits[j-1] = hits[j-1], hits[j]
		}
	}
	perm := make([]int, n)
	for newPos, h := range hits {
		perm[newPos] = h.idx
	}
	return perm, true
}

func TestNested_DifferentialAndOracle(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	var total, textDiff, permMismatch, nonIdentity, skipped int
	shapes := map[string]bool{}

	for i := 0; i < 20000; i++ {
		sql := genNested(r)
		want := rewriteReference(sql, "ok")
		if want == sql {
			skipped++ // 解析失败的构造，跳过
			continue
		}
		got, perm, ok := rewriteSQLWithParser(sql, "ok")
		if !ok {
			t.Fatalf("不应降级: %s", sql)
		}
		total++
		shapes[sql] = true

		if got != want {
			textDiff++
			if textDiff <= 2 {
				t.Errorf("SQL 文本变了\n前: %s\n后: %s", want, got)
			}
		}

		exp, valid := oraclePerm(sql, "ok")
		if !valid {
			continue
		}
		if len(exp) != len(perm) {
			permMismatch++
			if permMismatch <= 2 {
				t.Errorf("置换长度不符 sql=%s\n impl=%v oracle=%v", sql, perm, exp)
			}
			continue
		}
		for k := range exp {
			if exp[k] != perm[k] {
				permMismatch++
				if permMismatch <= 2 {
					t.Errorf("置换与预言机不符 sql=%s\n impl=%v oracle=%v", sql, perm, exp)
				}
				break
			}
			if k != perm[k] {
				// 记一次非恒等
			}
		}
		for k, p := range perm {
			if k != p {
				nonIdentity++
				break
			}
		}
	}
	fmt.Printf("嵌套/复杂查询 %d 条（%d 种形态，另跳过 %d 条解析失败构造）\n", total, len(shapes), skipped)
	fmt.Printf("  SQL 文本与改动前不一致: %d\n", textDiff)
	fmt.Printf("  置换与独立预言机不符  : %d\n", permMismatch)
	fmt.Printf("  发生参数重排的语句    : %d\n", nonIdentity)
}
