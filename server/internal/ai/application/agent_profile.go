package application

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/ai/tools"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
)

// maxStepKeySep Profile 缓存中「步数变体」的复合键分隔符：
// 同一 Profile 按不同 MaxStep 装配为独立实例（eino 工具/步数构造后不可刷新），
// 缓存键 = profileId + maxStepKeySep + N；ResetProfile 按此前缀一并失效。
const maxStepKeySep = "#maxstep="

// 内置 Profile ID 契约常量下沉至中立包 headless（供 flow/agentops 等消费方引用，
// 破除循环依赖）；此处以别名再导出，保持 application 包内既有引用习惯。
const (
	ProfileDiagnostician = headless.ProfileDiagnostician
	ProfileExecutor      = headless.ProfileExecutor
	ProfileReviewer      = headless.ProfileReviewer
	ProfileAuditor       = headless.ProfileAuditor
)

// AgentProfile 声明式 Agent 装配档
//
// 一个 Profile 装配为一个独立的进程内缓存 Agent 实例（尊重 eino「工具清单构造后
// 不可刷新」硬约束）；Profile/工具配置变更时经 ResetProfile 触发懒重建。
//
// 字段语义：
//   - Instruction 为空时回退内置 system_prompt.md（agent.SystemInstruction）
//   - ToolPolicy 为 nil 表示能力维度不限制（全能力放行）；只读档传 tools.ReadOnlyPolicy()
//   - MaxStep 为 0 时使用 NewAgent 默认步数
//   - RequireRunGrant 为 true 时该档必须由一次性运行许可驱动（见 tools.RunGrant）：
//     许可清单外的工具一律拒绝，没带许可则什么工具都不许调——受控执行档用它把
//     「只执行已批准的那一个动作」做成机制而不是提示词约束
type AgentProfile struct {
	Id          string
	Name        string
	Description string
	Instruction string
	ToolPolicy  *tools.ToolPolicy
	MaxStep     int
	// RequireRunGrant 本档运行必须携带一次性工具许可（无许可即拒绝一切工具调用）
	RequireRunGrant bool
}

var (
	profileMu sync.RWMutex
	// profiles Profile 定义注册表（内置档在此登记；业务可 RegisterProfile 追加）
	profiles = map[string]*AgentProfile{
		ProfileDiagnostician: {
			Id:          ProfileDiagnostician,
			Name:        "诊断专家",
			Description: "只读诊断档：仅可调用只读工具排查问题、产出诊断报告；任何变更操作须登记为变更提案并经人工审批",
			Instruction: diagnosticianInstruction,
			ToolPolicy:  tools.ReadOnlyPolicy(),
		},
		ProfileExecutor: {
			Id:          ProfileExecutor,
			Name:        "受控执行",
			Description: "受控执行档：仅在变更提案经人工审批后由回调一次性调用，且只能执行许可（run grant）列出的那一个动作",
			Instruction: executorInstruction,
			// 不按能力收窄（要能执行 ExecSql/MachineCommandExec 等高危动作），但必须由一次性许可驱动：
			// 每次运行由提案网关把「已批准的 action」解析成工具名下发。没有许可、或许可清单外的工具
			// 全部被护栏拒绝——旧形态在这里传 nil 策略且不挂护栏，「参数锁定」只剩提示词
			ToolPolicy:      nil,
			RequireRunGrant: true,
		},
		ProfileReviewer: {
			Id:          ProfileReviewer,
			Name:        "结果审查",
			Description: "审查档：只读复查执行结果、判定目标是否达成、给出回滚建议",
			Instruction: reviewerInstruction,
			ToolPolicy:  tools.ReadOnlyPolicy(),
		},
		ProfileAuditor: {
			Id:          ProfileAuditor,
			Name:        "工单审批",
			Description: "flow AI 审批节点承接档：只读依据审批规则对工单给出 allowExecute/suggestion 结论",
			// 审批档指令由 node_aitask 按审批规则动态注入（system 消息），此处静态指令留空回退默认
			ToolPolicy: tools.ReadOnlyPolicy(),
		},
	}

	// profileAgents Profile → 已装配 Agent 实例缓存（懒构建，ResetProfile 失效）
	profileAgents = map[string]*agent.Agent{}
)

// 内置档角色指令（简洁自持；审批档动态指令由调用方经 system 消息注入）
const (
	diagnosticianInstruction = `你是一名资深运维诊断专家（只读诊断模式）。
职责：使用只读工具排查问题、定位根因、产出结构化诊断报告。
硬性约束：
1. 你没有任何变更权限，禁止尝试执行写操作/高危操作（SQL 变更、机器命令等）；
2. 若诊断结论需要变更（重启、清理、改配置、执行 SQL 等），必须登记为「变更提案」，说明操作、参数、理由与风险，交由人工审批后执行；
3. 诊断报告应包含：现象、排查过程、根因判断、影响面、建议措施（区分只读可立即做的 与 需审批的变更）。`

	executorInstruction = `你是受控执行代理。你仅在变更提案经人工审批通过后被调用，且调用参数已锁定为审批通过的提案参数。
职责：严格按锁定参数执行既定操作，不得自行扩大范围或改动参数；执行后如实回报结果（成功/失败/关键输出）。`

	reviewerInstruction = `你是运维结果审查专家（只读）。
职责：在变更执行后复查目标资源状态/指标，判定预期目标是否达成，识别异常，并在未达成或出现副作用时给出回滚建议。
约束：仅使用只读工具，不得执行任何变更。`
)

// RegisterProfile 注册/覆盖 Profile 定义（业务扩展档；同名覆盖并失效其缓存实例）
func RegisterProfile(p *AgentProfile) {
	if p == nil || p.Id == "" {
		return
	}
	profileMu.Lock()
	defer profileMu.Unlock()
	profiles[p.Id] = p
	delete(profileAgents, p.Id)
}

// GetProfile 查询 Profile 定义（不存在返回 nil, false）
func GetProfile(id string) (*AgentProfile, bool) {
	profileMu.RLock()
	defer profileMu.RUnlock()
	p, ok := profiles[id]
	return p, ok
}

// ListProfiles 列出全部 Profile 定义（副本，避免外部改内部注册表）
func ListProfiles() []*AgentProfile {
	profileMu.RLock()
	defer profileMu.RUnlock()
	result := make([]*AgentProfile, 0, len(profiles))
	for _, p := range profiles {
		cp := *p
		result = append(result, &cp)
	}
	return result
}

// profileCatalogImpl headless.ProfileCatalog 端口的适配器：把注册表以对外元信息形式暴露，
// 供 agentops 等消费方装配表单选项与校验档位（不致于让消费方 import 本包）
type profileCatalogImpl struct{}

func (profileCatalogImpl) List() []headless.ProfileInfo {
	defs := ListProfiles()
	out := make([]headless.ProfileInfo, 0, len(defs))
	for _, p := range defs {
		out = append(out, headless.ProfileInfo{
			Id:              p.Id,
			Name:            p.Name,
			Description:     p.Description,
			RequireRunGrant: p.RequireRunGrant,
		})
	}
	// 注册表是 map，遍历顺序不定；下拉与校验结果需要稳定顺序
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out
}

func (profileCatalogImpl) Find(id string) (headless.ProfileInfo, bool) {
	p, ok := GetProfile(id)
	if !ok {
		return headless.ProfileInfo{}, false
	}
	return headless.ProfileInfo{
		Id:              p.Id,
		Name:            p.Name,
		Description:     p.Description,
		RequireRunGrant: p.RequireRunGrant,
	}, true
}

// GetProfileAgent 获取 Profile 对应的 Agent 实例（进程内缓存 + 懒构建，并发安全）
//
// 每个 Profile 装配为独立 Agent 实例（AgentId=ProfileId，注入工具贡献上下文，
// 作为条件工具组过滤点）；共享默认 Registry/ContextManager/CheckPointStore
// （经 AssembleDefault 缓存），差异化仅在指令/工具策略/步数。
// 构建失败不缓存，下次调用重试。
func GetProfileAgent(ctx context.Context, profileId string) (*agent.Agent, error) {
	return GetProfileAgentWithMaxStep(ctx, profileId, 0)
}

// GetProfileAgentWithMaxStep 获取 Profile 在指定最大步数下的 Agent 实例。
//
// maxStep<=0 时用 Profile 默认步数（缓存键即 profileId）；maxStep>0 时装配为独立
// 「步数变体」实例并以复合键缓存（同一 Profile 的不同步数互不干扰）。尊重 eino
// 「工具/步数构造后不可刷新」硬约束：变体一经装配即缓存，ResetProfile 按前缀一并失效。
func GetProfileAgentWithMaxStep(ctx context.Context, profileId string, maxStep int) (*agent.Agent, error) {
	cacheKey := profileId
	if maxStep > 0 {
		cacheKey = profileId + maxStepKeySep + strconv.Itoa(maxStep)
	}

	profileMu.RLock()
	cached, ok := profileAgents[cacheKey]
	profile, hasProfile := profiles[profileId]
	profileMu.RUnlock()
	if ok && cached != nil {
		return cached, nil
	}
	if !hasProfile || profile == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrUnknownAgentProfile, "profileId", profileId)
	}

	profileMu.Lock()
	defer profileMu.Unlock()
	// 双检：并发下可能已被其它 goroutine 构建
	if cached, ok := profileAgents[cacheKey]; ok && cached != nil {
		return cached, nil
	}

	ag, err := assembleProfileAgent(ctx, profile, maxStep)
	if err != nil {
		return nil, err
	}
	profileAgents[cacheKey] = ag
	logx.InfofContext(ctx, "[agent-profile] assembled profile agent: id=%s, maxStep=%d, tools_policy_restricted=%v, require_run_grant=%v",
		profileId, maxStep, profile.ToolPolicy.IsRestricted(), profile.RequireRunGrant)
	return ag, nil
}

// assembleProfileAgent 按 Profile 定义（+ 可选步数覆盖）装配一个全新 Agent 实例。
// 调用方须持有 profileMu 写锁（保证缓存双检与装配的原子性）。
func assembleProfileAgent(ctx context.Context, profile *AgentProfile, maxStep int) (*agent.Agent, error) {
	opts := []agent.Option{
		agent.WithId(profile.Id),
		agent.WithName(profile.Name),
		agent.WithDescription(profile.Description),
		agent.WithToolPolicy(profile.ToolPolicy),
		agent.WithRequireRunGrant(profile.RequireRunGrant),
	}
	if profile.Instruction != "" {
		opts = append(opts, agent.WithInstruction(profile.Instruction))
	}
	// 步数：显式覆盖优先，否则用 Profile 默认（均为 0 时由 NewAgent 取内置默认）
	step := maxStep
	if step <= 0 {
		step = profile.MaxStep
	}
	if step > 0 {
		opts = append(opts, agent.WithMaxStep(step))
	}
	return agent.NewAgent(ctx, opts...)
}

// ResetProfile 失效指定 Profile 的缓存实例（下次 GetProfileAgent 懒重建）
//
// 供 Profile 定义变更、工具配置变更（对齐 ResetDefaultAgent 模式）后刷新工具清单。
// 一并清除该 Profile 的全部「步数变体」缓存，避免定义变更后旧变体残留。
func ResetProfile(profileId string) {
	profileMu.Lock()
	defer profileMu.Unlock()
	delete(profileAgents, profileId)
	prefix := profileId + maxStepKeySep
	for k := range profileAgents {
		if strings.HasPrefix(k, prefix) {
			delete(profileAgents, k)
		}
	}
}

// ResetAllProfiles 失效全部 Profile 缓存实例（工具/插件全局变更后统一刷新）
func ResetAllProfiles() {
	profileMu.Lock()
	defer profileMu.Unlock()
	profileAgents = map[string]*agent.Agent{}
}
