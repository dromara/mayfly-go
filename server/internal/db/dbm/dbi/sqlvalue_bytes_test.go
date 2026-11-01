package dbi

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBinaryValueTwoPathsAgree 二进制列的中间值经「文本拼接」与「参数绑定」两条路径必须落同一字节序列。
//
// 历史形态：SQLValueBytes 对非 hex 文本退回普通字符串字面量（MySQL 隐式转字节、PG 报错），
// 而 BindValueBytes 直接报错 —— 同一任务换个冲突策略就在"脏写"与"失败"之间漂移。
// 现在两路径统一为「hex 解成字节 / 非 hex 取字面字节」。
func TestBinaryValueTwoPathsAgree(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want []byte
	}{
		{"[]byte", []byte("abc"), []byte("abc")},
		{"hex 字符串", "616263", []byte("abc")},
		{"非 hex 字符串", "not-hex-text", []byte("not-hex-text")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			text := SQLValueBytes(c.in)
			require.True(t, strings.HasPrefix(text, "X'") && strings.HasSuffix(text, "'"),
				"二进制列必须产出 X'..' 字面量，实际: %s", text)
			textBytes, err := hex.DecodeString(text[2 : len(text)-1])
			require.NoError(t, err)

			bound, err := BindValueBytes(c.in)
			require.NoError(t, err)
			boundBytes, ok := bound.([]byte)
			require.True(t, ok)
			require.Equal(t, textBytes, boundBytes, "两条路径必须落同一字节序列")
			require.Equal(t, c.want, textBytes, "非 hex 必须按字面字节写入")
		})
	}

	// 空串保持历史形态（空 hex 与空串文本上不可区分）
	require.Equal(t, "''", SQLValueBytes(""))
}
