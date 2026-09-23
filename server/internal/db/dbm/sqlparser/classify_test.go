package sqlparser

import (
	"errors"
	"io"
	"testing"

	mysqlparser "mayfly-go/internal/db/dbm/sqlparser/mysql"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"
	"mayfly-go/internal/pkg/utils"

	"github.com/stretchr/testify/assert"
)

// fakeSplitter 仅暴露分类所需的首词关键字，其余方法零实现
type fakeSplitter struct{ kw string }

func (fakeSplitter) SplitSQL(_ io.Reader, _ utils.StmtCallback) error { return nil }
func (f fakeSplitter) LeadingKeyword(_ string) string                 { return f.kw }
func (f fakeSplitter) MaskComments(s string) string                   { return s }

func TestAstStmtType(t *testing.T) {
	cases := map[string]struct {
		stmt sqlstmt.Stmt
		want sqlstmt.StmtType
	}{
		"select": {&sqlstmt.SelectStmt{}, sqlstmt.StmtTypeSelect},
		"with":   {&sqlstmt.WithStmt{}, sqlstmt.StmtTypeSelect},
		"update": {&sqlstmt.UpdateStmt{}, sqlstmt.StmtTypeUpdate},
		"delete": {&sqlstmt.DeleteStmt{}, sqlstmt.StmtTypeDelete},
		"insert": {&sqlstmt.InsertStmt{}, sqlstmt.StmtTypeInsert},
		"ddl":    {&sqlstmt.DdlStmt{}, sqlstmt.StmtTypeDDL},
		"other":  {&sqlstmt.OtherStmt{}, sqlstmt.StmtTypeOther},
	}
	for name, c := range cases {
		got, ok := astStmtType(c.stmt)
		assert.Truef(t, ok, "%s: 应命中映射", name)
		assert.Equalf(t, c.want, got, name)
	}
}

func TestStmtTypeFromKeyword(t *testing.T) {
	assert.Equal(t, sqlstmt.StmtTypeSelect, stmtTypeFromKeyword("select"))
	// 修复漂移：with(CTE) 应判为查询，与 AST 路径一致（历史上字符串路径把它归 other）
	assert.Equal(t, sqlstmt.StmtTypeSelect, stmtTypeFromKeyword("with"))
	assert.Equal(t, sqlstmt.StmtTypeDDL, stmtTypeFromKeyword("create"))
	assert.Equal(t, sqlstmt.StmtTypeDDL, stmtTypeFromKeyword("grant"))
	assert.Equal(t, sqlstmt.StmtTypeOther, stmtTypeFromKeyword("show"))
	// 未识别动词返回空串，交调用方兜底
	assert.Equal(t, sqlstmt.StmtType(""), stmtTypeFromKeyword("call"))
	assert.Equal(t, sqlstmt.StmtType(""), stmtTypeFromKeyword(""))
}

// Classify 优先级：有 AST 用 AST；无 AST 走关键字兜底；两者皆不可用返回空串
func TestClassify_Priority(t *testing.T) {
	// AST 优先：即便关键字兜底会给出不同结果，也应采信 AST
	got := Classify(nil, fakeSplitter{kw: "delete"}, "update t set a=1", &sqlstmt.UpdateStmt{}, nil)
	assert.Equal(t, sqlstmt.StmtTypeUpdate, got)

	// 解析失败：无 AST，走 LeadingKeyword 整词兜底
	notParse := errors.New("parse failed")
	got = Classify(nil, fakeSplitter{kw: "insert"}, "insert into t values(1)", nil, notParse)
	assert.Equal(t, sqlstmt.StmtTypeInsert, got)

	// 无 parser、无 splitter、无 AST：无从判定 → 空串
	got = Classify(nil, nil, "whatever", nil, notParse)
	assert.Equal(t, sqlstmt.StmtType(""), got)
}

// TestClassify_ExecutableCommentNoPanic 回归锁：mysqldump 头部的可执行注释（/*!40101 ... */ 等）
// 经切割后作为独立语句进入 Classify 时，AST 为 nil 且无解析错误。旧实现会调用方言 ClassifyStmt
// 对 nil AST 解引用而 panic（用户在 SQL 控制台粘贴一段 dump 即可触发，导致整批语句中断）。
// 现应安全落到关键字兜底、返回空串（交调用方按通用语句透传执行）。
func TestClassify_ExecutableCommentNoPanic(t *testing.T) {
	p := &mysqlparser.MysqlParser{}

	// 纵深防御：方言分类器自身对 nil AST 也必须安全
	assert.NotPanics(t, func() { _, _ = p.ClassifyStmt("/*!40101 SET NAMES utf8 */") })

	// 端到端：切割器会把可执行注释交付为独立语句，Classify 不得 panic
	sp := NewSplitter(tokenizer.MysqlConfig)
	for _, sql := range []string{
		"/*!40101 SET NAMES utf8 */",
		"/*!40000 ALTER TABLE `t` DISABLE KEYS */",
	} {
		assert.NotPanicsf(t, func() { _ = Classify(p, sp, sql, nil, nil) }, "Classify(%q) 不得 panic", sql)
	}
}
