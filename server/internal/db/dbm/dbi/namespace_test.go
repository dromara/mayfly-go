package dbi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// schemalessBackend 无独立 schema 层的方言（模拟 mysql/sqlite）：覆写 GetCapabilities 使 HasSchema=false。
type schemalessBackend struct {
	stubBackend
}

func (schemalessBackend) GetCapabilities() MetadataCapabilities {
	return NewAllCapabilities().WithNamespace(NamespaceHierarchy{HasDatabase: true, HasSchema: false})
}

const (
	testSchemaDt   = DbType("codec-test-with-schema")  // stubBackend：HasSchema=true（内嵌 DefaultBackend）
	testNoSchemaDt = DbType("codec-test-no-schema")    // schemalessBackend：HasSchema=false
	testUnknownDt  = DbType("codec-test-unregistered") // 未注册：GetBackend 返回 nil
)

func init() {
	RegisterBackend(testSchemaDt, &stubBackend{})
	RegisterBackend(testNoSchemaDt, &schemalessBackend{})
}

// ParseLocator 方言感知：有 schema 层才按首个 `/` 拆；无 schema 层整串即库名（含 `/` 不误拆）。
func TestParseLocator_DialectAware(t *testing.T) {
	// 有 schema 层
	db, schema := ParseLocator(testSchemaDt, "mydb/public")
	assert.Equal(t, "mydb", db)
	assert.Equal(t, "public", schema)

	db, schema = ParseLocator(testSchemaDt, "mydb")
	assert.Equal(t, "mydb", db)
	assert.Equal(t, "", schema)

	// 任意段数：首段为库名，其后全部为 schema（保 round-trip）
	db, schema = ParseLocator(testSchemaDt, "a/b/c")
	assert.Equal(t, "a", db)
	assert.Equal(t, "b/c", schema)

	// 无 schema 层：库名含 `/` 也不拆
	db, schema = ParseLocator(testNoSchemaDt, "a/b")
	assert.Equal(t, "a/b", db)
	assert.Equal(t, "", schema)

	// 未注册方言保守视为无 schema 层
	db, schema = ParseLocator(testUnknownDt, "mydb/public")
	assert.Equal(t, "mydb/public", db)
	assert.Equal(t, "", schema)

	// 空串
	db, schema = ParseLocator(testSchemaDt, "")
	assert.Equal(t, "", db)
	assert.Equal(t, "", schema)
}

// locator round-trip 保真：FormatLocator(ParseLocator(dt, s)) == s，对任意段数与方言成立。
func TestLocator_RoundTrip(t *testing.T) {
	locators := []string{"", "mydb", "mydb/public", "a/b/c", "weird/db/name"}
	for _, dt := range []DbType{testSchemaDt, testNoSchemaDt, testUnknownDt} {
		for _, s := range locators {
			db, schema := ParseLocator(dt, s)
			assert.Equal(t, s, FormatLocator(db, schema), "dt=%q locator=%q", dt, s)
		}
	}
	// 反向：ParseLocator(dt, FormatLocator(db, "")) == (db, "")（无 schema 场景无损）
	for _, dt := range []DbType{testSchemaDt, testNoSchemaDt} {
		db, schema := ParseLocator(dt, FormatLocator("mydb", ""))
		assert.Equal(t, "mydb", db)
		assert.Equal(t, "", schema)
	}
}

// connId 一致性：Conn() 用 FormatLocator(Database,Schema) 重建的 locator，
// 与 dbm 侧以原始 locator 算出的池键逐字节相等（否则连接池/CloseDb/实例反向索引错位）。
func TestConnId_RebuildMatchesLocator(t *testing.T) {
	for _, locator := range []string{"mydb", "mydb/public", "mydb/sales"} {
		db, schema := ParseLocator(testSchemaDt, locator)
		// Conn() 路径：结构化字段重建
		rebuilt := FormatLocator(db, schema)
		assert.Equal(t, GetDbConnId(5, locator), GetDbConnId(5, rebuilt), "locator=%q", locator)
	}
	// 同库不同 schema 必须是两个不同连接单例（schema_cache 不串味）
	assert.NotEqual(t, GetDbConnId(5, "mydb/public"), GetDbConnId(5, "mydb/sales"))
}
