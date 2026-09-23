//go:build it

package itest

// 异构 / 复杂源查询端到端同步集成测试（黑盒，经导出入口 SyncBatch）：
//   - 复杂聚合源（JOIN+GROUP BY+HAVING）→ pg 目标真实同步；
//   - 全类型×异构方向×大数据量：pg↔mysql 各 2 万行 bulk + 边界行，覆盖策略二次同步幂等。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/sync"
	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/dbm/dbi/value"
	"mayfly-go/internal/db/dbm/sqlparser/sqlstmt"
	"mayfly-go/internal/db/domain/entity"
)

func TestITDataSyncComplexJoinGroupBy(t *testing.T) {
	srcConn := mysqlConn(t)
	defer srcConn.Close()
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_cs_dept`")
	mustExec(t, srcConn, "DROP TABLE IF EXISTS `it_cs_emp`")
	mustExec(t, srcConn, "CREATE TABLE `it_cs_dept` (dept VARCHAR(32) PRIMARY KEY, city VARCHAR(64))")
	mustExec(t, srcConn, "CREATE TABLE `it_cs_emp` (id INT PRIMARY KEY, dept VARCHAR(32), salary INT)")
	mustExec(t, srcConn, "INSERT INTO `it_cs_dept` VALUES ('eng','bj'),('sales','sh')")
	mustExec(t, srcConn, "INSERT INTO `it_cs_emp` VALUES (1,'eng',3000),(2,'eng',5000),(3,'sales',4000)")

	dstConn := pgConn(t)
	defer dstConn.Close()
	mustExec(t, dstConn, "DROP TABLE IF EXISTS it_cs_target")
	mustExec(t, dstConn, "CREATE TABLE it_cs_target (dept VARCHAR(32) PRIMARY KEY, cnt INT)")

	complexSrcSQL := "SELECT e.dept, COUNT(*) AS cnt FROM `it_cs_emp` e JOIN `it_cs_dept` d ON d.dept = e.dept GROUP BY e.dept HAVING COUNT(*) >= 1"
	_, srcRes, err := srcConn.Query(complexSrcSQL)
	require.NoError(t, err)
	require.Len(t, srcRes, 2, "聚合源查询应返回2个分组")

	stmt, err := srcConn.GetDialect().GetSQLParser().Parse(complexSrcSQL)
	require.NoError(t, err)
	sel, ok := stmt.(*sqlstmt.SelectStmt)
	require.True(t, ok, "期望SelectStmt，得到%T", stmt)
	require.Len(t, sel.GroupBy, 1, "GROUP BY应被回填")
	require.NotNil(t, sel.Having, "HAVING应被回填")

	targetColumns := []dbi.Column{
		{ColumnName: "dept", DataType: "varchar", IsPrimaryKey: true},
		{ColumnName: "cnt", DataType: "int"},
	}
	fieldMap := []map[string]string{{"src": "dept", "target": "dept"}, {"src": "cnt", "target": "cnt"}}
	task := &entity.DataSyncTask{TargetTableName: "it_cs_target", DuplicateStrategy: 2}
	app, _ := newApp()
	require.NoError(t, syncBatch(app, srcRes, fieldMap, task, dstConn, targetColumns))

	_, rows, err := dstConn.Query("SELECT dept, cnt FROM it_cs_target ORDER BY dept")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "eng", fmt.Sprint(rows[0]["dept"]))
	assert.Equal(t, int64(2), toInt64(rows[0]["cnt"]))
}

const syncHeteroRows = 20000

type syncHeteroSpec struct {
	name       string
	srcType    string
	tgtType    string
	srcDDL     string
	tgtDDL     string
	cols       []string
	srcBulkSQL string
	boundRows  [][]any
}

var syncHeteroSpecs = []syncHeteroSpec{
	{
		name: "pg->mysql", srcType: "postgres", tgtType: "mysql",
		srcDDL: `CREATE TABLE it_sync_hetero (id BIGINT PRIMARY KEY, c_int INTEGER, c_big BIGINT, c_f64 DOUBLE PRECISION,
			c_dec NUMERIC(30,10), c_date DATE, c_dt TIMESTAMP(6), c_v VARCHAR(200), c_text TEXT,
			c_bytea BYTEA, c_bool BOOLEAN, c_jsonb JSONB)`,
		tgtDDL: "CREATE TABLE `it_sync_hetero_t` (id BIGINT PRIMARY KEY, c_int INT, c_big BIGINT, c_f64 DOUBLE, " +
			"c_dec DECIMAL(30,10), c_date DATE, c_dt DATETIME(6), c_v VARCHAR(200), c_text TEXT, " +
			"c_bytea LONGBLOB, c_bool TINYINT(1), c_jsonb JSON) DEFAULT CHARSET=utf8mb4",
		cols: []string{"c_int", "c_big", "c_f64", "c_dec", "c_date", "c_dt", "c_v", "c_text", "c_bytea", "c_bool", "c_jsonb"},
		srcBulkSQL: `INSERT INTO it_sync_hetero SELECT g, FLOOR(RANDOM()*2147483647)::int, FLOOR(RANDOM()*9223372036854775807)::bigint,
			RANDOM(), RANDOM()*1000000000, DATE '2000-01-01' + FLOOR(RANDOM()*9000)::int,
			TIMESTAMP '2020-01-01' + (FLOOR(RANDOM()*100000)::int || ' minutes')::interval,
			md5(random()::text), repeat(md5(random()::text), 10), decode(md5(random()::text), 'hex'),
			RANDOM()>0.5, json_build_object('k', FLOOR(RANDOM()*1000)::int, 'v', md5(random()::text))
			FROM generate_series(4, 20003) g`,
		boundRows: [][]any{
			{"1", "-2147483648", "-9223372036854775808", "-1.5", "12345678901234.1234567890", "2024-02-29",
				"2024-06-15 08:30:05.123456", "single'quote\"double\\back", "emoji😀🎉多字节\nline2",
				"00ff7f", "true", `{"b":true,"a":[1,2,"x'y"]}`},
			{"2", "2147483647", "9223372036854775807", "1e-10", "-0.0000000001", "1000-01-01",
				"2037-12-31 23:59:59.999999", strings.Repeat("x", 200), strings.Repeat("中", 234),
				"dead", "false", `{"k":"v"}`},
		},
	},
	{
		name: "mysql->pg", srcType: "mysql", tgtType: "postgres",
		srcDDL: "CREATE TABLE `it_sync_hetero` (id BIGINT PRIMARY KEY, c_int INT, c_big BIGINT, c_f64 DOUBLE, " +
			"c_dec DECIMAL(30,10), c_date DATE, c_dt DATETIME(6), c_v VARCHAR(200), c_text TEXT, " +
			"c_blob BLOB, c_json JSON) DEFAULT CHARSET=utf8mb4",
		tgtDDL: `CREATE TABLE it_sync_hetero_t (id BIGINT PRIMARY KEY, c_int INTEGER, c_big BIGINT, c_f64 DOUBLE PRECISION,
			c_dec NUMERIC(30,10), c_date DATE, c_dt TIMESTAMP(6), c_v VARCHAR(200), c_text TEXT,
			c_blob BYTEA, c_json JSONB)`,
		cols: []string{"c_int", "c_big", "c_f64", "c_dec", "c_date", "c_dt", "c_v", "c_text", "c_blob", "c_json"},
		srcBulkSQL: `INSERT INTO ` + "`it_sync_hetero`" + ` (id, c_int, c_big, c_f64, c_dec, c_date, c_dt, c_v, c_text, c_blob, c_json)
			SELECT FLOOR(RAND()*2147483647), FLOOR(RAND()*9223372036854775807), RAND(), RAND()*1000000000,
			DATE_ADD('2000-01-01', INTERVAL FLOOR(RAND()*9000) DAY), DATE_ADD('2020-01-01', INTERVAL FLOOR(RAND()*100000) MINUTE),
			MD5(RAND()), REPEAT(MD5(RAND()), 10), UNHEX(MD5(RAND())), JSON_OBJECT('k', FLOOR(RAND()*1000), 'v', MD5(RAND()))
			FROM ` + "`it_sync_hetero`" + ` WHERE id = 3`,
		boundRows: [][]any{
			{"1", "-2147483648", "-9223372036854775808", "-1.5", "12345678901234.1234567890", "2024-02-29",
				"2024-06-15 08:30:05.123456", "single'quote\"double\\back", "emoji😀🎉多字节\nline2",
				"00ff7f", `{"b":true,"a":[1,2,"x'y"]}`},
			{"2", "2147483647", "9223372036854775807", "1e-10", "-0.0000000001", "1000-01-01",
				"2037-12-31 23:59:59.999999", strings.Repeat("x", 200), strings.Repeat("中", 234),
				"dead", `{"k":"v"}`},
		},
	},
}

func TestITDataSyncHeteroFull(t *testing.T) {
	for _, spec := range syncHeteroSpecs {
		spec := spec
		t.Run(spec.name, func(t *testing.T) {
			srcInfo := &dbi.DbInfo{Type: dbi.DbType(spec.srcType), Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"}
			tgtInfo := &dbi.DbInfo{Type: dbi.DbType(spec.tgtType), Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"}
			if spec.srcType == "postgres" {
				srcInfo = &dbi.DbInfo{Type: "postgres", Host: "127.0.0.1", Port: 5432, Username: "postgres", Password: "postgres", Database: "mayfly_pg_it"}
				tgtInfo = &dbi.DbInfo{Type: "mysql", Host: "127.0.0.1", Port: 3306, Username: "root", Password: "111049", Database: "mayfly_dbm_it"}
			}
			srcConn := conn(t, srcInfo)
			defer srcConn.Close()
			tgtConn := conn(t, tgtInfo)
			defer tgtConn.Close()

			mustExec(t, srcConn, "DROP TABLE IF EXISTS it_sync_hetero")
			mustExec(t, srcConn, spec.srcDDL)
			mustExec(t, tgtConn, "DROP TABLE IF EXISTS it_sync_hetero_t")
			mustExec(t, tgtConn, spec.tgtDDL)

			for _, r := range spec.boundRows {
				cols := append([]string{"id"}, spec.cols...)
				phs := make([]string, len(cols))
				for ci := range cols {
					if spec.srcType == "postgres" {
						phs[ci] = fmt.Sprintf("$%d", ci+1)
					} else {
						phs[ci] = "?"
					}
				}
				_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
					quote(srcConn, "it_sync_hetero"), strings.Join(cols, ", "), strings.Join(phs, ", ")), r...)
				require.NoError(t, err, "边界行写入失败: %v", r)
			}
			mustExec(t, srcConn, seedSQLFor(spec.srcType))

			if spec.srcType == "postgres" {
				mustExec(t, srcConn, spec.srcBulkSQL)
			} else {
				target := syncHeteroRows + 3
				maxID := 3
				const stepLimit = 4096
				colsCopy := "c_int, c_big, c_f64, c_dec, c_date, c_dt, c_v, c_text, c_blob, c_json"
				for maxID < target {
					step := maxID - 2
					if step > stepLimit {
						step = stepLimit
					}
					if step > target-maxID {
						step = target - maxID
					}
					mustExec(t, srcConn, fmt.Sprintf(
						"INSERT INTO `it_sync_hetero` (id, %s) SELECT id + %d, %s FROM `it_sync_hetero` WHERE id > %d AND id <= %d",
						colsCopy, step, colsCopy, maxID-step, maxID))
					maxID += step
				}
			}

			_, srcRes, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(srcConn, "it_sync_hetero")))
			require.NoError(t, err)
			require.Len(t, srcRes, syncHeteroRows+len(spec.boundRows)+1)

			targetColumns := make([]dbi.Column, 0, len(spec.cols)+1)
			targetColumns = append(targetColumns, dbi.Column{ColumnName: "id", DataType: "bigint", IsPrimaryKey: true})
			for _, c := range spec.cols {
				targetColumns = append(targetColumns, dbi.Column{ColumnName: c, DataType: syncTgtColType(spec.tgtType, c)})
			}
			fieldMap := make([]map[string]string, 0, len(spec.cols)+1)
			fieldMap = append(fieldMap, map[string]string{"src": "id", "target": "id"})
			for _, c := range spec.cols {
				fieldMap = append(fieldMap, map[string]string{"src": c, "target": c})
			}
			task := &entity.DataSyncTask{TargetTableName: "it_sync_hetero_t", DuplicateStrategy: 2}
			meta := sync.BuildTargetTableMeta(tgtConn, "it_sync_hetero_t", targetColumns)
			app, _ := newApp()
			require.NoError(t, app.SyncBatch(context.Background(), srcRes, fieldMap, "", task, tgtConn, targetColumns, meta), "[%s] 全量同步失败", spec.name)

			_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS c FROM %s", quote(tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			require.EqualValues(t, syncHeteroRows+len(spec.boundRows)+1, rows[0]["c"], "[%s] 同步后行数不一致", spec.name)

			_, srcRows, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s WHERE id <= %d ORDER BY id", quote(srcConn, "it_sync_hetero"), len(spec.boundRows)))
			require.NoError(t, err)
			_, tgtRows, err := tgtConn.Query(fmt.Sprintf("SELECT * FROM %s WHERE id <= %d ORDER BY id", quote(tgtConn, "it_sync_hetero_t"), len(spec.boundRows)))
			require.NoError(t, err)
			require.Len(t, tgtRows, len(srcRows))
			for i := range srcRows {
				for _, c := range spec.cols {
					assert.Equal(t, syncNorm(t, c, srcRows[i][c]), syncNorm(t, c, tgtRows[i][c]), "[%s] 边界行%d 列[%s] 同步后不一致", spec.name, i+1, c)
				}
			}

			if spec.srcType == "postgres" {
				mustExec(t, srcConn, "UPDATE it_sync_hetero SET c_v = 'updated-v' WHERE id = 1")
			} else {
				mustExec(t, srcConn, "UPDATE `it_sync_hetero` SET c_v = 'updated-v' WHERE id = 1")
			}
			_, srcRes2, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(srcConn, "it_sync_hetero")))
			require.NoError(t, err)
			require.NoError(t, app.SyncBatch(context.Background(), srcRes2, fieldMap, "", task, tgtConn, targetColumns, meta))
			_, rows2, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS c FROM %s", quote(tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			require.EqualValues(t, syncHeteroRows+len(spec.boundRows)+1, rows2[0]["c"], "[%s] 二次同步后行数变化（覆盖策略未生效）", spec.name)
			_, updated, err := tgtConn.Query(fmt.Sprintf("SELECT c_v FROM %s WHERE id = 1", quote(tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			assert.Equal(t, "updated-v", fmt.Sprint(updated[0]["c_v"]))

			_, _ = srcConn.Exec("DROP TABLE IF EXISTS it_sync_hetero")
			_, _ = tgtConn.Exec("DROP TABLE IF EXISTS it_sync_hetero_t")
		})
	}
}

func seedSQLFor(srcType string) string {
	if srcType == "postgres" {
		return `INSERT INTO it_sync_hetero (id, c_int, c_big, c_f64, c_dec, c_date, c_dt, c_v, c_text, c_bytea, c_bool, c_jsonb)
			VALUES (3, 42, 42, 0.5, 0.5, DATE '2024-02-29', TIMESTAMP '2024-06-15 08:30:05.123456', 'seed', 'seed text', decode('00ff', 'hex'), true, '{"seed":1}')`
	}
	return "INSERT INTO `it_sync_hetero` (id, c_int, c_big, c_f64, c_dec, c_date, c_dt, c_v, c_text, c_blob, c_json) " +
		"VALUES (3, 42, 42, 0.5, 0.5, '2024-02-29', '2024-06-15 08:30:05.123456', 'seed', 'seed text', X'00ff', '{\"seed\":1}')"
}

func syncTgtColType(tgtType, col string) string {
	if tgtType == "mysql" {
		m := map[string]string{"c_int": "int", "c_big": "bigint", "c_f64": "double", "c_dec": "decimal", "c_date": "date",
			"c_dt": "datetime", "c_v": "varchar", "c_text": "text", "c_bytea": "longblob", "c_bool": "tinyint", "c_jsonb": "json"}
		return m[col]
	}
	m := map[string]string{"c_int": "integer", "c_big": "bigint", "c_f64": "double precision", "c_dec": "numeric", "c_date": "date",
		"c_dt": "timestamp", "c_v": "varchar", "c_text": "text", "c_blob": "bytea", "c_json": "jsonb"}
	return m[col]
}

func syncNorm(t *testing.T, col string, v any) string {
	t.Helper()
	if v == nil {
		return "<NIL>"
	}
	switch col {
	case "c_bool":
		b, ok := value.ValToBool(v)
		require.True(t, ok, "c_bool无法归一: %v", v)
		if b {
			return "true"
		}
		return "false"
	case "c_jsonb", "c_json":
		var m any
		require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf("%v", v)), &m))
		b, err := json.Marshal(m)
		require.NoError(t, err)
		return string(b)
	case "c_f64", "c_dec":
		f, ok := value.ValToFloat64(v)
		if ok {
			return fmt.Sprintf("%v", f)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return fmt.Sprintf("%x", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
