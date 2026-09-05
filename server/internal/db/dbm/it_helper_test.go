//go:build it

package dbm

// 集成测试公共基建（连接/断言/数据读取），仅供包内各 *_it_test.go 使用：
//   - 测试环境：本机mysql(3306, 测试库mayfly_dbm_it自动创建) + Docker pg(mayfly-pg-it容器
//     postgres:16@5432, 库mayfly_pg_it) + sqlite临时文件
//   - 断言：assertCell/assertRow 做跨方言值形态宽松归一（数值跨类型、[]byte/time归一）

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	_ "mayfly-go/internal/db/dbm/mysql"  // 注册mysql方言
	_ "mayfly-go/internal/db/dbm/sqlite" // 注册sqlite方言
)

const itMysqlDatabase = "mayfly_dbm_it"

const itPgDatabase = "mayfly_pg_it"

func itCtx() context.Context {
	return context.Background()
}

// 直接连接到mysql server（不指定业务库），确保测试库存在
func ensureMysqlDatabase(t *testing.T) {
	t.Helper()
	conn := connectDbInfo(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "information_schema"})
	defer conn.Close()
	_, err := conn.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARSET utf8mb4", itMysqlDatabase))
	require.NoError(t, err)
}

func connectDbInfo(t *testing.T, di *dbi.DbInfo) *dbi.DbConn {
	t.Helper()
	conn, err := Conn(itCtx(), di)
	require.NoError(t, err)
	return conn
}

func mysqlConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	ensureMysqlDatabase(t)
	return connectDbInfo(t, &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: itMysqlDatabase})
}

func pgConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	conn := connectDbInfo(t, &dbi.DbInfo{
		Type: "postgres", Host: "127.0.0.1", Port: 5432,
		Username: "postgres", Password: "postgres", Database: itPgDatabase,
	})
	require.NoError(t, conn.Ping())
	return conn
}

func sqliteConn(t *testing.T) *dbi.DbConn {
	t.Helper()
	path := filepath.Join(t.TempDir(), "dbm_it.sqlite")
	require.NoError(t, os.WriteFile(path, []byte{}, 0644))
	return connectDbInfo(t, &dbi.DbInfo{Type: "sqlite", Host: path})
}

func mustExec(t *testing.T, conn *dbi.DbConn, sql string) {
	t.Helper()
	if _, err := conn.Exec(sql); err != nil {
		t.Fatalf("exec failed [%s]: %s", sql, err.Error())
	}
}

// normalizeDbValue 归一化数据库驱动返回值，便于跨库对比
func normalizeDbValue(v any) any {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format("2006-01-02 15:04:05")
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case uint:
		return int64(val)
	case uint32:
		return int64(val)
	default:
		return v
	}
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int64:
		return float64(x), true
	case uint64:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// assertCell 断言单元格值一致：数值跨类型宽松对比（int64/float64/字符串形态的数值），其余严格对比
func assertCell(t *testing.T, col string, expect, actual any) {
	t.Helper()
	expect, actual = normalizeDbValue(expect), normalizeDbValue(actual)

	if expect == nil || actual == nil {
		assert.Nil(t, actual, "col [%s] expect NULL, actual: %v", col, actual)
		if expect == nil {
			assert.Nil(t, expect, "col [%s] unexpected", col)
		}
		return
	}

	ef, eok := toFloat(expect)
	af, aok := toFloat(actual)
	if eok && aok {
		assert.Equal(t, ef, af, "col [%s] numeric value not equal: expect %v actual %v", col, expect, actual)
		return
	}
	assert.Equal(t, fmt.Sprintf("%v", expect), fmt.Sprintf("%v", actual), "col [%s] value not equal", col)
}

func assertRow(t *testing.T, expect map[string]any, actual map[string]any, rowIdx int) {
	t.Helper()
	for col, expectVal := range expect {
		require.Contains(t, actual, col, "row %d missing column [%s]", rowIdx, col)
		assertCell(t, col, expectVal, actual[col])
	}
}

func readAllRows(t *testing.T, conn *dbi.DbConn, table string, orderBy string) []map[string]any {
	t.Helper()
	_, rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY %s", conn.GetDialect().Quoter().Quote(table), orderBy))
	require.NoError(t, err)
	return rows
}

// setupMysqlSourceTable 建立迁移/转义共用的全类型mysql源表并填充数据，返回列与期望数据
func setupMysqlSourceTable(t *testing.T, conn *dbi.DbConn, table string) ([]dbi.Column, []map[string]any) {
	t.Helper()
	quote := conn.GetDialect().Quoter().Quote
	mustExec(t, conn, "DROP TABLE IF EXISTS "+quote(table))
	mustExec(t, conn, fmt.Sprintf(`CREATE TABLE %s (
		id int NOT NULL,
		v_varchar varchar(128) NOT NULL DEFAULT 'it''s' COMMENT '姓''名',
		v_nullable varchar(64) DEFAULT NULL,
		v_text text COMMENT '文本''s',
		v_dec decimal(10,2) NOT NULL,
		v_uint int unsigned NOT NULL,
		v_ubig bigint unsigned NOT NULL,
		v_utiny tinyint unsigned NOT NULL,
		v_dt datetime DEFAULT NULL,
		v_blob blob DEFAULT NULL,
		PRIMARY KEY (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`, quote(table)))

	columns, err := conn.GetMetadata().GetColumns(table)
	require.NoError(t, err)
	require.Len(t, columns, 10)

	// 期望数据（key为列名）。注意：含反斜杠/单引号/双引号/换行/中文/emoji/NULL/空串。
	// v_blob按迁移链路真实形态传入：ValuerBytes回读二进制为hex编码字符串，
	// 写入侧经SQLValueBytes输出X'...'保真还原为原始字节
	rows := []map[string]any{
		{
			"id": 1, "v_varchar": "it's", "v_nullable": nil, "v_text": "line1\nline2\tend",
			"v_dec": "123.45", "v_uint": 4294967295, "v_ubig": 1844674407370955161,
			"v_utiny": 255, "v_dt": "2026-01-02 03:04:05", "v_blob": hex.EncodeToString([]byte("bin data")),
		},
		{
			"id": 2, "v_varchar": `a\b`, "v_nullable": `say "hi"`, "v_text": "中文'单引'与emoji🙂",
			"v_dec": "-99.99", "v_uint": 0, "v_ubig": 0,
			"v_utiny": 0, "v_dt": nil, "v_blob": nil,
		},
		{
			"id": 3, "v_varchar": "normal", "v_nullable": "", "v_text": "",
			"v_dec": "0.01", "v_uint": 1, "v_ubig": 1,
			"v_utiny": 1, "v_dt": "2026-12-31 23:59:59", "v_blob": hex.EncodeToString([]byte("blob'quote\\slash")),
		},
	}

	// 通过GenInsert生成插入SQL并真实执行（走完整的值转义链路）
	gen := conn.GetDialect().GetSQLGenerator()
	values := make([][]any, 0, len(rows))
	for _, row := range rows {
		rowVals := make([]any, 0, len(columns))
		for _, col := range columns {
			rowVals = append(rowVals, row[col.ColumnName])
		}
		values = append(values, rowVals)
	}
	insertSqls := gen.GenInsert(table, columns, values, dbi.DuplicateStrategyNone, nil)
	require.NotEmpty(t, insertSqls)
	for _, sql := range insertSqls {
		mustExec(t, conn, sql)
	}
	return columns, rows
}
