package vo

import (
	"time"

	"mayfly-go/internal/agentops/domain/entity"
)

// AgentTaskVO 定时计划任务列表/详情视图。
//
// 字段名与 entity.AgentTask 一致，供 model.PageResultConv（structx 按字段名拷贝）转换。
// 相对实体额外携带 Running（cron 是否已注册在调度器），供前端展示「运行中」状态。
// 表单可编辑字段必须在此齐全（含 dryRun/tokenBudget）：VO 缺字段 = 前端回填丢值 = 保存时把真实值刷成零值。
type AgentTaskVO struct {
	Id          uint64 `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Instruction string `json:"instruction"`
	// InstructionSegments 必须随 VO 回传：编辑表单靠它还原指令里的技能/资源芯片，
	// 缺该字段时下一次保存会把已有引用静默清掉
	InstructionSegments []entity.InstructionSegment `json:"instructionSegments"`
	Cron                string                      `json:"cron"`
	ProfileId           string                      `json:"profileId"`
	OwnerAccountId      uint64                      `json:"ownerAccountId"`
	OwnerUsername       string                      `json:"ownerUsername"`

	Status         int8  `json:"status"`
	TimeoutSeconds int   `json:"timeoutSeconds"`
	RetryTimes     int8  `json:"retryTimes"`
	MaxStep        int   `json:"maxStep"`
	TokenBudget    int64 `json:"tokenBudget"`
	// DryRun 必须随 VO 回传：表单从列表行/详情回填，缺该字段时开关永远显示「关闭」，
	// 而保存现在是「可编辑列整列真实写入」（零值不再被跳过），不把现值回传就会把
	// 一次普通编辑变成「静默关掉影子模式」（tokenBudget 同理会被清零）
	DryRun         bool       `json:"dryRun"`
	NotifyType     int8       `json:"notifyType"`
	NotifyTmplCode string     `json:"notifyTmplCode"`
	MisfirePolicy  int8       `json:"misfirePolicy"`
	LastFireTime   *time.Time `json:"lastFireTime"`
	Remark         string     `json:"remark"`

	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   string     `json:"modifier"`

	Running bool `json:"running" gorm:"-"`
}

// RunVO 运行记录（执行历史）列表视图。
//
// 相对实体裁剪 Instruction / Input（完整提示词与输入，体积大且列表无需展示），
// 保留 Summary（结论摘要）与 ConversationId（点击跳会话回放）。详情另走 RunDetailVO。
type RunVO struct {
	Id          uint64 `json:"id"`
	RunNo       string `json:"runNo"`
	TriggerType string `json:"triggerType"`
	TriggerRef  string `json:"triggerRef"`
	// ParentRunId 编排子运行的父运行 id（0=非编排子运行）：供前端从子运行回跳编排上下文
	ParentRunId uint64 `json:"parentRunId"`
	ProfileId   string `json:"profileId"`
	Title       string `json:"title"`
	// RunRole 运行形态（编排规划/步骤/审查、评测重放、提案执行/校验；空=普通运行）。
	// 前端据此渲染本地化形态标签——形态以前只写在标题中文前缀里，英文界面无法本地化
	RunRole string `json:"runRole"`
	Status  string `json:"status"`

	// Feedback 人类反馈（up/down/空）：必须随列表返回，否则界面看不出哪条运行已给过反馈，
	// 会重复反馈也无法按反馈态筛（取实体同名字段，字段名不一致会被 structx 静默漏拷）
	Feedback string `json:"feedback"`

	OwnerAccountId uint64 `json:"ownerAccountId"`
	OwnerUsername  string `json:"ownerUsername"`

	ConversationId uint64 `json:"conversationId"`
	TurnId         string `json:"turnId"`
	Summary        string `json:"summary"`

	PromptTokens     int64 `json:"promptTokens"`
	CompletionTokens int64 `json:"completionTokens"`
	TotalTokens      int64 `json:"totalTokens"`

	StartedAt  *time.Time `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	CostMs     int64      `json:"costMs"`

	Err        string     `json:"err"`
	CreateTime *time.Time `json:"createTime"`
}

// RunDetailVO 运行详情视图：列表字段 + 完整提示词/输入/影子模式标记。
//
// 不直接回实体：实体上还挂着 RunnerInstance（执行实例标识）、IdempotencyKey（去重键）与
// Extra（工具许可清单、工具级失败等运行期内部事实），它们不是展示契约；而「实体即响应」
// 会让今后新增的敏感列被自动带出，详情侧一个裁剪点都没有
type RunDetailVO struct {
	RunVO

	// Instruction / Input 完整提示词与输入：列表为省体积裁掉，详情用于复现与「带上下文追问」
	Instruction string `json:"instruction"`
	Input       string `json:"input"`
	// DryRun 影子模式（只读演练，不落地变更）
	DryRun bool `json:"dryRun"`
}

// TriggerVO 事件驱动触发器列表/详情视图。
//
// 字段名与 entity.Trigger 一致，供 model.PageResultConv（structx 按字段名拷贝）转换。
// webhookToken 只在详情接口回传（该接口与保存同一权限点把守）；列表接口一律清空，
// 因为它是免鉴权 webhook 路由的唯一凭据，而列表 GET 按平台惯例不带权限码。
// 同时：表单不携带该字段，因此列表不回传也不会造成「提交时丢字段 → 令牌被重置」。
type TriggerVO struct {
	Id         uint64 `json:"id"`
	Name       string `json:"name"`
	SourceType string `json:"sourceType"`
	MatchCond  string `json:"matchCond"`

	TaskId uint64 `json:"taskId"`
	// TaskName 关联任务名（列表与详情都回传，使界面能显示「关联任务：某巡检」而不是「关联任务 #18」）。
	// 任务已被删除时为空：删任务只把绑定它的触发器置为禁用、不解绑（见 AgentTaskService.Delete），
	// 此时前端按「任务已不存在」给文案，而不是亮一个没人认得的裸 id
	TaskName    string `json:"taskName"`
	Instruction string `json:"instruction"`
	// InstructionSegments 指令芯片引用段（编辑表单回填用）：VO 不回传则前端打开编辑抽屉看不到
	// 已有引用，用户保存一次就把已存的芯片段静默清空
	InstructionSegments []entity.InstructionSegment `json:"instructionSegments"`
	ProfileId           string                      `json:"profileId"`
	// NotifyType/NotifyTmplCode 内联模式的结果通知配置（与 AgentTaskVO 同源语义）
	NotifyType     int8   `json:"notifyType"`
	NotifyTmplCode string `json:"notifyTmplCode"`

	OwnerAccountId  uint64     `json:"ownerAccountId"`
	OwnerUsername   string     `json:"ownerUsername"`
	CooldownSeconds int        `json:"cooldownSeconds"`
	WebhookToken    string     `json:"webhookToken"`
	Enabled         int8       `json:"enabled"`
	LastFireTime    *time.Time `json:"lastFireTime"`
	Remark          string     `json:"remark"`

	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   string     `json:"modifier"`
}

// ProposalVO 变更提案列表/详情视图。
//
// 字段名与 entity.Proposal 一致，供 model.PageResultConv（structx 按字段名拷贝）转换。
// 携带审批工单关联（procInstId）与执行/校验运行关联（execRunId/verifyRunId），
// 供前端跳 flow 工单页 / 跳执行历史回放。
type ProposalVO struct {
	Id             uint64 `json:"id"`
	RunId          uint64 `json:"runId"`
	ConversationId uint64 `json:"conversationId"`

	Title  string `json:"title"`
	Action string `json:"action"`
	Params string `json:"params"`
	Risk   string `json:"risk"`
	Reason string `json:"reason"`
	Status string `json:"status"`

	OwnerAccountId uint64 `json:"ownerAccountId"`
	OwnerUsername  string `json:"ownerUsername"`

	ProcInstId  uint64 `json:"procInstId"`
	ProcDefId   uint64 `json:"procDefId"`
	ExecRunId   uint64 `json:"execRunId"`
	VerifyRunId uint64 `json:"verifyRunId"`

	ExecResult       string `json:"execResult"`
	VerifyConclusion string `json:"verifyConclusion"`
	Err              string `json:"err"`

	CreateTime *time.Time `json:"createTime"`
	Creator    string     `json:"creator"`
	UpdateTime *time.Time `json:"updateTime"`
	Modifier   string     `json:"modifier"`
}
