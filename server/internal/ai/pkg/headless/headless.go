package headless

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// headless 无头运行契约包（中立低层，仅依赖标准库 + eino schema）
//
// 存在动机（依赖倒置，破除循环依赖）：
//
//	flow/application → ai/application 会成环 —— ai/application 经
//	ai/application/resource → db/application → db/application/sqlexec 反向依赖
//	flow/application。故将「无头运行」的契约（请求/结果/Runner 端口）下沉到本中立包：
//	  - ai/application 作为适配器实现 Runner 并在 Init 期 Register；
//	  - flow/application（及后续 agentops）作为消费方调用 Run，仅依赖本契约包。
//
// 端口-适配器（Ports & Adapters）：本包定义端口，实现方运行期注入，消费方面向契约编程。

// DefaultConversationTitle 新建会话的默认标题哨兵：调用方未提供标题时由会话应用层兜底写入。
//
// 前端把它当作「尚未命名」的标记，命中时会用首条消息自动命名替换（见 ChatContainer 的 DEFAULT_TITLE），
// 改值需两端同步。无头运行不要自己按 RunKind 编中文标题：标题是一次性落库的展示文本，
// 写死中文会让英文界面永远看到中文（切语言也不变），新增 RunKind 还得回来改 case
const DefaultConversationTitle = "New Chat"

// 内置 Agent Profile ID（无头运行/编排/审批各角色的稳定标识）
const (
	// ProfileDiagnostician 诊断档：只读，产出诊断报告；任何变更须走提案审批
	ProfileDiagnostician = "diagnostician"
	// ProfileExecutor 执行档：受控写，仅在提案经人工审批后由回调一次性调用（参数锁定）
	ProfileExecutor = "executor"
	// ProfileReviewer 审查档：只读，审查执行结果/判定目标达成/给回滚建议
	ProfileReviewer = "reviewer"
	// ProfileAuditor 审批档：flow AI 审批节点（node_aitask）承接，只读产出审批建议
	ProfileAuditor = "auditor"
)

// 无头运行终态（RunResult.Status）
const (
	RunStatusSucceeded = "succeeded"
	RunStatusFailed    = "failed"
	RunStatusTimeout   = "timeout"
	RunStatusCanceled  = "canceled"
	// RunStatusInterrupted 本轮以「等待人工审批/补充参数」的中断挂起结束。
	//
	// 为什么必须单独成一个终态：中断不产生 error（agent 是正常挂起），因此会被
	// resolveRunStatus 归为 succeeded；而无头运行（运维任务、提案受控执行）里
	// 根本没有审批应答者——于是一次写操作都没发生的运行会被记成「已执行」，
	// 提案状态机随之落 executed（安全误导，比记成失败更糟）
	RunStatusInterrupted = "interrupted"
)

// 无头运行来源类型（RunReq.RunKind，审计/统计维度）
const (
	RunKindCron          = "cron"
	RunKindEvent         = "event"
	RunKindManual        = "manual"
	RunKindAudit         = "audit"
	RunKindOrchestration = "orchestration"
)

// RunReq 无头运行请求
//
// 与 WS 对话路径共用同一 Agent 运行时（Agent.Run + TurnRecorder），差异仅在：
// 无前端连接（publisher 传 nil，仅收集持久化不推送）、以服务身份（owner）运行、
// 可选是否落库会话/TurnItem。
type RunReq struct {
	// ProfileId Agent 装配档（空则默认 diagnostician 只读诊断档）
	ProfileId string
	// Instruction 任务级 system 提示词（可选；与 Profile 静态指令叠加，随消息下发）
	Instruction string
	// Input 用户输入（user 消息，可为结构化 JSON 文本）
	Input string
	// TurnId 轮次 ID（空则自动生成）
	TurnId string
	// OwnerAccountId 运行服务身份账号 ID（必填；fail-closed —— 缺省拒绝运行，
	// 使 resource/db/machine/memory 工具的权限过滤严格按 owner 收敛，不越权）
	OwnerAccountId uint64
	// OwnerUsername 运行服务身份账号名（日志/审计展示用）
	OwnerUsername string
	// Timeout 运行超时（>0 时以 ctx 超时约束，超时取消传播到 LLM 请求与工具执行）
	Timeout time.Duration
	// Persist 是否落库 Conversation/TurnItem（获得审计/回放能力；false 为无痕运行）
	Persist bool
	// ConversationId 复用既有会话（Persist 且 >0 时续写该会话；0 则新建）
	ConversationId uint64
	// Title 新建会话标题（Persist 且 ConversationId==0 时用；空则按 RunKind 兜底）
	Title string
	// MaxStep 本次运行的最大步数覆盖（>0 时按此装配独立 Agent 实例；0=用 Profile 默认步数）
	MaxStep int
	// MaxTotalTokens 单次运行 token 预算熔断（>0 时累计用量超限即中止运行，防失控消耗；0=不熔断）
	MaxTotalTokens int64
	// RunKind 运行来源（cron/event/manual/audit，审计/统计维度）
	RunKind string
	// ConversationSource 新建会话的来源标记（Persist 且新建会话时写入）：
	// 空=用户发起（user）；agentops 系统任务传 system，使用户会话列表不展示系统会话
	ConversationSource string
	// AllowedToolNames 本次运行的一次性工具许可（非空时只允许调用清单内的工具，其余一律拒绝）。
	// 用于「审批后受控执行」：把「只能执行已批准的那一个动作」从提示词约束变成机制约束。
	// 空=不下发许可（不因本次运行额外收窄）。ai 运行时不解释其业务语义，仅透传给护栏
	AllowedToolNames []string
	// ApprovedTools 已由人工批准的工具名 → 批准时锁定的参数 JSON（可空）。
	//
	// 与 AllowedToolNames 的分工：清单是天花板（清单外一律拒绝），本字段是「人已批准过
	// 这个工具的这个参数」的凭据，用于跳过工具内部的二次审批。ai 运行时同样不解释业务
	// 语义，仅透传：谁审批、批了什么由调用方（agentops 提案）决定
	ApprovedTools map[string]string
}

// RunResult 无头运行结果
type RunResult struct {
	// Output 最终输出文本
	Output string
	// Usage 本轮 token 用量累计（模型未上报时为 nil）
	Usage *schema.TokenUsage
	// ConversationId 关联会话 ID（Persist=false 时为 0）
	ConversationId uint64
	// TurnId 轮次 ID
	TurnId string
	// Status 运行终态（succeeded/failed/timeout/canceled/interrupted）
	Status string
	// ToolErrors 本轮失败的工具调用描述（空=工具全部成功）。
	//
	// 存在动机：模型可以把「工具报错」诚实写进回复，但那轮 LLM 调用本身是成功的（Status=succeeded）。
	// 只凭 Status 判定成败的业务方会把「没干成」记成「已执行」（审批后的受控执行就是这类调用方），
	// 故工具级事实必须由运行时一并上报。ai 不解释它的业务含义，由消费方自行决策
	ToolErrors []string
	// ProfileId 实际使用的装配档
	ProfileId string
	// RunKind 运行来源
	RunKind string
}

// Runner 无头运行端口（由 ai/application 适配器实现并在 Init 期 Register）
type Runner func(ctx context.Context, req *RunReq) (*RunResult, error)

var (
	// runnerMu 保护 runner 的读写：各模块 InitFunc 由 starter 以独立 goroutine 并发启动，
	// ai 在此注册 Runner 与 agentops 工作池轮询 Registered()/Run() 天然并发；同特性另一端口
	// （tools.SetProposalRecorder）已用 RWMutex，两个注入端口的防护强度必须一致，否则 -race
	// 会在启动窗口报数据竞态，且内存模型不保证读到新鲜值（可能看到旧 nil）
	runnerMu sync.RWMutex
	runner   Runner
)

// Register 注册无头运行实现（ai/application.Init 调用；后注册覆盖先注册）
func Register(r Runner) {
	runnerMu.Lock()
	defer runnerMu.Unlock()
	runner = r
}

// Registered 实现是否已注册（消费方可据此降级/给出友好错误）
func Registered() bool {
	runnerMu.RLock()
	defer runnerMu.RUnlock()
	return runner != nil
}

// Run 执行一次无头运行（面向契约编程；实现未注册时返回错误）
func Run(ctx context.Context, req *RunReq) (*RunResult, error) {
	runnerMu.RLock()
	r := runner
	runnerMu.RUnlock()
	if r == nil {
		return nil, errors.New("headless runner not registered (ai/application not initialized)")
	}
	return r(ctx, req)
}

// ProfileInfo Agent 装配档的对外元信息（供表单/筛选器装配与派发期校验）。
//
// 只暴露展示与判定所需的最小集：ToolPolicy / Instruction 等装配细节留在 ai 侧，
// 不让档位实现细节变成对外契约
type ProfileInfo struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// RequireRunGrant 本档必须由一次性运行许可驱动（无许可则护栏拒掉全部工具）。
	// 无人值守派发（cron/事件/手动）不会下发许可，所以这类档只能由审批后的提案执行链路驱动
	RequireRunGrant bool `json:"requireRunGrant"`
}

// ProfileCatalog 装配档目录端口（由 ai/application 在 Init 期注册实现）。
//
// 为什么需要这个端口：档位注册表是开放的（RegisterProfile 是公开扩展点），但消费方
// 此前把四档写死在前端枚举里——新增一档就要改前端，且保存时无人校验 profileId 是否存在，
// 任意字符串可入库。下沉为目录端口后，前端选项与派发期校验都从同一个注册表读出
type ProfileCatalog interface {
	// List 返回全部已注册档位（按 Id 升序，保证下拉顺序稳定）
	List() []ProfileInfo

	// Find 按 id 查档位；第二个返回值为 false 表示该档未注册
	Find(id string) (ProfileInfo, bool)
}

var (
	// catalogMu 与 runner 同一理由：各模块 InitFunc 并发启动，读写必须加锁
	catalogMu      sync.RWMutex
	profileCatalog ProfileCatalog
)

// RegisterProfileCatalog 注册档位目录实现（ai/application.Init 调用；后注册覆盖先注册）
func RegisterProfileCatalog(c ProfileCatalog) {
	catalogMu.Lock()
	defer catalogMu.Unlock()
	profileCatalog = c
}

// Profiles 取档位目录；未注册时返回 nil，调用方需自行兜底（不得当成「无档位」放行任意值）
func Profiles() ProfileCatalog {
	catalogMu.RLock()
	defer catalogMu.RUnlock()
	return profileCatalog
}

// 注：「哪些档可用于无人值守派发」是消费方（agentops 定时/事件触发）的业务规则，
// 不收在本包——本包是仅依赖标准库与 eino schema 的中立契约层，不引入业务错误类型

// RunAssoc 无头运行的业务关联标识（消费方注入，运行期工具据此把产物回链到发起运行的业务实体）。
//
// 存在动机（依赖倒置）：变更提案工具（ai/tools.propose_remediation）需要把提案回链到
// 所属的 agentops Run 与会话，但 ai 运行时不得反向依赖 agentops。故把「关联标识」下沉到
// 本中立契约包：agentops 作为消费方在发起运行前注入 BizRunId，ai 运行时仅透传并在解析
// 会话后回填 ConversationId，工具面向本契约读取，二者无直接依赖（无环）。
type RunAssoc struct {
	// BizRunId 发起本次运行的业务运行 id（如 agentops Run.Id；无关联时为 0）
	BizRunId uint64
	// ConversationId 本次运行关联的会话 id（由无头运行解析会话后回填；无痕运行时为 0）
	ConversationId uint64
}

type runAssocKey struct{}

// WithRunAssoc 把业务关联标识注入运行上下文（消费方在发起无头运行前调用）
func WithRunAssoc(ctx context.Context, assoc *RunAssoc) context.Context {
	if assoc == nil {
		return ctx
	}
	return context.WithValue(ctx, runAssocKey{}, assoc)
}

// RunAssocFrom 读取运行上下文中的业务关联标识（不存在返回 nil）
func RunAssocFrom(ctx context.Context) *RunAssoc {
	if ctx == nil {
		return nil
	}
	assoc, _ := ctx.Value(runAssocKey{}).(*RunAssoc)
	return assoc
}
