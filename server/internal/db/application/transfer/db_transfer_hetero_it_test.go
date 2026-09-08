//go:build it

package transfer

// 异构数据迁移 端到端真实链路集成测试（全类型+大数据量，三方向）：
//
//	mysql→pg、pg→mysql、mysql→sqlite
//
// 真实链路：源库建表写数据（边界行+bulk行） → DumpDbScript(TargetDbType=目标方言)导出到文件
//（dump内部完成源列类型→目标列类型转换ConvToTargetDbColumn，并生成目标方言的DROP/CREATE/INSERT）
// → 目标库importDumpStream从文件导入 → 逐值比对（列级归一抹平方言读回形态差异）
//
// 类型映射矩阵：int族/float/double/decimal/numeric/date/datetime(6)/timestamp(6)/varchar/text/
// blob→bytea→longblob/json→jsonb/boolean→tinyint(1)
//
// 比对语义：两侧（源库读回/目标库读回）经同一列级归一函数处理；decimal统一按float最短表示
// （sqlite NUMERIC亲和会物化浮点，物理限制）；JSON规范化（mysql JSON与pg JSONB均重排键序）。
//
// 运行：cd server && go test -tags it -count=1 -timeout 30m -run TestITHeteroMigrate ./internal/db/application/transfer/

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

const (
	hetRows  = 50000 // bulk行数（~40MB/方向）
	hetTable = "it_hetero"
)

// hetSpecialStrings 异构迁移特殊字符串集
var hetSpecialStrings = []string{
	"single'quote", "double\"quote", "back\\slash", "back\\\\slash\\\\x",
	"line\nbreak", "crlf\r\nline", "tab\there", "nul-like-placeholder-not-0x00",
	"emoji😀🎉end", "中文混排English mixed 123", "semi;colon", "--sql-comment",
	"/*block*/comment", "back`tick", "dollar$sign", "%s%v%d",
}

// hetSpec 单方向异构迁移规格
type hetSpec struct {
	name  string // 方向名
	table string // 本方向使用的表名（同源方言的多方向并行时不得共用源表）
	// src 源方言信息
	srcDialect string
	srcDDL     string
	srcSeed    string // bulk种子行业务列表达式（不含id与c_pad）
	srcPad     string // 源bulk pad表达式
	srcPH      string // 源边界行参数占位符（"?"或"$1"，后者实际用fmt生成递增）
	// tgt 目标方言信息
	tgtDialect string
	// cols 源业务列清单（比对与生成共用）
	cols []string
	// boundaryRows 边界行Go值（顺序=cols），id为行号
	boundaryRows [][]any
}

var hetSpecs = []hetSpec{
	{
		name:       "mysql->pg",
		table:      "it_het_a",
		srcDialect: "mysql",
		srcDDL: "CREATE TABLE `it_hetero` (id BIGINT PRIMARY KEY, c_int INT, c_big BIGINT, c_f32 FLOAT, c_f64 DOUBLE, " +
			"c_dec DECIMAL(30,10), c_date DATE, c_dt DATETIME(6), c_v VARCHAR(200), c_text TEXT, " +
			"c_blob BLOB, c_json JSON, c_pad VARCHAR(1000)) DEFAULT CHARSET=utf8mb4",
		srcSeed: "FLOOR(RAND()*2147483647), FLOOR(RAND()*9223372036854775807), RAND(), RAND(), RAND()*1000000000, " +
			"DATE_ADD('2000-01-01', INTERVAL FLOOR(RAND()*9000) DAY), DATE_ADD('2020-01-01', INTERVAL FLOOR(RAND()*1000*60) MINUTE), " +
			"MD5(RAND()), REPEAT(MD5(RAND()), 10), UNHEX(MD5(RAND())), JSON_OBJECT('k', FLOOR(RAND()*1000), 'v', MD5(RAND()))",
		srcPad:     "REPEAT(MD5(RAND()), 22)",
		srcPH:      "?",
		tgtDialect: "postgres",
		cols:       []string{"c_int", "c_big", "c_f32", "c_f64", "c_dec", "c_date", "c_dt", "c_v", "c_text", "c_blob", "c_json", "c_pad"},
		boundaryRows: [][]any{
			{int64(-2147483648), int64(-9223372036854775808), -1.5, 3.141592653589793, "12345678901234.1234567890",
				"2024-02-29", "2024-06-15 08:30:05.123456",
				"single'quote\"double\\back", "emoji😀🎉多字节\nline2", []byte{0x00, 0xFF, 0x7F}, `{"b":true,"a":[1,2,"x'y"]}`, "pad-min"},
			{int64(2147483647), int64(9223372036854775807), 1e10, 1e-10, "-0.0000000001",
				"1000-01-01", "2037-12-31 23:59:59.999999",
				strings.Repeat("x", 200), strings.Repeat("中", 234), []byte{0xDE, 0xAD}, `{"k":"v"}`, "pad-max"},
			{int64(0), int64(0), 0.0, 0.0, "0",
				nil, nil,
				nil, nil, nil, nil, "nulls-mixed"},
		},
	},
	{
		name:       "pg->mysql",
		table:      "it_hetero",
		srcDialect: "postgres",
		srcDDL: `CREATE TABLE it_hetero (id BIGINT PRIMARY KEY, c_int INTEGER, c_big BIGINT, c_f32 REAL, c_f64 DOUBLE PRECISION,
			c_dec NUMERIC(30,10), c_date DATE, c_dt TIMESTAMP(6), c_v VARCHAR(200), c_text TEXT,
			c_bytea BYTEA, c_bool BOOLEAN, c_jsonb JSONB, c_pad VARCHAR(1000))`,
		srcSeed: `FLOOR(RANDOM()*2147483647)::int, FLOOR(RANDOM()*9223372036854775807)::bigint, RANDOM(), RANDOM(), RANDOM()*1000000000,
			DATE '2000-01-01' + FLOOR(RANDOM()*9000)::int, TIMESTAMP '2020-01-01' + (FLOOR(RANDOM()*1000*60)::int || ' minutes')::interval,
			md5(random()::text), repeat(md5(random()::text), 10), decode(md5(random()::text), 'hex'), RANDOM()>0.5, json_build_object('k', FLOOR(RANDOM()*1000)::int, 'v', md5(random()::text))`,
		srcPad:     "repeat(md5(random()::text), 22)",
		srcPH:      "$",
		tgtDialect: "mysql",
		cols:       []string{"c_int", "c_big", "c_f32", "c_f64", "c_dec", "c_date", "c_dt", "c_v", "c_text", "c_bytea", "c_bool", "c_jsonb", "c_pad"},
		boundaryRows: [][]any{
			{int64(-2147483648), int64(-9223372036854775808), -1.5, 3.141592653589793, "12345678901234.1234567890",
				"2024-02-29", "2024-06-15 08:30:05.123456",
				"single'quote\"double\\back", "emoji😀🎉多字节\nline2", []byte{0x00, 0xFF, 0x7F}, true, `{"b":true,"a":[1,2,"x'y"]}`, "pad-min"},
			{int64(2147483647), int64(9223372036854775807), 1e10, 1e-10, "-0.0000000001",
				"1000-01-01", "2037-12-31 23:59:59.999999",
				strings.Repeat("x", 200), strings.Repeat("中", 234), []byte{0xDE, 0xAD}, false, `{"k":"v"}`, "pad-max"},
			{int64(0), int64(0), 0.0, 0.0, "0",
				nil, nil,
				nil, nil, nil, nil, nil, "nulls-mixed"},
		},
	},
	{
		name:       "mysql->sqlite",
		table:      "it_het_b",
		srcDialect: "mysql",
		srcDDL: "CREATE TABLE `it_hetero` (id BIGINT PRIMARY KEY, c_int INT, c_big BIGINT, c_f32 FLOAT, c_f64 DOUBLE, " +
			"c_dec DECIMAL(30,10), c_date DATE, c_dt DATETIME(6), c_v VARCHAR(200), c_text TEXT, " +
			"c_blob BLOB, c_json JSON, c_pad VARCHAR(1000)) DEFAULT CHARSET=utf8mb4",
		srcSeed: "FLOOR(RAND()*2147483647), FLOOR(RAND()*9223372036854775807), RAND(), RAND(), RAND()*1000000000, " +
			"DATE_ADD('2000-01-01', INTERVAL FLOOR(RAND()*9000) DAY), DATE_ADD('2020-01-01', INTERVAL FLOOR(RAND()*1000*60) MINUTE), " +
			"MD5(RAND()), REPEAT(MD5(RAND()), 10), UNHEX(MD5(RAND())), JSON_OBJECT('k', FLOOR(RAND()*1000), 'v', MD5(RAND()))",
		srcPad:     "REPEAT(MD5(RAND()), 22)",
		srcPH:      "?",
		tgtDialect: "sqlite",
		cols:       []string{"c_int", "c_big", "c_f32", "c_f64", "c_dec", "c_date", "c_dt", "c_v", "c_text", "c_blob", "c_json", "c_pad"},
		boundaryRows: [][]any{
			{int64(-2147483648), int64(-9223372036854775808), -1.5, 3.141592653589793, "12345678901234.1234567890",
				"2024-02-29", "2024-06-15 08:30:05.123456",
				"single'quote\"double\\back", "emoji😀🎉多字节\nline2", []byte{0x00, 0xFF, 0x7F}, `{"b":true,"a":[1,2,"x'y"]}`, "pad-min"},
			{int64(2147483647), int64(9223372036854775807), 1e10, 1e-10, "-0.0000000001",
				"1000-01-01", "2037-12-31 23:59:59.999999",
				strings.Repeat("x", 200), strings.Repeat("中", 234), []byte{0xDE, 0xAD}, `{"k":"v"}`, "pad-max"},
			{int64(0), int64(0), 0.0, 0.0, "0",
				nil, nil,
				nil, nil, nil, nil, "nulls-mixed"},
		},
	},
}

// TestITHeteroMigrate 异构迁移三方向（全类型+大数据量）
func TestITHeteroMigrate(t *testing.T) {
	nodes := map[string]func(t *testing.T) *dbi.DbConn{
		"mysql":    itMysqlNode,
		"postgres": itPgNode,
		"sqlite":   itSqliteNode,
	}
	for _, spec := range hetSpecs {
		spec := spec
		t.Run(spec.name, func(t *testing.T) {
			t.Parallel()
			srcConn := nodes[spec.srcDialect](t)
			defer srcConn.Close()
			tgtConn := nodes[spec.tgtDialect](t)
			defer tgtConn.Close()

			// 1. 源库建表写数据
			hetLoadSrc(t, srcConn, &spec)

			// 2. 备份前源库快照
			before := hetSnapshot(t, srcConn, &spec)

			// 3. 异构dump：TargetDbType=目标方言（内部完成类型转换+目标方言DDL/INSERT生成）
			p := filepath.Join(t.TempDir(), "hetero.sql")
			bf, err := os.Create(p)
			require.NoError(t, err)
			bw := bufio.NewWriterSize(bf, 1<<20)
			require.NoError(t, DumpDbScript(context.Background(), srcConn, &dto.DumpDb{
				DbName:       srcConn.Info.Database,
				Tables:       []string{spec.table},
				DumpDDL:      true,
				DumpData:     true,
				TargetDbType: dbi.DbType(spec.tgtDialect),
				Writer:       bw,
			}), "[%s] dump失败", spec.name)
			require.NoError(t, bw.Flush())
			require.NoError(t, bf.Close())

			// 4. 目标库从文件导入（dump内含目标方言DROP/CREATE，直接导入即完成迁移）
			rf, err := os.Open(p)
			require.NoError(t, err)
			defer rf.Close()
			app := &DbTransferAppImpl{}
			require.NoError(t, app.importDumpStream(context.Background(), 0, tgtConn, bufio.NewReaderSize(rf, 1<<20)),
				"[%s] 目标库导入失败", spec.name)

			// 5. 比对
			after := hetSnapshot(t, tgtConn, &spec)
			require.EqualValues(t, before.cnt, after.cnt, "[%s] 迁移后行数不一致", spec.name)
			require.EqualValues(t, before.sumID, after.sumID, "[%s] SUM(id)不一致", spec.name)
			for id, beforeRow := range before.rows {
				afterRow, ok := after.rows[id]
				require.True(t, ok, "[%s] 迁移后缺少id=%d", spec.name, id)
				for _, col := range spec.cols {
					require.Equal(t, beforeRow[col], afterRow[col],
						"[%s] id=%d 列[%s] 异构迁移后不一致", spec.name, id, col)
				}
			}
		})
	}
}

// hetLoadSrc 源库建表+边界行+bulk行（mysql用复制法，pg用generate_series）
func hetLoadSrc(t *testing.T, conn *dbi.DbConn, spec *hetSpec) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	tbl := quote(spec.table)
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl))
	require.NoError(t, err)
	// srcDDL常量内表名为it_hetero，动态替换为本方向专属表名（同源方言多方向并行不共用源表）
	srcDDL := strings.ReplaceAll(spec.srcDDL, "it_hetero", spec.table)
	_, err = conn.Exec(srcDDL)
	require.NoError(t, err, "[%s] 源建表失败", spec.name)

	// 边界行参数化写入
	isPg := spec.srcDialect == "postgres"
	allCols := append([]string{"id"}, spec.cols...)
	for ri, row := range spec.boundaryRows {
		phs := make([]string, len(allCols))
		for ci := range allCols {
			if isPg {
				phs[ci] = fmt.Sprintf("$%d", ci+1)
			} else {
				phs[ci] = "?"
			}
		}
		args := append([]any{int64(ri + 1)}, row...)
		_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			tbl, strings.Join(allCols, ", "), strings.Join(phs, ", ")), args...)
		require.NoError(t, err, "[%s] 边界行%d写入失败", spec.name, ri+1)
	}

	// bulk行
	target := len(spec.boundaryRows) + hetRows
	seedID := len(spec.boundaryRows) + 1
	if spec.srcDialect == "postgres" {
		// pg单语句generate_series
		bizCols := hetPgSrcCols(spec)
		_, err = conn.Exec(fmt.Sprintf(
			"INSERT INTO %s (id, %s, c_pad) SELECT g, %s, %s FROM generate_series(%d, %d) g",
			tbl, strings.Join(bizCols, ", "), spec.srcSeed, spec.srcPad, seedID, target))
		require.NoError(t, err, "[%s] bulk生成失败", spec.name)
		return
	}
	// mysql：seed+翻倍复制（封顶4096行/轮，防单语句过大压崩容器）
	bizCols := hetMysqlSrcCols(spec)
	_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s (id, %s) VALUES (%d, %s)",
		tbl, strings.Join(bizCols, ", "), seedID, spec.srcSeed))
	require.NoError(t, err, "[%s] bulk seed失败", spec.name)
	_, err = conn.Exec(fmt.Sprintf("UPDATE %s SET c_pad = %s WHERE id = %d", tbl, spec.srcPad, seedID))
	require.NoError(t, err, "[%s] bulk seed pad失败", spec.name)
	bulkBase := len(spec.boundaryRows)
	maxID := seedID
	const stepLimit = 4096
	for maxID < target {
		step := maxID - bulkBase
		if step > stepLimit {
			step = stepLimit
		}
		if step > target-maxID {
			step = target - maxID
		}
		_, err := conn.Exec(fmt.Sprintf(
			"INSERT INTO %s (id, %s, c_pad) SELECT id + %d, %s, %s FROM %s WHERE id > %d AND id <= %d",
			tbl, strings.Join(bizCols, ", "), step, strings.Join(bizCols, ", "), spec.srcPad,
			tbl, maxID-step, maxID))
		require.NoError(t, err, "[%s] bulk复制失败 maxID=%d", spec.name, maxID)
		maxID += step
	}
}

// hetMysqlSrcCols mysql源业务列（排除c_pad，复制语句SELECT侧使用）
func hetMysqlSrcCols(spec *hetSpec) []string {
	names := make([]string, 0, len(spec.cols))
	for _, c := range spec.cols {
		if c != "c_pad" {
			names = append(names, c)
		}
	}
	return names
}

// hetPgSrcCols pg源业务列（generate_series路径同mysql复制路径使用）
func hetPgSrcCols(spec *hetSpec) []string {
	return hetMysqlSrcCols(spec)
}

// hetSnapshot 全表快照：COUNT + SUM(id) + 边界行+抽样行读回（列级归一）
type hetSnapshotResult struct {
	cnt   int64
	sumID int64
	rows  map[int64]map[string]string
}

func hetSnapshot(t *testing.T, conn *dbi.DbConn, spec *hetSpec) hetSnapshotResult {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf(
		"SELECT COUNT(*) AS c1, SUM(id) AS c2 FROM %s", quote(spec.table)))
	require.NoError(t, err)
	res := hetSnapshotResult{rows: map[int64]map[string]string{}}
	res.cnt = mtVal(t, rows[0]["c1"], "COUNT")
	res.sumID = mtVal(t, rows[0]["c2"], "SUM(id)")

	// 读回边界行与抽样行（中间行+末行）
	total := len(spec.boundaryRows) + hetRows
	ids := make([]int64, 0, len(spec.boundaryRows)+2)
	for ri := range spec.boundaryRows {
		ids = append(ids, int64(ri+1))
	}
	ids = append(ids, int64(total/2), int64(total))
	cols := append([]string{"id"}, spec.cols...)
	for _, id := range ids {
		_, rs, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = %d",
			strings.Join(cols, ", "), quote(spec.table), id))
		require.NoError(t, err, "[%s] 读回id=%d失败", spec.name, id)
		require.Len(t, rs, 1, "[%s] id=%d无行", spec.name, id)
		row := map[string]string{}
		for _, c := range spec.cols {
			row[c] = hetNorm(t, c, rs[0][c])
		}
		res.rows[id] = row
	}
	return res
}

// hetNorm 异构比对列级归一：抹平各库读回形态与类型语义差异。
// 两侧（源/目标）经同一函数处理后比对，避免写入侧精度/展示语义被误判为迁移损失
func hetNorm(t *testing.T, col string, v any) string {
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
	case "c_f32":
		f, ok := dbi.ValToFloat64(v)
		require.True(t, ok, "c_f32无法归一: %v", v)
		return fmt.Sprintf("%v", float32(f))
	case "c_f64":
		// pg float8走numeric文本valuer（如"0.0000000001"），mysql double读回float64（"1e-10"），
		// 同一值两种文本表示，统一ParseFloat后按最短表示比对
		f, ok := dbi.ValToFloat64(v)
		require.True(t, ok, "c_f64无法归一: %v", v)
		return fmt.Sprintf("%v", f)
	case "c_dec":
		// sqlite NUMERIC亲和会把decimal字面量物化为浮点（物理限制），统一按float最短表示
		f, ok := dbi.ValToFloat64(v)
		if ok {
			return fmt.Sprintf("%v", f)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	case "c_json", "c_jsonb":
		return hetJSONNorm(t, fmt.Sprintf("%v", v))
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return fmt.Sprintf("%x", x) // 二进制列读回[]byte时转hex（与ValuerBytes形态一致）
	default:
		return fmt.Sprintf("%v", x)
	}
}

// hetJSONNorm JSON规范化（mysql JSON与pg JSONB均重排键序/空白）
func hetJSONNorm(t *testing.T, s string) string {
	t.Helper()
	var m any
	require.NoError(t, json.Unmarshal([]byte(s), &m), "非法JSON: %.200s", s)
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return string(b)
}
