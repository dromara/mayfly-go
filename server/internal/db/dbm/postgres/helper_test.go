package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
