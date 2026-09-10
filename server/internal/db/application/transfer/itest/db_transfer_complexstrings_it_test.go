package itest

// 复杂字符串四方言（mysql/postgres/sqlite/mssql）端到端真实链路集成测试。
//
// 完整业务链路：源库参数化写入复杂值 → dumpDbScript（真实dump：目标方言DDL+转义INSERT，
// 含行数/字节双预算分批）→ ImportDumpStream（真实方言切割器切分 + 事务批量导入）→
// 目标库回读，与源库逐行逐列规范化比对。
//
// 覆盖值形态：单引号/双引号/反斜杠（含结尾单反斜杠）/LF/CRLF/制表与回车/JSON（含转义与分号）/
// 分号/注释样式/中文全角/emoji与零宽连接符/控制字符/反引号/超长文本/NULL/空串/二进制（含\x00）。
// 覆盖方言有序对：mysql→pg、mysql→sqlite、mysql→mssql、pg→mysql、pg→sqlite、
// sqlite→mysql、sqlite→pg、mssql→mysql。同方言备份恢复由整库用例覆盖（本用例源/目标同库同表会相碰）。
//
// 运行：cd server && go test -tags it -count=1 -run TestITComplexStrings ./internal/db/application/

import (
	"mayfly-go/internal/db/application/transfer"
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/application/dto"
	"mayfly-go/internal/db/dbm/dbi"
)

// itComplexTexts 复杂字符串样本：按行轮转写入文本列
var itComplexTexts = []string{
	"plain",
	"",
	"it's a single quote",
	"'",
	"''",
	"'quoted pair'",
	`say "double" and 'single'`,
	`back\slash`,
	`trailing\`,
	`a\\b`,
	"line1\nline2",
	"line1\r\nline2\r\n",
	"tab\tsep\r",
	`{"key":"va'lue","path":"C:\\tmp\\a","arr":[1,2,{"k":";"}]}`,
	"[1,2,{\"n\":\"中文\"}]",
	"semi;colon;ends;",
	"'; delete from t; --",
	"-- like comment",
	"/* like block comment */",
	"# mysql hash",
	"中文，ＢＢＣ全角。",
	"emoji 😀🎉 and 👨‍👩‍👧 ZWJ",
	"%_wild%_",
	"0x4142 and X'41'",
	"`backquoted`",
	"\thas\ttabs\t",
	strings.Repeat("long混合文本;", 200),
}

// itComplexBlob 二进制样本：含\x00与高字节，必须以字节保真迁移
var itComplexBlob = []byte{0x00, 0x01, 0xfe, 'a', '\'', '\\', 0x7f, 0xff}

// 方言节点与连接工厂见 it_env_it_test.go（IT 公共底座）

// itCreateComplexTable 按方言类型建含文本/JSON/二进制/时间/数值/可空列的表
func itCreateComplexTable(t *testing.T, conn *dbi.DbConn, table string) {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	var ddl string
	switch conn.Info.Type {
	case "mysql":
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id int PRIMARY KEY, v_text text, v_varchar varchar(2000), v_json json, v_blob blob, "+
			"v_time datetime(3), v_num decimal(20,6), v_null text) DEFAULT CHARSET=utf8mb4", quote(table))
	case "postgres":
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id int PRIMARY KEY, v_text text, v_varchar varchar(2000), v_json jsonb, v_blob bytea, "+
			"v_time timestamp(3), v_num numeric(20,6), v_null text)", quote(table))
	case "sqlite":
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id INTEGER PRIMARY KEY, v_text TEXT, v_varchar TEXT, v_json TEXT, v_blob BLOB, "+
			"v_time TIMESTAMP, v_num NUMERIC, v_null TEXT)", quote(table))
	case "mssql":
		// 字符列统一nvarchar：本机SQL Server库默认排序规则为SQL_Latin1_General_CP1_CI_AS，
		// varchar/text列会在**写入阶段**就按库代码页将中文/emoji转为'?'，与dump转义无关而混淆归因
		ddl = fmt.Sprintf("CREATE TABLE %s ("+
			"id int PRIMARY KEY, v_text nvarchar(max), v_varchar nvarchar(2000), v_json nvarchar(max), v_blob varbinary(max), "+
			"v_time datetime2(3), v_num decimal(20,6), v_null nvarchar(max))", quote(table))
	default:
		t.Fatalf("unsupported dialect %s", conn.Info.Type)
	}
	_, err := conn.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", quote(table)))
	require.NoError(t, err)
	_, err = conn.Exec(ddl)
	require.NoError(t, err, "建表失败")
}

// itPlaceholder 方言参数占位符：postgres用$N，其余用?
func itPlaceholder(conn *dbi.DbConn, idx int) string {
	if conn.Info.Type == "postgres" {
		return fmt.Sprintf("$%d", idx)
	}
	return "?"
}

// itInsertComplexRows 参数化写入复杂值（不经SQL字面量拼接，确保源库存的是原始值）
func itInsertComplexRows(t *testing.T, conn *dbi.DbConn, table string) int {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	cols := []string{"id", "v_text", "v_varchar", "v_json", "v_blob", "v_time", "v_num", "v_null"}
	ph := make([]string, 0, len(cols))
	for i := range cols {
		ph = append(ph, itPlaceholder(conn, i+1))
	}
	insertSql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", quote(table), strings.Join(cols, ", "), strings.Join(ph, ", "))

	// JSON列取值：以合法JSON文本为主（非法JSON另由v_text覆盖）
	jsonVals := []string{
		`{"k":"va'lue","p":"C:\\tmp"}`,
		`{"arr":[1,2,{"s":"x;y"}],"u":"中文😀"}`,
		`{"nl":"line1\nline2","tab":"a\tb"}`,
		`{}`,
		`{"quote":"say \"hi\" and 'bye'"}`,
	}
	timeVal := "2024-06-15 08:30:05.123"
	numVal := "123456.456000"

	total := len(itComplexTexts)
	for i := 0; i < total; i++ {
		text := itComplexTexts[i]
		var nullVal any
		if i%3 == 0 {
			nullVal = nil // 覆盖NULL与非NULL混合
		} else {
			nullVal = text
		}
		var jsonArg any = jsonVals[i%len(jsonVals)]
		if conn.Info.Type == "sqlite" {
			jsonArg = jsonVals[i%len(jsonVals)]
		}
		_, err := conn.Exec(insertSql, int64(i+1), text, text, jsonArg, itComplexBlob, timeVal, numVal, nullVal)
		require.NoError(t, err, "写入第%d行失败, v_text=%q", i+1, text)
	}
	return total
}

// itReadRowsRaw 回读全表并按id升序返回驱动原始返值（列名统一小写）
func itReadRowsRaw(t *testing.T, conn *dbi.DbConn, table string) []map[string]any {
	t.Helper()
	quote := conn.GetDialect().Quoter().QuoteIdent
	_, rows, err := conn.Query(fmt.Sprintf("SELECT * FROM %s ORDER BY id", quote(table)))
	require.NoError(t, err)
	res := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		m := make(map[string]any, len(row))
		for k, v := range row {
			m[strings.ToLower(k)] = v
		}
		res = append(res, m)
	}
	return res
}

// itDumpScript 驱动真实dump生成目标方言脚本
func itDumpScript(t *testing.T, srcConn *dbi.DbConn, table string, targetDbType dbi.DbType) string {
	t.Helper()
	var buf bytes.Buffer
	err := transfer.DumpDbScript(context.Background(), srcConn, &dto.DumpDb{
		DbName:       srcConn.Info.Database,
		Tables:       []string{table},
		DumpDDL:      true,
		DumpData:     true,
		TargetDbType: targetDbType,
		Writer:       &buf,
	})
	require.NoError(t, err, "dump失败")
	script := buf.String()
	require.NotEmpty(t, script)
	return script
}

// itTextColumns 必须字节级保真的文本类列（本次测试的核心断言目标）
var itTextColumns = []string{"id", "v_text", "v_varchar", "v_json", "v_blob", "v_null"}

// itTypedColumns 时间/数值类列：驱动返值形态存在方言差异，同样以规范化后等值为准确保未丢精度
var itTypedColumns = []string{"v_time", "v_num"}

// TestITComplexStringsAcrossDialects 八种方言有序对的复杂字符串端到端保真验证
func TestITComplexStringsAcrossDialects(t *testing.T) {
	pairs := []itPair{
		{itMysql, itPg}, {itMysql, itSqlite}, {itMysql, itMssql},
		{itPg, itMysql}, {itPg, itSqlite},
		{itSqlite, itMysql}, {itSqlite, itPg},
		// mssql作为异构源：SQL Server的nvarchar/varbinary元数据与值形态进入dump链路
		{itMssql, itMysql},
	}

	for _, p := range pairs {
		t.Run(p.src.name+"->"+p.tgt.name, func(t *testing.T) {
			table := fmt.Sprintf("it_cx_%s_%s", p.src.name, p.tgt.name)

			srcConn := p.src.conn(t)
			defer srcConn.Close()
			tgtConn := p.tgt.conn(t)
			defer tgtConn.Close()

			itCreateComplexTable(t, srcConn, table)
			wantRows := itInsertComplexRows(t, srcConn, table)

			// 源库自校验：写入即可读回，且复杂值未被驱动改写
			srcRows := itReadRowsRaw(t, srcConn, table)
			require.Len(t, srcRows, wantRows, "源库回读行数不符")
			for i, text := range itComplexTexts {
				assert.Equal(t, text, itTextAt(srcRows[i], "v_text"), "源库第%d行文本被改写", i+1)
			}

			// 目标库与源库不同库/不同实例，按真实迁移语义使用同表名，dump脚本直接导入
			script := itDumpScript(t, srcConn, table, p.tgt.dbType)

			// 真实切割 + 批量导入
			app := &transfer.DbTransferAppImpl{}
			require.NoError(t, app.ImportDumpStream(context.Background(), 0, tgtConn, strings.NewReader(script)),
				"导入失败, dump脚本:\n%s", itTruncate(script, 4000))

			tgtRows := itReadRowsRaw(t, tgtConn, table)
			require.Len(t, tgtRows, wantRows, "目标库行数与源库不一致")

			sampleOf := func(i int) string {
				return itTruncate(itComplexTexts[i%len(itComplexTexts)], 60)
			}
			// 文本/二进制列：字节级严格保真（本次测试的核心断言）
			for i := 0; i < wantRows; i++ {
				for _, col := range itTextColumns {
					assert.Equal(t, itTextAt(srcRows[i], col), itTextAt(tgtRows[i], col),
						"第%d行(%q)列[%s]迁移失真, 源=%q 目标=%q", i+1, sampleOf(i), col,
						itTextAt(srcRows[i], col), itTextAt(tgtRows[i], col))
				}
				// 时间/数值列：两侧均为数值/时间类，按产品校验器同一套语义比对（容忍标度呈现差异）
				for _, col := range itTypedColumns {
					assert.True(t, dbi.CanonicalNumericEqual(srcRows[i][col], tgtRows[i][col]),
						"第%d行(%q)列[%s]迁移失真, 源=%q 目标=%q", i+1, sampleOf(i), col,
						dbi.CanonicalValue(srcRows[i][col]), dbi.CanonicalValue(tgtRows[i][col]))
				}
			}
		})
	}
}
