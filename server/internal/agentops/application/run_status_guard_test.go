package application

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"mayfly-go/internal/agentops/domain/entity"

	"github.com/stretchr/testify/require"
)

// 运行状态机的可达性守卫。
//
// 存在理由：状态常量曾在 entity/run.go 里声明过一个 waiting_approval，但**没有任何代码写入它**
// （提案的等待状态由 Proposal 聚合自持，受控执行是审批通过后另起的一条运行，不存在「一条运行
// 卡在审批里」）。后果不是报错，而是两处静默空洞：前端状态筛选多出「待审批」这个永远查不到
// 数据的选项（选它=空列表，看起来像系统没跑出过运行），治理分布多一个永远为零的键。
// 编译、类型检查、lint、既有单测全绿——只有人真的去筛一次才会发现。
//
// 所以把「声明的状态必须被运行时代码用到」钉成守卫：想加状态就得同时给出写入点，
// 否则本用例红。删状态时两侧同步删干净（本用例只拦「声明了没人用」，不拦「用了没声明」，
// 后者编译就过不去）

// TestEveryRunStatusHasRuntimeReference 每个声明的运行状态都必须在运行时源码里被引用
func TestEveryRunStatusHasRuntimeReference(t *testing.T) {
	declared := declaredRunStatuses(t)
	require.NotEmpty(t, declared, "未从 entity/run.go 解析出任何 RunStatus 常量：解析式已失配，本守卫形同虚设")

	// 与两个集合并起来必须完全一致：漏归类的状态既进不了「在途/可取消」判定，也进不了终态分布
	union := make(map[string]bool, len(entity.ActiveRunStatuses)+len(entity.TerminalRunStatuses))
	for _, s := range append(append([]string{}, entity.ActiveRunStatuses...), entity.TerminalRunStatuses...) {
		union[s] = true
	}
	for ident, value := range declared {
		require.Truef(t, union[value], "状态 %q（%s）未归类到 ActiveRunStatuses/TerminalRunStatuses", value, ident)
	}
	require.Lenf(t, union, len(declared), "集合里有 %d 个状态，常量块只声明了 %d 个：集合引用了不存在的状态", len(union), len(declared))

	// 写入点分布在两层：application（入队 queued、终态收敛、编排/评测收尾）与
	// persistence（认领时 queued→running 的条件更新）
	sources := runtimeSources(t, ".", filepath.Join("..", "infra", "persistence"))
	for ident, value := range declared {
		// 布尔断言而非 require.Contains(sources, …)：后者失配时会把拼接好的整包源码
		// 打进失败输出（实测 30KB+），把真正那条信息埋掉
		require.Truef(t, strings.Contains(sources, "entity."+ident),
			"运行状态 %q 在 application/persistence 的运行时源码里没有任何引用：它永远进不了库，"+
				"界面却会把它当成一个可筛选、可展示的状态（waiting_approval 曾如此）", value)
	}
}

// declaredRunStatuses 从 entity/run.go 的常量块解析「标识符 → 状态值」。
//
// 不手写清单：手写那份就是第二个真源，新增状态时漏登记不会红，守卫等于没有
func declaredRunStatuses(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "domain", "entity", "run.go"))
	require.NoError(t, err)
	// 只认顶格 tab 缩进的常量声明行，避免把注释里出现的字面量也算成声明
	re := regexp.MustCompile(`(?m)^\t(RunStatus\w+)\s*=\s*"([^"]+)"`)
	out := make(map[string]string)
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		require.Emptyf(t, out[m[1]], "常量 %s 重复声明", m[1])
		out[m[1]] = m[2]
	}
	return out
}

// runtimeSources 拼接指定目录下全部非测试 Go 源码（测试源码里引用不算写入点：
// 用例可以自己造一个状态值塞进库，那不代表系统真会走到）
func runtimeSources(t *testing.T, dirs ...string) string {
	t.Helper()
	var sb strings.Builder
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		require.NoErrorf(t, err, "读取目录 %s 失败", dir)
		for _, e := range entries {
			name := e.Name()
			if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			raw, err := os.ReadFile(filepath.Join(dir, name))
			require.NoError(t, err)
			sb.WriteString(string(raw))
			sb.WriteByte('\n')
		}
	}
	require.NotEmpty(t, sb.String(), "没读到任何运行时源码，断言会恒真")
	return sb.String()
}
