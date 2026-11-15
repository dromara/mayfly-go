package sqlutil

// 语句里表/列引用的 AST 级抽取（单一真源）。
//
// 两个消费者语义不同，但必须基于同一份抽取，否则同一句 SQL 在两处的「引用了什么」会漂移：
//   - 列级访问控制（sqlexec）：拒绝判定，宁可误拦不可放过；
//   - SQL 草稿校验（sqldraft）：标识符存在性核对，只产出提示，不参与安全判定。
//
// 边界：本文件是**近似清单**。子查询/CTE/INSERT…SELECT 的来源表无法在此完整枚举
// （WithStmt 不保留内层结构），故 StatementTables 对这类语句可能少报。安全判定一律走
// sqlexec 的语句级 ACL（它有 aclTableScope 的可信度判据），不得以本文件做放行依据。

import (
	"strings"

	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// ColumnRef 语句里显式写出的一处列引用。
//
// Table 允许为空：SELECT 列表里 `t.col` 拿得到表别名，但别名到真表的映射要跨 JOIN 解析；
// 列级判定按「任一目标表命中即拒」处理，宁可多拦一次，也不让 JOIN 写法成为旁路。
type ColumnRef struct {
	Table  string
	Column string
}

// StatementColumnRefs 语句里显式出现的列引用（用于列级拒绝判定）。
//
// 表达式里的列不会被结构化拆出来（COUNT(secret)、CONCAT(a, b)），因此对表达式文本
// 再做一次整词比对：这一侧宁可误拦（把同名列都算上），也不能让「外头包一层函数」成为旁路。
func StatementColumnRefs(stmt sqlstmt.Stmt) []ColumnRef {
	switch typed := stmt.(type) {
	case *sqlstmt.SelectStmt:
		refs := selectColumnRefs(typed)
		// 条件里引用无权列同样要拦：`WHERE phone LIKE '138%'` 能把值一点点试出来，
		// 「不返回这一列」并不等于「不能用这一列做条件」
		refs = append(refs, exprColumnRefs(typed.Where)...)
		// ORDER BY / GROUP BY / HAVING 同属「用这一列做条件」：
		// 按无权列排序就能二分出它的值，按它分组计数更直接暴露取值分布
		for _, group := range typed.GroupBy {
			refs = append(refs, ColumnRef{Column: group})
		}
		refs = append(refs, exprColumnRefs(typed.Having)...)
		for _, order := range typed.OrderBy {
			refs = append(refs, exprColumnRefs(&sqlstmt.Expr{Text: order.Text})...)
		}
		return refs
	case *sqlstmt.InsertStmt:
		// VALUES 里出现的列引用（如 `INSERT INTO t (a) SELECT phone FROM u` 由 Select 分支覆盖）
		refs := make([]ColumnRef, 0, len(typed.Columns))
		for _, column := range typed.Columns {
			refs = append(refs, ColumnRef{Table: typed.Table.Name, Column: column})
		}
		return refs
	case *sqlstmt.UpdateStmt:
		refs := make([]ColumnRef, 0, len(typed.Set)+8)
		for _, assignment := range typed.Set {
			refs = append(refs, ColumnRef{Column: assignment.Column})
		}
		return append(refs, exprColumnRefs(typed.Where)...)
	case *sqlstmt.DeleteStmt:
		// 删除条件引用无权列：删除行为本身会暴露「该列等于某值时有多少行」
		return exprColumnRefs(typed.Where)
	}
	return nil
}

// StatementTables 语句里可静态确定的表名（去重、保持出现顺序、忽略 schema 前缀）。
//
// 只处理「表引用一目了然」的四种形态；CTE / 子查询 / INSERT…SELECT 的来源表返回空，
// 由调用方据此把结论降级为「未能核对」而不是「核对通过」。
func StatementTables(stmt sqlstmt.Stmt) []string {
	var tables []string
	seen := make(map[string]struct{})
	add := func(ref sqlstmt.TableRef) {
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		tables = append(tables, name)
	}

	switch typed := stmt.(type) {
	case *sqlstmt.SelectStmt:
		collectSelectTables(typed, add)
	case *sqlstmt.InsertStmt:
		add(typed.Table)
	case *sqlstmt.UpdateStmt:
		for _, table := range typed.Tables {
			add(table)
		}
	case *sqlstmt.DeleteStmt:
		for _, table := range typed.Tables {
			add(table)
		}
	}
	return tables
}

// collectSelectTables 递归收 FROM / JOIN / UNION 各分支的表引用
func collectSelectTables(stmt *sqlstmt.SelectStmt, add func(sqlstmt.TableRef)) {
	if stmt == nil {
		return
	}
	for _, from := range stmt.From {
		add(from)
	}
	for _, join := range stmt.Joins {
		add(join.Table)
	}
	for _, union := range stmt.Unions {
		collectSelectTables(union.Select, add)
	}
}

// IdentifierWords 从表达式文本里取标识符（字母/下划线开头）。
//
// 不做关键字过滤：本函数只用于「与具体列名/表名比对」，关键字永远配不上真正的标识符，
// 多留几个词不会带来误判，却能避免维护一份与解析器不同步的关键字表。
func IdentifierWords(text string) []string {
	words := make([]string, 0, 8)
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	for _, r := range text {
		switch {
		case r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			current.WriteRune(r)
		default:
			flush()
		}
	}
	flush()
	return words
}

// exprColumnRefs 从条件表达式里取列引用（按文本整词，保守取全）
func exprColumnRefs(expr *sqlstmt.Expr) []ColumnRef {
	if expr == nil {
		return nil
	}
	words := IdentifierWords(expr.Text)
	refs := make([]ColumnRef, 0, len(words))
	for _, word := range words {
		refs = append(refs, ColumnRef{Column: word})
	}
	return refs
}

// selectColumnRefs 查询列表里的列引用（含 UNION 各分支）
func selectColumnRefs(stmt *sqlstmt.SelectStmt) []ColumnRef {
	if stmt == nil {
		return nil
	}
	refs := make([]ColumnRef, 0, len(stmt.Items))
	for _, item := range stmt.Items {
		if item.IsStar() {
			// 星号代表「全部列」：整表导出场景由 AssertWholeTableAllowed 拒，
			// 交互查询场景由结果集裁剪兜住，这里不产生具体列引用
			continue
		}
		if item.ColumnName != "" {
			refs = append(refs, ColumnRef{Table: item.TableAlias, Column: item.ColumnName})
		}
		for _, word := range IdentifierWords(item.Text) {
			refs = append(refs, ColumnRef{Table: item.TableAlias, Column: word})
		}
	}
	for _, union := range stmt.Unions {
		refs = append(refs, selectColumnRefs(union.Select)...)
	}
	return refs
}
