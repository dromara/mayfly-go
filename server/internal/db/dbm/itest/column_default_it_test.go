package itest

// 列默认值跨方言真实往返集成测试（9个源→目标组合：mysql/postgres/sqlite 两两含同方言）。
//
// 与既有的数据往返测试互补，本测试只验证「结构迁移中的列默认值」：
//  1. 手写源表DDL（各方言按自身字面量语义书写同一个语义值）；
//  2. 读源库元数据 → ConvToTargetDbColumn → 目标方言GenTableDDL → 真实建表；
//  3. 结构断言：目标DDL中该列必须仍带DEFAULT子句（防止默认值被静默丢弃）；
//  4. 语义断言：两侧均只插入主键、其余列走默认值，逐列比对库内实际值必须等于建表时书写的语义值。
//
// 用例覆盖：含单引号/连续单引号/双引号/反斜杠/分号与注释符/纯空白/首尾空白/空串/
// 内容含括号（'(0)'、'待确认(必填)'、'unknown (pending)'）/裸文本NULL/数值负数与0/
// 大整数/decimal/CURRENT_TIMESTAMP/CURRENT_TIMESTAMP(3)/CURRENT_DATE/
// 与SQL关键字同形的字符串默认值（'TRUE'、'CURRENT_TIMESTAMP'、'now()'、'0'）/
// 显式DEFAULT NULL/无默认值。
//
// 运行方式：cd server && go test -tags it -count=1 -run TestITColumnDefault ./internal/db/dbm/

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

const (
	dflMysql    = dbi.DbType("mysql")
	dflPostgres = dbi.DbType("postgres")
	dflSqlite   = dbi.DbType("sqlite")
)

// dflCase 单个默认值用例：kind决定各方言列类型，lit决定各方言DEFAULT子句书写形态，want为期望库内值
type dflCase struct {
	col   string
	kind  string // str/int/bigint/dec/ts/ts3/date
	lit   map[dbi.DbType]string
	want  any  // 期望库内实际值；nil表示期望NULL
	loose bool // 仅断言非NULL（默认值为当前时间的列，两侧写入时刻不同）
	// ddlSub 目标DDL该列行必须包含的片段（大写比对）：用于锁定「当前日期/时间默认值必须仍以关键字
	// 形态存活」，既不能被静默丢弃，也不能退化成写死的时间字面量或源库专有函数名
	ddlSub string
}

// dflStrLit 按各方言字符串字面量语义书写v：mysql默认开启反斜杠转义（需额外双写反斜杠），
// pg（standard_conforming_strings=on）与sqlite的字符串字面量中反斜杠是普通字符
func dflStrLit(dt dbi.DbType, v string) string {
	if dt == dflMysql {
		return "'" + dbi.QuoteEscapeBackslash(v) + "'"
	}
	return "'" + dbi.QuoteEscape(v) + "'"
}

// dflStrLitAll 对所有方言生成同一字面量（用于不含反斜杠、三方言语义一致的内容）
func dflStrLitAll(v string) map[dbi.DbType]string {
	return map[dbi.DbType]string{
		dflMysql:    dflStrLit(dflMysql, v),
		dflPostgres: dflStrLit(dflPostgres, v),
		dflSqlite:   dflStrLit(dflSqlite, v),
	}
}

var dflCases = []dflCase{
	{col: "d_plain", kind: "str", lit: dflStrLitAll("abc"), want: "abc"},
	{col: "d_quote", kind: "str", lit: dflStrLitAll("it's"), want: "it's"},
	{col: "d_two_quote", kind: "str", lit: dflStrLitAll("a''b"), want: "a''b"},
	{col: "d_dblquote", kind: "str", lit: dflStrLitAll(`say "hi"`), want: `say "hi"`},
	{col: "d_empty", kind: "str", lit: dflStrLitAll(""), want: ""},
	{col: "d_space", kind: "str", lit: dflStrLitAll(" "), want: " "},
	{col: "d_pad", kind: "str", lit: dflStrLitAll("  x  "), want: "  x  "},
	{col: "d_paren_num", kind: "str", lit: dflStrLitAll("(0)"), want: "(0)"},
	{col: "d_paren_cn", kind: "str", lit: dflStrLitAll("待确认(必填)"), want: "待确认(必填)"},
	{col: "d_pending", kind: "str", lit: dflStrLitAll("unknown (pending)"), want: "unknown (pending)"},
	{col: "d_semi", kind: "str", lit: dflStrLitAll("a;b--c/*x*/"), want: "a;b--c/*x*/"},
	{col: "d_backslash", kind: "str", lit: map[dbi.DbType]string{
		dflMysql:    `'a\\b'`,
		dflPostgres: `'a\b'`,
		dflSqlite:   `'a\b'`,
	}, want: `a\b`},
	// NOT NULL列的字符串默认值NULL：MySQL不允许NOT NULL DEFAULT NULL（建表即报1067），
	// 故元数据中的裸NULL必为字符串内容，不得当作无默认值丢弃
	{col: "d_nullstr", kind: "str", lit: dflStrLitAll("NULL"), want: "NULL"},
	// 可空列的显式DEFAULT NULL：等价于无默认值，插入省略该列必须得到NULL
	{col: "d_explicit_null", kind: "str", lit: map[dbi.DbType]string{
		dflMysql: "NULL", dflPostgres: "NULL", dflSqlite: "NULL",
	}, want: nil},
	{col: "d_no_default", kind: "str", lit: map[dbi.DbType]string{}, want: nil},
	{col: "d_int_neg", kind: "int", lit: map[dbi.DbType]string{
		dflMysql: "-1", dflPostgres: "-1", dflSqlite: "-1",
	}, want: int64(-1)},
	{col: "d_int_zero", kind: "int", lit: map[dbi.DbType]string{
		dflMysql: "0", dflPostgres: "0", dflSqlite: "0",
	}, want: int64(0)},
	{col: "d_bigint", kind: "bigint", lit: map[dbi.DbType]string{
		dflMysql: "9223372036854775807", dflPostgres: "9223372036854775807", dflSqlite: "9223372036854775807",
	}, want: int64(9223372036854775807)},
	{col: "d_dec", kind: "dec", lit: map[dbi.DbType]string{
		dflMysql: "12.34", dflPostgres: "12.34", dflSqlite: "12.34",
	}, want: "12.34"},
	{col: "d_ts", kind: "ts", lit: map[dbi.DbType]string{
		dflMysql: "CURRENT_TIMESTAMP", dflPostgres: "CURRENT_TIMESTAMP", dflSqlite: "CURRENT_TIMESTAMP",
	}, want: "NON_NULL", loose: true, ddlSub: "CURRENT_TIMESTAMP"},
	// 带小数秒精度的自动初始化默认值：MySQL要求默认值的fsp与列定义严格相等（不匹配即Error 1067），
	// 而sqlite不接受CURRENT_TIMESTAMP的参数，故源侧按各方言自身语法书写、目标侧必须重新归一
	{col: "d_ts3", kind: "ts3", lit: map[dbi.DbType]string{
		dflMysql: "CURRENT_TIMESTAMP(3)", dflPostgres: "CURRENT_TIMESTAMP(3)", dflSqlite: "CURRENT_TIMESTAMP",
	}, want: "NON_NULL", loose: true, ddlSub: "CURRENT_TIMESTAMP"},
	// 纯日期列的当日默认值：MySQL 8.0元数据以(CURRENT_DATE)表达式形态呈现，PG/sqlite为裸关键字，
	// 归一不当会在目标库产生Error 1064（裸CURRENT_DATE不是合法的MySQL默认值）而丢默认值
	{col: "d_date", kind: "date", lit: map[dbi.DbType]string{
		dflMysql: "(CURRENT_DATE)", dflPostgres: "CURRENT_DATE", dflSqlite: "CURRENT_DATE",
	}, want: "NON_NULL", loose: true, ddlSub: "CURRENT_DATE"},
	// 字符串列的默认值内容恰好与SQL关键字/函数/数字同形：MySQL 8.0元数据去引号呈现，
	// 若误当裸字面量拼入DDL，varchar DEFAULT TRUE会被目标库重新解释为'1'（静默数据损坏），
	// varchar DEFAULT CURRENT_TIMESTAMP则直接建表失败
	{col: "d_true_str", kind: "str", lit: dflStrLitAll("TRUE"), want: "TRUE"},
	{col: "d_ts_str", kind: "str", lit: dflStrLitAll("CURRENT_TIMESTAMP"), want: "CURRENT_TIMESTAMP"},
	{col: "d_now_str", kind: "str", lit: dflStrLitAll("now()"), want: "now()"},
	{col: "d_zero_str", kind: "str", lit: dflStrLitAll("0"), want: "0"},
}

// dflColTypeSql 返回该用例列在指定方言下的列类型与约束片段（不含DEFAULT）
func dflColTypeSql(dt dbi.DbType, c dflCase) string {
	strType := map[dbi.DbType]string{dflMysql: "varchar(64)", dflPostgres: "character varying(64)", dflSqlite: "text"}[dt]
	switch c.kind {
	case "int":
		return map[dbi.DbType]string{dflMysql: "int", dflPostgres: "integer", dflSqlite: "integer"}[dt] + " NOT NULL"
	case "bigint":
		return map[dbi.DbType]string{dflMysql: "bigint", dflPostgres: "bigint", dflSqlite: "integer"}[dt] + " NOT NULL"
	case "dec":
		return map[dbi.DbType]string{dflMysql: "decimal(10,2)", dflPostgres: "numeric(10,2)", dflSqlite: "numeric(10,2)"}[dt] + " NOT NULL"
	case "ts":
		return map[dbi.DbType]string{dflMysql: "timestamp", dflPostgres: "timestamp", dflSqlite: "timestamp"}[dt] + " NOT NULL"
	case "ts3":
		// sqlite无小数秒概念，列类型不带精度参数；mysql用datetime避开同表多个timestamp的特殊约束
		return map[dbi.DbType]string{dflMysql: "datetime(3)", dflPostgres: "timestamp(3)", dflSqlite: "timestamp"}[dt] + " NOT NULL"
	case "date":
		return map[dbi.DbType]string{dflMysql: "date", dflPostgres: "date", dflSqlite: "date"}[dt] + " NOT NULL"
	}
	// 字符串列：显式DEFAULT NULL与无默认值两例必须可空，其余非空以验证NULL字符串默认值
	switch c.col {
	case "d_explicit_null", "d_no_default":
		return strType + " NULL"
	}
	return strType + " NOT NULL"
}

// buildDflSourceDDL 按指定方言的字面量语义生成源表建表SQL
func buildDflSourceDDL(dt dbi.DbType, table string) string {
	quote := dbi.GetDialect(dt).Quoter().Quote
	var sb strings.Builder
	sb.WriteString("CREATE TABLE " + quote(table) + " (id INT NOT NULL,")
	for _, c := range dflCases {
		sb.WriteString("\n  " + quote(c.col) + " " + dflColTypeSql(dt, c))
		if lit, ok := c.lit[dt]; ok {
			sb.WriteString(" DEFAULT " + lit)
		}
		sb.WriteString(",")
	}
	sb.WriteString("\n  PRIMARY KEY (id)")
	if dt == dflMysql {
		sb.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4")
	} else {
		sb.WriteString(")")
	}
	return sb.String()
}

// dflShortCode 表名用的方言短码（pg标识符长度有限）
func dflShortCode(dt dbi.DbType) string {
	switch dt {
	case dflMysql:
		return "my"
	case dflPostgres:
		return "pg"
	default:
		return "lt"
	}
}

func dflConn(t *testing.T, dt dbi.DbType) *dbi.DbConn {
	t.Helper()
	switch dt {
	case dflMysql:
		return mysqlConn(t)
	case dflPostgres:
		return pgConn(t)
	default:
		return sqliteConn(t)
	}
}

// TestITColumnDefaultRoundtrip 9个源→目标组合的列默认值结构与语义往返
func TestITColumnDefaultRoundtrip(t *testing.T) {
	srcTypes := []dbi.DbType{dflMysql, dflPostgres, dflSqlite}
	dstTypes := []dbi.DbType{dflMysql, dflPostgres, dflSqlite}

	for _, srcType := range srcTypes {
		for _, dstType := range dstTypes {
			pair := fmt.Sprintf("%s2%s", dflShortCode(srcType), dflShortCode(dstType))
			t.Run(pair, func(t *testing.T) {
				srcConn := dflConn(t, srcType)
				dstConn := srcConn
				if dstType != srcType {
					dstConn = dflConn(t, dstType)
				}

				srcTable := "it_dfl_" + dflShortCode(srcType) + "_src"
				dstTable := "it_dfl_" + dflShortCode(srcType) + "_" + dflShortCode(dstType) + "_dst"
				if srcType == dstType {
					dstTable = srcTable + "_rebuild"
				}
				// t.Cleanup按后进先出执行：必须先注册Close、后注册DROP，否则DROP会在连接已关闭后执行而panic
				t.Cleanup(func() { _ = srcConn.Close() })
				if dstConn != srcConn {
					t.Cleanup(func() { _ = dstConn.Close() })
				}
				t.Cleanup(func() {
					_, _ = srcConn.Exec("DROP TABLE IF EXISTS " + srcConn.GetDialect().Quoter().Quote(srcTable))
					_, _ = dstConn.Exec("DROP TABLE IF EXISTS " + dstConn.GetDialect().Quoter().Quote(dstTable))
				})

				// 1. 建源表（手写DDL，按源方言字面量语义书写默认值）
				mustExec(t, srcConn, "DROP TABLE IF EXISTS "+srcConn.GetDialect().Quoter().Quote(srcTable))
				mustExec(t, srcConn, buildDflSourceDDL(srcType, srcTable))

				// 2. 读源库元数据并转换到目标方言类型
				srcCols, err := srcConn.GetMetadata().GetColumns(srcTable)
				require.NoError(t, err)
				require.Len(t, srcCols, len(dflCases)+1, "源表列数不符")

				dstDialect := dstConn.GetDialect()
				cols := make([]dbi.Column, 0, len(srcCols))
				for _, col := range srcCols {
					col.TableName = dstTable
					require.NoError(t, dbi.ConvToTargetDbColumn(srcConn.Info.Type, dstConn.Info.Type, dstDialect, &col),
						"convert column [%s] failed", col.ColumnName)
					cols = append(cols, col)
				}

				// 3. 目标方言生成DDL并真实执行
				ddls := dstDialect.GetSQLGenerator().GenTableDDL(dbi.Table{TableName: dstTable}, cols, true)
				for _, ddl := range ddls {
					mustExec(t, dstConn, ddl)
				}

				// 4. 结构断言：应有默认值的列，其DDL行必须仍含DEFAULT子句（防静默丢弃）
				createSql := strings.Join(ddls, "\n")
				for _, c := range dflCases {
					if c.want == nil {
						continue
					}
					line := sqlLineOfColumn(createSql, dstDialect.Quoter().QuoteIdent(c.col))
					require.NotEmpty(t, line, "目标DDL中未找到列 [%s]", c.col)
					assert.Contains(t, strings.ToUpper(line), "DEFAULT", "列 [%s] 默认值被静默丢弃: %s", c.col, line)
					if c.ddlSub != "" {
						assert.Contains(t, strings.ToUpper(line), c.ddlSub,
							"列 [%s] 的默认值形态不符（期望含 %s，实际 %s）", c.col, c.ddlSub, strings.TrimSpace(line))
					}
				}

				// 4b. 类型保真断言（从目标库回读元数据，不依赖生成的DDL文本）：
				// 定点数列不得因跨方言转换退化为浮点（金额类值静默失真），
				// 源列声明了小数秒精度时目标列必须同样携带该精度（否则秒以下数据静默截断）
				dstCols, err := dstConn.GetMetadata().GetColumns(dstTable)
				require.NoError(t, err)
				dstColByName := make(map[string]dbi.Column, len(dstCols))
				for _, col := range dstCols {
					dstColByName[strings.ToLower(col.ColumnName)] = col
				}
				for _, c := range dflCases {
					dstCol, ok := dstColByName[strings.ToLower(c.col)]
					if !ok {
						continue
					}
					switch c.kind {
					case "dec":
						ct := dbi.GetDbDataType(dstType, dstCol.DataType).CommonType
						assert.True(t, ct == dbi.CTDecimal || ct == dbi.CTNumeric,
							"定点数列 [%s] 在目标库退化为非精确数值类型: %s", c.col, dstCol.GetColumnType())
					case "ts3":
						// sqlite无小数秒概念（源与目标均不适用），其余组合必须携带精度3
						if srcType != dflSqlite && dstType != dflSqlite {
							assert.Contains(t, dstCol.GetColumnType(), "(3)",
								"小数秒精度未带入目标列 [%s]: %s", c.col, dstCol.GetColumnType())
						}
					}
				}

				// 5. 语义断言：两侧均只插入主键，逐列比对库内默认值
				insert := func(conn *dbi.DbConn, table string) {
					t.Helper()
					mustExec(t, conn, "INSERT INTO "+conn.GetDialect().Quoter().Quote(table)+"(id) VALUES (1)")
				}
				insert(srcConn, srcTable)
				insert(dstConn, dstTable)

				srcRow := mustQueryRow(t, srcConn, srcTable)
				dstRow := mustQueryRow(t, dstConn, dstTable)
				for _, c := range dflCases {
					if c.loose {
						assert.NotNil(t, srcRow[c.col], "源表 [%s] 默认值未生效", c.col)
						assert.NotNil(t, dstRow[c.col], "目标表 [%s] 默认值未生效", c.col)
						continue
					}
					assertCell(t, "src."+c.col, c.want, srcRow[c.col])
					assertCell(t, "dst."+c.col, c.want, dstRow[c.col])
				}
			})
		}
	}
}

// sqlLineOfColumn 从建表SQL中取出包含指定列引用的那一行（用于按列做DEFAULT存在性断言）
func sqlLineOfColumn(createSql, quotedCol string) string {
	for _, line := range strings.Split(createSql, "\n") {
		if strings.Contains(line, quotedCol) {
			return line
		}
	}
	return ""
}

func mustQueryRow(t *testing.T, conn *dbi.DbConn, table string) map[string]any {
	t.Helper()
	_, rows, err := conn.Query("SELECT * FROM " + conn.GetDialect().Quoter().Quote(table) + " WHERE id = 1")
	require.NoError(t, err)
	require.Len(t, rows, 1, "表 [%s] 应恰有一行默认值数据", table)
	return rows[0]
}
