//go:build it

package transfer

// 多表×多字段类型 大数据量"备份恢复"端到端真实链路集成测试（总量≈1GB，均匀分配到8张表）：
//
//	8张表各自拥有丰富的字段类型组合（整数族/浮点与高精度decimal/时间族/字符串族/大文本/
//	二进制族/JSON与逻辑位/NULL混合），每表附c_pad文本列使行宽可控在~760B，表大小均匀~100MB
//	→ 每表id 1..N边界值行（类型极值+复杂字符串，参数化写入）+ 服务端生成bulk行（种子+步长复制）
//	→ 逐表DumpDbScript流式导出到真实.sql文件 → 聚合快照留存 → DROP源表
//	→ importDumpStream从文件恢复 → 逐表聚合精确比对 + 边界行全量回读比对 + 抽样一致性比对
//
// 类型覆盖矩阵（per方言DDL见mtSpecs）：
//	mysql: tinyint~bigint(含unsigned)/float/double/decimal/date/datetime(6)/timestamp(6)/
//	       time(3)/year/char/varchar/text/mediumtext/longtext/varbinary/tinyblob/blob/
//	       mediumblob/longblob/bit/json/enum/set
//	pg:    int2/int4/int8/real/float8/numeric/date/time(6)/timestamp(6)/bpchar/varchar/
//	       text/bytea/boolean/json/jsonb
//	sqlite: integer/real/text/blob（弱类型亲和）
//
// 运行：cd server && go test -tags it -count=1 -timeout 40m -run TestITMultiTableBigData ./internal/db/application/transfer/

import (
	"bufio"
	"context"
	"encoding/hex"
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

// mtBigTableSpec 单张多类型大表规格
type mtBigTableSpec struct {
	name string
	rows int // bulk行数
	// ddl 各方言建表语句（id列统一首列，均含c_pad列；charset由mtCreateAndLoad按方言拼接）
	ddl map[string]string
	// seedExprs 各方言bulk种子行业务列表达式（列顺序=boundaryCols中非c_pad列顺序）
	seedExprs map[string]string
	// 边界行：boundaryCols为列清单（含c_pad），boundaryRows为对应Go值（id 1..N）
	boundaryCols []string
	boundaryRows [][]any
	// 参与SUM(LENGTH)聚合校验的列
	aggCols []string
}

var mtSpecialStrings = []string{
	"single'quote", "double\"quote", "back\\slash", "back\\\\slash\\\\x",
	"line\nbreak", "crlf\r\nline", "tab\there", "nul-like-placeholder-not-0x00",
	"emoji😀🎉end", "中文混排English mixed 123", "semi;colon", "--sql-comment",
	"/*block*/comment", "back`tick", "dollar$sign", "%s%v%d",
	"  leading and trailing  ",
	strings.Repeat("x", 700),
	strings.Repeat("田", 233),
	strings.Repeat("q'\\\" mixed", 60),
}

// mtPadExpr 各方言生成~700B pad文本的bulk表达式
var mtPadExpr = map[string]string{
	"mysql":    "REPEAT(MD5(RAND()), 22)",
	"postgres": "repeat(md5(random()::text), 22)",
	"sqlite":   "lower(hex(randomblob(350)))",
}

var mtSpecs = []mtBigTableSpec{
	{
		name: "mt_int_types",
		rows: 140000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_tiny TINYINT, c_small SMALLINT, c_med MEDIUMINT, c_int INT, c_big BIGINT, c_utiny TINYINT UNSIGNED, c_uint INT UNSIGNED, c_ubig BIGINT UNSIGNED, c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_tiny SMALLINT, c_small SMALLINT, c_med INTEGER, c_int INTEGER, c_big BIGINT, c_utiny SMALLINT, c_uint BIGINT, c_ubig BIGINT, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_tiny INTEGER, c_small INTEGER, c_med INTEGER, c_int INTEGER, c_big INTEGER, c_utiny INTEGER, c_uint INTEGER, c_ubig INTEGER, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "FLOOR(RAND()*127), FLOOR(RAND()*32767), FLOOR(RAND()*8388607), FLOOR(RAND()*2147483647), FLOOR(RAND()*9223372036854775807), FLOOR(RAND()*255), FLOOR(RAND()*4294967295), FLOOR(RAND()*1000000000)",
			"postgres": "FLOOR(RANDOM()*127)::int, FLOOR(RANDOM()*32767)::int, FLOOR(RANDOM()*8388607)::int, FLOOR(RANDOM()*2147483647)::int, FLOOR(RANDOM()*9223372036854775807)::bigint, FLOOR(RANDOM()*255)::int, FLOOR(RANDOM()*4294967295)::bigint, FLOOR(RANDOM()*1000000000)::bigint",
			"sqlite":   "abs(random())%127, abs(random())%32767, abs(random())%8388607, abs(random())%2147483647, abs(random())%9223372036854775807, abs(random())%255, abs(random())%4294967295, abs(random())%1000000000",
		},
		boundaryCols: []string{"c_tiny", "c_small", "c_med", "c_int", "c_big", "c_utiny", "c_uint", "c_ubig", "c_pad"},
		boundaryRows: [][]any{
			{int64(-128), int64(-32768), int64(-8388608), int64(-2147483648), int64(-9223372036854775808), int64(0), int64(0), int64(0), "min-values"},
			{int64(127), int64(32767), int64(8388607), int64(2147483647), int64(9223372036854775807), int64(255), int64(4294967295), int64(9223372036854775807), "max-values"},
			{int64(0), int64(-1), int64(1), int64(-1), int64(0), int64(1), int64(1), int64(1), "zeros-ones"},
		},
		aggCols: []string{"c_pad"},
	},
	{
		name: "mt_float_decimal",
		rows: 140000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_f32 FLOAT, c_f64 DOUBLE, c_dec36 DECIMAL(36,10), c_dec10 DECIMAL(10,3), c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_f32 REAL, c_f64 DOUBLE PRECISION, c_dec36 NUMERIC(36,10), c_dec10 NUMERIC(10,3), c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_f32 REAL, c_f64 REAL, c_dec36 NUMERIC, c_dec10 NUMERIC, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "RAND(), RAND(), RAND()*1000000000, RAND()*1000",
			"postgres": "RANDOM(), RANDOM(), RANDOM()*1000000000, RANDOM()*1000",
			"sqlite":   "abs(random())/9223372036854775807.0, abs(random())/9223372036854775807.0, abs(random())%1000000000, abs(random())%1000",
		},
		boundaryCols: []string{"c_f32", "c_f64", "c_dec36", "c_dec10", "c_pad"},
		boundaryRows: [][]any{
			{0.1, 0.1, "12345678901234.1234567890", "-999.999", "float-frac"},
			{-123456.789, 3.141592653589793, "-0.0000000001", "0.001", "float-neg"},
			{1e-10, 1e-10, "99999999999999.9999999999", "0.000", "float-exp"},
		},
		aggCols: []string{"c_pad"},
	},
	{
		name: "mt_time_types",
		rows: 140000,
		ddl: map[string]string{
			// mysql TIMESTAMP上限2038，boundary的c_ts值统一取2037内；c_dt DATETIME(6)另覆盖datetime
			"mysql":    "(id BIGINT PRIMARY KEY, c_date DATE, c_dt DATETIME(6), c_ts TIMESTAMP(6) NULL, c_time TIME(3), c_year YEAR, c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_date DATE, c_dt TIMESTAMP(6), c_ts TIMESTAMP(6), c_time TIME(3), c_year SMALLINT, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_date TEXT, c_dt TEXT, c_ts TEXT, c_time TEXT, c_year INTEGER, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "DATE_ADD('2000-01-01', INTERVAL FLOOR(RAND()*9000) DAY), DATE_ADD('2020-01-01', INTERVAL FLOOR(RAND()*2000*60) MINUTE), CURRENT_TIMESTAMP(6), SEC_TO_TIME(FLOOR(RAND()*86400)), FLOOR(1990+RAND()*30)",
			"postgres": "(DATE '2000-01-01' + FLOOR(RANDOM()*9000)::int), TIMESTAMP '2020-01-01' + (FLOOR(RANDOM()*2000*60)::int || ' minutes')::interval, now(), (FLOOR(RANDOM()*86400)::int || ' seconds')::interval, 1990+FLOOR(RANDOM()*30)::int",
			"sqlite":   "date('2000-01-01', '+' || (abs(random())%9000) || ' days'), datetime('2020-01-01', '+' || (abs(random())%120000) || ' minutes'), datetime('now'), time('00:00', '+' || (abs(random())%86400) || ' seconds'), 1990+abs(random())%30",
		},
		boundaryCols: []string{"c_date", "c_dt", "c_ts", "c_time", "c_year", "c_pad"},
		boundaryRows: [][]any{
			{"2024-02-29", "2024-06-15 08:30:05.123456", "2037-12-31 23:59:59.999999", "23:59:59.999", int64(2024), "leap-micro9"},
			{"1000-01-01", "1970-01-01 00:00:00.000001", "2024-02-29 12:00:00.000000", "00:00:00.001", int64(1901), "min-micro1"},
			{nil, nil, nil, nil, nil, "all-null-time"},
		},
		aggCols: []string{"c_pad"},
	},
	{
		name: "mt_string_types",
		rows: 40000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_char CHAR(36), c_v255 VARCHAR(255), c_v2000 VARCHAR(2000), c_text TEXT, c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_char CHAR(36), c_v255 VARCHAR(255), c_v2000 VARCHAR(2000), c_text TEXT, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_char TEXT, c_v255 TEXT, c_v2000 TEXT, c_text TEXT, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "REPEAT(MD5(RAND()), 1), REPEAT(MD5(RAND()), 7), REPEAT(MD5(RAND()), 62), REPEAT(MD5(RAND()), 62)",
			"postgres": "repeat(md5(random()::text), 1), repeat(md5(random()::text), 7), repeat(md5(random()::text), 62), repeat(md5(random()::text), 62)",
			"sqlite":   "lower(hex(randomblob(16))), lower(hex(randomblob(128))), lower(hex(randomblob(1000))), lower(hex(randomblob(1000)))",
		},
		boundaryCols: []string{"c_char", "c_v255", "c_v2000", "c_text", "c_pad"},
		boundaryRows: mtStringBoundaryRows(),
		aggCols:      []string{"c_v2000", "c_text", "c_pad"},
	},
	{
		name: "mt_big_text",
		rows: 9000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_med MEDIUMTEXT, c_long LONGTEXT, c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_med TEXT, c_long TEXT, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_med TEXT, c_long TEXT, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "REPEAT(MD5(RAND()), 400), REPEAT(MD5(RAND()), 400)",
			"postgres": "repeat(md5(random()::text), 400), repeat(md5(random()::text), 400)",
			"sqlite":   "lower(hex(randomblob(6000))), lower(hex(randomblob(6000)))",
		},
		boundaryCols: []string{"c_med", "c_long", "c_pad"},
		boundaryRows: [][]any{
			{strings.Repeat("中", 4000), strings.Repeat("e😀", 2000), "bigtext-multibyte"},
			{strings.Repeat("a'b\\c\n", 500), strings.Repeat("\"';--\t", 500), "bigtext-special"},
			{nil, strings.Repeat("x", 20000), "bigtext-null"},
		},
		aggCols: []string{"c_med", "c_long", "c_pad"},
	},
	{
		name: "mt_binary",
		rows: 50000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_bin VARBINARY(16), c_varbin VARBINARY(255), c_tblob TINYBLOB, c_blob BLOB, c_mblob MEDIUMBLOB, c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_bin BYTEA, c_varbin BYTEA, c_tblob BYTEA, c_blob BYTEA, c_mblob BYTEA, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_bin BLOB, c_varbin BLOB, c_tblob BLOB, c_blob BLOB, c_mblob BLOB, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "UNHEX(REPEAT('a1', 8)), UNHEX(REPEAT('b2', 64)), UNHEX(REPEAT('c3', 64)), UNHEX(REPEAT('d4', 1024)), UNHEX(REPEAT('e5', 1024))",
			"postgres": "decode(repeat('a1', 8), 'hex'), decode(repeat('b2', 64), 'hex'), decode(repeat('c3', 64), 'hex'), decode(repeat('d4', 1024), 'hex'), decode(repeat('e5', 1024), 'hex')",
			"sqlite":   "randomblob(8), randomblob(64), randomblob(64), randomblob(1024), randomblob(1024)",
		},
		boundaryCols: []string{"c_bin", "c_varbin", "c_tblob", "c_blob", "c_mblob", "c_pad"},
		boundaryRows: [][]any{
			{[]byte{0x00}, []byte{0x00, 0xFF}, []byte{0x80}, []byte{0x00, 0x01, 0xFE, 0xFF}, []byte{0xFF, 0x00}, "bin-0x00-pair"},
			{[]byte("\x7fELF-mock"), []byte{0xCA, 0xFE, 0xBA, 0xBE}, []byte{0xDE, 0xAD}, []byte{0xBE, 0xEF}, []byte{0x01}, "bin-magic"},
			{nil, nil, nil, nil, nil, "bin-all-null"},
		},
		aggCols: []string{"c_pad"},
	},
	{
		name: "mt_json_misc",
		rows: 140000,
		ddl: map[string]string{
			// bit类型仅mysql承载（pg未注册bit的dump映射）；enum/set为mysql特有，其余方言以varchar承载
			"mysql":    "(id BIGINT PRIMARY KEY, c_json JSON, c_bit BIT(8), c_bool TINYINT(1), c_enum ENUM('a','b','c'), c_set SET('x','y','z'), c_pad VARCHAR(1000))",
			"postgres": "(id BIGINT PRIMARY KEY, c_json JSONB, c_bit SMALLINT, c_bool BOOLEAN, c_enum VARCHAR(10), c_set VARCHAR(20), c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_json TEXT, c_bit INTEGER, c_bool INTEGER, c_enum TEXT, c_set TEXT, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "JSON_OBJECT('k', FLOOR(RAND()*1000), 'v', MD5(RAND())), FLOOR(RAND()*255), RAND()>0.5, ELT(FLOOR(1+RAND()*3),'a','b','c'), IF(RAND()>0.5, ELT(FLOOR(1+RAND()*3),'x','y','z'), '')",
			"postgres": "json_build_object('k', FLOOR(RANDOM()*1000)::int, 'v', md5(random()::text)), FLOOR(RANDOM()*255)::int, RANDOM()>0.5, (ARRAY['a','b','c'])[FLOOR(1+RANDOM()*3)::int], FLOOR(RANDOM()*7)::int::text",
			"sqlite":   "json_object('k', abs(random())%1000, 'v', lower(hex(randomblob(8)))), abs(random())%255, abs(random())%2, case abs(random())%3 when 0 then 'a' when 1 then 'b' else 'c' end, abs(random())%7",
		},
		boundaryCols: []string{"c_json", "c_bit", "c_bool", "c_enum", "c_set", "c_pad"},
		boundaryRows: [][]any{
			{`{"e":"😀","msg":"it's \"quoted\"","note":"a\nb","path":"C:\\tmp"}`, int64(255), true, "a", "x,y,z", "json-special"},
			{`{"arr":[1,2,"x'y"],"b":true,"nested":{"k":"v"}}`, int64(0), false, "c", "x", "json-nested"},
			{nil, nil, nil, nil, nil, "json-null"},
		},
		aggCols: []string{"c_pad"},
	},
	{
		name: "mt_null_mixed",
		rows: 140000,
		ddl: map[string]string{
			"mysql":    "(id BIGINT PRIMARY KEY, c_int INT NULL, c_f64 DOUBLE NULL, c_dec DECIMAL(20,6) NULL, c_dt DATETIME(6) NULL, c_v VARCHAR(500) NULL, c_text TEXT NULL, c_blob BLOB NULL, c_pad VARCHAR(1000) NULL)",
			"postgres": "(id BIGINT PRIMARY KEY, c_int INTEGER, c_f64 DOUBLE PRECISION, c_dec NUMERIC(20,6), c_dt TIMESTAMP(6), c_v VARCHAR(500), c_text TEXT, c_blob BYTEA, c_pad VARCHAR(1000))",
			"sqlite":   "(id INTEGER PRIMARY KEY, c_int INTEGER, c_f64 REAL, c_dec NUMERIC, c_dt TEXT, c_v TEXT, c_text TEXT, c_blob BLOB, c_pad TEXT)",
		},
		seedExprs: map[string]string{
			"mysql":    "NULL, IF(RAND()>0.5, RAND(), NULL), IF(RAND()>0.5, RAND()*1e6, NULL), IF(RAND()>0.5, NOW(6), NULL), IF(RAND()>0.5, MD5(RAND()), NULL), IF(RAND()>0.5, MD5(RAND()), NULL), IF(RAND()>0.5, UNHEX(MD5(RAND())), NULL)",
			"postgres": "NULL, CASE WHEN RANDOM()>0.5 THEN RANDOM() END, CASE WHEN RANDOM()>0.5 THEN RANDOM()*1e6 END, CASE WHEN RANDOM()>0.5 THEN now() END, CASE WHEN RANDOM()>0.5 THEN md5(random()::text) END, CASE WHEN RANDOM()>0.5 THEN md5(random()::text) END, CASE WHEN RANDOM()>0.5 THEN decode(md5(random()::text),'hex') END",
			"sqlite":   "NULL, case when abs(random())%2=0 then abs(random())/9223372036854775807.0 end, case when abs(random())%2=0 then abs(random())%1000000 end, case when abs(random())%2=0 then datetime('now') end, case when abs(random())%2=0 then lower(hex(randomblob(8))) end, case when abs(random())%2=0 then lower(hex(randomblob(8))) end, case when abs(random())%2=0 then randomblob(8) end",
		},
		boundaryCols: []string{"c_int", "c_f64", "c_dec", "c_dt", "c_v", "c_text", "c_blob", "c_pad"},
		boundaryRows: [][]any{
			{int64(42), 3.14, "12345.678901", "2024-06-15 08:30:05.123456", "some varchar", "some text", []byte{0x01, 0x02}, "all-set"},
			{nil, nil, nil, nil, nil, nil, nil, "all-null"},
		},
		aggCols: []string{"c_pad"},
	},
}

func mtStringBoundaryRows() [][]any {
	rows := make([][]any, 0, len(mtSpecialStrings))
	for i, s := range mtSpecialStrings {
		v255 := mtTruncateRunes(strings.Repeat(s+"|", 40), 255)
		v2000 := mtTruncateRunes(strings.Repeat(s+"\n", 100), 2000)
		rows = append(rows, []any{fmt.Sprintf("char-%02d", i), v255, v2000, s + " /// " + s, s})
	}
	return rows
}

// mtTruncateRunes 按**字符**（rune）截断到n个字符：varchar(255)等长度语义是字符数而非字节数，
// 按字节切会在多字节字符（emoji/中文）中间截断产生非法UTF-8（实测pg报invalid byte sequence）
func mtTruncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// TestITMultiTableBigData 多表×多类型大数据量备份恢复roundtrip（三方言并行）
func TestITMultiTableBigData(t *testing.T) {
	nodes := []struct {
		name string
		conn func(t *testing.T) *dbi.DbConn
	}{
		{"mysql", itMysqlNode},
		{"pg", itPgNode},
		{"sqlite", itSqliteNode},
	}
	for _, n := range nodes {
		n := n
		t.Run(n.name, func(t *testing.T) {
			t.Parallel()
			conn := n.conn(t)
			defer conn.Close()
			quote := conn.GetDialect().Quoter().QuoteIdent
			dialect := string(conn.Info.Type)

			for i := range mtSpecs {
				mtCreateAndLoad(t, conn, &mtSpecs[i], dialect)
			}

			backupDir := t.TempDir()
			totalBytes := int64(0)
			for i := range mtSpecs {
				spec := &mtSpecs[i]
				before := mtAgg(t, conn, spec)

				p := filepath.Join(backupDir, fmt.Sprintf("%s.sql", spec.name))
				mtDump(t, conn, spec, p)
				st, err := os.Stat(p)
				require.NoError(t, err)
				totalBytes += st.Size()
				t.Logf("[%s][%s] dump=%dMB", dialect, spec.name, st.Size()>>20)

				// 每表独立"毁库+恢复"（表粒度备份恢复语义）
				_, err = conn.Exec(fmt.Sprintf("DROP TABLE %s", quote(spec.name)))
				require.NoError(t, err)
				mtRestore(t, conn, p)

				after := mtAgg(t, conn, spec)
				require.EqualValues(t, before.cnt, after.cnt, "[%s] 恢复后行数不一致", spec.name)
				require.EqualValues(t, before.sumID, after.sumID, "[%s] SUM(id)不一致", spec.name)
				for col, beforeLen := range before.lengths {
					require.EqualValues(t, beforeLen, after.lengths[col],
						"[%s] 列[%s] SUM(LENGTH)不一致（内容截断/改写）", spec.name, col)
				}
				mtCheckBoundary(t, conn, spec, before.boundarySnap)
				mtCheckSample(t, conn, spec, before.sample)
			}
			t.Logf("[%s] 总备份体积≈%dMB", dialect, totalBytes>>20)
			require.Greater(t, totalBytes, int64(700<<20), "[%s] 总备份体积异常偏小（数据量不足或静默截断）", dialect)
		})
	}
}

// mtCreateAndLoad 建表+边界行（参数化）+bulk行（种子+步长复制）
func mtCreateAndLoad(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec, dialect string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	tbl := quote(spec.name)
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tbl))
	require.NoError(t, err)
	ddl := "CREATE TABLE " + tbl + " " + spec.ddl[dialect]
	if dialect == "mysql" {
		// 复杂字符串含emoji等多字节字符，mysql表必须显式utf8mb4（否则默认utf8mb3写入即报1366）
		ddl += " DEFAULT CHARSET=utf8mb4"
	}
	_, err = conn.Exec(ddl)
	require.NoError(t, err, "[%s] 建表失败", spec.name)

	// 边界行（id 1..N）参数化写入：复杂字符串与类型极值。
	// 注意pg占位符必须递增$1..$N（同号参数会被推断为不同类型报inconsistent types deduced）
	ph := "?"
	if dialect == "postgres" {
		ph = "$%d"
	}
	allCols := append([]string{"id"}, spec.boundaryCols...)
	for ri, row := range spec.boundaryRows {
		phs := make([]string, len(allCols))
		for ci := range allCols {
			if dialect == "postgres" {
				phs[ci] = fmt.Sprintf(ph, ci+1)
			} else {
				phs[ci] = ph
			}
		}
		args := append([]any{int64(ri + 1)}, row...)
		_, err := conn.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			tbl, strings.Join(allCols, ", "), strings.Join(phs, ", ")), args...)
		require.NoError(t, err, "[%s] 边界行%d写入失败", spec.name, ri+1)
	}

	// bulk行：种子1行业务列随机表达式，pad单独UPDATE（避免列序对齐错误）
	bizCols := mtBizCols(spec)
	seedID := len(spec.boundaryRows) + 1
	_, err = conn.Exec(fmt.Sprintf("INSERT INTO %s (id, %s) VALUES (%d, %s)",
		tbl, strings.Join(bizCols, ", "), seedID, spec.seedExprs[dialect]))
	require.NoError(t, err, "[%s] bulk seed失败", spec.name)
	_, err = conn.Exec(fmt.Sprintf("UPDATE %s SET c_pad = %s WHERE id = %d", tbl, mtPadExpr[dialect], seedID))
	require.NoError(t, err, "[%s] bulk seed pad失败", spec.name)

	// 步长复制：翻倍增长至stepLimit封顶，之后每轮固定复制最近4096行。
	// 不变量：每轮offset=step=本轮复制行数，且 (maxID-step, maxID] 区间内行全部连续存在，
	// 新id区间恰好衔接 (maxID, maxID+step]（内容每行重新随机，不会与源行重复）。
	// 单条INSERT..SELECT过大（500MB级）实测会压崩资源受限的mysql容器，故封顶4096行（≈31MB）
	target := len(spec.boundaryRows) + spec.rows
	bulkBase := len(spec.boundaryRows)
	maxID := seedID
	const stepLimit = 4096
	for maxID < target {
		step := maxID - bulkBase // 现有bulk行数（翻倍阶段）
		if step > stepLimit {
			step = stepLimit
		}
		if step > target-maxID {
			step = target - maxID
		}
		_, err := conn.Exec(fmt.Sprintf(
			"INSERT INTO %s (id, %s, c_pad) SELECT id + %d, %s, %s FROM %s WHERE id > %d AND id <= %d",
			tbl, strings.Join(bizCols, ", "), step, strings.Join(bizCols, ", "), mtPadExpr[dialect],
			tbl, maxID-step, maxID))
		require.NoError(t, err, "[%s] bulk复制失败 maxID=%d", spec.name, maxID)
		maxID += step
	}
}

// mtBizCols 业务列（boundaryCols去掉c_pad）
func mtBizCols(spec *mtBigTableSpec) []string {
	names := make([]string, 0, len(spec.boundaryCols))
	for _, c := range spec.boundaryCols {
		if c != "c_pad" {
			names = append(names, c)
		}
	}
	return names
}

// mtAgg 聚合快照：COUNT + SUM(id) + 各aggCol的SUM(LENGTH) + 边界行快照 + 抽样集
type mtAggResult struct {
	cnt          int64
	sumID        int64
	lengths      map[string]int64
	boundarySnap map[int64]map[string]string
	sample       map[int64]map[string]string
}

func mtAgg(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec) mtAggResult {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	lengthExprs := make([]string, 0, len(spec.aggCols))
	for _, c := range spec.aggCols {
		lengthExprs = append(lengthExprs, fmt.Sprintf("SUM(LENGTH(%s)) AS len_%s", c, c))
	}
	_, rows, err := conn.Query(fmt.Sprintf("SELECT COUNT(*) AS c1, SUM(id) AS c2, %s FROM %s",
		strings.Join(lengthExprs, ", "), quote(spec.name)))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	res := mtAggResult{lengths: map[string]int64{}}
	res.cnt = mtVal(t, rows[0]["c1"], "COUNT")
	res.sumID = mtVal(t, rows[0]["c2"], "SUM(id)")
	for _, c := range spec.aggCols {
		res.lengths[c] = mtVal(t, rows[0]["len_"+c], "SUM(LENGTH:"+c+")")
	}
	res.sample = mtSample(t, conn, spec)
	res.boundarySnap = mtBoundarySnap(t, conn, spec)
	return res
}

// mtBoundarySnap 边界行读回快照：以"导出前源库读值"为基准而非Go侧硬编码写入值——
// FLOAT列写入本身就有精度损失、时间列.999999 layout会去除末尾零，这些是写入/展示语义
// 而非备份恢复损失；备份保真的正确语义是恢复后读值==备份前读值
func mtBoundarySnap(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec) map[int64]map[string]string {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	cols := append([]string{"id"}, spec.boundaryCols...)
	m := make(map[int64]map[string]string, len(spec.boundaryRows))
	for ri := range spec.boundaryRows {
		id := int64(ri + 1)
		_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = %d",
			strings.Join(cols, ", "), quote(spec.name), id))
		require.NoError(t, err, "[%s] 边界行%d读回失败", spec.name, id)
		require.Len(t, rows, 1, "[%s] 边界行id=%d缺失", spec.name, id)
		row := map[string]string{}
		for _, c := range spec.boundaryCols {
			row[c] = mtNormCell(t, c, rows[0][c])
		}
		m[id] = row
	}
	return m
}

// mtSample 抽样3行（种子行、中间行、末行）全部业务列内容
func mtSample(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec) map[int64]map[string]string {
	t.Helper()
	total := len(spec.boundaryRows) + spec.rows
	ids := []int64{int64(len(spec.boundaryRows) + 1), int64(total / 2), int64(total)}
	quote := conn.GetDialect().Quoter().QuoteIdent
	cols := append([]string{"id"}, spec.boundaryCols...)
	m := make(map[int64]map[string]string, len(ids))
	for _, id := range ids {
		_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = %d",
			strings.Join(cols, ", "), quote(spec.name), id))
		require.NoError(t, err, "[%s] 抽样id=%d", spec.name, id)
		require.Len(t, rows, 1, "[%s] 抽样id=%d无行", spec.name, id)
		row := map[string]string{}
		for _, c := range spec.boundaryCols {
			row[c] = mtNormCell(t, c, rows[0][c])
		}
		m[id] = row
	}
	return m
}

// mtNormCell 列级规范化：抹平驱动返值形态与各库类型语义差异后输出规范文本。
// 两侧（导出前快照/恢复后回读）经同一函数处理后方可比对。
// 注意：列级归一（bool/float32/decimal/json等）必须先于string短路返回，否则字符串形态的
// 期望值会绕过归一导致两侧不一致（如sqlite把decimal字面量物化为浮点）
func mtNormCell(t *testing.T, col string, v any) string {
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
		return fmt.Sprintf("%v", float32(f)) // FLOAT/REAL为float32精度
	case "c_dec36", "c_dec10", "c_dec":
		// sqlite弱类型会把decimal字面量物化为浮点（高精度串精度丢失，sqlite物理限制）；
		// 两侧统一按float最短表示比对（高精度保真由单表全类型roundtrip IT覆盖）
		f, ok := dbi.ValToFloat64(v)
		if ok {
			return fmt.Sprintf("%v", f)
		}
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	case "c_json":
		return mtJSONNorm(t, fmt.Sprintf("%v", v))
	case "c_char":
		return strings.TrimRight(fmt.Sprintf("%v", v), " ") // CHAR(36)右补空格语义
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return hex.EncodeToString(x)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// mtJSONNorm JSON文本规范化（键序重排/空白差异抹平：mysql JSON与pg JSONB均会重排键序）
func mtJSONNorm(t *testing.T, s string) string {
	t.Helper()
	var m any
	require.NoError(t, json.Unmarshal([]byte(s), &m), "非法JSON: %.200s", s)
	b, err := json.Marshal(m)
	require.NoError(t, err)
	return string(b)
}

func mtVal(t *testing.T, v any, name string) int64 {
	t.Helper()
	i, ok := dbi.ValToInt64(v)
	require.True(t, ok, "聚合值[%s]无法归一: %v", name, v)
	return i
}

func mtDump(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec, path string) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	bw := bufio.NewWriterSize(f, 1<<20)
	require.NoError(t, DumpDbScript(context.Background(), conn, &dto.DumpDb{
		DbName:   conn.Info.Database,
		Tables:   []string{spec.name},
		DumpDDL:  true,
		DumpData: true,
		Writer:   bw,
	}), "[%s] dump失败", spec.name)
	require.NoError(t, bw.Flush())
	require.NoError(t, f.Sync())
}

func mtRestore(t *testing.T, conn *dbi.DbConn, path string) {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	app := &DbTransferAppImpl{}
	require.NoError(t, app.importDumpStream(context.Background(), 0, conn, bufio.NewReaderSize(f, 1<<20)),
		"从备份文件恢复失败")
}

// mtCheckBoundary 边界行全量回读比对：以备份前源库读值为基准（FLOAT写入精度损失与
// 时间末零去除属写入/展示语义，不应作为备份恢复失败判定）
func mtCheckBoundary(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec, before map[int64]map[string]string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	cols := append([]string{"id"}, spec.boundaryCols...)
	for id := range before {
		_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = %d",
			strings.Join(cols, ", "), quote(spec.name), id))
		require.NoError(t, err, "[%s] 边界行%d回读失败", spec.name, id)
		require.Len(t, rows, 1, "[%s] 边界行id=%d恢复后缺失", spec.name, id)
		for _, col := range spec.boundaryCols {
			require.Equal(t, before[id][col], mtNormCell(t, col, rows[0][col]),
				"[%s] 边界行%d列[%s]恢复后不一致", spec.name, id, col)
		}
	}
}

// mtGoValueAsDriver 将Go侧期望值转为驱动返值同形态（[]byte→hex string，与ValuerBytes一致）
func mtGoValueAsDriver(v any) any {
	switch x := v.(type) {
	case []byte:
		return hex.EncodeToString(x)
	default:
		return v
	}
}

// mtCheckSample 抽样行一致性比对
func mtCheckSample(t *testing.T, conn *dbi.DbConn, spec *mtBigTableSpec, before map[int64]map[string]string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	cols := append([]string{"id"}, spec.boundaryCols...)
	for id, beforeRow := range before {
		_, rows, err := conn.Query(fmt.Sprintf("SELECT %s FROM %s WHERE id = %d",
			strings.Join(cols, ", "), quote(spec.name), id))
		require.NoError(t, err)
		require.Len(t, rows, 1, "[%s] 抽样行id=%d恢复后缺失", spec.name, id)
		for _, col := range spec.boundaryCols {
			require.Equal(t, beforeRow[col], mtNormCell(t, col, rows[0][col]),
				"[%s] 抽样行id=%d列[%s]恢复后不一致", spec.name, id, col)
		}
	}
}
