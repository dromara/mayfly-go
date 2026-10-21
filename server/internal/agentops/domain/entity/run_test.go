package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// allRunStatuses 声明的全部运行状态（新增状态必须同时归类到终态或非终态，否则本用例失败）。
// 这里不得登记「声明了但没人写入」的状态：那样的状态进不了库，却会在前端筛选项与
// 治理分布里各占一个永远为空的位置（waiting_approval 曾如此，已删；写入点的守卫见
// application.TestEveryRunStatusHasRuntimeReference）
var allRunStatuses = []string{
	RunStatusQueued, RunStatusRunning,
	RunStatusSucceeded, RunStatusFailed, RunStatusCanceled,
}

// ActiveRunStatuses 必须是 IsTerminal 的严格补集。
//
// 为什么单独钉这条：取消白名单、急停清场、重叠防护、治理分布都改读这一个集合，集合与
// IsTerminal 一旦漂移（例如新增终态忘了归类），就会出现「已终态的运行仍被当成在途去取消」
// 或「在途运行不在清场范围」这类静默失真——正是本轮修掉的那类问题，不能靠人记忆维护
func TestActiveRunStatusesComplementIsTerminal(t *testing.T) {
	active := make(map[string]bool, len(ActiveRunStatuses))
	for _, s := range ActiveRunStatuses {
		active[s] = true
	}

	for _, s := range allRunStatuses {
		if active[s] && IsTerminal(s) {
			t.Fatalf("status %q 同时在 ActiveRunStatuses 与终态判定里，状态机分类冲突", s)
		}
		if !active[s] && !IsTerminal(s) {
			t.Fatalf("status %q 既不在 ActiveRunStatuses 也不是终态，会同时逃过取消/清场/重叠防护判定", s)
		}
	}
}

// TerminalRunStatuses 必须恰好是 IsTerminal==true 的全集。
//
// 为什么单独钉：这个集合只服务「需要列出全部终态」的地方（SQL 的 IN、看板分布键、成功率
// 分母）。新增终态时若只改了 IsTerminal 忘了改集合，编译与界面都不红，只表现为该状态的运行
// 不进分布、不进分母、却进 RunTotal——ΣRunByStatus 与 RunTotal 自相矛盾，运维看到的成功率是虚高的
func TestTerminalRunStatusesMatchesIsTerminal(t *testing.T) {
	inSet := make(map[string]bool, len(TerminalRunStatuses))
	for _, s := range TerminalRunStatuses {
		if inSet[s] {
			t.Fatalf("终态集合里 %q 重复，成功率分母会重复计入", s)
		}
		inSet[s] = true
		if !IsTerminal(s) {
			t.Fatalf("%q 在 TerminalRunStatuses 里但 IsTerminal 为 false，集合与判定漂移", s)
		}
	}
	for _, s := range allRunStatuses {
		if IsTerminal(s) && !inSet[s] {
			t.Fatalf("终态 %q 未列入 TerminalRunStatuses，它不会出现在看板分布与成功率分母里", s)
		}
	}
}

// TestApprovedActionsRoundTrip 批准参数必须随运行自持（存进 Extra 并能原样读回）。
//
// 若只存在入队参数里，工作池异步认领执行时就读不到 → 同一次入队走同步/异步两条路径会得到
// 不同的审批凭据（一条能执行、一条挂在审批上），是典型的双真源。
// 空 map 不写入：否则 Extra 里会留一个「有键无内容」的假凭据，让人误以为已批准
func TestApprovedActionsRoundTrip(t *testing.T) {
	actions := map[string]string{"MachineCommandExec": `{"command":"true","authCertName":"tVUcRFKwOC"}`}

	run := &Run{}
	run.SetApprovedActions(actions)
	got := run.ApprovedActions()
	if len(got) != len(actions) {
		t.Fatalf("往返后项数变了：want %d, got %d", len(actions), len(got))
	}
	for name, params := range actions {
		if got[name] != params {
			t.Fatalf("工具 %q 的批准参数往返不一致：want %s, got %s", name, params, got[name])
		}
	}

	empty := &Run{}
	empty.SetApprovedActions(nil)
	if empty.ApprovedActions() != nil {
		t.Fatalf("空许可不得写入 Extra，否则留下无法比对却看似已批准的假凭据")
	}

	// 脏数据按未批准处理（宁可不执行，不能误放行）
	dirty := &Run{}
	dirty.SetExtraValue("approvedActions", "not-a-map")
	if dirty.ApprovedActions() != nil {
		t.Fatalf("脏数据必须归为未批准")
	}
}

// TestRunRoleRoundTripViaExtra 运行形态必须经 Extra 落库并能原样读回。
//
// BeforeSave/AfterFind 少挂一行时表现是「列表里形态标签永远消失」，编译与接口都不报错
func TestRunRoleRoundTripViaExtra(t *testing.T) {
	run := &Run{Title: "慢查询巡检", RunRole: RunRoleOrchStep}
	require.NoError(t, run.BeforeSave(nil))
	require.Equal(t, RunRoleOrchStep, run.GetExtraString(ExtraKeyRunRole), "形态须按键增量写进 Extra")

	restored := &Run{}
	restored.ExtraData = run.ExtraData
	require.NoError(t, restored.AfterFind(nil))
	require.Equal(t, RunRoleOrchStep, restored.RunRole)

	// Extra 键必须与对外 json 字段名一致：改常量标识符不会自动改它的字符串值，
	// 两者一旦分叉，前端按 runRole 读永远是 undefined（回填用例正是这么抓到的）
	require.Equal(t, "runRole", ExtraKeyRunRole, "Extra 键必须等于 Run.RunRole 的 json tag")

	// 普通运行不写形态：空值是「未声明」而不是一个角色，不能凭空造出 `{}`
	plain := &Run{Title: "t"}
	require.NoError(t, plain.BeforeSave(nil))
	require.Nil(t, plain.GetExtraVal(ExtraKeyRunRole), "空形态不得写入 Extra")
}

// TestDetectLegacyRunRoleTitle 存量标题的形态识别必须覆盖全部历史前缀且互不误判。
//
// 这张表只服务存量数据；它一旦漏项或顺序错，老运行会被归成别的角色（`[编排步骤 ` 排在
// `[编排] ` 之后就会全部命中父运行），而新代码已不再按标题分类，没人会再发现它错了
func TestDetectLegacyRunRoleTitle(t *testing.T) {
	cases := []struct {
		title     string
		wantRole  string
		wantTitle string
	}{
		{"[编排步骤 3] 清理临时表", RunRoleOrchStep, "清理临时表"},
		{"[编排规划] 定位慢查询", RunRoleOrchPlan, "定位慢查询"},
		{"[编排审查] 定位慢查询", RunRoleOrchReview, "定位慢查询"},
		{"[编排] 定位慢查询", RunRoleOrchParent, "定位慢查询"},
		{"[评测回放] 巡检", RunRoleEvalReplay, "巡检"},
		{"[执行] 重启 nginx", RunRoleExec, "重启 nginx"},
		{"[校验] 重启 nginx", RunRoleVerify, "重启 nginx"},
		// 未带前缀：普通运行，标题一字不动
		{"每日运维日报", "", "每日运维日报"},
		{"", "", ""},
	}
	for _, c := range cases {
		role, bare := DetectLegacyRunRoleTitle(c.title)
		require.Equalf(t, c.wantRole, role, "标题 %q 的形态识别错了", c.title)
		require.Equalf(t, c.wantTitle, bare, "标题 %q 去前缀结果错了", c.title)
	}
}

// TestLegacyRunRolePrefixesAreReachable 历史前缀表里不得留死项，也不得漏掉已声明的角色。
//
// 死项（库里从未出现过的形态）会被误当成「已覆盖历史」而让后续判断走偏；漏项则让该角色的
// 存量行永远识别不出形态
func TestLegacyRunRolePrefixesAreReachable(t *testing.T) {
	declared := map[string]bool{
		RunRoleEvalReplay: true, RunRoleOrchParent: true, RunRoleOrchPlan: true,
		RunRoleOrchStep: true, RunRoleOrchReview: true, RunRoleExec: true, RunRoleVerify: true,
	}
	seen := map[string]bool{}
	for _, it := range LegacyRunRoleTitlePrefixes {
		require.Truef(t, declared[it.Role], "历史前缀表引用了未声明的角色 %q", it.Role)
		require.Falsef(t, seen[it.Role], "角色 %q 在历史前缀表里重复", it.Role)
		seen[it.Role] = true
	}
	for role := range declared {
		require.Truef(t, seen[role], "角色 %q 没有对应的历史标题前缀（要么补表，要么它本就不是标题形态）", role)
	}
}
