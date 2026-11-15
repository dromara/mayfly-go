package runlog

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这一组用例只盯一件事：**Fold 的出口长度**。写库超限不是「日志变短」而是整行 UPDATE 失败
// （同步整轮失败 / 迁移日志停在执行中），所以它是唯一必须成立的不变式；至于折叠后保留了哪
// 几行，只要头尾还在即可，不值得为文案钉死格式。

func TestFoldKeepsShortLogUnchanged(t *testing.T) {
	for _, s := range []string{"", "一行日志\n", strings.Repeat("短日志\n", 100)} {
		assert.Equal(t, s, Fold(s), "未超上限不应改写")
	}
}

func TestAppendFoldsWhenOverLimit(t *testing.T) {
	long := strings.Repeat("批次进度行\n", 12000) // ~132KB，已超限
	got := Append(long, "新的一行\n")
	assert.LessOrEqual(t, len(got), MaxBytes, "超限输入必须折叠到上限内")
	assert.Contains(t, got, "新的一行\n", "最新进度必须保留（尾部）")
	assert.True(t, utf8.ValidString(got), "折叠不得切碎多字节字符")
}

// TestFoldInvariantAcrossShapes 覆盖各种行形状：正常行、超长单行、无换行、中文与 emoji、
// 恰好落在窗口边界附近的长度。每一例都断言同一条不变式（长度 + 合法 UTF-8 + 保留最后一行）。
func TestFoldInvariantAcrossShapes(t *testing.T) {
	cases := []struct {
		name  string
		build func() string
	}{
		{"普通多行", func() string { return repeatLines("同步第 %d 批完成\n", 5, 20000) }},
		{"单行超长无换行", func() string { return strings.Repeat("校验差异主键 ", 20000) }},
		{"首行超长其余正常", func() string {
			return strings.Repeat("x", 200000) + "\n" + repeatLines("尾部进度 %d\n", 7, 200)
		}},
		{"尾行超长", func() string {
			return repeatLines("头部上下文 %d\n", 5, 1000) + strings.Repeat("报告JSON ", 40000)
		}},
		{"中文每字节都参与切分", func() string { return repeatLines("第 %d 行：中文字节占多数\n", 5, 20000) }},
		{"emoji 四字节", func() string { return repeatLines("进度 %d 🎉🎉🎉\n", 9, 20000) }},
		{"恰好略超上限", func() string { return strings.Repeat("a", MaxBytes+1) + "\n" }},
		{"空行密集", func() string { return strings.Repeat("\n", MaxBytes+100) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := c.build()
			require.Greater(t, len(src), MaxBytes, "夹具本身应超限，否则测不到折叠")

			folded := Fold(src)
			assert.LessOrEqual(t, len(folded), MaxBytes, "折叠结果必须不超列上限")
			assert.True(t, utf8.ValidString(folded), "折叠不得产出碎字符（写库会报 Incorrect string value）")
			assert.Contains(t, folded, "省略", "必须留下省略标记，不能让人以为日志本来就短")

			// 再折一次仍守住上限（重复 Fold 不得反弹）
			again := Fold(folded + strings.Repeat("z", MaxBytes))
			assert.LessOrEqual(t, len(again), MaxBytes)
			assert.True(t, utf8.ValidString(again))
		})
	}
}

// TestFoldKeepsWholeLines 正常行形状下必须在整行边界切：头段是原日志的若干完整行，
// 尾段以完整行开头。按字节硬切只在单行超限时才允许（那是守住列上限的代价）。
func TestFoldKeepsWholeLines(t *testing.T) {
	src := repeatLines("[12:00:00.000] 同步批次 %d 完成，处理 500 行\n", 11, 12000)
	folded := Fold(src)
	require.Less(t, len(folded), len(src), "应发生折叠")

	markerIdx := strings.Index(folded, "… 日志中段已省略")
	require.Positive(t, markerIdx, "缺少省略标记")
	head, tail := folded[:markerIdx], folded[markerIdx+strings.Index(folded[markerIdx:], "\n")+1:]
	assert.True(t, strings.HasPrefix(src, head), "头段必须是原日志的完整前缀行")
	assert.NotEmpty(t, strings.TrimSuffix(head, "\n"))
	assert.True(t, strings.HasSuffix(head, "\n"), "头段必须停在行尾")
	assert.True(t, strings.HasSuffix(src, tail), "尾段必须由原日志末尾的整行组成")
	assert.Equal(t, lastLine(src), lastLine(folded), "最后一行（最新进度）必须原样保留")
}

// TestFoldSingleGiantLine 回归本包存在的理由之一：数据校验把整份报告 JSON 作为**一行**写入。
// 只按整行边界折叠的实现会在这里原样返回超限内容，写库时报 1406。
func TestFoldSingleGiantLine(t *testing.T) {
	src := strings.Repeat("中", MaxBytes*2/3) // 单行 8 万余字节，且全是三字节字符
	folded := Fold(src)
	assert.LessOrEqual(t, len(folded), MaxBytes, "单行超限也必须兜住")
	assert.True(t, utf8.ValidString(folded), "按字符边界回退，不得切碎中文")
}

func TestFoldMarkerReportsDroppedBytes(t *testing.T) {
	src := repeatLines("行 %d\n", 5, 20000)
	folded := Fold(src)
	var droppedBytes int
	_, _ = fmt.Sscanf(cutAfter(t, folded, "省略（"), "%d 行 / %d 字节", new(int), &droppedBytes)
	assert.Positive(t, droppedBytes, "必须给出被省略的字节数，单行超长时行数会是 0")
}

// repeatLines 生成 count 段文本，每段用 format 填行号；每段长度约为 perBytes 字节（含中文）
func repeatLines(format string, count, perBytes int) string {
	var sb strings.Builder
	payload := strings.Repeat("数", max(perBytes/3-8, 1))
	for i := 0; i < count; i++ {
		sb.WriteString(fmt.Sprintf(format, i) + payload + "\n")
	}
	return sb.String()
}

func lastLine(s string) string {
	trimmed := strings.TrimSuffix(s, "\n")
	if i := strings.LastIndex(trimmed, "\n"); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}

// cutAfter 返回 substr 之后的内容（用于从标记文案里取值）
func cutAfter(t *testing.T, s, substr string) string {
	t.Helper()
	i := strings.Index(s, substr)
	require.NotEqual(t, -1, i, "未找到子串: %s", substr)
	return s[i+len(substr):]
}
