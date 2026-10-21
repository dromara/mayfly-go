package dbtool

import (
	"context"
	"os"
	"strings"
	"testing"

	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/db/domain/entity"

	"github.com/stretchr/testify/require"
)

// TestNonEmptyDbNamesDropsPlaceholderItems 兜底产生的空库名项不得算作候选。
//
// 真机形态：sqlite 资产既没配指定库、又连不上去枚举，listDbNamesOfAsset 兜底给出一条 dbName=""
// 的项；不过滤就得到 [""]，len>0 成立 → 输出「真实库名为 []，只能取其中之一」这种谎话提示
func TestNonEmptyDbNamesDropsPlaceholderItems(t *testing.T) {
	items := []dbOptionItem{
		{dbName: "", label: "tokhub"}, // 兜底项
		{dbName: "   ", label: "空白项"}, // 脏数据
		{dbName: "mayfly_go", label: "主库"},
		{dbName: "sys", label: "系统库"},
	}
	require.Equal(t, []string{"mayfly_go", "sys"}, nonEmptyDbNames(items))

	// 只有兜底项时必须返回空切片，上层才会走「无法枚举该实例的库清单」那条正确文案
	require.Empty(t, nonEmptyDbNames([]dbOptionItem{{dbName: "", label: "tokhub"}}))
	require.Empty(t, nonEmptyDbNames(nil))
	require.Empty(t, nonEmptyDbNames([]dbOptionItem{}))
}

// TestCandidateDbNamesGoesThroughFilter 静态守卫：candidateDbNames 必须经过 nonEmptyDbNames。
//
// 为什么用静态断言：这条路径要连上真实资产才能触发，行为用例覆盖不到（上一轮就是这么漏的——
// 加了「空清单不说只能选一个」的分支，却因为候选里混着空串，那个分支从来没被执行过）
func TestCandidateDbNamesGoesThroughFilter(t *testing.T) {
	src, err := os.ReadFile("options.go")
	require.NoError(t, err, "读取 options.go 失败")
	var codeLines []string
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			continue
		}
		codeLines = append(codeLines, line)
	}
	code := strings.Join(codeLines, "\n")

	// 定位 candidateDbNames 函数体，正向断言它调用了 nonEmptyDbNames——而不是锚定某句具体绕过代码文本：
	// 后者换一种绕过写法就漏网、一次合法重构就误红，正是 db_names.go 自身注释批评过的措辞型守卫
	start := strings.Index(code, "func candidateDbNames(")
	require.Greater(t, start, 0, "candidateDbNames 必须存在")
	body := code[start:]
	if nl := strings.Index(body, "\nfunc "); nl >= 0 {
		body = body[:nl]
	}
	require.Contains(t, body, "nonEmptyDbNames(",
		"candidateDbNames 的返回必须经 nonEmptyDbNames 过滤，否则兜底空项会混进候选清单")
}

// TestAssignModeDbNamesAreConnectableNames 「指定库名」模式的候选必须是可连接的真实库名，
// 且按空格分隔逐个展开成独立候选。
//
// 这与机器侧「候选里给了机器 code 而不是凭证名」是同一类缺陷：资产显示名不能用于建连，用户在
// 补全卡片里选中照样失败。该规则此前只有 options.go 里的一句注释在撑——注释会腐化，断言不会
func TestAssignModeDbNamesAreConnectableNames(t *testing.T) {
	r := &resource.Resource{
		Name: "本地mysql",
		Extra: map[string]any{
			resource.ExtraKeyGetDatabaseMode: entity.DbGetDatabaseModeAssign,
			resource.ExtraKeyDatabase:        "mayfly_go sys",
		},
	}

	items := listDbNamesOfAsset(context.Background(), r)
	require.Len(t, items, 2, "配置了多个库名时每个都必须是独立候选")
	require.Equal(t, "mayfly_go", items[0].dbName)
	require.Equal(t, "sys", items[1].dbName)
	for _, item := range items {
		require.NotEqual(t, r.Name, item.dbName, "资产显示名不是可连接库名，选中之后的建连必然失败")
		require.Contains(t, item.label, item.dbName, "多库时 label 必须带库名，否则用户无从区分")
	}
}
