package dbi

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mayfly-go/internal/db/dbm/sqlparser"
)

// complexStrSamples 复杂字符串样本：引号/反斜杠/换行/回车/制表/JSON/分号/注释样式/
// 控制字符/中文全角/emoji/零宽连接符/超长文本，覆盖迁移dump转义的所有易错形态
var complexStrSamples = []string{
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
	`a\\\b`,
	"line1\nline2",
	"line1\r\nline2\r\n",
	"tab\tsep\r",
	`{"key":"va'lue","path":"C:\\tmp\\a","arr":[1,2,{"k":";"}]}`,
	"semi;colon;ends;",
	"'; delete from t; --",
	"-- like comment",
	"/* like block comment */",
	"# mysql hash",
	"中文，ＢＢＣ全角。",
	"emoji 😀🎉 and 👨‍👩‍👧 ZWJ",
	"%_wild%_",
	"0x4142 and X'41'",
	"\x01\x02 control chars",
	"`backquoted`",
	"\thas\ttabs\t",
	strings.Repeat("long混合文本;", 300),
}

// decodeStdLiteral 按标准SQL语义还原字符串字面量（单引号双写还原为单引号）
func decodeStdLiteral(t *testing.T, lit string) string {
	t.Helper()
	require.True(t, strings.HasPrefix(lit, "'") && strings.HasSuffix(lit, "'") && len(lit) >= 2, "字面量形态错误: %q", lit)
	body := lit[1 : len(lit)-1]
	return strings.ReplaceAll(body, "''", "'")
}

// decodeMysqlLiteral 按mysql语义还原字符串字面量（反斜杠转义下一字符，单引号双写还原为单引号）
func decodeMysqlLiteral(t *testing.T, lit string) string {
	t.Helper()
	require.True(t, strings.HasPrefix(lit, "'") && strings.HasSuffix(lit, "'") && len(lit) >= 2, "字面量形态错误: %q", lit)
	body := lit[1 : len(lit)-1]
	var sb strings.Builder
	for i := 0; i < len(body); i++ {
		switch c := body[i]; c {
		case '\\':
			// 反斜杠转义下一字符
			if i+1 < len(body) {
				i++
				sb.WriteByte(body[i])
			} else {
				sb.WriteByte(c)
			}
		case '\'':
			// '' 为字面单引号
			if i+1 < len(body) && body[i+1] == '\'' {
				i++
			}
			sb.WriteByte('\'')
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

// 转义可逆性：标准SQL语义下转义后的字面量必须能无损还原为原值
func TestSQLValueString_RoundTrip(t *testing.T) {
	for _, s := range complexStrSamples {
		lit := SQLValueString(s)
		assert.Equal(t, s, decodeStdLiteral(t, lit), "标准SQL转义还原失败, 原值=%q 字面量=%s", s, lit)
	}
}

// 转义可逆性：mysql语义（反斜杠转义）下同样必须无损还原
func TestSQLValueStringEscapeBackslash_RoundTrip(t *testing.T) {
	for _, s := range complexStrSamples {
		lit := SQLValueStringEscapeBackslash(s)
		assert.Equal(t, s, decodeMysqlLiteral(t, lit), "mysql转义还原失败, 原值=%q 字面量=%s", s, lit)
	}
}

// splitStmts 使用指定切割器切割脚本，返回语句列表
func splitStmts(t *testing.T, splitter sqlparser.SQLSplitter, script string) []string {
	t.Helper()
	stmts := make([]string, 0, 8)
	require.NoError(t, splitter.SplitSQL(strings.NewReader(script), func(stmt string) error {
		stmts = append(stmts, stmt)
		return nil
	}))
	return stmts
}

// dump闭环：复杂值生成的INSERT语句必须被对应方言切割器识别为恰好一条语句，
// 且语句内容未被改写；否则迁移导入会错切SQL导致语法错误或数据丢失
func TestComplexValueInsertNotMissplit(t *testing.T) {
	for _, s := range complexStrSamples {
		// 目标为标准SQL方言（postgres/sqlite）：值用标准转义，语句用标准切割器
		stdScript := "INSERT INTO t (a) VALUES (" + SQLValueString(s) + ");"
		stdStmts := splitStmts(t, sqlparser.NewStdSQLSplitter(), stdScript)
		require.Len(t, stdStmts, 1, "标准SQL语句被错切, 原值=%q", s)
		assert.Equal(t, stdScript[:len(stdScript)-1], stdStmts[0], "切割改写了语句内容")

		// 目标为mysql：值用反斜杠转义，语句用mysql切割器
		myScript := "INSERT INTO `t` (a) VALUES (" + SQLValueStringEscapeBackslash(s) + ");"
		myStmts := splitStmts(t, sqlparser.NewMysqlSplitter(), myScript)
		require.Len(t, myStmts, 1, "mysql语句被错切, 原值=%q", s)
		assert.Equal(t, myScript[:len(myScript)-1], myStmts[0], "切割改写了语句内容")
	}
}

// 多条复杂值语句混合：切割条数必须与语句数一致，且顺序内容不变
func TestComplexValuesMultiStmtSplit(t *testing.T) {
	var sb strings.Builder
	stmtsWant := make([]string, 0, len(complexStrSamples))
	for _, s := range complexStrSamples {
		one := "INSERT INTO t (a, b) VALUES (1, " + SQLValueString(s) + ")"
		stmtsWant = append(stmtsWant, one)
		sb.WriteString(one + ";\n")
	}
	got := splitStmts(t, sqlparser.NewStdSQLSplitter(), sb.String())
	assert.Equal(t, stmtsWant, got)
}

// 二进制值：hex形态输出X'...'字面量，非hex文本退化为字符串转义
func TestSQLValueBytes(t *testing.T) {
	raw := []byte{0x00, 0x01, 0xff, 'a'}
	// 小写hex（迁移链路ValuerBytes实际产物）
	assert.Equal(t, "X'0001ff61'", SQLValueBytes(encodeHexLower(raw)))
	// 大写hex同样识别为二进制字面量，输出保持原形态
	assert.Equal(t, "X'0001FF61'", SQLValueBytes(strings.ToUpper(encodeHexLower(raw))))
	// 非hex形态退化为字符串转义
	assert.Equal(t, `'not-hex'`, SQLValueBytes("not-hex"))
	assert.Equal(t, "NULL", SQLValueBytes(nil))
	// 空hex无长度，无法与空串区分，退化为字符串字面量
	assert.Equal(t, "''", SQLValueBytes(""))
}

func encodeHexLower(b []byte) string {
	const digits = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(digits[c>>4])
		sb.WriteByte(digits[c&0x0f])
	}
	return sb.String()
}

// IsHexString 边界：空串/奇数长度/非hex字符均判否
func TestIsHexString(t *testing.T) {
	assert.False(t, IsHexString(""))
	assert.False(t, IsHexString("abc"))
	assert.False(t, IsHexString("ag"))
	assert.True(t, IsHexString("abcd"))
	assert.True(t, IsHexString("ABCD0189"))
}
