package entity

import (
	"time"

	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/jsonx"

	"gorm.io/gorm"
)

// Run 一次 Agent 无头运行的聚合根（一等公民）。
//
// cron / event / manual 三类触发统一经 RunService.Enqueue 产出 Run，收敛「运行生命周期」这一
// 事实源：状态机、token 成本、关联会话（回放）、幂等去重都在此聚合，替代割裂的多套记录。
//
// 时序模型：追加写 + 状态流转（queued→running→terminal），运行期需要更新，故用 ModelNLD
// （含创建/修改审计、无逻辑删除）；历史清理走硬删除，与机器指标等运维记录口径一致。
// idempotency_key 唯一索引即在此模型上成立（无 is_deleted 干扰 InsertIfAbsent 去重）。
type Run struct {
	model.ModelNLD
	// ExtraData 非查询维度的扩展信息容器（一次性工具许可等运行期约束随运行自持，不单独立列）
	model.ExtraData

	// RunNo 运行业务编号（对外稳定标识，唯一）
	RunNo string `json:"runNo" gorm:"size:64;not null;uniqueIndex:uk_aiops_run_no;comment:运行编号"`

	// TriggerType 触发类型：cron/event/manual（见 TriggerType* 常量）
	TriggerType string `json:"triggerType" form:"triggerType" gorm:"size:16;not null;index:idx_aiops_run_trigger;comment:触发类型"`
	// TriggerRef 触发源引用：task_id / event_id / 空（manual）
	TriggerRef string `json:"triggerRef" form:"triggerRef" gorm:"size:64;index:idx_aiops_run_trigger;comment:触发源引用"`

	// ProfileId Agent 装配档（diagnostician/executor/reviewer/...）
	ProfileId string `json:"profileId" form:"profileId" gorm:"size:64;not null;comment:Agent装配档"`

	// Title 运行标题（任务名/事件名快照；用于会话标题、结果通知与列表展示，使运行自持）
	Title string `json:"title" gorm:"size:255;comment:运行标题"`

	// Status 运行状态（见 RunStatus* 常量）
	Status string `json:"status" form:"status" gorm:"size:24;not null;index:idx_aiops_run_status;comment:运行状态"`

	// Instruction 任务级 system 提示词（可空；与 Profile 静态指令叠加）
	Instruction string `json:"instruction" gorm:"type:text;comment:任务system提示词"`
	// Input 用户输入（user 消息，可为结构化 JSON 文本）——使运行自持、可重放/重启后续跑
	Input string `json:"input" gorm:"type:text;comment:运行输入"`

	// OwnerAccountId 运行服务身份账号 ID（工具权限过滤按此收敛）
	OwnerAccountId uint64 `json:"ownerAccountId" form:"ownerAccountId" gorm:"not null;index:idx_aiops_run_owner;comment:运行身份账号id"`
	// OwnerUsername 运行服务身份账号名（展示/审计）。瞬态字段：落库收敛在 Extra（非查询维度，
	// 列表与详情展示依旧契约由 AfterFind 还原）
	OwnerUsername string `json:"ownerUsername" gorm:"-"`

	// RunRole 运行形态（编排规划/步骤/审查、评测重放/打分、提案执行/校验；空=普通运行）。
	// 瞬态字段：收敛在 Extra，由 BeforeSave/AfterFind 同步。形态以前只写在标题中文前缀里，
	// 使英文界面直出中文、且迁移拿 title LIKE 做分类谓词（值域与理由见 run_role.go）
	RunRole string `json:"runRole" gorm:"-"`

	// ParentRunId 父运行 id（仅编排子运行写入）：编排轨迹靠「触发类型 + 本列」筛取与聚合，
	// 不依赖 trigger_ref 字符串前缀模糊匹配（仓储保持精确匹配，不放开 LIKE 查询面）
	ParentRunId uint64 `json:"parentRunId" form:"parentRunId" gorm:"index:idx_aiops_run_parent;comment:父运行id"`

	// ConversationId 关联会话 ID（回放入口；无痕运行时为 0）。
	// form tag 不可省：仓储按本列过滤（见 runRepoImpl.GetPageList），而 Gin 的 BindQuery 只认 form，
	// 缺失时退回**字段名**（大小写敏感），于是 ?conversationId=1 绑不进来——过滤条件被静默丢弃、
	// 列表返回全量，接口照返 200，调用方只会以为「这个会话没有运行记录」
	ConversationId uint64 `json:"conversationId" form:"conversationId" gorm:"index:idx_aiops_run_conv;comment:关联会话id"`
	// TurnId 关联轮次 ID（回放定位）
	TurnId string `json:"turnId" gorm:"size:64;comment:关联轮次id"`

	// Summary 运行结论摘要（最终输出文本）
	Summary string `json:"summary" gorm:"type:text;comment:运行结论摘要"`

	// PromptTokens 输入 token 数
	PromptTokens int64 `json:"promptTokens" gorm:"comment:输入token"`
	// CompletionTokens 输出 token 数
	CompletionTokens int64 `json:"completionTokens" gorm:"comment:输出token"`
	// TotalTokens 总 token 数
	TotalTokens int64 `json:"totalTokens" gorm:"comment:总token"`

	// TimeoutSeconds 单次运行超时秒数（0=用服务默认）
	TimeoutSeconds int `json:"timeoutSeconds" gorm:"comment:运行超时秒数"`

	// MaxStep 单次运行最大步数（0=用 Profile 默认；触发时快照，使运行自持可重放）
	MaxStep int `json:"maxStep" gorm:"comment:最大步数"`

	// TokenBudget 单次运行 token 预算熔断（0=不熔断；>0 时累计用量超限中止运行）
	TokenBudget int64 `json:"tokenBudget" gorm:"comment:token预算熔断"`

	// DryRun 影子模式（演练）：只读执行，产出「将执行什么」而不落地任何变更
	DryRun bool `json:"dryRun" gorm:"comment:影子模式"`

	// RetryTimes 失败重试次数（仅超时/连接类可重试；0=不重试）
	RetryTimes int8 `json:"retryTimes" gorm:"comment:失败重试次数"`

	// NotifyType 结果通知方式（0不通知 1仅失败 2总是；见 TaskNotify* 常量）
	NotifyType int8 `json:"notifyType" gorm:"comment:结果通知方式"`
	// NotifyTmplCode 结果通知消息模板 code（渠道由模板关联决定）
	NotifyTmplCode string `json:"notifyTmplCode" gorm:"size:64;comment:通知模板code"`

	// StartedAt 开始执行时间（queued→running 时落）
	StartedAt *time.Time `json:"startedAt" gorm:"comment:开始时间"`
	// FinishedAt 结束时间（进入终态时落）
	FinishedAt *time.Time `json:"finishedAt" gorm:"comment:结束时间"`
	// CostMs 执行耗时毫秒（started→finished）
	CostMs int64 `json:"costMs" gorm:"comment:执行耗时ms"`

	// IdempotencyKey 幂等键（唯一索引；如 alert:<eventId>，重复触发直接短路去重）
	IdempotencyKey string `json:"idempotencyKey" gorm:"size:191;not null;uniqueIndex:uk_aiops_run_idem;comment:幂等键"`

	// Err 失败/超时/取消原因（终态非成功时填充）
	Err string `json:"err" gorm:"size:2000;comment:错误信息"`

	// Feedback 人类反馈（学习闭环）：up=点赞 down=点踩 空=未反馈（可按此筛选统计，故独立列）
	Feedback string `json:"feedback" form:"feedback" gorm:"size:16;comment:人类反馈"`
	// FeedbackNote 反馈备注（原因说明，随经验沉淀进记忆系统）。瞬态字段：收敛在 Extra
	FeedbackNote string `json:"feedbackNote" gorm:"-"`

	// RunnerInstance 认领并执行该运行的实例标识（仅 RunSync/工作池认领时写入）。
	// 只供运维定位「谁在跑」，不参与清扫判定：残留按租约收敛（started_at + 超时×阶段数），
	// 实例 id 每进程随机，按归属收敛会让上一进程留下的行永久无人收尾并阻断重叠防护
	// （口径真源见 RunService.SweepInterruptedRuns）
	RunnerInstance string `json:"runnerInstance" gorm:"size:64;index:idx_aiops_run_instance;comment:执行实例标识"`
}

// BeforeSave / AfterFind：与 AgentTask 同一机制，把展示型瞬态字段与 Extra 双向同步。
//
// Run 的 Extra 还携带一次性工具许可与工具级失败记录（allowedToolNames / toolErrors），
// 因此同步只能按键增量写入，不能重建整个 Extra 容器
func (r *Run) BeforeSave(*gorm.DB) error {
	r.SetExtraValue(ExtraKeyOwnerUsername, r.OwnerUsername)
	r.SetExtraValue(ExtraKeyFeedbackNote, r.FeedbackNote)
	r.SetExtraValue(ExtraKeyRunRole, r.RunRole)
	return nil
}

// AfterFind 从 Extra 还原瞬态展示字段（VO 按名拷贝与 JSON 输出保持原契约）
func (r *Run) AfterFind(*gorm.DB) error {
	r.OwnerUsername = r.GetExtraString(ExtraKeyOwnerUsername)
	r.FeedbackNote = r.GetExtraString(ExtraKeyFeedbackNote)
	r.RunRole = r.GetExtraString(ExtraKeyRunRole)
	return nil
}

// TableName 指定表名
func (r *Run) TableName() string {
	return "t_aiops_run"
}

// 触发类型
//
// orchestration 是多 Agent 编排的专属触发类型（父运行与规划/步骤/审查子运行共用）：
// 旧实现把它们归入 manual，仅靠 trigger_ref 前缀区分，而列表按 trigger_ref 精确匹配，
// 导致「编排轨迹无法在执行历史里筛出来」、且触发方式列与实际来源不符（成本归属看不清）。
const (
	TriggerTypeCron          = "cron"          // 定时计划任务
	TriggerTypeEvent         = "event"         // 事件驱动（告警/webhook）
	TriggerTypeManual        = "manual"        // 人工手动触发
	TriggerTypeOrchestration = "orchestration" // 多 Agent 编排（父运行与子运行）
)

// 运行状态机
//
//	queued ──claim──> running ──> succeeded
//	                   │  ├──────> failed
//	                   │  └──────> canceled
//	                   └──(取消)──> canceled
//
// 没有「等待审批」态：诊断运行产出提案后就正常收尾（succeeded），等待审批是 Proposal 聚合
// 自己的状态（pending），受控执行由审批通过后**另起一条**运行承载，不存在「一条运行卡在审批里」。
// 此处曾声明过 waiting_approval 但始终没有写入点，代价是前端状态筛选多出一个永远查不到数据的
// 选项、治理分布多一个永远为零的键：声明一个系统进不去的状态，等于给界面和统计各留一个静默空洞
const (
	RunStatusQueued    = "queued"    // 已入队，等待工作池认领
	RunStatusRunning   = "running"   // 执行中
	RunStatusSucceeded = "succeeded" // 成功
	RunStatusFailed    = "failed"    // 失败（含超时）
	RunStatusCanceled  = "canceled"  // 已取消（急停/人工取消）
)

// ActiveRunStatuses 非终态（进行中）状态集合：IsTerminal 的补集，是「仍在途 / 可取消 / 占重叠防护位」
// 的唯一真源——仓储重叠防护、急停清场、取消白名单、治理分布全部引用它，不得再手写子集
// （曾出现急停清场与取消白名单各自凑子集 → 0 行更新却报已取消）。
// 集合与 IsTerminal 的互补关系由 domain 层用例守护（新增状态时漏归类会被用例拦下）。
var ActiveRunStatuses = []string{RunStatusQueued, RunStatusRunning}

// TerminalRunStatuses 终态集合的**可枚举**形态：只给「需要列出全部终态」的地方用（SQL 的 IN、
// 治理看板的分布键与成功率分母）。新增终态时改这一处即可让统计自动覆盖，不必再去 SQL 字符串
// 和 governance 的逐键映射里各拄一份（曾因此出现「新终态不进分布、不进分母、却进总数」，
// 看板 ΣRunByStatus 与 RunTotal 自相矛盾）。
//
// 判定仍一律走 IsTerminal（不反向由集合派生，未知/脏状态必须稳定落在非终态侧），
// 两者的等价关系由 domain 用例守住。
var TerminalRunStatuses = []string{RunStatusSucceeded, RunStatusFailed, RunStatusCanceled}

// IsTerminal 是否为终态（终态不可再流转，幂等重入直接短路）
func IsTerminal(status string) bool {
	switch status {
	case RunStatusSucceeded, RunStatusFailed, RunStatusCanceled:
		return true
	default:
		return false
	}
}

// 反馈取值（Run.Feedback）
const (
	FeedbackUp   = "up"   // 点赞：结论准确/有帮助
	FeedbackDown = "down" // 点踩：结论有误/无帮助
)

// IsValidFeedback 反馈取值是否合法
func IsValidFeedback(f string) bool {
	return f == FeedbackUp || f == FeedbackDown
}

// extraAllowedToolNames Run.Extra 中一次性工具许可的键
const extraAllowedToolNames = "allowedToolNames"

// extraToolErrors Run.Extra 中本轮失败工具调用的键
const extraToolErrors = "toolErrors"

// SetAllowedToolNames 记录本次运行允许调用的工具集（空列表=不下发许可）。
//
// 存在动机：受控执行必须「运行自持」——许可随运行落库，工作池认领后执行的才是那条
// 运行自己携带的约束，而不是调用方内存里的参数（否则同一个入队请求走同步/异步两条路径
// 会得到不同的权限，是典型的双真源）
func (r *Run) SetAllowedToolNames(names []string) {
	if len(names) == 0 {
		return
	}
	r.SetExtraValue(extraAllowedToolNames, names)
}

// AllowedToolNames 读取本运行的一次性工具许可（未设置返回空，表示不按许可收窄）
func (r *Run) AllowedToolNames() []string {
	return r.GetExtraStringSlice(extraAllowedToolNames)
}

// extraApprovedActions Run.Extra 中「已获人工批准的工具→批准参数 JSON」的键
const extraApprovedActions = "approvedActions"

// SetApprovedActions 记录提案审批已放行的工具及其锁定参数（空 map 不写入）。
//
// 为什么必须随运行自持而不是只活在入队参数里：许可要在执行那一刻还在——工作池认领与
// 同步执行两条路径都必须拿到同一份约束，否则会出现「同一次入队两条路径权限不同」的双真源。
// 参数一并持久化是因为「批准的参数」就是审批对象本身，丢了它就只能按工具名放行
func (r *Run) SetApprovedActions(actions map[string]string) {
	if len(actions) == 0 {
		return
	}
	r.SetExtraValue(extraApprovedActions, actions)
}

// ApprovedActions 读取已批准的工具与参数（未设置或脏数据返回 nil，一律按未批准处理）
func (r *Run) ApprovedActions() map[string]string {
	raw := r.GetExtraVal(extraApprovedActions)
	if raw == nil {
		return nil
	}
	parsed, err := jsonx.ToByStr[map[string]string](jsonx.ToStr(raw))
	if err != nil || parsed == nil {
		return nil
	}
	return *parsed
}

// SetToolErrors 记录本轮失败的工具调用描述（空列表=不写入）。
//
// 为什么必须随运行自持：模型可以把工具报错写进回复，但那轮 LLM 调用本身成功（run 仍为 succeeded）。
// 不落库则「受控执行其实失败」这个事实只活在文本摘要里，提案状态机会把它记成已执行
func (r *Run) SetToolErrors(errs []string) {
	if len(errs) == 0 {
		return
	}
	r.SetExtraValue(extraToolErrors, errs)
}

// ToolErrors 读取本轮失败的工具调用描述
func (r *Run) ToolErrors() []string {
	return r.GetExtraStringSlice(extraToolErrors)
}
