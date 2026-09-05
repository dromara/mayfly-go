package dbi

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubMeta 测试用Meta实现
type stubMeta struct {
	dialect Dialect
}

func (m *stubMeta) GetSqlDb(ctx context.Context, d *DbInfo) (*sql.DB, error) { return nil, nil }
func (m *stubMeta) GetDialect(d *DbConn) Dialect                             { return m.dialect }
func (m *stubMeta) GetMetadata(d *DbConn) Metadata                           { return nil }
func (m *stubMeta) GetDbDataTypes() []*DbDataType {
	return []*DbDataType{NewDbDataType("stub_varchar", DTString).WithCT(CTVarchar)}
}
func (m *stubMeta) GetCommonTypeConverter() CommonTypeConverter { return &testConverter{} }

var _ Meta = (*stubMeta)(nil)

func TestRegister_NilMetaPanic(t *testing.T) {
	assert.Panics(t, func() {
		Register(DbType("test-nil-meta"), nil)
	})
}

func TestGetMeta_Unregistered(t *testing.T) {
	assert.Nil(t, GetMeta(DbType("test-never-registered")))
	assert.Nil(t, GetDialect(DbType("test-never-registered")))
}

func TestGetMeta_RegisterAndInit(t *testing.T) {
	testDbType := DbType("test-stub-db")
	meta := &stubMeta{}
	Register(testDbType, meta)

	// 首次获取触发类型与转换器注册
	got := GetMeta(testDbType)
	assert.NotNil(t, got)
	assert.Equal(t, meta, got)

	// 再次获取返回同一实例
	assert.Equal(t, meta, GetMeta(testDbType))

	// initMeta已注册列类型
	dt := GetDbDataType(testDbType, "stub_varchar")
	assert.Equal(t, "stub_varchar", dt.Name)

	// initMeta已注册公共类型转换器
	cts := getCommonTypeConverters(testDbType)
	assert.NotNil(t, cts)
	assert.NotNil(t, cts[CTVarchar])
}

func TestGetMeta_Concurrent(t *testing.T) {
	testDbType := DbType("test-concurrent-db")
	Register(testDbType, &stubMeta{})

	// 并发获取不应panic且均能拿到meta（验证读写锁正确性）
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotNil(t, GetMeta(testDbType))
			assert.NotNil(t, GetDbDataType(testDbType, "stub_varchar"))
		}()
	}
	wg.Wait()
}
