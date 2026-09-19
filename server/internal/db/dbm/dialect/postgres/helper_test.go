package postgres

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/dbi"
)

// TestFixColumnDefault pg元数据默认值形态归一的表驱动测试。
//
// pg的information_schema.column_default为pg_get_expr输出，实测形态（postgres:16）：
//
//	DEFAULT 'abc'      → 'abc'::character varying
//	DEFAULT ''         → ''::character varying
//	DEFAULT -1(int)    → '-1'::integer
//	DEFAULT NULL(varchar) → NULL::character varying
//	无默认值            → SQL NULL（扫描为nil，cast后为空串）
//
// 必须保留字面量书写引号（区分空串默认值与无默认值），且NULL::cast必须归一为空。
// isExpr为归一后列应被标记的「表达式默认值」状态：该标记决定后续生成DDL时是省略默认值还是还原为字面量
func TestFixColumnDefault(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   string
		isExpr bool
	}{
		{"无默认值", "", "", false},
		{"字符串字面量剥cast保留引号", "'abc'::character varying", "'abc'", false},
		{"空串字面量保留引号形态", "''::character varying", "''", false},
		{"含双写引号的字面量", "'it''s'::character varying", "'it''s'", false},
		{"数值列的字符串形态负数", "'-1'::integer", "'-1'", false},
		{"内容含括号的字面量", "'(0)'::character varying", "'(0)'", false},
		{"内容含cast文本的字面量", "'a::b'::character varying", "'a::b'", false},
		{"显式DEFAULT NULL归一为无默认值", "NULL::character varying", "", false},
		{"显式DEFAULT NULL-numeric", "NULL::numeric", "", false},
		{"裸NULL归一为无默认值", "NULL", "", false},
		{"jsonb字面量", "'{\"k\":1}'::jsonb", "'{\"k\":1}'", false},
		{"uuid函数调用不处理", "gen_random_uuid()", "gen_random_uuid()", true},
		{"带schema前缀的函数调用", "pg_catalog.now()", "pg_catalog.now()", true},
		{"自增序列默认值不处理", "nextval('t_id_seq'::regclass)", "nextval('t_id_seq'::regclass)", true},
		{"关键字默认值原样", "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP", false},
		// 可归一的当前时间函数不是「不可还原表达式」，否则timestamp列的now()默认值会被整族丢弃
		{"now函数可跨库还原", "now()", "now()", false},
		{"无cast的裸数字原样", "0", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &dbi.Column{ColumnDefault: tt.raw}
			FixColumnDefault(col)
			assert.Equal(t, tt.want, col.ColumnDefault)
			assert.Equal(t, tt.isExpr, col.IsExprDefault, "表达式默认值标记不符")
		})
	}
}

// TestFixColumnDefault_GeneratedSql 归一后的形态经统一默认值生成器必须产出正确DDL片段
func TestFixColumnDefault_GeneratedSql(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		dataType string
		want     string
	}{
		{"显式NULL不生成默认值", "NULL::character varying", "character varying", ""},
		{"空串默认值不得退化为无默认值", "''::character varying", "character varying", " DEFAULT ''"},
		{"含括号内容按字面量保留", "'(0)'::character varying", "character varying", " DEFAULT '(0)'"},
		{"含引号内容重新转义", "'it''s'::character varying", "character varying", " DEFAULT 'it''s'"},
		{"jsonb默认值必须引用", "'{\"k\":1}'::jsonb", "jsonb", " DEFAULT '{\"k\":1}'"},
		{"自增序列跳过", "nextval('t_id_seq'::regclass)", "integer", ""},
		// pg的now()/now与CURRENT_TIMESTAMP完全等价（同为事务开始时间），必须归一为标准关键字保留
		{"now()默认值保留", "now()", "timestamp", " DEFAULT CURRENT_TIMESTAMP"},
		{"now裸形态默认值保留", "now", "timestamp", " DEFAULT CURRENT_TIMESTAMP"},
		{"带精度参数归一为无参关键字", "CURRENT_TIMESTAMP(3)", "timestamp", " DEFAULT CURRENT_TIMESTAMP"},
		{"纯日期列取日期分量", "CURRENT_TIMESTAMP", "date", " DEFAULT CURRENT_DATE"},
		// 字符串列的内容可能就是这段文本，不得被改写成SQL关键字
		{"varchar列的同形文本必须引用", "CURRENT_TIMESTAMP", "character varying", " DEFAULT 'CURRENT_TIMESTAMP'"},
		{"uuid函数默认值跳过", "gen_random_uuid()", "uuid", ""},
		// text列的函数默认值最危险：不标记会被写成 DEFAULT 'gen_random_uuid()'，建表成功但默认值静默变成一串文本
		{"text列的函数默认值必须跳过而非引用", "gen_random_uuid()", "character varying", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col := &dbi.Column{ColumnDefault: tt.raw, DataType: tt.dataType}
			FixColumnDefault(col)
			// 必须走方言SQLGenerator实际使用的Of入口（携带源侧表达式标记），裸文本入口无法体现该语义
			assert.Equal(t, tt.want, dbi.GenColumnDefaultSqlOf(col, tt.dataType, dbi.QuoteEscape))
		})
	}
}

// itAfterInsert 调用生产校正语句生成器并返回产物文本
func itAfterInsert(t *testing.T, table string, columns []dbi.Column) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, (&DumpHelper{}).AfterInsert(&buf, table, columns))
	return buf.String()
}

// TestAfterInsertSequenceCorrection 自增序列校正语句的形态守卫。
//
// 并发下调竞态在IT侧只能概率性暴露（实测20轮约半数命中），本用例钉死语句形态，
// 使「去掉GREATEST」「去掉NULL守卫」这类回归在单测阶段即变红。三项语义要求（详见AfterInsert注释）：
//   - 只增不减：写入值必须与本段读到的max及序列当前值取GREATEST，否则大表分片并行迁移时
//     后完成但id更小的分片会把序列下调，迁移后首条业务插入即撞主键；
//   - 空表零副作用：max为NULL时必须靠WHERE整条跳过，不能让GREATEST忽略NULL后把is_called置true（首条插入平白跳号）；
//   - 不携带事务包装：导入方自管事务，脚本内BEGIN/COMMIT会破坏外层事务
func TestAfterInsertSequenceCorrection(t *testing.T) {
	t.Run("无自增列不输出任何语句", func(t *testing.T) {
		assert.Empty(t, itAfterInsert(t, "t_plain", []dbi.Column{{ColumnName: "id"}, {ColumnName: "v"}}))
	})

	t.Run("校正语句形态", func(t *testing.T) {
		out := itAfterInsert(t, "t_order", []dbi.Column{{ColumnName: "id", AutoIncrement: true}, {ColumnName: "v"}})
		t.Log(out)
		assert.Equal(t,
			"SELECT setval('\"t_order_id_seq\"', GREATEST((SELECT max(\"id\") FROM \"t_order\"), "+
				"(SELECT last_value FROM \"t_order_id_seq\")), true) "+
				"WHERE (SELECT max(\"id\") FROM \"t_order\") IS NOT NULL;\n", out)
		assert.NotContains(t, out, "BEGIN", "不得输出事务包装语句")
		assert.NotContains(t, out, "COMMIT", "不得输出事务包装语句")
	})

	t.Run("多个自增列各自一条校正", func(t *testing.T) {
		out := itAfterInsert(t, "t_multi", []dbi.Column{
			{ColumnName: "id", AutoIncrement: true},
			{ColumnName: "ver", AutoIncrement: true},
			{ColumnName: "v"},
		})
		assert.Equal(t, 2, bytes.Count([]byte(out), []byte("SELECT setval(")), "每个自增列各输出一条:\n"+out)
		assert.Contains(t, out, `setval('"t_multi_id_seq"'`)
		assert.Contains(t, out, `setval('"t_multi_ver_seq"'`)
	})

	// 序列名同时出现在字符串字面量（需转义）与FROM标识符位（需引用）两处，两者转义方式不同
	t.Run("含引号表名的双位引用", func(t *testing.T) {
		out := itAfterInsert(t, "it's_tbl", []dbi.Column{{ColumnName: "id", AutoIncrement: true}})
		assert.Contains(t, out, `setval('"it''s_tbl_id_seq"'`, "序列名在字符串字面量内必须按字面量转义:\n"+out)
		assert.Contains(t, out, `(SELECT last_value FROM "it's_tbl_id_seq")`, "序列名在FROM位必须按标识符引用:\n"+out)
	})

	// 列名作为主键参与max子查询，含特殊字符时必须引用，否则切割与执行均语法错误
	t.Run("含空格列名必须引用", func(t *testing.T) {
		out := itAfterInsert(t, "t_sp", []dbi.Column{{ColumnName: "my id", AutoIncrement: true}})
		assert.Contains(t, out, `(SELECT max("my id") FROM "t_sp")`, out)
	})
}
