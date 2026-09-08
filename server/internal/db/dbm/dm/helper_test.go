package dm

import (
	"bytes"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/assert"
)

// 仅含自增列的表才输出set identity_insert，否则达梦报错；
// 表名用双引号quote避免保留字冲突
func TestDumpHelper_BeforeInsertSql(t *testing.T) {
	helper := &DumpHelper{}

	// 无自增列不输出
	assert.Equal(t, "", helper.BeforeInsertSql("t1", []dbi.Column{
		{ColumnName: "id"}, {ColumnName: "name"},
	}))
	// 有自增列输出on语句
	assert.Equal(t, "set identity_insert \"t1\" on;\n", helper.BeforeInsertSql("t1", []dbi.Column{
		{ColumnName: "id", AutoIncrement: true},
	}))
}

// AfterInsert与BeforeInsertSql必须成对输出on/off，否则会话保持on状态，
// 后续其他含自增表的on语句会报错
func TestDumpHelper_AfterInsert(t *testing.T) {
	helper := &DumpHelper{}

	buf := &bytes.Buffer{}
	assert.NoError(t, helper.AfterInsert(buf, "t1", []dbi.Column{{ColumnName: "name"}}))
	assert.Equal(t, "", buf.String())

	buf2 := &bytes.Buffer{}
	assert.NoError(t, helper.AfterInsert(buf2, "t1", []dbi.Column{
		{ColumnName: "id", AutoIncrement: true},
	}))
	assert.Equal(t, "set identity_insert \"t1\" off;\n", buf2.String())
}
