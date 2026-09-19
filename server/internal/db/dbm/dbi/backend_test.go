package dbi

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stubBackend 测试用DbBackend实现
type stubBackend struct {
	BaseBackend
	dialect Dialect
}

func (m *stubBackend) GetSqlDb(ctx context.Context, d *DbInfo) (*sql.DB, error) { return nil, nil }
func (m *stubBackend) GetDialect(d *DbInfo) Dialect                             { return m.dialect }
func (m *stubBackend) GetServerInfo(d *DbInfo) ServerInfo                       { return nil }
func (m *stubBackend) GetMetadataProvider(d *DbInfo) MetadataProvider           { return nil }

var _ DbBackend = (*stubBackend)(nil)

func TestRegisterBackend_NilBackendPanic(t *testing.T) {
	assert.Panics(t, func() {
		RegisterBackend(DbType("test-nil-backend"), nil)
	})
}

func TestGetBackend_Unregistered(t *testing.T) {
	assert.Nil(t, GetBackend(DbType("test-never-registered")))
	assert.Nil(t, GetDialect(DbType("test-never-registered")))
}

func TestGetBackend_RegisterAndRetrieve(t *testing.T) {
	testDbType := DbType("test-stub-db")
	backend := &stubBackend{}
	RegisterBackend(testDbType, backend)

	got := GetBackend(testDbType)
	assert.NotNil(t, got)
	assert.Equal(t, backend, got)

	// 再次获取返回同一实例
	assert.Equal(t, backend, GetBackend(testDbType))
}

func TestGetBackend_Concurrent(t *testing.T) {
	testDbType := DbType("test-concurrent-db")
	RegisterBackend(testDbType, &stubBackend{})

	// 并发获取不应panic且均能拿到backend（验证读写锁正确性）
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			assert.NotNil(t, GetBackend(testDbType))
		}()
	}
	wg.Wait()
}
