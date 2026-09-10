package sqlparser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"mayfly-go/internal/db/dbm/sqlparser/tokenizer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// splitAll 切割成功并返回语句列表（出现错误则测试失败）
func splitAll(t *testing.T, splitter SQLSplitter, sql string) []string {
	t.Helper()
	stmts := make([]string, 0)
	require.NoError(t, splitter.SplitSQL(strings.NewReader(sql), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	}))
	return stmts
}

// splitErrAll 切割并返回（语句列表, 错误），用于断言未闭合区域等错误场景
func splitErrAll(t *testing.T, splitter SQLSplitter, sql string) ([]string, error) {
	t.Helper()
	stmts := make([]string, 0)
	err := splitter.SplitSQL(strings.NewReader(sql), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	})
	return stmts, err
}

func stdSplitter(delimiter ...rune) SQLSplitter {
	return NewSplitter(tokenizer.StdConfig, delimiter...)
}

func mysqlSplitter(delimiter ...rune) SQLSplitter {
	return NewSplitter(tokenizer.MysqlConfig, delimiter...)
}

// 各方言切割器装配断言：标准SQL方言（sqlite/postgres/mssql/oracle）反斜杠为普通字符，
// mysql 反斜杠转义且支持 # 注释
func TestSplitterDialectWiring(t *testing.T) {
	script := `insert into t values ('\'); insert into t values ('x');`

	std := []string{`insert into t values ('\')`, `insert into t values ('x')`}
	assert.Equal(t, std, splitAll(t, stdSplitter(), script), "标准SQL语义应正确切割")
	assert.Equal(t, std, splitAll(t, stdSplitter('@'), `insert into t values ('\')@insert into t values ('x')@`), "自定义分隔符不影响反斜杠语义")

	// mysql语义：\' 不结束字符串，直到 ('x') 的开引号才闭合，其后再开串至EOF未闭合，
	// 与mysql真实词法一致（mysql客户端亦报未闭合字符串），必须报错而非静默合并整段
	_, err := splitErrAll(t, mysqlSplitter(), script)
	assertUnterminated(t, err, 1, tokenizer.KindString)

	// mysql # 行注释：注释原文保留在其后语句中，注释内的分号不参与切割
	assert.Equal(t, []string{"select 1", "# c;oment\nselect 2"},
		splitAll(t, mysqlSplitter(), "select 1; # c;oment\nselect 2;"))
	// 标准SQL方言不处理 # 注释（# 可能是操作符），# 后文本作为普通语句内容保留
	assert.Equal(t, []string{"select 1", "# c\nselect 2"},
		splitAll(t, stdSplitter(), "select 1; # c\nselect 2;"))

	// 字符串内的分号两种语义均不切割
	assert.Equal(t, []string{`insert into t values ('a;b')`},
		splitAll(t, mysqlSplitter(), `insert into t values ('a;b');`))
	assert.Equal(t, []string{`insert into t values ('a;b')`},
		splitAll(t, stdSplitter(), `insert into t values ('a;b');`))
}

// 注释必须原样保留：Oracle hint 与 mysql 可执行注释均会被服务端执行，剥离即丢语义；
// 且注释不得与其后的 token 粘连（切割不再改写语句文本）
func TestSplitterPreservesComments(t *testing.T) {
	assert.Equal(t, []string{"select 1 -- trailing\nfrom t", "select 2"},
		splitAll(t, stdSplitter(), "select 1 -- trailing\nfrom t; select 2;"))
	assert.Equal(t, []string{"select /*+ LEADING(t) */ 1"},
		splitAll(t, stdSplitter(), "select /*+ LEADING(t) */ 1;"))
	assert.Equal(t, []string{"/*!40101 SET NAMES utf8 */", "select 1"},
		splitAll(t, mysqlSplitter(), "/*!40101 SET NAMES utf8 */;\nselect 1;"))
	// 注释块与多余分隔符单独成段时不构成语句
	assert.Empty(t, splitAll(t, mysqlSplitter(), "-- dump header\n;\n/* block; */\n;;"))
}

// 未闭合的字面量/注释返回带行号的明确错误（此前三套切割器静默吞并后续脚本）
func TestSplitterUnterminated(t *testing.T) {
	_, err := splitErrAll(t, stdSplitter(), "select 1;\ninsert into t values ('abc")
	assertUnterminated(t, err, 2, tokenizer.KindString)

	_, err = splitErrAll(t, stdSplitter(), "select 1;\n/* unclosed\ninsert into t values (1);")
	assertUnterminated(t, err, 2, tokenizer.KindBlockComment)

	_, err = splitErrAll(t, mysqlSplitter(), "select * from `abc")
	assertUnterminated(t, err, 1, tokenizer.KindIdentQuote)

	// 闭合正常则无错误
	stmts, err := splitErrAll(t, stdSplitter(), "insert into t values ('abc');")
	require.NoError(t, err)
	assert.Equal(t, []string{"insert into t values ('abc')"}, stmts)
}

// assertUnterminated 断言错误为未闭合区域错误，且行号与区域类型符合预期
func assertUnterminated(t *testing.T, err error, line int, kind string) {
	t.Helper()
	var ue *tokenizer.UnterminatedError
	require.ErrorAs(t, err, &ue)
	assert.Equal(t, line, ue.Line, "未闭合区域行号")
	assert.Equal(t, kind, ue.Kind, "未闭合区域类型")
}

// TestSplitSQLCallbackError 回调执行错误必须原样透传（而非转为切割错误），
// 否则导入时某条语句的业务错误会被误报为“SQL 切割失败”
func TestSplitSQLCallbackError(t *testing.T) {
	execErr := errors.New("duplicate key")
	var got []string
	err := mysqlSplitter().SplitSQL(strings.NewReader("select 1; select 2;"), func(stmt string) error {
		got = append(got, stmt)
		if len(got) == 2 {
			return execErr
		}
		return nil
	})
	assert.Same(t, execErr, err, "回调错误应原样返回")
	assert.Equal(t, []string{"select 1", "select 2"}, got)

	// SplitError 不得包装修改非切割类错误
	assert.Same(t, execErr, SplitError(context.Background(), err))
}

// TestSplitSQLBomAndLargeValue 导入文件常见形态：带 UTF-8 BOM 的脚本与单行超长字面量
func TestSplitSQLBomAndLargeValue(t *testing.T) {
	var got []string
	require.NoError(t, stdSplitter().SplitSQL(strings.NewReader("\ufeffselect 1;\nselect '"+strings.Repeat("a", 1<<16)+"';"), func(stmt string) error {
		got = append(got, stmt)
		return nil
	}))
	require.Len(t, got, 2)
	assert.Equal(t, "select 1", got[0], "BOM 不得附着在首条语句上")
	assert.Len(t, got[1], (1<<16)+9, "超长值整条保留")
}
