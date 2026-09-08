package dbi

import "mayfly-go/internal/db/dbm/sqlparser"

// stubDialect 测试用完整Dialect实现（复用DefaultDialect的Quoter等默认实现，仅补齐剩余接口方法）
type stubDialect struct {
	DefaultDialect
}

func (d *stubDialect) CopyTable(copy *DbCopyTable) error { return nil }
func (d *stubDialect) GetSQLGenerator() SQLGenerator     { return nil }
func (d *stubDialect) GetSQLParser() sqlparser.SqlParser { return d.DefaultDialect.GetSQLParser() }

var _ Dialect = (*stubDialect)(nil)

func newStubDialect() *stubDialect { return &stubDialect{} }
