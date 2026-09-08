package mssql

import (
	"bytes"
	"testing"

	"mayfly-go/internal/db/dbm/dbi"

	"github.com/stretchr/testify/assert"
)

// 仅含自增列的表才输出set identity_insert on，否则mssql报
// "does not have the identity property"错误
func TestDumpHelper_BeforeInsertSql(t *testing.T) {
	helper := &DumpHelper{}

	// 无自增列不输出
	assert.Equal(t, "", helper.BeforeInsertSql("t1", []dbi.Column{
		{ColumnName: "id"}, {ColumnName: "name"},
	}))
	// 有自增列输出on语句，表名用方括号quote
	assert.Equal(t, "set identity_insert [t1] on;\n", helper.BeforeInsertSql("t1", []dbi.Column{
		{ColumnName: "id", AutoIncrement: true},
	}))
}

// AfterInsert必须与BeforeInsertSql成对：自增表输出off语句，
// 否则会话保持on状态，后续其他含自增表的on语句报"already ON for table"错误
func TestDumpHelper_AfterInsert(t *testing.T) {
	helper := &DumpHelper{}

	// 无自增列不输出
	buf := &bytes.Buffer{}
	assert.NoError(t, helper.AfterInsert(buf, "t1", []dbi.Column{{ColumnName: "name"}}))
	assert.Equal(t, "", buf.String())

	// 有自增列输出off语句
	buf2 := &bytes.Buffer{}
	assert.NoError(t, helper.AfterInsert(buf2, "t1", []dbi.Column{
		{ColumnName: "id", AutoIncrement: true},
	}))
	assert.Equal(t, "set identity_insert [t1] off;\n", buf2.String())
}

// BeforeInsert不输出BEGIN包装（mssql批量导入不支持begin/commit语句）
func TestDumpHelper_BeforeInsert(t *testing.T) {
	buf := &bytes.Buffer{}
	assert.NoError(t, (&DumpHelper{}).BeforeInsert(buf, "t1"))
	assert.Equal(t, "", buf.String())
}
