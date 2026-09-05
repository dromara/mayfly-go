package dbm

// 各方言字符串转义矩阵单元测试（无环境依赖）：
// 验证各方言字符串/JSON类型的SQLValue转义接线正确——
//   - mysql/clickhouse为反斜杠转义语义（'双写 + \双写，其余原样）
//   - 标准SQL方言（pg/sqlite/mssql/oracle/dm）为'双写语义（反斜杠是普通字符，绝不能转义，否则数据损坏）
// 复杂字符谱系：单引号 双引号 反斜杠 回车CR 换行LF CRLF tab NUL 中文 emoji 多行JSON

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"mayfly-go/internal/db/dbm/clickhouse"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dm"
	"mayfly-go/internal/db/dbm/mssql"
	"mayfly-go/internal/db/dbm/mysql"
	"mayfly-go/internal/db/dbm/oracle"
	"mayfly-go/internal/db/dbm/postgres"
	"mayfly-go/internal/db/dbm/sqlite"
)

// complexVal 复杂字符串：单引号/双引号/反斜杠/CR/CRLF/LF/tab/中文/emoji/多行JSON
var complexVal = "it's \"q\" \\back\rreturn\nlf\r\nline2\tend中文🙂\n" + `{"msg": "it's \"quoted\""}`

var nulVal = "nul\x00end"

// mysqlEscape mysql/CH转义规则：先\双写再'双写，其余（\r \n \t \0 " 等）原样
func mysqlEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `''`)
	return s
}

// stdEscape 标准SQL转义规则：仅'双写，反斜杠与控制字符一律原样
func stdEscape(s string) string {
	return strings.ReplaceAll(s, `'`, `''`)
}

func assertMysqlStyle(t *testing.T, name string, dt *dbi.DbDataType, val string) {
	t.Helper()
	want := "'" + mysqlEscape(val) + "'"
	assert.Equal(t, want, dt.DataType.SQLValue(val), "%s mysql式转义输出不符", name)
	// 关键：反斜杠必须双写（否则值会被静默解释为转义字符）
	assert.Contains(t, dt.DataType.SQLValue(val), `\\`, "%s 应双写反斜杠", name)
}

func assertStdStyle(t *testing.T, name string, dt *dbi.DbDataType, val string) {
	t.Helper()
	want := "'" + stdEscape(val) + "'"
	assert.Equal(t, want, dt.DataType.SQLValue(val), "%s 标准式转义输出不符", name)
	// 关键反例：反斜杠必须原样（若双写会在目标库被解释为转义导致数据损坏）
	assert.NotContains(t, dt.DataType.SQLValue(val), `\\`, "%s 反斜杠应原样不得双写", name)
	// 回车换行原样保留（不转义为\r\n字面量）
	assert.Contains(t, dt.DataType.SQLValue(val), "\r\n", "%s 应原样保留CRLF", name)
}

// TestDialectStringEscapeMatrix 各方言字符串类型转义接线矩阵
func TestDialectStringEscapeMatrix(t *testing.T) {
	// mysql系：反斜杠转义语义
	assertMysqlStyle(t, "mysql.Varchar", mysql.Varchar, complexVal)
	assertMysqlStyle(t, "mysql.Text", mysql.Text, complexVal)
	assertMysqlStyle(t, "mysql.JSON", mysql.JSON, complexVal)
	assertMysqlStyle(t, "clickhouse.String", clickhouse.String, complexVal)
	assertMysqlStyle(t, "clickhouse.FixedString", clickhouse.FixedString, complexVal)

	// 标准SQL系：仅单引号双写
	assertStdStyle(t, "postgres.Varchar", postgres.Varchar, complexVal)
	assertStdStyle(t, "postgres.Text", postgres.Text, complexVal)
	assertStdStyle(t, "postgres.Jsonb", postgres.Jsonb, complexVal)
	assertStdStyle(t, "sqlite.Text", sqlite.Text, complexVal)
	assertStdStyle(t, "mssql.Varchar", mssql.Varchar, complexVal)
	assertStdStyle(t, "oracle.VARCHAR2", oracle.VARCHAR2, complexVal)
	assertStdStyle(t, "dm.VARCHAR", dm.VARCHAR, complexVal)
}

// TestDialectNulCharMatrix NUL字节（\0）在字符串中的转义输出：
// mysql/clickhouse与标准SQL方言均原样输出（SQL文本中嵌入NUL字节由驱动与目标库决定可否存储：
// mysql/sqlite可存，pg text与oracle等硬性禁止——属数据库固有限制而非转义缺陷）
func TestDialectNulCharMatrix(t *testing.T) {
	// NUL无需转义，原样嵌入字符串字面量
	assert.Equal(t, "'nul\x00end'", mysql.Varchar.DataType.SQLValue(nulVal))
	assert.Equal(t, "'nul\x00end'", postgres.Varchar.DataType.SQLValue(nulVal))
	assert.Equal(t, "'nul\x00end'", sqlite.Text.DataType.SQLValue(nulVal))
}

// TestMysqlToJsonValueRoundtrip mysql JSON值的导出转义正确性：
// JSON值含 " ' \ \n 等字符，经mysql转义输出后应能还原为同一JSON文本
func TestMysqlToJsonValueRoundtrip(t *testing.T) {
	// 导出侧：mysql JSON列回读的规范化JSON文本 → SQL字符串字面量
	jsonText := `{"msg": "it's \"quoted\"", "path": "C:\\tmp", "note": "a\nb"}`
	out := mysql.JSON.DataType.SQLValue(jsonText)
	// ' → ''、\ → \\，其余原样
	assert.Equal(t, `'{"msg": "it''s \\"quoted\\"", "path": "C:\\\\tmp", "note": "a\\nb"}'`, out)
}
