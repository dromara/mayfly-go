package persistence

import (
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/ai/domain/entity"

	"github.com/stretchr/testify/require"
)

// TestHistoryQueryExcludesNonMessageItemTypes 重建 LLM 上下文的查询必须排除「完全产不出消息」的类型。
//
// 为什么用源码守卫：这条仓储的真调用方（session 历史重建）测的是 fake 仓储，跑不到真实 SQL。
// 不产消息的类型被取回后各占一个 limit 窗口槽位，表现是「用户多停几次、上下文被多压几次，
// 模型逐渐忘掉前面的对话」，很难反查回一条 SQL。
//
// 判据分两层：① SQL 必须由 nonMessageItemTypes 这一个真源驱动（写死字面量列表就是双真源）；
// ② 真源本身要覆盖已知的所有非消息类型。上一版把占位符个数钉成魔数 4，补上 compaction（应为 5）
// 反而会把正确修复判红，所以这里改成对着真源长度断言。
func TestHistoryQueryExcludesNonMessageItemTypes(t *testing.T) {
	source, err := os.ReadFile("turn_item.go")
	require.NoError(t, err, "读取 turn_item.go 失败")
	// 剔除注释再判：函数上方的说明注释里原样写着被排除的类型名，不剔就是拿自己的注释当缺陷
	query := funcSegment(t, stripLineComments(string(source)), "SelectRecentByConvId")

	require.Contains(t, query, "item_type NOT IN", "必须用 NOT IN 排除多个非消息类型；改回 != 单值就会把新登记的类型放进窗口")
	require.Contains(t, query, "nonMessageItemTypes",
		"排除集必须由单一真源变量驱动；在 SQL 或实参里重写一遍类型就是第二真源，新增类型时必漏一处")
	// 占位符与实参必须同源于同一个排除集：上一版直接数源码里的 `?` 个数，而排除集的占位符现在是
	// 运行期生成的（数不到），那条断言当场自己红了 —— 同源不靠数量巧合，靠两个位置都引用同一个列表
	require.Contains(t, query, "nonMessagePlaceholders()", "NOT IN 的占位符必须由排除集长度生成")
	require.Contains(t, query, "args...", "实参必须直接展开那个由排除集构造的参数列表，不得手写逐个对齐")
}

// TestNonMessagePlaceholdersMatchExclusionSet 生成的占位符个数必须等于排除集长度。
//
// 这是真正会出事的那个点：占位符与实参个数错位时，SQL 不报错、只是静默查错东西（比如把 limit
// 当成一个类型绑进 NOT IN），界面表现为「历史里凭空少了几条」且无任何日志
func TestNonMessagePlaceholdersMatchExclusionSet(t *testing.T) {
	marks := strings.Split(nonMessagePlaceholders(), ", ")
	require.Len(t, marks, len(nonMessageItemTypes), "占位符个数与排除集长度不一致")
	for _, mark := range marks {
		require.Equal(t, "?", mark, "占位符串只能由 ? 组成")
	}
}

// TestNonMessageItemTypesCoverKnownKinds 真源自身要覆盖全部已知的非消息类型。
//
// 这三项都是「写进同一张 t_ai_turn_item、但 itemToMessages 没有对应分支」的类型：漏一项就是少排一个、
// 白占一个上下文窗口槽位。新增类型时必须加进 nonMessageItemTypes，本用例负责让它响
func TestNonMessageItemTypesCoverKnownKinds(t *testing.T) {
	require.ElementsMatch(t,
		[]string{entity.ItemTypeReasoning, entity.ItemTypeTurnNotice, entity.ItemTypeCompaction},
		nonMessageItemTypes,
		"非消息类型排除集与实体登记的类型不一致：漏排的类型会占掉 limit 窗口槽位却产不出消息")
}

// funcSegment 取某个方法从签名到下一个「func 」之间的源码；找不到就致命报错，避免扫描式空跑
func funcSegment(t *testing.T, text, methodName string) string {
	t.Helper()
	start := strings.Index(text, "func (t *turnItemRepoImpl) "+methodName)
	if start < 0 {
		t.Fatalf("未找到 %s（改名或搬走必须同步本守卫）", methodName)
	}
	segment := text[start:]
	if next := strings.Index(segment, "\nfunc "); next >= 0 {
		segment = segment[:next]
	}
	return segment
}

// stripLineComments 去掉行注释。
//
// 前提（显式写下）：本 helper 不识别字符串字面量，被扫文件若出现含 `//` 的字符串（URL、SQL 注释）
// 会被拦腰截断；届时应改用 AST 级判据，而不是把断言放宽
func stripLineComments(source string) string {
	var kept []string
	for _, line := range strings.Split(source, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
