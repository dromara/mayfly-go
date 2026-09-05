package application

import (
	"context"
	"database/sql"
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeMetadata 测试用元数据实现（仅GetTableIndex参与断言，其余返回零值）
type fakeMetadata struct {
	indexes  []dbi.Index
	indexErr error
}

func (f *fakeMetadata) GetTableIndex(tableName string) ([]dbi.Index, error) {
	return f.indexes, f.indexErr
}

func (f *fakeMetadata) GetDbServer() (*dbi.DbServer, error)                  { return nil, nil }
func (f *fakeMetadata) GetCompatibleDbVersion() dbi.DbVersion                { return "" }
func (f *fakeMetadata) GetDefaultDb() string                                 { return "" }
func (f *fakeMetadata) GetSchemas() ([]string, error)                        { return nil, nil }
func (f *fakeMetadata) GetDbNames() ([]string, error)                        { return nil, nil }
func (f *fakeMetadata) GetTables(tableNames ...string) ([]dbi.Table, error)  { return nil, nil }
func (f *fakeMetadata) GetColumns(tableNames ...string) ([]dbi.Column, error) { return nil, nil }
func (f *fakeMetadata) GetPrimaryKey(tableName string) (string, error)       { return "", nil }
func (f *fakeMetadata) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return "", nil
}

var _ dbi.Metadata = (*fakeMetadata)(nil)

func newFakeConn(md dbi.Metadata) *dbi.DbConn {
	return &dbi.DbConn{Info: &dbi.DbInfo{Meta: fakeMeta{md: md}}}
}

type fakeMeta struct {
	md dbi.Metadata
}

func (m fakeMeta) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) { return nil, nil }
func (m fakeMeta) GetDialect(d *dbi.DbConn) dbi.Dialect                         { return nil }
func (m fakeMeta) GetMetadata(d *dbi.DbConn) dbi.Metadata                       { return m.md }
func (m fakeMeta) GetDbDataTypes() []*dbi.DbDataType                            { return nil }
func (m fakeMeta) GetCommonTypeConverter() dbi.CommonTypeConverter              { return nil }

var _ dbi.Meta = fakeMeta{}

func TestBuildTargetTableMeta_PrimaryKeyFirst(t *testing.T) {
	// 有主键时优先使用主键列，不查询唯一索引
	columns := []dbi.Column{
		{ColumnName: "id", IsPrimaryKey: true, AutoIncrement: true},
		{ColumnName: "name"},
	}
	meta := buildTargetTableMeta(newFakeConn(&fakeMetadata{}), "t1", columns)
	assert.Equal(t, []string{"id"}, meta.UniqueColumns)
	assert.Equal(t, []string{"id"}, meta.IdentityColumns)
}

func TestBuildTargetTableMeta_MultiPrimaryKey(t *testing.T) {
	// 联合主键全部纳入冲突检测列
	columns := []dbi.Column{
		{ColumnName: "k1", IsPrimaryKey: true},
		{ColumnName: "k2", IsPrimaryKey: true},
		{ColumnName: "val"},
	}
	meta := buildTargetTableMeta(newFakeConn(&fakeMetadata{}), "t1", columns)
	assert.Equal(t, []string{"k1", "k2"}, meta.UniqueColumns)
}

func TestBuildTargetTableMeta_SingleUniqueIndex(t *testing.T) {
	// 无主键且仅一个唯一索引：取该索引列（含联合索引列拆分与空格清理）
	columns := []dbi.Column{{ColumnName: "code"}, {ColumnName: "org"}}
	md := &fakeMetadata{
		indexes: []dbi.Index{
			{IndexName: "uk_code_org", ColumnName: "code, org", IsUnique: true},
			{IndexName: "idx_org", ColumnName: "org", IsUnique: false},
		},
	}
	meta := buildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Equal(t, []string{"code", "org"}, meta.UniqueColumns)
}

func TestBuildTargetTableMeta_MultiUniqueIndexDegenerate(t *testing.T) {
	// 无主键且存在多个唯一索引：无法确定冲突语义，UniqueColumns为空退化为直接插入
	columns := []dbi.Column{{ColumnName: "code"}, {ColumnName: "org"}}
	md := &fakeMetadata{
		indexes: []dbi.Index{
			{IndexName: "uk_code", ColumnName: "code", IsUnique: true},
			{IndexName: "uk_org", ColumnName: "org", IsUnique: true},
		},
	}
	meta := buildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Empty(t, meta.UniqueColumns)
}

func TestBuildTargetTableMeta_IndexQueryError(t *testing.T) {
	// 索引查询失败：不中断流程，退化为直接插入（由数据库约束报错提示）
	columns := []dbi.Column{{ColumnName: "code"}}
	md := &fakeMetadata{indexErr: assert.AnError}
	meta := buildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Empty(t, meta.UniqueColumns)
}
