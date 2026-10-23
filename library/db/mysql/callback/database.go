package callback

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/format"
	testdriver "github.com/pingcap/tidb/pkg/parser/test_driver"
	"github.com/coderyw/gomicro-util/library/env"
	"github.com/coderyw/gomicro-util/library/namespace"
	"gorm.io/gorm"
	gormcallbacks "gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
)

const PluginName = "tenant_db_plugin"

const (
	cbQuery  = "tenant:query"
	cbRow    = "tenant:row"
	cbRaw    = "tenant:raw"
	cbCreate = "tenant:create"
	cbUpdate = "tenant:update"
	cbDelete = "tenant:delete"
)

type NamespaceResolver func(ctx context.Context) string

// TableDBResolver 用于处理 join 里只有裸表名的情况。
// 比如关联 JOIN 里可能只拿到 b_vip_user，需要反推出它原始属于哪个库。
type TableDBResolver func(tableName string) (dbName string, ok bool)

type Config struct {
	NamespaceResolver NamespaceResolver
	TableDBResolver   TableDBResolver

	// 是否启用 SQL parser 兜底
	EnableRawSQLRewrite bool
}

type Plugin struct {
	cfg Config
}

func NewPlugin(cfg Config) *Plugin {
	return &Plugin{cfg: cfg}
}

func (p *Plugin) Name() string {
	return PluginName
}

func (p *Plugin) Initialize(db *gorm.DB) error {
	// Query 用 replace，保证能拿到 Build 后的最终 SQL
	if err := db.Callback().Query().Replace("gorm:query", p.queryWithRewrite); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("gorm:query").Register(cbQuery, p.rewrite); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("gorm:row").Register(cbRow, p.rewrite); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("gorm:raw").Register(cbRaw, p.rewrite); err != nil {
		return err
	}
	if err := db.Callback().Create().Before("gorm:create").Register(cbCreate, p.rewrite); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("gorm:update").Register(cbUpdate, p.rewrite); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("gorm:delete").Register(cbDelete, p.rewrite); err != nil {
		return err
	}
	return nil
}

func (p *Plugin) queryWithRewrite(db *gorm.DB) {
	//if db.Error == nil {
	//		BuildQuerySQL(db)
	//
	//		if !db.DryRun && db.Error == nil {
	//			rows, err := db.Statement.ConnPool.QueryContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)
	//			if err != nil {
	//				db.AddError(err)
	//				return
	//			}
	//			defer func() {
	//				db.AddError(rows.Close())
	//			}()
	//			gorm.Scan(rows, db, 0)
	//
	//			if db.Statement.Result != nil {
	//				db.Statement.Result.RowsAffected = db.RowsAffected
	//			}
	//		}
	//	}
	if db.Error != nil {
		return
	}

	stmt := db.Statement
	if stmt == nil || stmt.Context == nil {
		return
	}

	ns := ""
	if p.cfg.NamespaceResolver != nil {
		ns = namespace.NormalizeTenantName(p.cfg.NamespaceResolver(stmt.Context))
	}
	// 配置了兜底租户（环境变量 TENANT_FALLBACK 已设）的 pod，数据保持原始库名，强制不加租户前缀
	if env.HasFallbackTenant() {
		ns = ""
	}

	// 先让 GORM 按原逻辑把 SQL 完整 build 出来
	gormcallbacks.BuildQuerySQL(db)
	if db.Error != nil {
		return
	}

	// 这里拿到的 SQL 已经包含：
	// - 主表
	// - Joins("BVipUser") 生成的 LEFT JOIN
	// - where/order/limit
	// - 子查询
	if ns != "" && stmt.SQL.Len() > 0 {
		// 改写 SQL 的同时必须同步重排 stmt.Vars，二者是位置绑定的。
		applySQLRewrite(db, stmt, ns)
	}

	if !db.DryRun && db.Error == nil {
		rows, err := stmt.ConnPool.QueryContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if err != nil {
			db.AddError(err)
			return
		}
		defer func() {
			db.AddError(rows.Close())
		}()

		gorm.Scan(rows, db, 0)
		if stmt.Result != nil {
			stmt.Result.RowsAffected = db.RowsAffected
		}
	}
}

func (p *Plugin) rewrite(tx *gorm.DB) {
	if tx == nil || tx.Statement == nil || tx.Statement.Context == nil {
		return
	}

	stmt := tx.Statement

	// 忽略 AutoMigrate 等场景
	if stmt.Schema != nil && stmt.Schema.Table == "" {
		return
	}

	// 配置了兜底租户（环境变量 TENANT_FALLBACK 已设）的 pod，数据保持原始库名，不做命名空间改写
	if env.HasFallbackTenant() {
		return
	}
	ns := ""
	if p.cfg.NamespaceResolver != nil {
		ns = namespace.NormalizeTenantName(p.cfg.NamespaceResolver(stmt.Context))
	}
	if ns == "" {
		return
	}

	// 1. 先做结构化重写
	p.rewriteStructured(tx, ns)

	// 2. SQL 兜底
	// 主要处理 Raw / Exec / 无法结构化识别的 raw join
	if p.cfg.EnableRawSQLRewrite && stmt.SQL.Len() > 0 {
		// 同上：SQL 与 Vars 必须一起改。
		applySQLRewrite(tx, stmt, ns)
	}
}

func (p *Plugin) rewriteStructured(tx *gorm.DB, ns string) {
	stmt := tx.Statement
	if stmt == nil {
		return
	}

	p.rewriteMainTable(stmt, ns)
	p.rewriteFromClause(stmt, ns)
	p.rewriteInsertClause(stmt, ns)
	p.rewriteUpdateClause(stmt, ns)
	p.rewriteWhereSubQuery(stmt, ns)
	p.rewriteSelectClause(stmt, ns)
	p.rewriteOrderByClause(stmt, ns)
	p.rewriteGroupByClause(stmt, ns)
}

// =========================
// 主表改写
// =========================

func (p *Plugin) rewriteMainTable(stmt *gorm.Statement, ns string) {
	if stmt == nil {
		return
	}

	// 1. 优先从 TableExpr 取，因为 db.Table("db.table") 的场景里
	// stmt.Table 往往已经只是裸表名，库名仍在 TableExpr 里
	if stmt.TableExpr != nil && strings.TrimSpace(stmt.TableExpr.SQL) != "" {
		if dbName, tableName, ok := extractDBTableFromSQL(stmt.TableExpr.SQL); ok {
			newDB := rewriteDBName(ns, dbName)
			stmt.Table = tableName
			stmt.TableExpr = &clause.Expr{
				SQL: quoteTable(stmt, newDB, tableName),
			}
			return
		}
	}
	// 3. 最后看 Schema.Table
	if stmt.Schema != nil && stmt.Schema.Table != "" {
		if dbName, tableName, ok := splitDBTable(stmt.Schema.Table); ok {
			newDB := rewriteDBName(ns, dbName)
			stmt.Table = tableName
			stmt.TableExpr = &clause.Expr{
				SQL: quoteTable(stmt, newDB, tableName),
			}
			return
		}
	}
	// 2. 再看 stmt.Table
	if stmt.Table != "" {
		if dbName, tableName, ok := splitDBTable(stmt.Table); ok {
			newDB := rewriteDBName(ns, dbName)
			stmt.Table = tableName
			stmt.TableExpr = &clause.Expr{
				SQL: quoteTable(stmt, newDB, tableName),
			}
			return
		}
	}

}

// =========================
// FROM / JOIN 改写
// =========================

func (p *Plugin) rewriteFromClause(stmt *gorm.Statement, ns string) {
	c, ok := stmt.Clauses["FROM"]
	if !ok || c.Expression == nil {
		return
	}

	from, ok := c.Expression.(clause.From)
	if !ok {
		return
	}

	for i := range from.Tables {
		from.Tables[i] = p.rewriteClauseTable(stmt, from.Tables[i], ns)
	}

	for i := range from.Joins {
		j := from.Joins[i]

		// 兼容关联 JOIN，如 Joins("BVipUser")
		if j.Table.Name != "" {
			j.Table = p.rewriteClauseTable(stmt, j.Table, ns)
		}

		// ON 条件里若有子查询
		if j.ON.Exprs != nil {
			p.rewriteExprs(stmt, j.ON.Exprs, ns)
		}

		from.Joins[i] = j
	}

	// 只改 Expression，不重建 Clause
	c.Expression = from
	stmt.Clauses["FROM"] = c
}

func (p *Plugin) rewriteClauseTable(stmt *gorm.Statement, t clause.Table, ns string) clause.Table {
	if t.Name == "" {
		return t
	}

	alias := t.Alias

	// 1. 正常 db.table
	if dbName, tableName, ok := splitDBTable(t.Name); ok {
		newDB := rewriteDBName(ns, dbName)
		t.Name = quoteTable(stmt, newDB, tableName)
		t.Raw = true
		t.Alias = alias
		return t
	}

	// 2. 兼容 `db`.`table`
	if dbName, tableName, ok := extractDBTableFromSQL(t.Name); ok {
		newDB := rewriteDBName(ns, dbName)
		t.Name = quoteTable(stmt, newDB, tableName)
		t.Raw = true
		t.Alias = alias
		return t
	}

	// 3. 关联 JOIN 可能只有裸表名，此时走 resolver
	if p.cfg.TableDBResolver != nil {
		if dbName, ok := p.cfg.TableDBResolver(normalizeIdent(t.Name)); ok {
			newDB := rewriteDBName(ns, dbName)
			t.Name = quoteTable(stmt, newDB, normalizeIdent(t.Name))
			t.Raw = true
			t.Alias = alias
			return t
		}
	}

	return t
}

// =========================
// INSERT / UPDATE 改写
// =========================

func (p *Plugin) rewriteInsertClause(stmt *gorm.Statement, ns string) {
	c, ok := stmt.Clauses["INSERT"]
	if !ok || c.Expression == nil {
		return
	}

	ins, ok := c.Expression.(clause.Insert)
	if !ok {
		return
	}

	if ins.Table.Name != "" {
		ins.Table = p.rewriteClauseTable(stmt, ins.Table, ns)
		c.Expression = ins
		stmt.Clauses["INSERT"] = c
	}
}

func (p *Plugin) rewriteUpdateClause(stmt *gorm.Statement, ns string) {
	c, ok := stmt.Clauses["UPDATE"]
	if !ok || c.Expression == nil {
		return
	}

	upd, ok := c.Expression.(clause.Update)
	if !ok {
		return
	}

	if upd.Table.Name != "" {
		upd.Table = p.rewriteClauseTable(stmt, upd.Table, ns)
		c.Expression = upd
		stmt.Clauses["UPDATE"] = c
	}
}

// =========================
// WHERE / 子查询递归改写
// =========================

func (p *Plugin) rewriteWhereSubQuery(stmt *gorm.Statement, ns string) {
	for k, c := range stmt.Clauses {
		switch expr := c.Expression.(type) {
		case clause.Where:
			p.rewriteExprs(stmt, expr.Exprs, ns)
			c.Expression = expr
			stmt.Clauses[k] = c

		case clause.Set:
			for i := range expr {
				expr[i].Value = p.rewriteValue(expr[i].Value, ns)
			}
			c.Expression = expr
			stmt.Clauses[k] = c
		}
	}
}

func (p *Plugin) rewriteSelectClause(stmt *gorm.Statement, ns string) {
	if ns == "" {
		return
	}
	c, ok := stmt.Clauses["SELECT"]
	if !ok || c.Expression == nil {
		return
	}
	sel, ok := c.Expression.(clause.Select)
	if !ok {
		return
	}
	for i := range sel.Columns {
		sel.Columns[i] = rewriteQualifiedColumn(ns, sel.Columns[i])
	}
	c.Expression = sel
	stmt.Clauses["SELECT"] = c
}

func (p *Plugin) rewriteOrderByClause(stmt *gorm.Statement, ns string) {
	if ns == "" {
		return
	}
	c, ok := stmt.Clauses["ORDER BY"]
	if !ok || c.Expression == nil {
		return
	}
	ob, ok := c.Expression.(clause.OrderBy)
	if !ok {
		return
	}
	for i := range ob.Columns {
		ob.Columns[i].Column = rewriteQualifiedColumn(ns, ob.Columns[i].Column)
	}
	c.Expression = ob
	stmt.Clauses["ORDER BY"] = c
}

func (p *Plugin) rewriteGroupByClause(stmt *gorm.Statement, ns string) {
	if ns == "" {
		return
	}
	c, ok := stmt.Clauses["GROUP BY"]
	if !ok || c.Expression == nil {
		return
	}
	gb, ok := c.Expression.(clause.GroupBy)
	if !ok {
		return
	}
	for i := range gb.Columns {
		gb.Columns[i] = rewriteQualifiedColumn(ns, gb.Columns[i])
	}
	if len(gb.Having) > 0 {
		p.rewriteExprs(stmt, gb.Having, ns)
	}
	c.Expression = gb
	stmt.Clauses["GROUP BY"] = c
}

func (p *Plugin) rewriteExprs(stmt *gorm.Statement, exprs []clause.Expression, ns string) {
	for i := range exprs {
		switch e := exprs[i].(type) {
		case clause.Expr:
			for j := range e.Vars {
				e.Vars[j] = p.rewriteValue(e.Vars[j], ns)
			}
			exprs[i] = e

		case clause.NamedExpr:
			for j := range e.Vars {
				e.Vars[j] = p.rewriteValue(e.Vars[j], ns)
			}
			exprs[i] = e

		case clause.IN:
			e.Column = rewriteColumnInterface(ns, e.Column)
			for j := range e.Values {
				e.Values[j] = p.rewriteValue(e.Values[j], ns)
			}
			exprs[i] = e

		case clause.Eq:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Neq:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Gt:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Gte:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Lt:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Lte:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.Like:
			e.Column = rewriteColumnInterface(ns, e.Column)
			e.Value = p.rewriteValue(e.Value, ns)
			exprs[i] = e

		case clause.AndConditions:
			p.rewriteExprs(stmt, e.Exprs, ns)
			exprs[i] = e

		case clause.OrConditions:
			p.rewriteExprs(stmt, e.Exprs, ns)
			exprs[i] = e

		case clause.NotConditions:
			p.rewriteExprs(stmt, e.Exprs, ns)
			exprs[i] = e
		}
	}
}

func (p *Plugin) rewriteValue(v interface{}, ns string) interface{} {
	sub, ok := v.(*gorm.DB)
	if !ok || sub == nil || sub.Statement == nil {
		return v
	}

	stmt := sub.Statement

	// 先改子查询自己的主表
	p.rewriteMainTable(stmt, ns)

	// 再改子查询自己的结构化 clause
	p.rewriteFromClause(stmt, ns)
	p.rewriteInsertClause(stmt, ns)
	p.rewriteUpdateClause(stmt, ns)
	p.rewriteWhereSubQuery(stmt, ns)
	p.rewriteSelectClause(stmt, ns)
	p.rewriteOrderByClause(stmt, ns)
	p.rewriteGroupByClause(stmt, ns)

	// 如果子查询 SQL 已经生成，则再 parser 兜底（同样要同步重排子查询自己的 Vars）
	if stmt.SQL.Len() > 0 {
		applySQLRewrite(sub, stmt, ns)
	}

	return sub
}

// =========================
// 库名改写：幂等，防重复前缀
// =========================

func rewriteDBName(ns, dbName string) string {
	dbName = normalizeIdent(dbName)
	if dbName == "" || ns == "" {
		return dbName
	}

	if shouldSkipSchema(dbName) {
		return dbName
	}

	if isDBNameRewritten(ns, dbName) {
		return dbName
	}

	return namespaceFmt(ns, dbName)
}

func isDBNameRewritten(ns, dbName string) bool {
	prefix := ns + "_"
	return strings.HasPrefix(dbName, prefix)
}

func shouldSkipSchema(schema string) bool {
	switch strings.ToLower(normalizeIdent(schema)) {
	case "information_schema", "mysql", "performance_schema", "sys":
		return true
	default:
		return false
	}
}

// namespaceFmt 这里包一层，方便你后面替换成自己的 namespace.FmtByType
func namespaceFmt(ns, db string) string {
	// 替换成你自己的：
	// return namespace.FmtByType(namespace.NAMESPACE_TYPE_MYSQL, ns, db)
	return ns + "_" + db
}

// =========================
// 解析 / 工具函数
// =========================

func splitDBTable(s string) (dbName, tableName string, ok bool) {
	s = normalizeIdent(s)
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func extractDBTableFromSQL(sql string) (dbName, tableName string, ok bool) {
	s := normalizeIdent(sql)
	parts := strings.Split(s, ".")
	if len(parts) != 2 {
		return "", "", false
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func normalizeIdent(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "\"", "")
	return s
}

func quoteTable(stmt *gorm.Statement, dbName, tableName string) string {
	var sb strings.Builder
	stmt.DB.Dialector.QuoteTo(&sb, dbName)
	sb.WriteByte('.')
	stmt.DB.Dialector.QuoteTo(&sb, tableName)
	return sb.String()
}

// rewriteQualifiedColumn 将 clause.Column.Table 中「逻辑库.表」的库段按租户改写为物理库，与 rewriteMainTable / FROM 一致。
// gorm-gen 会把 Table 设为「逻辑库.表」单字符串；MySQL dialector 会按 '.' 拆成多段反引号，若库名非真实库会 1054。
// 仅当 Table 恰好为 db.table 两段时改写；表别名等无法拆分的 Table 保持不变。
func rewriteQualifiedColumn(ns string, col clause.Column) clause.Column {
	if ns == "" || strings.TrimSpace(col.Table) == "" {
		return col
	}
	dbName, tableName, ok := splitDBTable(col.Table)
	if !ok {
		return col
	}
	newDB := rewriteDBName(ns, dbName)
	col.Table = newDB + "." + tableName
	return col
}

func rewriteColumnInterface(ns string, col interface{}) interface{} {
	switch c := col.(type) {
	case clause.Column:
		return rewriteQualifiedColumn(ns, c)
	default:
		return col
	}
}

// =========================
// TiDB Parser SQL 兜底
// =========================

var parserPool = sync.Pool{
	New: func() any {
		return parser.New()
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

// rewriteSQLWithParser 使用 TiDB parser 解析 SQL，在 AST 中给所有表引用的 Schema（库名）加前缀，再 Restore 回 SQL。
// 解析失败时返回原 SQL。
//
// ⚠️ Restore 只保证**语义**等价，不保证**占位符（?）的相对位置**不变。
// 典型如分页子句：GORM 生成的 `LIMIT ? OFFSET ?` 会被 Restore 成 MySQL 的 `LIMIT ?,?`
// （(*ast.Limit).Restore 固定先写 Offset 再写 Count），两个占位符位置互换。而调用方的
// args 是按**原 SQL** 中 ? 的出现顺序排列的（GORM 写一个 ? 就追加一个 var），
// 不同步调整就会张冠李戴——limit 与 offset 的值互换，后台列表翻到第 3 页开始重复第 2 页的数据。
//
// 因此这里在 Restore 前把每个 ? 换成携带「原 SQL 字节偏移」的标记字面量，Restore 后按标记
// 在新 SQL 中的出现顺序算出置换，再把标记还原成 ?。调用方据此重排 args（见 permuteVars）。
// 这个做法与「哪条语法会被重排」无关，parser 未来改变行为也自动适配。
//
// 返回值：
//   - newSQL：改写后的 SQL；与入参相同表示未改写
//   - perm：参数置换，newArgs[i] = oldArgs[perm[i]]；nil 表示参数顺序无需调整
//   - ok：false 表示置换不可信，newSQL 是**降级**改写结果（等同本次改动前的算法：
//     照常加库名前缀，但不重排参数），调用方应原样沿用参数。
//     这样最坏情况也只退化成改动前的行为，不会因为放弃改写而丢掉租户前缀。
func rewriteSQLWithParser(sql, prefix string) (string, []int, bool) {
	if prefix == "" || strings.TrimSpace(sql) == "" {
		return sql, nil, true
	}

	// 简单快速判定，没点号通常没必要改
	if !strings.Contains(sql, ".") {
		return sql, nil, true
	}

	restored, injectedOffsets, ok := restoreWithPrefix(sql, prefix, true)
	if !ok {
		// 解析 / Restore 失败：与改动前一致，返回原 SQL
		return sql, nil, true
	}

	if out, perm, valid := extractParamOrder(restored, injectedOffsets, sql); valid {
		return out, perm, true
	}

	// 置换不可信 → 降级成「不注入标记」的改写，行为与改动前完全一致
	return rewriteSQLPlain(sql, prefix), nil, false
}

// rewriteSQLPlain 只加库名前缀、不做占位符标记，等价于本次改动前的 rewriteSQLWithParser。
// 作为一致性校验失败时的降级路径：保证最坏结果不超出改动前的错误范围。
func rewriteSQLPlain(sql, prefix string) string {
	plain, _, ok := restoreWithPrefix(sql, prefix, false)
	if !ok {
		return sql
	}
	return plain
}

// restoreWithPrefix 解析 SQL、给库名加前缀、再 Restore。
// markParams 为 true 时同时把每个占位符替换成可识别的标记，返回值 injected 为替换个数。
func restoreWithPrefix(sql, prefix string, markParams bool) (restored string, injectedOffsets []int, ok bool) {
	p := parserPool.Get().(*parser.Parser)
	defer func() {
		p.Reset()
		parserPool.Put(p)
	}()

	stmts, _, err := p.Parse(sql, "", "")
	if err != nil || len(stmts) == 0 {
		return sql, nil, false
	}

	v := &schemaPrefixVisitor{prefix: prefix, markParams: markParams}

	out := bufferPool.Get().(*bytes.Buffer)
	out.Reset()
	defer bufferPool.Put(out)

	for i, stmt := range stmts {
		if i > 0 {
			out.WriteByte(';')
		}
		stmt.Accept(v)
		ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, out)
		if err := stmt.Restore(ctx); err != nil {
			return sql, nil, false
		}
	}
	return out.String(), v.paramOffsets, true
}

type schemaPrefixVisitor struct {
	ast.Node
	prefix string
	// markParams 为 false 时退化成改动前的 visitor：只改库名，不碰占位符。
	markParams bool
	// paramOffsets 记录本次替换掉的每个占位符在原 SQL 中的偏移，用于 Restore 后的一致性校验。
	paramOffsets []int
}

func (v *schemaPrefixVisitor) Enter(in ast.Node) (ast.Node, bool) {
	switch n := in.(type) {
	case *ast.TableName:
		if n.Schema.O != "" {
			if !shouldSkipSchema(n.Schema.O) && !isDBNameRewritten(v.prefix, n.Schema.O) {
				n.Schema.O = namespaceFmt(v.prefix, n.Schema.O)
				n.Schema.L = strings.ToLower(n.Schema.O)
			}
		}
	case *ast.ColumnName:
		if n.Schema.O != "" {
			if !shouldSkipSchema(n.Schema.O) && !isDBNameRewritten(v.prefix, n.Schema.O) {
				n.Schema.O = namespaceFmt(v.prefix, n.Schema.O)
				n.Schema.L = strings.ToLower(n.Schema.O)
			}
		}
	}
	return in, false
}

// Leave 把每个占位符（?）替换成携带原 SQL 字节偏移的标记字面量，使它在 Restore 之后
// 仍然可以被认出来、从而算出参数置换。见 rewriteSQLWithParser 的说明。
//
// 注：不能用 ast.Node 的 OriginTextPosition()——实测它只对 WHERE 等表达式位置有值，
// 对 LIMIT / OFFSET 的占位符恒为 0（语法里没走 yySetOffset），而那正是会被重排的位置。
// test_driver.ParamMarkerExpr.Offset 是构造时写入的，四个位置都准确。
func (v *schemaPrefixVisitor) Leave(in ast.Node) (ast.Node, bool) {
	if !v.markParams {
		return in, true
	}
	if pm, ok := in.(*testdriver.ParamMarkerExpr); ok {
		v.paramOffsets = append(v.paramOffsets, pm.Offset)
		return ast.NewValueExpr(paramMarker(pm.Offset), "", ""), true
	}
	return in, true
}

// paramMarkerSalt 每个进程启动时随机生成，拼进标记里。
// 目的是让外部内容（用户数据被内联进 SQL 文本的情况）无法伪造出与本进程一致的标记，
// 从而无法干扰占位符识别。Go 1.24 起 crypto/rand.Read 不再返回错误（内部失败直接 panic），
// 这里的错误分支只是防御性写法。
var paramMarkerSalt = func() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "0"
	}
	return strconv.FormatUint(uint64(binary.BigEndian.Uint32(b[:])), 36)
}()

var paramMarkerPrefix = "__CGP" + paramMarkerSalt + "_"

func paramMarker(offset int) string {
	return paramMarkerPrefix + strconv.Itoa(offset) + "__"
}

// nextParamMarker 从 s[from:] 起找下一个标记字面量，返回它在 s 中的
// [start,end) 区间与解析出的偏移。没找到时 ok=false。
//
// 标记的完整形态是 [字符集导引符]'<prefix><digits>__'，例如 _UTF8MB4'__CGP1a2b_75__'。
// 这里手写单遍扫描而不用正则：大 IN 列表动辄上千个占位符，正则扫描是热点。
func nextParamMarker(s string, from int) (start, end, offset int, ok bool) {
	for {
		i := strings.Index(s[from:], paramMarkerPrefix)
		if i < 0 {
			return 0, 0, 0, false
		}
		i += from
		from = i + len(paramMarkerPrefix)

		// 前一个字符必须是引号
		if i == 0 || s[i-1] != '\'' {
			continue
		}

		// 数字 + "__'"
		j := from
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		// j == from（数字段为空）与下面的 Atoi 判据等价，留着只是提前短路
		if j == from || j+3 > len(s) || s[j] != '_' || s[j+1] != '_' || s[j+2] != '\'' {
			continue
		}
		off, err := strconv.Atoi(s[from:j])
		if err != nil {
			continue
		}

		// 向前吃掉可能存在的字符集导引符，如 _UTF8MB4'...'
		st := i - 1
		k := st - 1
		for k >= 0 && isIdentByte(s[k]) {
			k--
		}
		if k+1 < st && s[k+1] == '_' {
			st = k + 1
		}

		return st, j + 3, off, true
	}
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// extractParamOrder 把 Restore 输出里的标记还原成 ?，并算出参数置换。
//
// 标记内容是该占位符在**原 SQL** 中的字节偏移：偏移升序 = 调用方 args 的原始顺序，
// 标记在**新 SQL** 中的出现顺序 = args 需要被重排成的顺序，两者比对即得置换。
//
// 原 SQL 里可能本来就存在长得像标记的字符串字面量（如 WHERE name='__CGP_5__'），
// 因此这里额外用 origSQL 校验：标记里的数字必须真的指向原 SQL 中的一个 ?。
// 对不上的按普通字面量放过，既不会被误当成占位符，也不影响本次库名改写。
//
// 一致性校验不通过时返回 (原SQL, nil, false)，由调用方放弃改写——
// 宁可不加租户前缀，也不能拿着对不上的参数去执行。
func extractParamOrder(restored string, injectedOffsets []int, origSQL string) (string, []int, bool) {
	if len(injectedOffsets) == 0 && !strings.Contains(restored, paramMarkerPrefix) {
		return restored, nil, true
	}

	// 只认本次真正注入过的偏移，且同一偏移至多出现一次。
	pending := make(map[int]bool, len(injectedOffsets))
	for _, off := range injectedOffsets {
		pending[off] = true
	}
	consumed := make(map[int]bool, len(injectedOffsets))

	offsets := make([]int, 0, len(injectedOffsets))
	var b strings.Builder
	b.Grow(len(restored))
	prev, scan := 0, 0
	for {
		start, end, off, found := nextParamMarker(restored, scan)
		if !found {
			break
		}
		scan = end

		if consumed[off] {
			// 同一偏移的标记出现两次：要么 Restore 复制了占位符，要么外部字面量抢占了
			// 配额。继续替换会静默产出错误的 SQL 与参数绑定，直接判不可信。
			return origSQL, nil, false
		}
		if !pending[off] {
			// 不是本次注入的标记 → 当普通字面量原样保留
			continue
		}
		if start < prev {
			// 与上一个已消费标记重叠（导引符回扫越过了边界）。不替换，交给后面的
			// 数量校验判成不可信，同时避免 restored[prev:start] 切片越界 panic。
			continue
		}
		delete(pending, off)
		consumed[off] = true
		offsets = append(offsets, off)
		b.WriteString(restored[prev:start])
		b.WriteByte('?')
		prev = end
	}
	b.WriteString(restored[prev:])

	if len(offsets) != len(injectedOffsets) {
		// 注入了 n 个标记却只认回 m 个，说明 Restore 吞掉了占位符，结果不可信
		return origSQL, nil, false
	}

	perm := permutationFromOffsets(offsets)
	if perm == nil {
		return origSQL, nil, false
	}
	return b.String(), perm, true
}

// permutationFromOffsets 由「标记的原始偏移」序列推出置换：perm[i] = 第 i 个新位置该取的旧下标。
// 偏移必须两两不同，否则无法建立一一对应，返回 nil。
func permutationFromOffsets(offsets []int) []int {
	sorted := make([]int, len(offsets))
	copy(sorted, offsets)
	sort.Ints(sorted)

	rank := make(map[int]int, len(sorted))
	for i, off := range sorted {
		if _, dup := rank[off]; dup {
			return nil
		}
		rank[off] = i
	}

	perm := make([]int, len(offsets))
	for i, off := range offsets {
		perm[i] = rank[off]
	}
	return perm
}

// permuteVars 按 perm 重排参数：newVars[i] = vars[perm[i]]。
// perm 为空表示无需重排；长度对不上或不是合法置换时返回 ok=false。
func permuteVars(vars []interface{}, perm []int) ([]interface{}, bool) {
	if len(perm) == 0 {
		return vars, true
	}
	if len(perm) != len(vars) {
		return nil, false
	}

	out := make([]interface{}, len(vars))
	seen := make([]bool, len(vars))
	for i, p := range perm {
		if p < 0 || p >= len(vars) || seen[p] {
			return nil, false
		}
		seen[p] = true
		out[i] = vars[p]
	}
	return out, true
}

// applySQLRewrite 把「改写 SQL + 同步重排参数」这件事收成一处，供各回调复用。
// 校验不通过时保持 stmt 原样不动，并记录错误日志。
// applySQLRewrite 改写 SQL 并同步重排参数——两者位置绑定，必须一起改。
//
// 降级原则：占位符置换一旦不可信，就退回本次改动前的行为（照常加库名前缀、参数不重排），
// 而不是放弃改写。放弃改写会让 SQL 带着裸库名执行，可能读到别的租户，
// 那是改动前不存在的新故障；降级则最坏只退化成改动前就有的参数错位。
// 两种降级都会打 error 日志。
func applySQLRewrite(db *gorm.DB, stmt *gorm.Statement, ns string) {
	// 子查询路径上的 *gorm.DB 由调用方塞进 clause，理论上可能是手工构造的裸对象，
	// 此时 Config/Logger 为 nil。降级日志不能因此 panic。
	logf := func(format string, args ...interface{}) {
		if db == nil || db.Config == nil || db.Logger == nil {
			return
		}
		db.Logger.Error(stmt.Context, format, args...)
	}

	oldSQL := stmt.SQL.String()
	newSQL, perm, ok := rewriteSQLWithParser(oldSQL, ns)
	if newSQL == "" || newSQL == oldSQL {
		return
	}

	// 置换不可信：newSQL 已经是降级改写结果，参数原样沿用
	if !ok {
		logf(
			"[tenant] 占位符一致性校验未通过，已降级为不重排参数的改写（行为同修复前，分页可能错位）: %s",
			oldSQL)
		writeSQL(stmt, newSQL, stmt.Vars)
		return
	}

	// perm 为 nil 说明一个占位符都没识别到。若语句本身带参数，就意味着占位符节点没被
	// 认出来（例如日后换了 parser driver，Leave 里的类型断言不再命中）——此时参数不会
	// 被重排，等于静默退回修复前的行为。SQL 照常改写，但必须报出来。
	if len(stmt.Vars) > 0 && len(perm) == 0 {
		logf(
			"[tenant] SQL 带 %d 个参数却未识别到任何占位符，占位符识别可能已失效: %s",
			len(stmt.Vars), oldSQL)
		writeSQL(stmt, newSQL, stmt.Vars)
		return
	}

	newVars, ok := permuteVars(stmt.Vars, perm)
	if !ok {
		logf(
			"[tenant] 参数与占位符数量不匹配，已降级为不重排参数的改写: sql=%s vars=%d perm=%v",
			oldSQL, len(stmt.Vars), perm)
		writeSQL(stmt, rewriteSQLPlain(oldSQL, ns), stmt.Vars)
		return
	}

	writeSQL(stmt, newSQL, newVars)
}

func writeSQL(stmt *gorm.Statement, sql string, vars []interface{}) {
	stmt.SQL.Reset()
	stmt.SQL.WriteString(sql)
	stmt.Vars = vars
}
