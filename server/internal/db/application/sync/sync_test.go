package sync

import (
	"context"
	"database/sql"
	_ "mayfly-go/internal/db/dbm"
	"mayfly-go/internal/db/dbm/dbi"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeMetadataProvider 测试用元数据实现（仅GetTableIndex参与断言，其余返回零值）
type fakeMetadataProvider struct {
	indexes  []dbi.Index
	indexErr error
}

func (f *fakeMetadataProvider) GetTableIndex(tableName string) ([]dbi.Index, error) {
	return f.indexes, f.indexErr
}

func (f *fakeMetadataProvider) GetSchemas() ([]string, error)                       { return nil, nil }
func (f *fakeMetadataProvider) GetDbNames() ([]string, error)                       { return nil, nil }
func (f *fakeMetadataProvider) GetTables(tableNames ...string) ([]dbi.Table, error) { return nil, nil }
func (f *fakeMetadataProvider) GetColumns(tableNames ...string) ([]dbi.Column, error) {
	return nil, nil
}
func (f *fakeMetadataProvider) GetPrimaryKey(tableName string) (string, error) { return "", nil }
func (f *fakeMetadataProvider) GetTableDDL(tableName string, dropBeforeCreate bool) (string, error) {
	return "", nil
}

var _ dbi.MetadataProvider = (*fakeMetadataProvider)(nil)

// fakeServerInfo 测试用服务器信息实现
type fakeServerInfo struct{}

func (f *fakeServerInfo) GetDbServer() (*dbi.DbServer, error)   { return nil, nil }
func (f *fakeServerInfo) GetCompatibleDbVersion() dbi.DbVersion { return "" }
func (f *fakeServerInfo) GetDefaultDb() string                  { return "" }

var _ dbi.ServerInfo = (*fakeServerInfo)(nil)

func newFakeConn(md dbi.MetadataProvider) *dbi.DbConn {
	return &dbi.DbConn{Info: &dbi.DbInfo{Backend: fakeBackend{mp: md}}}
}

type fakeBackend struct {
	mp dbi.MetadataProvider
}

func (m fakeBackend) GetSqlDb(ctx context.Context, d *dbi.DbInfo) (*sql.DB, error) { return nil, nil }
func (m fakeBackend) GetDialect(d *dbi.DbInfo) dbi.Dialect                         { return nil }
func (m fakeBackend) GetServerInfo(d *dbi.DbInfo) dbi.ServerInfo                   { return &fakeServerInfo{} }
func (m fakeBackend) GetMetadataProvider(d *dbi.DbInfo) dbi.MetadataProvider       { return m.mp }
func (m fakeBackend) CommitTargetTx(conn *dbi.DbConn, tx *sql.Tx) error            { return nil }
func (m fakeBackend) GetCapabilities() dbi.MetadataCapabilities                    { return dbi.MetadataCapabilities{} }

var _ dbi.DbBackend = fakeBackend{}

func TestBuildTargetTableMeta_PrimaryKeyFirst(t *testing.T) {
	// 有主键时优先使用主键列，不查询唯一索引
	columns := []dbi.Column{
		{ColumnName: "id", IsPrimaryKey: true, AutoIncrement: true},
		{ColumnName: "name"},
	}
	meta := dbi.BuildTargetTableMeta(newFakeConn(&fakeMetadataProvider{}), "t1", columns)
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
	meta := dbi.BuildTargetTableMeta(newFakeConn(&fakeMetadataProvider{}), "t1", columns)
	assert.Equal(t, []string{"k1", "k2"}, meta.UniqueColumns)
}

func TestBuildTargetTableMeta_SingleUniqueIndex(t *testing.T) {
	// 无主键且仅一个唯一索引：取该索引列（含联合索引列拆分与空格清理）
	columns := []dbi.Column{{ColumnName: "code"}, {ColumnName: "org"}}
	md := &fakeMetadataProvider{
		indexes: []dbi.Index{
			{IndexName: "uk_code_org", ColumnName: "code, org", IsUnique: true},
			{IndexName: "idx_org", ColumnName: "org", IsUnique: false},
		},
	}
	meta := dbi.BuildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Equal(t, []string{"code", "org"}, meta.UniqueColumns)
}

func TestBuildTargetTableMeta_MultiUniqueIndexDegenerate(t *testing.T) {
	// 无主键且存在多个唯一索引：无法确定冲突语义，UniqueColumns为空退化为直接插入
	columns := []dbi.Column{{ColumnName: "code"}, {ColumnName: "org"}}
	md := &fakeMetadataProvider{
		indexes: []dbi.Index{
			{IndexName: "uk_code", ColumnName: "code", IsUnique: true},
			{IndexName: "uk_org", ColumnName: "org", IsUnique: true},
		},
	}
	meta := dbi.BuildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Empty(t, meta.UniqueColumns)
}

// normalizeBatchSize：存量任务的PageSize可能为0，分批取模前必须兑底，避免除零panic
func TestNormalizeBatchSize(t *testing.T) {
	assert.Equal(t, 500, normalizeBatchSize(0))
	assert.Equal(t, 500, normalizeBatchSize(-1))
	// 合法值原样返回
	assert.Equal(t, 1, normalizeBatchSize(1))
	assert.Equal(t, 1000, normalizeBatchSize(1000))
}

func TestBuildTargetTableMeta_IndexQueryError(t *testing.T) {
	// 索引查询失败：不中断流程，退化为直接插入（由数据库约束报错提示）
	columns := []dbi.Column{{ColumnName: "code"}}
	md := &fakeMetadataProvider{indexErr: assert.AnError}
	meta := dbi.BuildTargetTableMeta(newFakeConn(md), "t1", columns)
	assert.Empty(t, meta.UniqueColumns)
}
