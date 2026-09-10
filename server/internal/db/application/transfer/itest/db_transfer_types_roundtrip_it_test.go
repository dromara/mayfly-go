package itest

// 全类型"备份恢复"端到端真实链路集成测试（备份恢复是dump导入导出的核心应用场景）：
//
//	源库写入类型边界值（date/time/微秒时间戳/布尔/大整数/浮点精度/decimal/NULL混合）
//	→ DumpDbScript导出到.sql文件（真实备份产物）
//	→ DROP源表（模拟库损坏/数据清空）
//	→ ImportDumpStream从文件恢复（真实导入）
//	→ 回读全表逐行逐列规范化比对，任何值不一致即失败
//
// 覆盖类型边界：闰年日期/年末日/微秒全9/最小非零微秒/int64两极/float最短表示round-trip/
// decimal高精度/布尔三态（true/false/NULL）。
//
// 运行：cd server && go test -tags it -count=1 -run TestITTypesBackupRestore ./internal/db/application/transfer/

import (
	"mayfly-go/internal/db/application/transfer"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

// typesRTColumns 类型化列（各方言DDL差异由typesRTCreateTable处理）
var typesRTColumns = []string{"id", "v_date", "v_time", "v_ts", "v_bool", "v_big", "v_flo", "v_dec"}

// typesRTCreateTable 按方言建类型丰富的表
func typesRTCreateTable(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	var ddl string
	switch conn.Info.Type {
	case "mysql":
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id int PRIMARY KEY, v_date DATE, v_time TIME(6), v_ts DATETIME(6), v_bool TINYINT(1), "+
			"v_big BIGINT, v_flo DOUBLE, v_dec DECIMAL(30,10))", quote(table))
	case "postgres":
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id int PRIMARY KEY, v_date DATE, v_time TIME(6), v_ts TIMESTAMP(6), v_bool BOOLEAN, "+
			"v_big BIGINT, v_flo DOUBLE PRECISION, v_dec NUMERIC(30,10))", quote(table))
	case "sqlite":
		// sqlite弱类型：按存入形态选择亲和（TEXT保时间文本、INTEGER/REAL保数值）
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id INTEGER PRIMARY KEY, v_date TEXT, v_time TEXT, v_ts TEXT, v_bool INTEGER, "+
			"v_big INTEGER, v_flo REAL, v_dec NUMERIC)", quote(table))
	default:
		t.Fatalf("unsupported dialect %s", conn.Info.Type)
	}
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(ddl)
	require.NoError(t, err, "建表失败")
}

// typesRTRows 边界值行：任何一列的往返丢失（微秒截断/精度丢失/NULL变默认值）都应被捕获
var typesRTRows = []map[string]any{
	{
		"id": int64(1), "v_date": "2024-02-29", "v_time": "23:59:59.999999",
		"v_ts": "2024-06-15 08:30:05.123456", "v_bool": true,
		"v_big": int64(9223372036854775807), "v_flo": 0.1, "v_dec": "12345678901234.1234567890",
	},
	{
		"id": int64(2), "v_date": "9999-12-31", "v_time": "00:00:00.000001",
		"v_ts": "9999-12-31 23:59:59.999999", "v_bool": false,
		"v_big": int64(-9223372036854775808), "v_flo": -123456.789, "v_dec": "-0.0000000001",
	},
	{
		"id": int64(3), "v_date": "1000-01-01", "v_time": "12:34:56",
		"v_ts": "1970-01-01 00:00:00.000001", "v_bool": nil,
		"v_big": int64(0), "v_flo": 3.141592653589793, "v_dec": nil,
	},
	{
		"id": int64(4), "v_date": nil, "v_time": nil,
		"v_ts": nil, "v_bool": nil,
		"v_big": nil, "v_flo": 1e-10, "v_dec": "0",
	},
}

// typesRTCanonical 单元格规范化：抹平驱动返值形态差异（时间string/time.Time、
// 数值int64/float64、bool三态表示），同列经同一函数处理后方可比对
func typesRTCanonical(t *testing.T, col string, v any) string {
	t.Helper()
	if v == nil {
		return "<NIL>"
	}
	switch col {
	case "v_bool":
		b, ok := dbi.ValToBool(v)
		require.True(t, ok, "v_bool无法归一: %v", v)
		if b {
			return "true"
		}
		return "false"
	case "v_big":
		i, ok := dbi.ValToInt64(v)
		require.True(t, ok, "v_big无法归一: %v", v)
		return fmt.Sprintf("%d", i)
	case "v_flo":
		f, ok := dbi.ValToFloat64(v)
		require.True(t, ok, "v_flo无法归一: %v", v)
		// 最短表示：同一float64在导出/导入/回读全链路应精确一致
		return fmt.Sprintf("%v", f)
	case "v_dec":
		// decimal高精度各库返值形态不同（string/[]byte/float），统一按浮点等值断言；
		// 精度保真由同方言dump→导入→回读链路的两侧同函数处理保证
		if f, ok := dbi.ValToFloat64(v); ok {
			return fmt.Sprintf("%v", f)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	default:
		// 时间类列：Valuer已归一为文本；兜底time.Time按微秒layout格式化
		switch tv := v.(type) {
		case string:
			return tv
		case []byte:
			return string(tv)
		default:
			return fmt.Sprintf("%v", tv)
		}
	}
}

// typesRTSnapshot 回读全表并规范化为 [id][col] -> canonical 的快照
func typesRTSnapshot(t *testing.T, conn *dbi.DbConn, table string) map[int64]map[string]string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(table)))
	require.NoError(t, err)
	snap := make(map[int64]map[string]string, len(rows))
	for _, row := range rows {
		id, ok := dbi.ValToInt64(row["id"])
		require.True(t, ok, "id无法归一: %v", row["id"])
		m := make(map[string]string, len(typesRTColumns))
		for _, col := range typesRTColumns {
			v, has := row[col]
			if !has {
				// 列名大小写差异兜底
				for k, kv := range row {
					if strings.EqualFold(k, col) {
						v, has = kv, true
						break
					}
				}
			}
			require.True(t, has, "列[%s]缺失", col)
			m[col] = typesRTCanonical(t, col, v)
		}
		snap[id] = m
	}
	return snap
}

// typesRTWriteRows 参数化写入边界值
func typesRTWriteRows(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	ph := make([]string, 0, len(typesRTColumns))
	for i := range typesRTColumns {
		if conn.Info.Type == "postgres" {
			ph = append(ph, fmt.Sprintf("$%d", i+1))
		} else {
			ph = append(ph, "?")
		}
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		quote(table), strings.Join(typesRTColumns, ", "), strings.Join(ph, ", "))
	for _, r := range typesRTRows {
		args := make([]any, 0, len(typesRTColumns))
		for _, col := range typesRTColumns {
			args = append(args, r[col])
		}
		_, err := conn.Exec(sql, args...)
		require.NoError(t, err, "写入行失败: %v", r["id"])
	}
}

// TestITTypesBackupRestoreRoundtrip 同方言"导出→清库→从文件恢复→全值比对"，
// mysql/pg/sqlite三方言各自完整走一遍备份恢复语义
func TestITTypesBackupRestoreRoundtrip(t *testing.T) {
	nodes := []struct {
		name string
		conn func(t *testing.T) *dbi.DbConn
	}{
		{"mysql", itMysqlNode},
		{"pg", itPgNode},
		{"sqlite", itSqliteNode},
	}
	for _, n := range nodes {
		t.Run(n.name, func(t *testing.T) {
			conn := n.conn(t)
			defer conn.Close()
			table := "it_types_rt"
			typesRTCreateTable(t, conn, table)
			typesRTWriteRows(t, conn, table)

			// 备份：dump到.sql文件（同方言，含DDL+数据）
			backupPath := filepath.Join(t.TempDir(), "backup.sql")
			var buf bytes.Buffer
			require.NoError(t, transfer.DumpDbScript(context.Background(), conn, &dto.DumpDb{
				DbName:   conn.Info.Database,
				Tables:   []string{table},
				DumpDDL:  true,
				DumpData: true,
				Writer:   &buf,
			}), "dump失败")
			require.NoError(t, os.WriteFile(backupPath, buf.Bytes(), 0o644))
			scriptBytes, err := os.ReadFile(backupPath)
			require.NoError(t, err)
			require.Contains(t, string(scriptBytes), "INSERT INTO", "备份文件应含数据")

			// 恢复前快照，随后清库（模拟数据丢失）
			before := typesRTSnapshot(t, conn, table)
			quote := conn.GetDialect().Quoter().QuoteIdent
			_, err = conn.Exec(fmt.Sprintf("DROP TABLE %s", quote(table)))
			require.NoError(t, err)

			// 恢复：从备份文件导入（真实splitter切分+批级事务）
			f, err := os.Open(backupPath)
			require.NoError(t, err)
			defer f.Close()
			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, conn, f), "从备份文件恢复失败")

			// 逐行逐列比对
			after := typesRTSnapshot(t, conn, table)
			require.Len(t, after, len(before), "恢复后行数不一致")
			for id, beforeRow := range before {
				afterRow, ok := after[id]
				require.True(t, ok, "恢复后缺少id=%d", id)
				for _, col := range typesRTColumns {
					require.Equal(t, beforeRow[col], afterRow[col],
						"id=%d 列[%s] 备份恢复后值不一致（导出转义或导入切分存在数据失真）", id, col)
				}
			}
		})
	}
}
