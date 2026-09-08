//go:build it

package sync

// 数据同步 全类型×异构方向×大数据量 端到端真实链路集成测试：
//
//	pg源→mysql目标（boolean/jsonb/bytea/timestamp等全类型映射为tinyint(1)/json/longblob/datetime）
//	mysql源→pg目标（json→jsonb、blob→bytea、datetime→timestamp等）
//	各20000行bulk + 边界行，DuplicateStrategy=覆盖，二次同步验证幂等（不产生重复行、值更新）
//
// 同步链路的值序列化与迁移共用GenInsert/SQLValue层，但upsert冲突处理（ON CONFLICT/
// replace into/insert or replace）为同步独有路径。
//
// 运行：cd server && go test -tags it -count=1 -timeout 20m -run TestITDataSyncHeteroFull ./internal/db/application/sync/

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
	"mayfly-go/internal/db/domain/entity"
)

const syncHeteroRows = 20000

type syncHeteroSpec struct {
	name       string
	srcType    string // "postgres"/"mysql"
	tgtType    string
	srcDDL     string
	tgtDDL     string
	cols       []string
	srcBulkSQL string // bulk生成语句（generate_series/递归）
	boundRows  [][]any
}

var syncHeteroSpecs = []syncHeteroSpec{
	{
		name:    "pg->mysql",
		srcType: "postgres",
		tgtType: "mysql",
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
		name:    "mysql->pg",
		srcType: "mysql",
		tgtType: "postgres",
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
			srcConn := syncTestConn(t, srcInfo)
			defer srcConn.Close()
			tgtConn := syncTestConn(t, tgtInfo)
			defer tgtConn.Close()

			mustSyncExec(t, srcConn, "DROP TABLE IF EXISTS it_sync_hetero")
			mustSyncExec(t, srcConn, spec.srcDDL)
			mustSyncExec(t, tgtConn, "DROP TABLE IF EXISTS it_sync_hetero_t")
			mustSyncExec(t, tgtConn, spec.tgtDDL)

			// 边界行（id 1..N）
			for _, r := range spec.boundRows {
				cols := append([]string{"id"}, spec.cols...)
				ph := "?"
				if spec.srcType == "postgres" {
					ph = "$1" // 用const占位，见下方手动构造
				}
				_ = ph
				phs := make([]string, len(cols))
				for ci := range cols {
					if spec.srcType == "postgres" {
						phs[ci] = fmt.Sprintf("$%d", ci+1)
					} else {
						phs[ci] = "?"
					}
				}
				_, err := srcConn.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
					quoteSync(t, srcConn, "it_sync_hetero"), strings.Join(cols, ", "), strings.Join(phs, ", ")), strToAnyA(r)...)
				require.NoError(t, err, "边界行写入失败: %v", r)
			}
			// bulk种子行 id=3（后续bulk以它为源复制/generate从4起）
			mustSyncExec(t, srcConn, seedSQLFor(spec.srcType))

			// bulk：pg源 generate_series 从4到20003；mysql源 从种子翻倍复制
			if spec.srcType == "postgres" {
				mustSyncExec(t, srcConn, spec.srcBulkSQL)
			} else {
				// mysql源：翻倍增长封顶4096行/轮（offset=step保证id无缝衔接）
				target := syncHeteroRows + 3
				maxID := 3 // 种子行id（bulk起点）
				const stepLimit = 4096
				colsCopy := "c_int, c_big, c_f64, c_dec, c_date, c_dt, c_v, c_text, c_blob, c_json"
				for maxID < target {
					// 现有bulk行数=maxID-(种子id-1)；step不得为0（否则空范围死循环）
					step := maxID - 2
					if step > stepLimit {
						step = stepLimit
					}
					if step > target-maxID {
						step = target - maxID
					}
					mustSyncExec(t, srcConn, fmt.Sprintf(
						"INSERT INTO `it_sync_hetero` (id, %s) SELECT id + %d, %s FROM `it_sync_hetero` WHERE id > %d AND id <= %d",
						colsCopy, step, colsCopy, maxID-step, maxID))
					maxID += step
				}
			}

			// 源快照
			_, srcRes, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id",
				quoteSync(t, srcConn, "it_sync_hetero")))
			require.NoError(t, err)
			require.Len(t, srcRes, syncHeteroRows+len(spec.boundRows)+1)

			targetColumns := make([]dbi.Column, 0, len(spec.cols)+1)
			targetColumns = append(targetColumns, dbi.Column{ColumnName: "id", DataType: "bigint", IsPrimaryKey: true})
			for _, c := range spec.cols {
				targetColumns = append(targetColumns, dbi.Column{ColumnName: c, DataType: syncTgtColType(spec.tgtType, c)})
			}
			targetMeta := dbi.BuildTargetTableMeta(tgtConn, "it_sync_hetero_t", targetColumns)

			fieldMap := make([]map[string]string, 0, len(spec.cols)+1)
			fieldMap = append(fieldMap, map[string]string{"src": "id", "target": "id"})
			for _, c := range spec.cols {
				fieldMap = append(fieldMap, map[string]string{"src": c, "target": c})
			}
			task := &entity.DataSyncTask{TargetTableName: "it_sync_hetero_t", DuplicateStrategy: 2}

			app := &DataSyncAppImpl{}
			require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes, fieldMap, "", task, tgtConn, targetColumns, targetMeta),
				"[%s] 全量同步失败", spec.name)

			// 目标COUNT
			_, rows, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS c FROM %s", quoteSync(t, tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			require.EqualValues(t, syncHeteroRows+len(spec.boundRows)+1, rows[0]["c"], "[%s] 同步后行数不一致", spec.name)

			// 边界行值比对（源读回 vs 目标读回，列级归一）
			_, srcRows, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s WHERE id <= %d ORDER BY id",
				quoteSync(t, srcConn, "it_sync_hetero"), len(spec.boundRows)))
			require.NoError(t, err)
			_, tgtRows, err := tgtConn.Query(fmt.Sprintf("SELECT * FROM %s WHERE id <= %d ORDER BY id",
				quoteSync(t, tgtConn, "it_sync_hetero_t"), len(spec.boundRows)))
			require.NoError(t, err)
			require.Len(t, tgtRows, len(srcRows))
			for i := range srcRows {
				for _, c := range spec.cols {
					assert.Equal(t, syncNorm(t, c, srcRows[i][c]), syncNorm(t, c, tgtRows[i][c]),
						"[%s] 边界行%d 列[%s] 同步后不一致", spec.name, i+1, c)
				}
			}

			// 二次同步幂等：更新源一行后重同步，覆盖不重复
			if spec.srcType == "postgres" {
				mustSyncExec(t, srcConn, "UPDATE it_sync_hetero SET c_v = 'updated-v' WHERE id = 1")
			} else {
				mustSyncExec(t, srcConn, "UPDATE `it_sync_hetero` SET c_v = 'updated-v' WHERE id = 1")
			}
			_, srcRes2, err := srcConn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id",
				quoteSync(t, srcConn, "it_sync_hetero")))
			require.NoError(t, err)
			require.NoError(t, app.srcData2TargetDb(context.Background(), srcRes2, fieldMap, "", task, tgtConn, targetColumns, targetMeta))
			_, rows2, err := tgtConn.Query(fmt.Sprintf("SELECT COUNT(*) AS c FROM %s", quoteSync(t, tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			require.EqualValues(t, syncHeteroRows+len(spec.boundRows)+1, rows2[0]["c"], "[%s] 二次同步后行数变化（覆盖策略未生效）", spec.name)
			_, updated, err := tgtConn.Query(fmt.Sprintf("SELECT c_v FROM %s WHERE id = 1", quoteSync(t, tgtConn, "it_sync_hetero_t")))
			require.NoError(t, err)
			assert.Equal(t, "updated-v", fmt.Sprint(updated[0]["c_v"]), "[%s] 二次同步后值应更新", spec.name)

			// 清理
			_, _ = srcConn.Exec("DROP TABLE IF EXISTS it_sync_hetero")
			_, _ = tgtConn.Exec("DROP TABLE IF EXISTS it_sync_hetero_t")
		})
	}
}

func quoteSync(t *testing.T, conn *dbi.DbConn, name string) string {
	return conn.GetDialect().Quoter().QuoteIdent(name)
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

// syncNorm 同步比对列级归一
func syncNorm(t *testing.T, col string, v any) string {
	t.Helper()
	if v == nil {
		return "<NIL>"
	}
	switch col {
	case "c_bool":
		b, ok := dbi.ValToBool(v)
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
		f, ok := dbi.ValToFloat64(v)
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

func strToAnyA(ss []any) []any {
	return ss
}
