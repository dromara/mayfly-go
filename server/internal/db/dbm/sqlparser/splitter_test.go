package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func splitAll(t *testing.T, splitter SQLSplitter, sql string) []string {
	t.Helper()
	stmts := make([]string, 0)
	require.NoError(t, splitter.SplitSQL(strings.NewReader(sql), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	}))
	return stmts
}

// 各方言切割器装配断言：标准SQL方言（sqlite/postgres/mssql/oracle）反斜杠为普通字符，
// mysql 反斜杠转义且支持 # 注释
func TestSplitterDialectWiring(t *testing.T) {
	script := `insert into t values ('\'); insert into t values ('x');`

	std := []string{`insert into t values ('\')`, `insert into t values ('x')`}
	assert.Equal(t, std, splitAll(t, NewStdSQLSplitter(), script), "标准SQL语义应正确切割")
	assert.Equal(t, std, splitAll(t, NewStdSQLSplitter('@'), `insert into t values ('\')@insert into t values ('x')@`), "自定义分隔符不影响反斜杠语义")

	// mysql语义：\' 不结束字符串，直到 ('x') 的开引号才闭合，整段为单条语句（含尾部未切分号，与mysql真实解析行为一致）
	mysqlExpected := []string{`insert into t values ('\'); insert into t values ('x');`}
	assert.Equal(t, mysqlExpected, splitAll(t, NewMysqlSplitter(), script))
	assert.Equal(t, mysqlExpected, splitAll(t, NewDefaultSplitter(), script))

	// mysql # 行注释
	assert.Equal(t, []string{"select 1", "select 2"},
		splitAll(t, NewMysqlSplitter(), "select 1; # c;oment\nselect 2;"))
	// 标准SQL方言不处理 # 注释（# 可能是操作符），# 后文本作为普通语句内容保留
	assert.Equal(t, []string{"select 1", "# c\nselect 2"},
		splitAll(t, NewStdSQLSplitter(), "select 1; # c\nselect 2;"))

	// 字符串内的分号两种语义均不切割
	assert.Equal(t, []string{`insert into t values ('a;b')`},
		splitAll(t, NewMysqlSplitter(), `insert into t values ('a;b');`))
	assert.Equal(t, []string{`insert into t values ('a;b')`},
		splitAll(t, NewStdSQLSplitter(), `insert into t values ('a;b');`))
}
