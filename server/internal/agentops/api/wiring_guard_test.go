package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// 归属/可见面断言的接线守卫。
//
// 存在理由：本模块的越权防线全部是「写入口取旧行 + assertResourceOwned / mustGetVisible」的
// 接线形态，一次重构（挪函数、改签名、复制入口）就可能把断言静默摘掉——编译、单测、界面
// 全都不会红（功能照常，只是任何 id 都能操作他人资源）。这类缺陷已在本模块真实发生过
// （读侧收窄、写侧漏拦被一轮全面审查抓出 6 处），因此把「每个写入口必须带断言」钉成守卫。

// readSource 读取本包源文件
func readSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(".", name))
	require.NoError(t, err)
	return string(b)
}

// funcBody 截取以 prefix 开头的函数体（到下一个顶格 func/} 收尾的粗略形态，够接线断言用）
func funcBody(t *testing.T, source, prefix string) string {
	t.Helper()
	i := strings.Index(source, prefix)
	require.GreaterOrEqual(t, i, 0, "函数 %s 不存在（改名/删除必须同步本守卫）", prefix)
	rest := source[i:]
	// 函数以「\n}」结尾（Go 顶层函数闭合括号顶格）
	end := strings.Index(rest, "\n}")
	require.Greater(t, end, 0)
	return rest[:end]
}

func TestTaskSaveGuardsOwnership(t *testing.T) {
	body := funcBody(t, readSource(t, "agent_task.go"), "func (a *AgentTask) Save(")
	require.Contains(t, body, "assertResourceOwned", "任务 Save 的更新分支必须断言归属（否则可整列覆盖他人任务并把 owner 改成自己）")
}

func TestTriggerSaveGuardsOwnershipAndTaskBinding(t *testing.T) {
	body := funcBody(t, readSource(t, "trigger.go"), "func (t *Trigger) Save(")
	require.Contains(t, body, "assertResourceOwned", "触发器 Save 必须断言归属")
	// 关联任务模式：绑定任务必须对调用者可见，否则可借免鉴权 webhook 以他人（admin）身份驱动运行
	require.Contains(t, body, "agentTaskApp.GetById", "触发器 Save 必须读取绑定任务以做可见归属断言")
	require.GreaterOrEqual(t, strings.Count(body, "assertResourceOwned"), 2, "任务绑定与更新归属是两条独立断言，缺一不可")
}

func TestRunWriteEntriesGuardVisibility(t *testing.T) {
	source := readSource(t, "run.go")
	for _, fn := range []string{
		"func (r *Run) Cancel(",
		"func (r *Run) Feedback(",
		"func (r *Run) ReplayEval(",
		"func (r *Run) GetById(",
	} {
		require.Contains(t, funcBody(t, source, fn), "mustGetVisible", "%s 必须经可见面断言（application 层签名不带调用者身份，漏接就等于知道 id 即可操作）", fn)
	}
	helper := funcBody(t, source, "func (r *Run) mustGetVisible(")
	require.Contains(t, helper, "canQueryAllRuns")
	require.Contains(t, helper, "runCallerId")
}

// TestProposalWriteEntriesGuardOwnership 提案的详情与删除必须断言归属。
//
// 为什么单独钉：提案带着「谁批准了什么高危变更」的审计证据，删掉它就是销毁证据。此前守卫
// 只钉了 Task/Trigger/Run 三类，proposal.go 里的两条 assertResourceOwned 摘掉后编译、单测、
// 界面全绿（同类「读侧已收窄、写侧漏拦」形态的第 4~5 处）
func TestProposalWriteEntriesGuardOwnership(t *testing.T) {
	source := readSource(t, "proposal.go")
	for _, fn := range []string{
		"func (p *Proposal) GetById(",
		"func (p *Proposal) Delete(",
	} {
		require.Contains(t, funcBody(t, source, fn), "assertResourceOwned",
			"%s 必须断言归属（否则任何持提案权限的账号可读/销毁他人高危变更的审计记录）", fn)
	}
	// 列表必须按调用者收窄，且覆写发生在真正查询之前（先查后覆写等于没收窄）。
	// 锚点取仓储查询调用而不是函数名：funcBody 从函数名处开始截，拿「GetPageList(」当锚会先
	// 匹配到函数签名自身，位置永远小于收窄行，断言恒真、没有牙
	listBody := funcBody(t, source, "func (p *Proposal) GetPageList(")
	require.Contains(t, listBody, "scopeOwnerToCaller", "提案列表必须按调用者收窄可见面")
	require.Less(t, strings.Index(listBody, "scopeOwnerToCaller"), strings.Index(listBody, "proposalApp.GetPageList("),
		"收窄必须在查询前生效：查询后才覆写 owner 条件等于没拦")
}

// TestReadSideEntriesScopeToCaller 任务/触发器列表同样不得跨账号全量可读：
// 不收窄时非全局可见者虽改不了他人资源，却能读到其完整指令、调度配置与绑定关系。
func TestReadSideEntriesScopeToCaller(t *testing.T) {
	for _, tc := range []struct{ file, fn string }{
		{"agent_task.go", "func (a *AgentTask) GetPageList("},
		{"trigger.go", "func (t *Trigger) GetPageList("},
	} {
		body := funcBody(t, readSource(t, tc.file), tc.fn)
		require.Contains(t, body, "scopeOwnerToCaller", "%s 必须按调用者收窄列表可见面", tc.fn)
	}
}

// TestTriggerTaskNamesFilledOnBothReadEntries 触发器的关联任务名必须在列表与详情两个读入口都补上。
//
// 只补列表的表现：列表能显示任务名，而打开编辑抽屉（走详情接口）时「关联任务」下拉找不到
// 匹配 option、退化成直接显示裸 id——同一个字段在两个界面上给出两种可信度。本仓已多次踩
// 「同一份数据的两条路径只改一条」（Enqueue/RunSync、两个表单默认值），故钉住两个入口
func TestTriggerTaskNamesFilledOnBothReadEntries(t *testing.T) {
	source := readSource(t, "trigger.go")
	for _, fn := range []string{
		"func (t *Trigger) GetPageList(",
		"func (t *Trigger) GetById(",
	} {
		require.Contains(t, funcBody(t, source, fn), "fillTaskNames", "%s 必须补关联任务名，否则该界面上只剩裸 id", fn)
	}
	// 取数收在一处：两处各写一遍就会漂移，且容易一处退化成逐行回查（N+1）
	helper := funcBody(t, source, "func (t *Trigger) fillTaskNames(")
	require.Contains(t, helper, "agentTaskApp.GetByIds", "任务名必须一次批量回查，不得逐行 GetById")
}

// TestGlobalDangerousActionsRecordTheirEffect 全局性危险动作的审计必须记到「产生了什么后果」。
//
// 只记「某人点了这个按钮」的审计等于没记：事后无法回答「当时到底影响了哪几行」。真机查
// t_sys_log 确认过：一键清场那几行的 req_param / resp / extra 三个字段全是空，而同文件的
// 急停开关则记了 engaged 入参——同一份里两种口径，说明是漏写而非有意设计
func TestGlobalDangerousActionsRecordTheirEffect(t *testing.T) {
	source := readSource(t, "governance.go")

	kill := funcBody(t, source, "func (g *Governance) SetKillSwitch(")
	require.Contains(t, kill, "rc.ReqParam", "急停开关必须把 engaged 入参写进审计")

	cancel := funcBody(t, source, "func (g *Governance) CancelAll(")
	require.Contains(t, cancel, "rc.ReqParam",
		"一键清场必须把取消结果写进审计：只记「有人点了」事后无法对账影响了哪几条运行")
	// 审计里的数量必须来自同一次执行结果，不允许审计一套、响应另一套
	require.Less(t, strings.Index(cancel, "CancelAllActive("), strings.Index(cancel, `rc.ReqParam = collx.Kvs("canceled"`),
		"审计结果得取自本次 CancelAllActive 的真实返回值，不得另算一份")
}
