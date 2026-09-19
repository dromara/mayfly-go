package postgres

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// libpq的keyword/value格式：值含空白时必须单引号包裹，值中的单引号与反斜杠需转义。
// 原实现未转义，密码含空格/特殊字符时会导致DSN解析错误或连接参数被截断
func TestEscapeLibpqParam(t *testing.T) {
	// 普通值原样返回（不加引号）
	assert.Equal(t, "localhost", escapeLibpqParam("localhost"))
	assert.Equal(t, "my_db", escapeLibpqParam("my_db"))

	// 含空格需包裹
	assert.Equal(t, "'my pass word'", escapeLibpqParam("my pass word"))

	// 含tab/换行同样包裹
	assert.Equal(t, "'a\tb'", escapeLibpqParam("a\tb"))
	assert.Equal(t, "'a\nb'", escapeLibpqParam("a\nb"))

	// 单引号转义（值含空白判定命中因含单引号，需包裹）
	assert.Equal(t, `'it\'s'`, escapeLibpqParam(`it's`))

	// 反斜杠转义
	assert.Equal(t, `'a\\b'`, escapeLibpqParam(`a\b`))

	// 单引号与反斜杠混合：先转反斜杠再转单引号
	assert.Equal(t, `'p\\\'w'`, escapeLibpqParam(`p\'w`))

	// 空串原样返回
	assert.Equal(t, "", escapeLibpqParam(""))
}
