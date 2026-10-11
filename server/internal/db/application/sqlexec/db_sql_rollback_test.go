package sqlexec

// 回滚 SQL 生成的单元测试（无需连库的部分）。
//
// 这里钉的是「渲染旧值」这件事上的静默失真：BIGINT 主键丢位、二进制列把 hex 文本误当值解码写回、
// NULL 被渲染成空串。它们生成的语句都能正常执行，错的只是数据——而回滚 SQL 一旦改错数据，
// 用户看到的却是「执行成功」。
//
// 需要真表事实的部分（列/主键/表结构变化）在 db_sql_rollback_it_test.go

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

// TestRollbackLiteralKeepsBigIntKeyDigits 超过 float64 精度的 BIGINT 主键必须逐位保留。
//
// 默认 JSON 解析会把数字读成 float64，9007199254740993 变成 ...992，
// 生成的 WHERE 落到另一行（或 0 行），而 0 行在界面上与成功还原完全同形
func TestRollbackLiteralKeepsBigIntKeyDigits(t *testing.T) {
	rows, err := decodeOldValueRows(context.Background(), `[{"id":9007199254740993,"name":"a"}]`)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	if _, isNumber := rows[0]["id"].(json.Number); !isNumber {
		t.Fatalf("the key must be decoded as json.Number, got %T", rows[0]["id"])
	}

	column := dbi.Column{ColumnName: "id", DataType: "bigint"}
	got := rollbackLiteral(dbi.DbType("mysql"), column, rows[0]["id"])
	if got != "9007199254740993" {
		t.Fatalf("bigint key literal = %q, want the exact digits", got)
	}
}

// TestRollbackLiteralRendersNullAndBinary 三个容易走偏的取值形态。
//
// 二进制这条钉的是 hex 而非 base64：旧值里的二进制列经 dbi 的 ValuerBytes 归一为 **hex 文本**，
// 渲染必须原样包成各方言的十六进制字面量（mysql X'...'、pg '\x...'）。绝不能对 hex 文本再做
// base64 解码——hex 字符集是 base64 的子集，长度为 4 的倍数的 hex 串（偶数字节值极常见）会被
// base64 静默解成完全不同的字节，回滚的 SET 值与 CAS 条件双双失真，且失真后的语句仍能「执行成功」
func TestRollbackLiteralRendersNullAndBinary(t *testing.T) {
	if got := rollbackLiteral(dbi.DbType("mysql"), dbi.Column{ColumnName: "c", DataType: "varchar"}, nil); got != "NULL" {
		t.Fatalf("a nil old value must render as NULL, got %q", got)
	}

	// "deadbeef" = 4 字节的 hex 编码：长度为 4 的倍数且全是 base64 合法字符，
	// 正是旧 base64 解码分支会静默失真（解出 6 个错误字节）的形态
	const payloadHex = "deadbeef"
	binaryColumn := dbi.Column{ColumnName: "payload", DataType: "blob"}
	if got := rollbackLiteral(dbi.DbType("mysql"), binaryColumn, payloadHex); got != "X'"+payloadHex+"'" {
		t.Fatalf("mysql binary hex value must render as X'hex' verbatim, got %q", got)
	}
	// 跨方言保真：pg bytea 的 hex 字面量形态不同（'\x...'），但同样必须原样保留 hex 而非解码
	pgColumn := dbi.Column{ColumnName: "payload", DataType: "bytea"}
	if got := rollbackLiteral(dbi.DbType("postgres"), pgColumn, payloadHex); got != "'\\x"+payloadHex+"'" {
		t.Fatalf("pg binary hex value must render as '\\xhex' verbatim, got %q", got)
	}

	// 数字落到文本列时必须带引号，否则执行时才报类型错误（而回滚 SQL 是给人复核的）
	text := rollbackLiteral(dbi.DbType("mysql"), dbi.Column{ColumnName: "code", DataType: "varchar"}, json.Number("12"))
	if !strings.HasPrefix(text, "'") {
		t.Fatalf("a numeric value for a varchar column must be quoted, got %q", text)
	}
}

// TestRollbackLiteralKeepsBitValueNumeric 位列（BIT）必须按数值渲染，不能被塌缩成布尔字面量。
//
// mysql BIT(8) 可存 0..255，旧值经 ValuerBit 归一为整数。曾对 TCBit 强走 SQLValueBool，
// 把 255 塌缩成 true（mysql 里 true=1）——回滚 SET 写回 1、CAS 条件 flags=true 也匹配不上 255，
// 数据被静默改写。渲染必须委托方言 codec（DTBit→SQLValueNumeric）原样输出数值
func TestRollbackLiteralKeepsBitValueNumeric(t *testing.T) {
	bitColumn := dbi.Column{ColumnName: "flags", DataType: "bit"}
	for _, tc := range []struct{ in, want string }{
		{"255", "255"},
		{"1", "1"},
		{"0", "0"},
	} {
		if got := rollbackLiteral(dbi.DbType("mysql"), bitColumn, json.Number(tc.in)); got != tc.want {
			t.Fatalf("mysql BIT value %s must render as the bare number %s, got %q", tc.in, tc.want, got)
		}
	}
}

// TestLookupRowValueToleratesAliasPrefix 旧值行里的键可能带别名前缀（SELECT t.name 的返回形态），
// 定位主键时按末段匹配，否则会误判「旧值里缺主键」而拒绝生成
func TestLookupRowValueToleratesAliasPrefix(t *testing.T) {
	row := map[string]any{"t.id": json.Number("7"), "t.name": "a"}
	value, found := LookupRowValue(row, "id")
	if !found {
		t.Fatalf("the aliased key must be found by its last segment")
	}
	if number, ok := value.(json.Number); !ok || number.String() != "7" {
		t.Fatalf("unexpected value: %#v", value)
	}

	if _, found := LookupRowValue(row, "missing"); found {
		t.Fatal("an absent column must not be reported as found")
	}
}

// TestColumnsOfRejectsDroppedColumn 旧值里的列已被删掉时必须停下：
// 继续生成会让值与列错位，把 A 列的旧值写进 B 列
func TestColumnsOfRejectsDroppedColumn(t *testing.T) {
	table := &rollbackTable{
		name:        "t",
		order:       []dbi.Column{{ColumnName: "id", DataType: "int"}, {ColumnName: "name", DataType: "varchar"}},
		byName:      map[string]dbi.Column{"id": {ColumnName: "id", DataType: "int"}, "name": {ColumnName: "name", DataType: "varchar"}},
		primaryKeys: []string{"id"},
	}

	if _, err := table.columnsOf(context.Background(), map[string]any{"id": json.Number("1"), "gone_col": "x"}); err == nil {
		t.Fatal("a column that no longer exists must block the generation")
	}

	columns, err := table.columnsOf(context.Background(), map[string]any{"NAME": "a", "id": json.Number("1")})
	if err != nil {
		t.Fatalf("case-insensitive keys must resolve: %v", err)
	}
	// 顺序按表定义给出而不是 map 遍历顺序：同一份旧值两次生成的文本必须一致，否则无法比对复核
	if len(columns) != 2 || columns[0].ColumnName != "id" || columns[1].ColumnName != "name" {
		t.Fatalf("columns must follow the table definition order, got %+v", columns)
	}
}

// TestSqlExecTypeNameNamesTheBlockedTypes 「该记录不能回滚」的提示要指名类型，
// 只说「不支持」会让人以为平台没有回滚能力，而实际上是 INSERT 的旧值里根本没有新行主键
func TestSqlExecTypeNameNamesTheBlockedTypes(t *testing.T) {
	for execType, want := range map[entity.DbSQLExecType]string{
		entity.DbSQLExecTypeInsert: "INSERT",
		entity.DbSQLExecTypeQuery:  "QUERY",
		entity.DbSQLExecTypeDDL:    "DDL",
	} {
		if got := sqlExecTypeName(execType); got != want {
			t.Fatalf("type %d name = %q, want %q", execType, got, want)
		}
	}
}

// rollbackGuardRepo 只提供 GenRollback 前置判定要的那一步取记录：其余方法一个也不给实现，
// 所以判定一旦被跳过，下一步就会去 nil 的 dbApp 接口上取连接而当场 panic，用例即红
func rollbackGuardRepo(record *entity.DbSQLExec) *stubRollbackRecordRepo {
	return &stubRollbackRecordRepo{RepoImpl: &base.RepoImpl[*entity.DbSQLExec]{}, record: record}
}

type stubRollbackRecordRepo struct {
	*base.RepoImpl[*entity.DbSQLExec]

	record *entity.DbSQLExec
}

func (r *stubRollbackRecordRepo) GetById(_ uint64, _ ...string) (*entity.DbSQLExec, error) {
	return r.record, nil
}

func (r *stubRollbackRecordRepo) GetPageList(*entity.DbSQLExecQuery, ...string) (*model.PageResult[*entity.DbSQLExec], error) {
	return nil, nil
}

// TestGenRollbackRejectsRecordThatNeverTookEffect 只为「执行成功」的记录生成回滚。
//
// 记录里带旧值不等于那次变更已提交：语句可以属于一个最终被整批回滚的原子事务，也可以是
// 待执行/已取消的工单。据这些旧值生成 UPDATE/INSERT 并让人照抄执行，会把陈旧值盖进现网数据
// （覆盖他人之后的修改）或把没删掉的行再插一遍，而界面显示「执行成功」
func TestGenRollbackRejectsRecordThatNeverTookEffect(t *testing.T) {
	for _, status := range []entity.DbSQLExecStatus{entity.DbSQLExecStatusFail, entity.DbSQLExecStatusWait, entity.DbSQLExecStatusNo} {
		record := &entity.DbSQLExec{
			Model: model.Model{CreateModel: model.CreateModel{Id: 7}},
			Type:  entity.DbSQLExecTypeUpdate,
			Table: "t_order",
			// 旧值齐备：若只靠「采到过旧值」就把住关，这条记录完全能走下去，守卫等于没有
			OldValue: `[{"id":1,"status":"paid"}]`,
			Status:   status,
		}
		app := &DbSQLExecAppImpl{dbSQLExecRepo: rollbackGuardRepo(record)}

		_, err := app.GenRollback(context.Background(), record.Id)
		if err == nil {
			t.Fatalf("status %d must be rejected, got nil error", status)
		}
		var bizErr *errorx.BizError
		if !errors.As(err, &bizErr) {
			t.Fatalf("rejection must be a business error (a reason the operator can read), got %T: %v", err, err)
		}
		// 拒绝必须指得出是哪条记录：只给一句「不能回滚」会让人在记录列表里逐条试
		if !strings.Contains(err.Error(), "7") {
			t.Fatalf("the rejection must name the record id so the operator can find it, got %q", err.Error())
		}
	}
}

// rbTestTable 一张「id 主键 + val 文本列」的表侧事实，供纯函数用例复用
func rbTestTable() *rollbackTable {
	order := []dbi.Column{{ColumnName: "id", DataType: "int"}, {ColumnName: "val", DataType: "varchar"}}
	return &rollbackTable{
		name:        "t",
		order:       order,
		byName:      map[string]dbi.Column{"id": order[0], "val": order[1]},
		primaryKeys: []string{"id"},
	}
}

// TestRollbackEqConditionRendersNullAsIsNull CAS/WHERE 条件里的 NULL 必须写 IS NULL：
// col = NULL 在三元逻辑里恒为 UNKNOWN，条件永不匹配，回滚静默落 0 行却显示「执行成功」
func TestRollbackEqConditionRendersNullAsIsNull(t *testing.T) {
	column := dbi.Column{ColumnName: "val", DataType: "varchar"}
	if got := rollbackEqCondition("`val`", nil, dbi.DbType("mysql"), column); got != "`val` IS NULL" {
		t.Fatalf("a nil value must render IS NULL, got %q", got)
	}
	if got := rollbackEqCondition("`val`", "a", dbi.DbType("mysql"), column); got != "`val` = 'a'" {
		t.Fatalf("a text value must render an equality, got %q", got)
	}
}

// TestCanonicalValueTextNormalizesNumberForms 配对键必须让「同一逻辑值」在存储解码侧（json.Number）
// 与实时回查侧（int64/string）得到同一文本，否则并发校验永远配不上、把未改过的行误判成缺失
func TestCanonicalValueTextNormalizesNumberForms(t *testing.T) {
	if canonicalValueText(json.Number("7")) != canonicalValueText(int64(7)) {
		t.Fatalf("json.Number and int64 of the same value must canonicalize identically")
	}
	if canonicalValueText(json.Number("9007199254740993")) != "9007199254740993" {
		t.Fatalf("a bigint key must keep every digit, got %q", canonicalValueText(json.Number("9007199254740993")))
	}
}

// TestPkRowKeyJoinsValuesAndRejectsMissing 主键配对键：多列主键以 \x00 连接（值含分隔符也不串键），
// 任一主键列缺失或为 NULL 都必须判为配不上（ok=false），否则会拿不完整键去比对
func TestPkRowKeyJoinsValuesAndRejectsMissing(t *testing.T) {
	table := rbTestTable()
	table.primaryKeys = []string{"id", "val"}

	key, _, ok := pkRowKey(table, map[string]any{"id": json.Number("1"), "val": "a"})
	if !ok || key != "1\x00a" {
		t.Fatalf("composite key = %q (ok=%v), want the values joined by \\x00", key, ok)
	}
	if _, bad, ok := pkRowKey(table, map[string]any{"id": json.Number("1"), "val": nil}); ok || bad != "val" {
		t.Fatalf("a NULL primary key column must not pair, bad=%q ok=%v", bad, ok)
	}
	if _, _, ok := pkRowKey(table, map[string]any{"id": json.Number("1")}); ok {
		t.Fatal("a missing primary key column must not pair")
	}
}

// TestVerifyUpdateRowsDegradesWithoutSnapshot 没有执行后快照（升级前的历史记录）时必须降级而非拒绝：
// verified=false、旧值原样透传，让回滚对历史记录仍然可用，风险由生成侧的 Note 说明。
// 该路径不触碰连接，conn 传 nil 即可验证「降级不查库」
func TestVerifyUpdateRowsDegradesWithoutSnapshot(t *testing.T) {
	table := rbTestTable()
	rows := []map[string]any{{"id": json.Number("1"), "val": "a"}}

	verification, err := verifyUpdateRows(context.Background(), nil, table, rows, "")
	if err != nil {
		t.Fatalf("an absent snapshot must degrade, not fail: %v", err)
	}
	if verification.verified {
		t.Fatal("without a snapshot the rows cannot be verified")
	}
	if len(verification.rows) != 1 || verification.rows[0].snapshot != nil {
		t.Fatalf("degraded rows must pass the old values through without a snapshot, got %+v", verification.rows)
	}
	if len(verification.conflicts) != 0 {
		t.Fatalf("degrading is not a conflict, got %+v", verification.conflicts)
	}
}

// TestGenInsertRollbackRejectsTableWithoutPrimaryKey 无主键表不能生成 INSERT 的回滚 DELETE：
// 没有主键就谈不上「补真实主键精确定位」，凭值删会误伤同值行。守卫在用连接前就拦下，conn 传 nil 即可
func TestGenInsertRollbackRejectsTableWithoutPrimaryKey(t *testing.T) {
	table := rbTestTable()
	table.primaryKeys = nil

	_, _, err := genInsertRollback(context.Background(), nil, table, []map[string]any{{"val": "'a'"}})
	if err == nil {
		t.Fatal("a table without a primary key must be rejected")
	}
	var bizErr *errorx.BizError
	if !errors.As(err, &bizErr) {
		t.Fatalf("the rejection must be a business error, got %T: %v", err, err)
	}
}

// TestBuildInsertDeleteSQLReplaysLiteralText 主键齐备时按「主键 + 全部插入值」直接生成 DELETE，
// 且字符串单元格按字面量原文回放（含引号），不做「解码再编码」的二次改写。该路径不查库，conn 传 nil
func TestBuildInsertDeleteSQLReplaysLiteralText(t *testing.T) {
	table := rbTestTable()
	quote := func(s string) string { return "`" + s + "`" }
	row := map[string]any{"id": "1", "val": "'a'"}

	statement, reason, err := buildInsertDeleteSQL(context.Background(), nil, table, row, quote)
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if reason != "" {
		t.Fatalf("a primary-key-ready row must not report a conflict, got %q", reason)
	}
	if statement != "DELETE FROM `t` WHERE `id` = 1 AND `val` = 'a'" {
		t.Fatalf("unexpected statement: %q", statement)
	}
}

// TestInsertSnapshotOfKeepsLiteralTextAndNull 插入值快照必须保留字面量原文（字符串带引号）与 NULL 语义：
// 快照是回滚 DELETE 的唯一线索，把 'a' 存成 a、把 NULL 存成空串都会让条件匹配不上或误配
func TestInsertSnapshotOfKeepsLiteralTextAndNull(t *testing.T) {
	stmt := &sqlstmt.InsertStmt{
		Columns: []string{"id", "val"},
		ValueRows: []sqlstmt.InsertValueRow{
			{{Text: "1"}, {Text: "'a'"}},
			{{Text: "2"}, {Null: true}},
		},
	}
	snapshot := insertSnapshotOf(stmt)
	rows, err := decodeOldValueRows(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("the snapshot must be valid JSON: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("want 2 rows, got %d", len(rows))
	}
	if rows[0]["val"] != "'a'" {
		t.Fatalf("the string literal must keep its quotes, got %#v", rows[0]["val"])
	}
	if rows[1]["val"] != nil {
		t.Fatalf("a NULL cell must decode to nil, got %#v", rows[1]["val"])
	}

	// 行列数不齐时整体放弃：硬凑会让值与列错位
	mismatch := &sqlstmt.InsertStmt{Columns: []string{"id", "val"}, ValueRows: []sqlstmt.InsertValueRow{{{Text: "1"}}}}
	if got := insertSnapshotOf(mismatch); got != "" {
		t.Fatalf("a ragged value row must yield no snapshot, got %q", got)
	}
}
