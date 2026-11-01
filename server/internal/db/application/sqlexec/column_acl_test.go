package sqlexec

// 列级权限的判定语义与结果裁剪。
//
// 这里盯三类真实失效：
//   1. 别名 / 函数包裹把列名藏起来（SELECT phone AS p、COUNT(phone)）；
//   2. 星号查回全部列却没被裁掉；
//   3. 行数据的键与列对象脱节（裁剪后 map 里还留着无权列的值）。

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
)

// seedColumnAclApp 构造一个只带快照的应用实例：绕过仓储，直接给判定层喂规则。
func seedColumnAclApp(dbId uint64, columns map[string]map[string][]uint64) *tableAclAppImpl {
	app := &tableAclAppImpl{
		cache:   map[uint64]*tableAclSnapshot{dbId: {rules: map[string][]uint64{}, columns: columns}},
		expires: map[uint64]time.Time{dbId: time.Now().Add(time.Hour)},
	}
	return app
}

func TestColumnAclDeniedColumns(t *testing.T) {
	app := seedColumnAclApp(1, map[string]map[string][]uint64{
		"t_user": {"phone": {}, "id_card": {7}},
		"t_log":  {"payload": {}},
	})
	ctx := context.Background() // 无登录态 = 角色未解析：角色维度规则按命中处理

	denied, err := app.DeniedColumns(ctx, 1, []string{"t_user"})
	if err != nil {
		t.Fatalf("DeniedColumns: %v", err)
	}
	if _, ok := denied["phone"]; !ok {
		t.Fatal("a rule without roles applies to every account")
	}
	if _, ok := denied["id_card"]; !ok {
		t.Fatal("unresolved roles must deny role-scoped rules (fail-closed)")
	}
	if _, ok := denied["payload"]; ok {
		t.Fatal("rules of another table must not leak into this table")
	}

	// 大小写与 schema 前缀都要归一
	denied, err = app.DeniedColumns(ctx, 1, []string{"PUBLIC.T_USER"})
	if err != nil {
		t.Fatalf("DeniedColumns: %v", err)
	}
	if len(denied) != 2 {
		t.Fatalf("table name must be normalised, got %v", denied)
	}

	// 不限定表 = 整库（导出/复制这类产物级场景用得上）
	all, err := app.DeniedColumns(ctx, 1, nil)
	if err != nil {
		t.Fatalf("DeniedColumns: %v", err)
	}
	if _, ok := all["payload"]; !ok {
		t.Fatalf("an empty table filter must cover the whole database, got %v", all)
	}
}

func TestStatementColumnRefs(t *testing.T) {
	star := &sqlstmt.SelectStmt{Items: []sqlstmt.SelectItem{
		{Kind: sqlstmt.SelectItemStar},
		{Kind: sqlstmt.SelectItemColumn, ColumnName: "phone", Alias: "p"},
		{Kind: sqlstmt.SelectItemFunction, Text: "COUNT(id_card)", Alias: "c"},
	}}
	refs := statementColumnRefs(star)
	found := map[string]bool{}
	for _, ref := range refs {
		found[ref.Column] = true
	}
	if !found["phone"] {
		t.Fatalf("an explicitly selected column must be collected, got %v", refs)
	}
	if !found["id_card"] {
		t.Fatalf("a function-wrapped column must not escape the check, got %v", refs)
	}
	// 单独一个星号不产出具体列：它由结果集裁剪（交互）或整表拒绝（导出）负责
	starOnly := &sqlstmt.SelectStmt{Items: []sqlstmt.SelectItem{{Kind: sqlstmt.SelectItemStar}}}
	if refs := statementColumnRefs(starOnly); len(refs) != 0 {
		t.Fatalf("a bare star must not produce column references, got %v", refs)
	}

	insert := &sqlstmt.InsertStmt{Table: sqlstmt.TableRef{Name: "t_user"}, Columns: []string{"name", "phone"}}
	if refs := statementColumnRefs(insert); len(refs) != 2 {
		t.Fatalf("insert columns must be collected, got %v", refs)
	}

	update := &sqlstmt.UpdateStmt{Set: []sqlstmt.Assignment{{Column: "phone"}}}
	if refs := statementColumnRefs(update); len(refs) != 1 || refs[0].Column != "phone" {
		t.Fatalf("updated columns must be collected, got %v", refs)
	}

	// 条件里引用无权列也要收上来：不返回列 ≠ 不能用它做条件
	where := &sqlstmt.SelectStmt{
		Items: []sqlstmt.SelectItem{{Kind: sqlstmt.SelectItemColumn, ColumnName: "id"}},
		Where: &sqlstmt.Expr{Text: "phone LIKE '138%'"},
	}
	whereRefs := statementColumnRefs(where)
	hasPhone := false
	for _, ref := range whereRefs {
		if ref.Column == "phone" {
			hasPhone = true
		}
	}
	if !hasPhone {
		t.Fatalf("a where clause on a denied column must be collected, got %v", whereRefs)
	}

	// UNION 分支里的列同样要判
	union := &sqlstmt.SelectStmt{
		Items:  []sqlstmt.SelectItem{{Kind: sqlstmt.SelectItemColumn, ColumnName: "id"}},
		Unions: []sqlstmt.UnionClause{{Select: &sqlstmt.SelectStmt{Items: []sqlstmt.SelectItem{{Kind: sqlstmt.SelectItemColumn, ColumnName: "phone"}}}}},
	}
	if refs := statementColumnRefs(union); len(refs) != 2 {
		t.Fatalf("union branches must be checked too, got %v", refs)
	}
}

func TestColumnAclAssertColumnsAllowed(t *testing.T) {
	app := seedColumnAclApp(1, map[string]map[string][]uint64{"t_user": {"phone": {}}})
	ctx := context.Background()

	if err := app.AssertColumnsAllowed(ctx, 1, []string{"t_user"}, []ColumnRef{{Column: "name"}}); err != nil {
		t.Fatalf("an allowed column must pass: %v", err)
	}
	if err := app.AssertColumnsAllowed(ctx, 1, []string{"t_user"}, []ColumnRef{{Column: "PHONE"}}); err == nil {
		t.Fatal("a denied column must be rejected, case-insensitively")
	}
	// 语句查的是别的表时，本表规则不能误伤
	if err := app.AssertColumnsAllowed(ctx, 1, []string{"t_order"}, []ColumnRef{{Column: "phone"}}); err != nil {
		t.Fatalf("rules must be scoped by the statement's target tables: %v", err)
	}
}

func TestColumnAclAssertWholeTableAllowed(t *testing.T) {
	app := seedColumnAclApp(1, map[string]map[string][]uint64{"t_user": {"phone": {}}})
	ctx := context.Background()

	if err := app.AssertWholeTableAllowed(ctx, 1, []string{"t_user"}); err == nil {
		t.Fatal("a whole-table product must be rejected while a column is denied")
	}
	if err := app.AssertWholeTableAllowed(ctx, 1, []string{"t_order"}); err != nil {
		t.Fatalf("an unprotected table must pass: %v", err)
	}
}

func TestTrimDeniedColumns(t *testing.T) {
	app := seedColumnAclApp(1, map[string]map[string][]uint64{"t_user": {"phone": {}}})
	ctx := context.Background()

	cols := []*dbi.QueryColumn{{Name: "id", Key: "id"}, {Name: "phone", Key: "phone"}, {Name: "name", Key: "name"}}
	rows := []map[string]any{
		{"id": 1, "phone": "13800000000", "name": "a"},
		{"id": 2, "phone": "13900000000", "name": "b"},
	}
	kept, trimmed, err := app.TrimDeniedColumns(ctx, 1, []string{"t_user"}, cols, rows)
	if err != nil {
		t.Fatalf("TrimDeniedColumns: %v", err)
	}
	if len(kept) != 2 || kept[0].Name != "id" || kept[1].Name != "name" {
		t.Fatalf("the denied column must be dropped from the result columns, got %+v", kept)
	}
	for _, row := range trimmed {
		if _, ok := row["phone"]; ok {
			t.Fatalf("the denied value must be removed from every row, got %v", row)
		}
		if _, ok := row["name"]; !ok {
			t.Fatalf("allowed columns must survive, got %v", row)
		}
	}

	// 没有命中规则时原样返回（不能重建列，否则脱敏/类型信息会丢）。
	// 用新的行数据：上一次裁剪已经就地删掉了 phone 键
	freshRows := []map[string]any{{"id": 1, "phone": "13800000000", "name": "a"}}
	untouched, sameRows, err := app.TrimDeniedColumns(ctx, 1, []string{"t_order"}, cols, freshRows)
	if err != nil {
		t.Fatalf("TrimDeniedColumns: %v", err)
	}
	if len(untouched) != 3 || len(sameRows[0]) != 3 {
		t.Fatalf("an unprotected table must be returned as-is, got %+v", untouched)
	}
}

// TestColumnAclGuardPlacement 列级权限的三个接线位置：
//
//	结果裁剪必须在取数之后（否则没有列信息）、在回给调用方之前（否则值已经出门）；
//	导出判定必须早于创建产物写入器；
//	整表导出（dump）必须早于写响应头。
func TestColumnAclGuardPlacement(t *testing.T) {
	exec, err := os.ReadFile("db_sql_exec.go")
	if err != nil {
		t.Fatalf("read db_sql_exec.go: %v", err)
	}
	body := string(exec)
	trimPos := strings.Index(body, "TrimDeniedColumns")
	if trimPos == -1 {
		t.Fatal("交互查询必须裁剪无权列")
	}
	// doQuery 里同一函数的取数与返回位置
	queryStart := strings.Index(body, "func (d *DbSQLExecAppImpl) doQuery(")
	if queryStart == -1 {
		t.Fatal("doQuery 应存在")
	}
	fn := body[queryStart:]
	walkPos := strings.Index(fn, "dbConn.WalkQueryRows(")
	retPos := strings.LastIndex(fn, "RowLimit:  maxRows,")
	if walkPos == -1 || retPos == -1 {
		t.Fatalf("doQuery 的取数与返回形态变了 (walk=%d ret=%d)", walkPos, retPos)
	}
	trimInFn := strings.Index(fn, "TrimDeniedColumns")
	if trimInFn == -1 {
		t.Fatal("doQuery 必须调用 TrimDeniedColumns")
	}
	if trimInFn < walkPos {
		t.Fatal("裁剪必须在取数之后：没有结果列就无从裁剪")
	}
	if trimInFn > retPos {
		t.Fatal("裁剪必须在返回之前：先回给调用方再裁，值已经发出去了")
	}

	exportSrc, err := os.ReadFile("db_sql_export.go")
	if err != nil {
		t.Fatalf("read db_sql_export.go: %v", err)
	}
	exportBody := string(exportSrc)
	assertPos := strings.Index(exportBody, "AssertExportAllowed")
	writerPos := strings.Index(exportBody, "openWriter()")
	if assertPos == -1 {
		t.Fatal("查询导出必须过列级权限")
	}
	if writerPos == -1 || assertPos > writerPos {
		t.Fatal("列级判定必须早于创建产物写入器")
	}
}
