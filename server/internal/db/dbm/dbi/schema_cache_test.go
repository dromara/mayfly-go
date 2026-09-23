package dbi

import (
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingProvider 记录 GetTables/GetColumns 实际下推次数，用于断言缓存命中不再查库。
type countingProvider struct {
	fakeMetaProvider
	tablesCalls  int
	columnsCalls int
}

func (p *countingProvider) GetTables(...string) ([]Table, error) {
	p.tablesCalls++
	return []Table{{TableName: "t_user"}}, nil
}

func (p *countingProvider) GetColumns(names ...string) ([]Column, error) {
	p.columnsCalls++
	name := ""
	if len(names) > 0 {
		name = names[0]
	}
	return []Column{{TableName: name, ColumnName: "id"}}, nil
}

// errProvider 的 GetColumns 恒失败，用于断言错误结果不入任何缓存。
type errProvider struct {
	fakeMetaProvider
	columnsCalls int
}

func (p *errProvider) GetColumns(...string) ([]Column, error) {
	p.columnsCalls++
	return nil, errors.New("boom")
}

func TestSchemaCache_SetGetInvalidate(t *testing.T) {
	c := newSchemaCache()
	c.set("k", 42)
	v, ok := c.get("k")
	assert.True(t, ok)
	assert.Equal(t, 42, v)

	c.invalidate()
	_, ok = c.get("k")
	assert.False(t, ok, "invalidate 后应未命中")
}

func TestSchemaCache_TTLExpiry(t *testing.T) {
	c := newSchemaCache()
	// 写入一个已过 TTL 的条目，get 须判未命中并顺手删除
	c.items["k"] = schemaEntry{val: 1, at: time.Now().Add(-schemaCacheTTL - time.Minute)}
	_, ok := c.get("k")
	assert.False(t, ok)
	_, existed := c.items["k"]
	assert.False(t, existed, "过期条目应被删除，避免常驻内存")
}

func TestMetadataReader_SchemaCache(t *testing.T) {
	sc := newSchemaCache()
	p := &countingProvider{}

	// 单表列：两次跨请求（两个 reader）只下推一次
	_, err := NewMetadataReader(nil, p, nil, sc).GetColumns("t_user")
	assert.NoError(t, err)
	assert.Equal(t, 1, p.columnsCalls)

	_, err = NewMetadataReader(nil, p, nil, sc).GetColumns("t_user")
	assert.NoError(t, err)
	assert.Equal(t, 1, p.columnsCalls, "单表列应命中跨请求缓存，不再查库")

	// 无参全量表清单：同样跨请求缓存
	_, err = NewMetadataReader(nil, p, nil, sc).GetTables()
	assert.NoError(t, err)
	assert.Equal(t, 1, p.tablesCalls)
	_, err = NewMetadataReader(nil, p, nil, sc).GetTables()
	assert.NoError(t, err)
	assert.Equal(t, 1, p.tablesCalls, "全量表清单应命中跨请求缓存")

	// 多表批量不进 schema 缓存：换 reader 各下推一次（累计 +2）
	_, err = NewMetadataReader(nil, p, nil, sc).GetColumns("a", "b")
	assert.NoError(t, err)
	_, err = NewMetadataReader(nil, p, nil, sc).GetColumns("a", "b")
	assert.NoError(t, err)
	assert.Equal(t, 3, p.columnsCalls, "多表批量不应进入跨请求缓存")
}

func TestMetadataReader_NilSchemaCache(t *testing.T) {
	p := &countingProvider{}
	// schemaCache 为 nil（如纯 SQL 生成场景）时不跨请求缓存，每 reader 各查一次
	_, _ = NewMetadataReader(nil, p, nil, nil).GetColumns("t")
	_, _ = NewMetadataReader(nil, p, nil, nil).GetColumns("t")
	assert.Equal(t, 2, p.columnsCalls, "无 schema 缓存时每请求都查库")
}

func TestSchemaCache_ErrorNotCached(t *testing.T) {
	p := &errProvider{}
	sc := newSchemaCache()
	m := NewMetadataReader(nil, p, nil, sc)

	_, err := m.GetColumns("t")
	assert.Error(t, err)
	_, err = m.GetColumns("t")
	assert.Error(t, err)
	assert.Equal(t, 2, p.columnsCalls, "错误不应入任何缓存，第二次仍查库")
	_, ok := sc.get("columns:t")
	assert.False(t, ok, "错误结果不得写入 schema 缓存")
}

// schemaCacheSeededKey 测试用缓存键：invalidate 清空全部条目，断言不依赖真实键名
const schemaCacheSeededKey = "tables:test-seeded"

// newSchemaCacheConn 构造带方言后端与元数据缓存的 mock 连接，并预置一条缓存条目
func newSchemaCacheConn(t *testing.T) (*DbConn, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	info := &DbInfo{
		Type:        DbType("test-schema-cache"),
		db:          db,
		Backend:     &stubBackend{dialect: newStubDialect()},
		schemaCache: newSchemaCache(),
	}
	info.schemaCache.set(schemaCacheSeededKey, []Table{{TableName: "t_user"}})
	return &DbConn{Id: "test-schema-cache-conn", Info: info}, mock
}

// schemaCacheSeeded 断言用：预置条目是否仍在缓存中
func schemaCacheSeeded(conn *DbConn) bool {
	_, ok := conn.Info.schemaCache.get(schemaCacheSeededKey)
	return ok
}

// TestExecDDLInvalidatesSchemaCache 结构变更后必须即时失效：否则补全/资源树最长一个 TTL 都看不到新表
func TestExecDDLInvalidatesSchemaCache(t *testing.T) {
	for _, execSQL := range []string{
		"CREATE TABLE t_new (id INT)",
		"ALTER TABLE t_user ADD COLUMN age INT",
		"DROP TABLE t_old",
		"TRUNCATE TABLE t_log",
		"-- 新增索引\nCREATE INDEX idx_age ON t_user(age)",
	} {
		conn, mock := newSchemaCacheConn(t)
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))

		_, err := conn.Exec(execSQL)
		require.NoError(t, err)
		assert.False(t, schemaCacheSeeded(conn), "DDL 执行成功后元数据缓存必须失效: %s", execSQL)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// TestExecNonDDLKeepsSchemaCache DML/查询不改变表清单与列结构，不得无谓失效（否则缓存失去意义）
func TestExecNonDDLKeepsSchemaCache(t *testing.T) {
	for _, execSQL := range []string{
		"INSERT INTO t_user (id) VALUES (1)",
		"UPDATE t_user SET age = 1 WHERE id = 1",
		"DELETE FROM t_user WHERE id = 1",
		"SELECT * FROM t_user",
	} {
		conn, mock := newSchemaCacheConn(t)
		mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 1))

		_, err := conn.Exec(execSQL)
		require.NoError(t, err)
		assert.True(t, schemaCacheSeeded(conn), "非 DDL 语句不得清缓存: %s", execSQL)
		require.NoError(t, mock.ExpectationsWereMet())
	}
}

// TestFailedDDLKeepsSchemaCache 执行失败的语句未改变结构，不该因错误而反复清缓存
func TestFailedDDLKeepsSchemaCache(t *testing.T) {
	conn, mock := newSchemaCacheConn(t)
	mock.ExpectExec(".*").WillReturnError(errors.New("table exists"))

	_, err := conn.Exec("CREATE TABLE t_user (id INT)")
	require.Error(t, err)
	assert.True(t, schemaCacheSeeded(conn), "DDL 失败时保持缓存不变")
}

// TestDbInfoExecInvalidatesSchemaCache 方言内部（如 CopyTable 的 create table like）经 DbInfo.Exec 执行，同样须失效
func TestDbInfoExecInvalidatesSchemaCache(t *testing.T) {
	conn, mock := newSchemaCacheConn(t)
	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := conn.Info.Exec("CREATE TABLE t_user_copy_1 LIKE t_user")
	require.NoError(t, err)
	assert.False(t, schemaCacheSeeded(conn))
}

// TestExecWithoutCacheOrBackendNoPanic 未建立连接/无方言后端的 DbInfo（如纯测试构造）执行 DDL 不得 panic
func TestExecWithoutCacheOrBackendNoPanic(t *testing.T) {
	conn, mock := newMockConn(t)
	mock.ExpectExec(".*").WillReturnResult(sqlmock.NewResult(0, 0))

	assert.NotPanics(t, func() {
		_, err := conn.Exec("CREATE TABLE t1 (id INT)")
		assert.NoError(t, err)
	})
	assert.Nil(t, conn.Info.schemaCache)
}
