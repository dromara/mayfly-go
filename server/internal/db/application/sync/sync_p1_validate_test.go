package sync

import (
	"testing"

	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseSelect 用 mysql 方言解析 SQL，返回顶层 SELECT 供 singleTableForIndex 探测。
func parseSelect(t *testing.T, sql string) *sqlstmt.SelectStmt {
	t.Helper()
	dialect := dbi.GetDialect("mysql")
	require.NotNil(t, dialect, "mysql dialect must be registered via blank import of dbm")
	stmt, err := dialect.GetSQLParser().Parse(sql)
	require.NoError(t, err)
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	require.True(t, ok, "not a SELECT stmt: %T", stmt)
	return sel
}

func TestSingleTableForIndex(t *testing.T) {
	cases := []struct {
		name       string
		sql        string
		wantOk     bool
		wantSchema string
		wantTable  string
	}{
		{"plain single table", "SELECT * FROM users", true, "", "users"},
		{"schema qualified", "SELECT * FROM mydb.users", true, "mydb", "users"},
		{"with alias", "SELECT * FROM users u", true, "", "users"},
		{"JOIN rejected", "SELECT * FROM a JOIN b ON a.id=b.id", false, "", ""},
		{"multi-table FROM rejected", "SELECT * FROM a, b", false, "", ""},
		{"UNION rejected", "SELECT * FROM a UNION SELECT * FROM b", false, "", ""},
		{"subquery FROM rejected", "SELECT * FROM (SELECT 1) t", false, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sel := parseSelect(t, c.sql)
			schema, table, ok := singleTableForIndex(sel)
			assert.Equal(t, c.wantOk, ok)
			assert.Equal(t, c.wantSchema, schema)
			assert.Equal(t, c.wantTable, table)
		})
	}
}

// TestSetSkipIndexValidation 校验逃生阀的 Extra 读写语义：
// false 应清 key（保持 Extra 干净，与默认行为等价），true 写入 bool true。
func TestSetSkipIndexValidation(t *testing.T) {
	task := &entity.DataSyncTask{}
	assert.False(t, task.GetSkipIndexValidation(), "空 Extra 默认 false")

	task.SetSkipIndexValidation(true)
	assert.True(t, task.GetSkipIndexValidation())
	assert.Equal(t, true, task.Extra[entity.ExtraKeySkipIndexValidation])

	task.SetSkipIndexValidation(false)
	assert.False(t, task.GetSkipIndexValidation())
	_, existed := task.Extra[entity.ExtraKeySkipIndexValidation]
	assert.False(t, existed, "false 时必须清除 key，避免脏数据留在 Extra")
}
