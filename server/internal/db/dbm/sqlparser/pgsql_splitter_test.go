package sqlparser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pgSplitAll(t *testing.T, sql string) []string {
	t.Helper()
	stmts := make([]string, 0)
	require.NoError(t, NewPgsqlSplitter().SplitSQL(strings.NewReader(sql), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	}))
	return stmts
}

func TestPgsqlSplitter_Basic(t *testing.T) {
	assert.Equal(t, []string{"select 1", "select 2"}, pgSplitAll(t, "select 1; select 2;"))
	assert.Equal(t, []string{"select 1", "select 2"}, pgSplitAll(t, "select 1;\nselect 2"))
	assert.Empty(t, pgSplitAll(t, ";;; ;"))
	// 纯注释不产生语句
	assert.Empty(t, pgSplitAll(t, "-- comment; here\n/* block; */"))
}

func TestPgsqlSplitter_Strings(t *testing.T) {
	// 普通字符串：分号不切割，反斜杠为普通字符
	assert.Equal(t, []string{`insert into t values ('a;b')`}, pgSplitAll(t, `insert into t values ('a;b');`))
	assert.Equal(t, []string{`insert into t values ('a\;b')`}, pgSplitAll(t, `insert into t values ('a\;b');`))
	assert.Equal(t, []string{`select "a;b"`}, pgSplitAll(t, `select "a;b";`))
	// 双写引号转义
	assert.Equal(t, []string{`insert into t values ('it''s;x')`}, pgSplitAll(t, `insert into t values ('it''s;x');`))
	assert.Equal(t, []string{`select "a""b"`}, pgSplitAll(t, `select "a""b";`))
}

// PG专属 E'...' 转义字符串：反斜杠为转义符
func TestPgsqlSplitter_EscapeString(t *testing.T) {
	// \' 不结束字符串，分号在字符串内
	assert.Equal(t, []string{`insert into t values (E'it\'s;x')`}, pgSplitAll(t, `insert into t values (E'it\'s;x');`))
	// 小写 e 前缀同样生效
	assert.Equal(t, []string{`insert into t values (e'a\;b')`}, pgSplitAll(t, `insert into t values (e'a\;b');`))
	// E'\\' ：反斜杠转义反斜杠后字符串正常闭合
	assert.Equal(t, []string{`insert into t values (E'a\\;b')`}, pgSplitAll(t, `insert into t values (E'a\\;b');`))
	// E'' 双写引号依然生效
	assert.Equal(t, []string{`insert into t values (E'it''s;x')`}, pgSplitAll(t, `insert into t values (E'it''s;x');`))
	// E'\'' ：被转义的引号不闭合，其后直到下一个未转义引号才闭合
	// E-string未闭合导致整段吞入，尾分号随语句保留（与pg词法行为一致）
	assert.Equal(t, []string{`insert into t values (E'\'); insert into t values ('x');`},
		pgSplitAll(t, `insert into t values (E'\'); insert into t values ('x');`))
	// 普通字符串紧邻 E 标识（如 type_e'..' 场景不合法，无需特殊处理）；注释后接E-string正常
	// 注释移除后保留两侧原有空格
	assert.Equal(t, []string{`insert  into t values (E'a;b')`},
		pgSplitAll(t, `insert /* E */ into t values (E'a;b');`))
}

// PG专属 dollar-quoted 字符串：$$...$$ / $tag$...$tag$（函数体/DO块内分号不切割）
func TestPgsqlSplitter_DollarQuote(t *testing.T) {
	// 空tag
	assert.Equal(t, []string{`insert into t values ($$a;b$$)`}, pgSplitAll(t, `insert into t values ($$a;b$$);`))
	// 带tag
	assert.Equal(t, []string{`insert into t values ($fn$a;b$fn$)`}, pgSplitAll(t, `insert into t values ($fn$a;b$fn$);`))
	// $$;$$ ：body含分号
	assert.Equal(t, []string{`$$;$$`}, pgSplitAll(t, `$$;$$;`))
	// 函数定义（典型导入脚本场景：体内多个分号）
	fnScript := `CREATE OR REPLACE FUNCTION add_one(a integer) RETURNS integer AS $fn$
BEGIN
    RETURN a + 1; -- comment; here
END;
$fn$ LANGUAGE plpgsql;`
	// 尾部分号被切割消费
	assert.Equal(t, []string{strings.TrimSuffix(fnScript, ";")}, pgSplitAll(t, fnScript))
	// DO块
	doScript := `DO $$ BEGIN
    update t set a = 1;
END $$;`
	assert.Equal(t, []string{strings.TrimSuffix(doScript, ";")}, pgSplitAll(t, doScript))
	// tag后紧跟闭合（空函数体）
	assert.Equal(t, []string{`$a$$a$`}, pgSplitAll(t, `$a$$a$;`))
	// 字符串内的 $ 不是dollar-quote
	assert.Equal(t, []string{`insert into t values ('a$b;c')`}, pgSplitAll(t, `insert into t values ('a$b;c');`))
	// dollar-quote内的引号与注释文本不影响状态
	assert.Equal(t, []string{`select $tag$it's -- not comment /* not */$tag$`},
		pgSplitAll(t, `select $tag$it's -- not comment /* not */$tag$;`))
	// dollar-quote外的分号正常切割
	assert.Equal(t, []string{`select $$a$$`, "select 2"}, pgSplitAll(t, `select $$a$$; select 2;`))
}

// PG支持嵌套块注释
func TestPgsqlSplitter_NestedComment(t *testing.T) {
	// 注释移除后保留两侧原有空格
	assert.Equal(t, []string{"select  1"}, pgSplitAll(t, `select /* /* ; */ ; */ 1;`))
	assert.Equal(t, []string{"select  1", "select 2"}, pgSplitAll(t, `select /* outer /* inner ; */ ; */ 1; select 2;`))
	// 嵌套注释跨行
	assert.Equal(t, []string{"select  1"}, pgSplitAll(t, "select /* l1\n/* inner ; */\nl2 ; */ 1;"))
	// 注释内的引号不进入字符串状态
	assert.Equal(t, []string{"select  1"}, pgSplitAll(t, `select /* it's "quoted" ; */ 1;`))
}

func TestPgsqlSplitter_Multibyte(t *testing.T) {
	assert.Equal(t, []string{"insert into t values ('中文；分号🙂;semi')"},
		pgSplitAll(t, "insert into t values ('中文；分号🙂;semi');"))
	assert.Equal(t, []string{"select '中文；'"}, pgSplitAll(t, "select '中文；';"))
}

// TestPgsqlSplitter_BadUtf8 非法UTF8字节容错：坏字节原样透传，不得中断解析导致后续语句丢失
func TestPgsqlSplitter_BadUtf8(t *testing.T) {
	// 坏字节夹在语句中间：语句原样保留（含坏字节），后续语句正常切割
	assert.Equal(t, []string{"insert into t values ('a\xffb')", "select 2"},
		pgSplitAll(t, "insert into t values ('a\xffb'); select 2;"))
	// 注释中的坏字节随注释丢弃
	assert.Equal(t, []string{"select 1", "select 中文", "select 3"},
		pgSplitAll(t, "select 1; -- \xff 注释\nselect 中文; select 3;"))
}
