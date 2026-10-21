package application

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	aientity "mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/pkg/headless"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/i18n"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"
)

const (
	// defaultRunWorkerPoolSize 有界工作池默认大小：固定 worker 数即运行中 Run 的并发上限，
	// 防告警风暴/批量触发派生无界 goroutine（每个 Run 都占一次 LLM 调用，代价高昂）
	defaultRunWorkerPoolSize = 4

	// defaultRunTimeout 单次运行默认超时（Run.TimeoutSeconds<=0 时使用）
	defaultRunTimeout = 5 * time.Minute

	// runWorkerPollInterval 工作池兜底轮询间隔：信号丢失/重启后残留 queued 由此被重新驱动，
	// 空闲期仅每 worker 每 2s 一次轻量查询，负载可忽略
	runWorkerPollInterval = 2 * time.Second

	// runSweepGraceSeconds 租约宽限期：在运行自身超时之上再留的收尾余量。
	// 超过「started_at + 超时 + 宽限期」仍是 running，即认定其执行者已消失（被杀/崩溃/
	// 没等到优雅停机），收敛为 failed。不按 runner_instance 归属判定：实例 id 每进程随机，
	// 按归属清扫会让上一个进程留下的行永远无人收敛，而它把同任务的重叠防护（HasActiveRun）
	// 永久判真，后续每次 cron/事件/手动派发都被静默跳过
	runSweepGraceSeconds = 120

	// runSweepInterval 租约清扫周期：SIGKILL/容器强杀下优雅停机不生效，靠周期清扫自愈
	runSweepInterval = 5 * time.Minute

	// maxRunErrLen err 列截断长度（与实体 size:2000 对齐）
	maxRunErrLen = 2000

	// runRetryBackoffBase 重试退避基数（第 n 次重试前等待 n×base，与机器计划任务口径一致）
	runRetryBackoffBase = time.Second
)

// EnqueueReq 运行入队请求（cron/event/manual 统一入口的输入契约）
type EnqueueReq struct {
	// TriggerType 触发类型（空则默认 manual）
	TriggerType string
	// TriggerRef 触发源引用（task_id / event_id / 空）
	TriggerRef string
	// ParentRunId 父运行 id（仅多 Agent 编排的子运行写入）：用于把一次编排的轨迹筛出与成本汇总，
	// 不依赖 TriggerRef 字符串前缀模糊匹配
	ParentRunId uint64
	// ProfileId Agent 装配档（空则默认 diagnostician 只读诊断档）
	ProfileId string
	// Instruction 任务级 system 提示词（可空）
	Instruction string
	// Input 用户输入（user 消息，可为结构化 JSON 文本）
	Input string
	// OwnerAccountId 运行服务身份账号 ID（必填；fail-closed）
	OwnerAccountId uint64
	// OwnerUsername 运行服务身份账号名
	OwnerUsername string
	// Title 运行标题（任务名/事件名快照；用于会话标题与结果通知）
	Title string
	// RunRole 运行形态（编排规划/步骤/审查、评测重放、提案执行/校验；空=普通运行）。
	// 形态必须由结构化字段承载：把它拼成标题中文前缀会让英文界面直出中文，且分类谓词只能靠
	// LIKE 文本匹配（值域与理由见 entity.RunRole）
	RunRole string
	// TimeoutSeconds 单次运行超时秒数（0=用服务默认）
	TimeoutSeconds int
	// MaxStep 单次运行最大步数（0=用 Profile 默认）
	MaxStep int
	// TokenBudget 单次运行 token 预算熔断（0=不熔断；>0 时累计用量超限中止运行）
	TokenBudget int64
	// DryRun 影子模式（演练）：强制降为只读档执行，仅产出「将执行什么」不落地任何变更
	DryRun bool
	// RetryTimes 失败重试次数（仅超时/连接类可重试；0=不重试）
	RetryTimes int8
	// NotifyType 结果通知方式（0不通知 1仅失败 2总是）
	NotifyType int8
	// NotifyTmplCode 结果通知消息模板 code
	NotifyTmplCode string
	// IdempotencyKey 幂等键（如 alert:<eventId>；空则自动生成天然唯一键，即不去重）
	IdempotencyKey string
	// AllowedToolNames 本次运行的一次性工具许可（非空即只允许调用清单内工具，其余被护栏拒绝）。
	// 用于审批后受控执行：把「只能执行已批准的那一个动作」做成机制约束。快照进 Run（Extra）使运行自持
	AllowedToolNames []string
	// ApprovedActions 提案审批已放行的「工具名 → 批准时锁定的参数 JSON」（可空）。
	//
	// 与 AllowedToolNames 的分工：清单是天花板（清单外一律拒），本字段是「人已批准过
	// 这个工具的这个参数」的凭据，让受控执行能真的执行（而不是挂在等一个不会来的审批上）
	ApprovedActions map[string]string
}

// RunService 运行聚合应用服务：统一入队 + 有界工作池执行 + 状态机流转 + 取消
type RunService interface {
	base.App[*entity.Run]

	// Enqueue 统一入队（cron/event/manual/misfire 都经此）：急停守护 → 幂等去重 → 落库 queued → 唤醒工作池。
	// 命中幂等键时短路返回既有运行（不重复入队）。本入口是「产出新 Run」的结构收口点：
	// 急停守护就在这里判定，新增派发源不可能漏加（详见方法注释）
	Enqueue(ctx context.Context, req *EnqueueReq) (*entity.Run, error)

	// RunSync 同步创建并执行一次运行（阻塞至终态），返回终态 Run。
	// 供审批后受控执行、编排、评测等在专用 goroutine 里需即时拿到结果的场景（区别于 Enqueue 的异步入队）：
	// 直接以 running 落库并在调用方 goroutine 内跑完 headless（不经工作池排队），
	// 复用 execute 的重试/终态写库/通知/在途取消跟踪。命中幂等键时短路返回既有运行。
	//
	// 本入口绕过有界工作池：不得直接接在 HTTP 请求链路上（并发上限失效且会挂住请求），
	// 需先拿信号量或投递到专用异步体（见 Orchestrator 与 ProposalGateway 的做法）
	RunSync(ctx context.Context, req *EnqueueReq) (*entity.Run, error)

	// GetPageList 分页查询运行记录
	GetPageList(condition *entity.Run, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Run], error)

	// Cancel 取消一个排队/运行中的运行（急停/人工取消）。
	// 已终态或已被其它路径收尾时返回 ErrRunNotCancelable（调用方据此区分「无事发生」与失败）
	Cancel(ctx context.Context, runId uint64) error

	// CancelAllActive 取消全部非终态运行（急停附带的一键清场）：逐个走 Cancel（含在途 ctx 取消）。
	// 返回实际取消数量；单个失败不中断其余。跨实例在途运行仅置库态，由其自身超时收敛。
	CancelAllActive(ctx context.Context) (int, error)

	// HasActiveRun 判断指定触发源是否存在非终态运行（集合真源见 entity.ActiveRunStatuses）。
	// 供 cron 触发的重叠防护：同一任务上一次运行尚未收尾时跳过本次触发。
	HasActiveRun(ctx context.Context, triggerType, triggerRef string) (bool, error)

	// Feedback 人类反馈（学习闭环，阶段 4.3）：对终态运行点赞/点踩 + 备注，
	// 落库并沉淀为操作员的经验记忆（经 ai memory 系统，供后续会话召回）
	Feedback(ctx context.Context, runId uint64, feedback, note string) error

	// ReplayEval 离线评测回放（阶段 4.3）：复制终态运行的原始输入用诊断档重跑一次，
	// 再由审查档对比新旧结论打分，评测运行异步执行（trigger_ref=eval:<origId>）
	ReplayEval(ctx context.Context, runId uint64) (*entity.Run, error)

	// Start 启动有界工作池（应用初始化时调用一次，幂等）
	Start()

	// Stop 优雅停止工作池（等待在途运行收尾）
	Stop()

	// SweepInterruptedRuns 收敛「租约过期」的 running 运行：按各行自己的超时 + 宽限期判定，
	// 过期即置 failed（应用初始化时调一次，并由工作池内的 sweeper 周期重跑）。
	// 工作池兜底轮询只重驱 queued，重启/崩溃时在途的 running 无人收敛——会永久悬挂在
	// running（治理 KPI 的「运行中」虚高、编排父运行/评测回放等 goroutine 驱动的运行尤甚，
	// 并且会永久阻断同一任务的后续派发）。
	// 按租约而非实例归属判定：既不误杀其他实例正在跑的运行（它们未超租约，终态由它自己写入），
	// 也不会因「实例 id 每进程随机」而永远漏掉上一个进程的残留行
	SweepInterruptedRuns(ctx context.Context) error
}

// runInstanceId 本进程的实例标识：认领/同步创建运行时写入 runner_instance，供运维定位
// 「这条运行是谁在跑」。**不作为清扫归属条件**——每进程随机值意味着上一个进程留下的行
// 新进程永远匹配不到，重启残留靠租约过期判定收敛（见 SweepInterruptedRuns）。
var runInstanceId = stringx.SortableUUID()

// ErrRunNotCancelable 运行已终态/不在可取消状态：明确报错而非「0 行更新也返回成功」，
// 否则前端弹「已取消」但库里什么都没变（急停清场计数也会跟着虚高）
var ErrRunNotCancelable = errorx.NewBiz(i18n.T(imsg.ErrRunNotCancelable))

type runServiceAppImpl struct {
	base.AppImpl[*entity.Run, repository.Run]

	// msgTmplApp 结果通知发送器（与告警/机器计划任务同一发送器，渠道由消息模板关联决定）。
	// 单测绕过 IOC 注入时为 nil，notifyResult 已做 nil 守卫。
	msgTmplApp msgapp.MsgTmpl `inject:"T"`

	// governance 治理服务：Enqueue 的急停守护收口点（单测未装配时为 nil，guardDispatch 已做 nil 守卫）
	governance Governance `inject:"T"`

	// PoolSize 工作池大小（<=0 时用默认；可在注册后按需覆盖）
	PoolSize int

	signalCh chan struct{} // 入队唤醒信号（缓冲 1，多次入队合并为一次唤醒）
	stopCh   chan struct{} // 停止信号
	wg       sync.WaitGroup

	startMu sync.Mutex
	started bool

	// inflight 本实例运行中的 runId → cancel（支持取消/急停；跨实例运行由其自身超时收敛）
	inflightMu sync.Mutex
	inflight   map[uint64]context.CancelFunc
	// cancelReq 已请求取消但 execute 尚未注册 cancel 的 runId 集合：闭合「Cancel 早于
	// trackInflight」的竞态——trackInflight 命中即 born-canceled，避免取消信号丢失致运行悬挂
	cancelReq map[uint64]struct{}
}

var _ RunService = (*runServiceAppImpl)(nil)

// buildRunEntity 运行实体的**唯一构造点**（Enqueue 与 RunSync 共用）。
//
// 为什么必须单点：两条创建路径此前各自拼一份 18 字段字面量，历史上已经因此漏过一次——
// RunRole 只补了 Enqueue，同步跑出来的运行（提案的受控执行/校验）永远没有形态标签。
// 两处手拄的字段清单没有任何机制保证同步，新增字段时漏一处也不报错，只是静默少一个值。
// 现在只在本函数里补字段；调用方只负责自己的差异：初始状态、startedAt、runnerInstance。
func buildRunEntity(req *EnqueueReq, status string) *entity.Run {
	triggerType := req.TriggerType
	if triggerType == "" {
		triggerType = entity.TriggerTypeManual
	}
	profileId := req.ProfileId
	if profileId == "" {
		profileId = headless.ProfileDiagnostician
	}
	runNo := stringx.SortableUUID()
	// 未提供幂等键：以 runNo 生成天然唯一键（每次都是新运行，不参与去重）
	idemKey := req.IdempotencyKey
	if idemKey == "" {
		idemKey = "auto:" + runNo
	}

	run := &entity.Run{
		RunNo:          runNo,
		TriggerType:    triggerType,
		TriggerRef:     req.TriggerRef,
		ParentRunId:    req.ParentRunId,
		ProfileId:      profileId,
		Title:          req.Title,
		RunRole:        req.RunRole,
		Status:         status,
		Instruction:    req.Instruction,
		Input:          req.Input,
		OwnerAccountId: req.OwnerAccountId,
		OwnerUsername:  req.OwnerUsername,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxStep:        req.MaxStep,
		TokenBudget:    req.TokenBudget,
		DryRun:         req.DryRun,
		RetryTimes:     req.RetryTimes,
		NotifyType:     req.NotifyType,
		NotifyTmplCode: req.NotifyTmplCode,
		IdempotencyKey: idemKey,
	}
	run.SetAllowedToolNames(req.AllowedToolNames)
	run.SetApprovedActions(req.ApprovedActions)
	return run
}

// Enqueue 统一入队：守护判定 → 构造 queued 运行 → 幂等插入 → 唤醒工作池
//
// 急停守护就构在此结构收口点：任何「产出新 Run」的派发源都必经本入口，因而新增派发源
// 不可能漏加守护（旧实现把判定写在 5 个入口，启动 misfire 补跑作为第 6 个入口就漏了）。
// 各业务入口若需要在入队前整批拒绝（事件批量、编排发起），调的是同一个谓词
// Governance.GuardDispatch，不是重复实现。
func (r *runServiceAppImpl) Enqueue(ctx context.Context, req *EnqueueReq) (*entity.Run, error) {
	if req == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunEnqueueReqNil)
	}
	// fail-closed：无运行身份拒绝入队（工具权限过滤依赖 owner 账号，缺省会越权）
	if req.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunEnqueueOwnerRequired)
	}
	if req.Input == "" && req.Instruction == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunEnqueueInstructionRequired)
	}
	// 急停守护（唯一结构收口点，见函数注释）
	if err := guardDispatch(r.governance); err != nil {
		return nil, err
	}

	run := buildRunEntity(req, entity.RunStatusQueued)
	idemKey := run.IdempotencyKey

	// 幂等入队：命中唯一键冲突则跳过插入（原子去重，抗并发重复触发/告警风暴）
	if err := r.InsertIfAbsent(ctx, run, "idempotency_key"); err != nil {
		return nil, err
	}
	if run.Id == 0 {
		// 冲突跳过 → 已存在同幂等键运行：回查既有运行返回（短路，不重复入队/不重复消耗 LLM）
		existing, err := r.GetRepo().GetByIdempotencyKey(idemKey)
		if err != nil {
			return nil, err
		}
		logx.DebugfContext(ctx, "[agentops-run] enqueue deduplicated by idempotency key: %s (runNo=%s)", idemKey, existing.RunNo)
		return existing, nil
	}

	r.signal()
	logx.InfofContext(ctx, "[agentops-run] enqueued: runNo=%s, trigger=%s/%s, profile=%s", run.RunNo, run.TriggerType, run.TriggerRef, run.ProfileId)
	return run, nil
}

// RunSync 同步创建并执行一次运行（阻塞至终态），返回终态 Run。
//
// 与 Enqueue 的异步入队不同：直接以 running 落库（不经工作池排队，worker 只认领 queued，
// 故不会被重复执行），在调用方 goroutine 内跑完 execute。execute 内部用独立 baseCtx（不受
// 调用方 ctx 取消影响，与 flow 审批回放 detached 语义一致），仍支持经 Cancel(runId) 急停。
//
// 不受急停守护：审批已通过的变更必须能收尾（见 Governance 接口注释），否则急停会把已批准的
// 变更卡在无人执行状态，比多跑一次更难收拾。
//
// 代价提醒：本入口绕过有界工作池，调用方并发多大就同时跑多少个 LLM 运行。需要并发上限的
// 场景（编排、审批后受控执行）必须自己先拿信号量/队列，不得直接把它接在请求链路上（见
// ProposalGateway 的受控执行异步化）。
func (r *runServiceAppImpl) RunSync(ctx context.Context, req *EnqueueReq) (*entity.Run, error) {
	if req == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunReqNil)
	}
	if req.OwnerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunSyncOwnerRequired)
	}
	if req.Input == "" && req.Instruction == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrRunSyncInstructionRequired)
	}

	// 与 Enqueue 共用构造器：字段清单只有一份，新增字段不会只落在一条路径上
	now := time.Now()
	run := buildRunEntity(req, entity.RunStatusRunning)
	// 同步路径不经工作池：开始时间与实例归属在本函数内就定下（worker 只认领 queued）
	run.StartedAt = &now
	run.RunnerInstance = runInstanceId
	idemKey := run.IdempotencyKey

	if err := r.InsertIfAbsent(ctx, run, "idempotency_key"); err != nil {
		return nil, err
	}
	if run.Id == 0 {
		// 命中幂等键：已存在同键运行，回查返回（短路，不重复执行/不重复消耗 LLM）
		existing, err := r.GetRepo().GetByIdempotencyKey(idemKey)
		if err != nil {
			return nil, err
		}
		logx.DebugfContext(ctx, "[agentops-run] run sync deduplicated by idempotency key: %s (runNo=%s)", idemKey, existing.RunNo)
		return existing, nil
	}

	logx.InfofContext(ctx, "[agentops-run] run sync executing: runNo=%s, profile=%s, trigger=%s/%s",
		run.RunNo, run.ProfileId, run.TriggerType, run.TriggerRef)
	terminal, won := r.execute(run)

	// 回查终态行（summary/token/关联会话等真值由 execute 写库，内存对象没有）。
	// 旧写法「回查失败就 return run, nil」会把内存里初始置好的 running 当作结果告人，
	// 调用方（提案链路）据 Status 判成败 → 变更已真实执行却把提案标 failed
	finished, err := r.GetById(run.Id)
	switch {
	case err == nil && finished != nil:
		return finished, nil
	case won:
		// 终态是本次写入的，回查失败仅缺展示字段，状态可信。结束时刻取当下而非
		// 函数入口的 now（那是运行开始的时刻），否则耗时永远显示 0ms
		run.Status = terminal
		finishedAt := time.Now()
		run.FinishedAt = &finishedAt
		return run, nil
	default:
		// 终态由他人写入（Cancel/启动清扫）且回查失败：无法给出可信状态，明确报错，不拿内存陈值冒充
		return nil, errorx.NewBizI(ctx, imsg.ErrRunTerminalUndeterminable, "runNo", run.RunNo, "err", err.Error())
	}
}

// GetPageList 分页查询运行记录
func (r *runServiceAppImpl) GetPageList(condition *entity.Run, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Run], error) {
	return r.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

// HasActiveRun 判断触发源是否存在非终态运行（重叠防护）
func (r *runServiceAppImpl) HasActiveRun(ctx context.Context, triggerType, triggerRef string) (bool, error) {
	return r.GetRepo().ExistsActiveByTriggerRef(ctx, triggerType, triggerRef)
}

// Cancel 取消排队/运行中的运行：置为 canceled，并取消本实例在途 ctx。
// 已终态返回 ErrRunNotCancelable（不抹平成「成功」，否则前端与清场计数都会拿到假结果）
func (r *runServiceAppImpl) Cancel(ctx context.Context, runId uint64) error {
	run, err := r.GetById(runId)
	if err != nil {
		return err
	}
	if entity.IsTerminal(run.Status) {
		return ErrRunNotCancelable
	}

	// 先标记取消请求并取本地在途 cancel：即便 execute 此刻尚未注册 cancel（认领后、注册前的
	// 窗口），trackInflight 也会命中该标记立即取消，信号不丢
	r.inflightMu.Lock()
	if r.cancelReq == nil {
		r.cancelReq = make(map[uint64]struct{})
	}
	r.cancelReq[runId] = struct{}{}
	cancel := r.inflight[runId]
	r.inflightMu.Unlock()

	now := time.Now()
	patch := base.NewPatch().
		Set("status", entity.RunStatusCanceled).
		Set("finishedAt", now).
		Set("err", "canceled")
	// 已起跑的运行必须补写 costMs：不写则它以 0 计入「平均耗时」的终态分母，
	// 点一次急停清场取消几十条慢运行后看板会显示系统「变快」（finish 的 CAS 要求非终态，
	// 取消后不会再有人替它补这个值）。未起跑的 queued 运行 StartedAt 为空，不写（由聚合侧排除）
	if run.StartedAt != nil {
		patch = patch.Set("costMs", now.Sub(*run.StartedAt).Milliseconds())
	}
	// 仅非终态可取消（集合真源见 entity.ActiveRunStatuses，与急停清场/重叠防护同一份）；
	// 条件更新避免与「认领」「终态写入」竞争
	rows, err := r.GetRepo().UpdateColsByCond(ctx, patch,
		model.NewCond().EqStrict("id", runId).In("status", entity.ActiveRunStatuses))
	if err != nil {
		return err
	}
	if rows != 1 {
		// 未改到行（已被其它路径置终态）：撤销取消标记，避免无谓残留
		r.inflightMu.Lock()
		delete(r.cancelReq, runId)
		r.inflightMu.Unlock()
		return ErrRunNotCancelable
	}
	switch {
	case cancel != nil:
		// 本实例在途则立即取消其 ctx，令 headless.Run 尽快返回
		cancel()
	case run.Status == entity.RunStatusQueued:
		// 取消的是尚未被认领的排队运行：本进程不会为它注册 cancel，其它进程认领也会因状态已终态
		// 而判赢失败 → 标记可安全回收（旧写法直到这里不删 → 长期运行+频繁取消时 map 只增不减）。
		// running 态绝不能删：那是「认领后、注册前」窗口，必须由 trackInflight 消费该标记
		r.inflightMu.Lock()
		delete(r.cancelReq, runId)
		r.inflightMu.Unlock()
	}
	logx.InfofContext(ctx, "[agentops-run] canceled: runId=%d, prevStatus=%s", runId, run.Status)
	return nil
}

// CancelAllActive 取消全部非终态运行（急停清场）：查出后逐个走 Cancel，
// 复用其条件更新与在途 ctx 取消逻辑；单个失败记日志继续。
// 返回实际取消数量：“已终态不可取消”属正常无事发生，不计入成功数（旧写法 Cancel 返回 nil
// 使计数把没改到行的运行也算进去，看板与前端都拿到夸大的清场结果）。
func (r *runServiceAppImpl) CancelAllActive(ctx context.Context) (int, error) {
	runs, err := r.GetRepo().SelectByCond(model.NewCond().In("status", entity.ActiveRunStatuses))
	if err != nil {
		return 0, err
	}
	canceled := 0
	for _, run := range runs {
		switch err := r.Cancel(ctx, run.Id); {
		case err == nil:
			canceled++
		case errors.Is(err, ErrRunNotCancelable):
			// 已被其它路径收尾：正常态，只记低优日志供对账
			logx.DebugfContext(ctx, "[agentops-run] cancel all: run already terminal: runId=%d", run.Id)
		default:
			logx.ErrorfContext(ctx, "[agentops-run] cancel all: cancel run failed: runId=%d, err=%v", run.Id, err)
		}
	}
	if canceled > 0 {
		logx.InfofContext(ctx, "[agentops-run] cancel all: total=%d, canceled=%d", len(runs), canceled)
	}
	return canceled, nil
}

// Start 启动有界工作池（幂等）
func (r *runServiceAppImpl) Start() {
	r.startMu.Lock()
	defer r.startMu.Unlock()
	if r.started {
		return
	}
	r.started = true

	if r.signalCh == nil {
		r.signalCh = make(chan struct{}, 1)
	}
	// stopCh 无条件重建：Stop 会 close 它，停机后再次 Start（测试重建实例、未来支持热重启）
	// 若沿用已关闭的 channel，新 worker 一启动就收到停止信号直接退出，工作池表现为「起了但永不取任」
	r.stopCh = make(chan struct{})
	if r.inflight == nil {
		r.inflight = make(map[uint64]context.CancelFunc)
	}
	if r.cancelReq == nil {
		r.cancelReq = make(map[uint64]struct{})
	}
	poolSize := r.PoolSize
	if poolSize <= 0 {
		poolSize = defaultRunWorkerPoolSize
	}
	for i := 0; i < poolSize; i++ {
		r.wg.Add(1)
		go r.worker()
	}
	// 租约清扫随工作池同生命周期启停（纳入同一个 wg/stopCh，不另建一套启停语义）
	r.wg.Add(1)
	go r.sweeper()
	logx.Infof("[agentops] run worker pool started: size=%d", poolSize)
}

// Stop 优雅停止工作池（幂等：停机钩子与测试收尾可能重复调用）
func (r *runServiceAppImpl) Stop() {
	r.startMu.Lock()
	if !r.started {
		r.startMu.Unlock()
		return
	}
	// 必须同时置位：否则二次 Stop 会 close 已关闭的 channel 直接 panic，而 panic 正好落在服务退出路径上
	r.started = false
	r.startMu.Unlock()

	close(r.stopCh)
	r.wg.Wait()
	logx.Infof("[agentops] run worker pool stopped")
}

// SweepInterruptedRuns 收敛租约过期的 running 运行（语义见接口注释）
func (r *runServiceAppImpl) SweepInterruptedRuns(ctx context.Context) error {
	now := time.Now()
	// 只按状态取候选，租约是否过期在 Go 侧按各行自己的超时判定：
	// 「started_at + timeout + 宽限期」里的 timeout 是 per-run 列，写进 SQL 要碰
	// MySQL/SQLite 两种时间运算方言且更易错；running 行数量有界（工作池并发上限），全取代价可忽略
	runs, err := r.GetRepo().SelectByCond(model.NewCond().EqStrict("status", entity.RunStatusRunning))
	if err != nil {
		return err
	}
	swept := 0
	for _, run := range runs {
		startedAt := run.StartedAt
		if startedAt == nil {
			// 异常形态（认领后未落开始时间）：退回创建时间，不让这类行永远扫不掉
			startedAt = run.CreateTime
			if startedAt == nil {
				continue
			}
		}
		lease := time.Duration(runLeaseSeconds(run)+runSweepGraceSeconds) * time.Second
		if now.Sub(*startedAt) < lease {
			continue // 租约未过期：可能确有实例在跑（多实例部署下的他人运行），不动它
		}
		// CAS 带 status=running：判定与写入之间若被真实执行者收尾/取消，本次即空操作（以库中终态为准）
		rows, err := r.GetRepo().UpdateColsByCond(ctx,
			base.NewPatch().
				Set("status", entity.RunStatusFailed).
				Set("finishedAt", now).
				Set("err", "run lease expired: no terminal state written within timeout"),
			model.NewCond().EqStrict("id", run.Id).EqStrict("status", entity.RunStatusRunning))
		if err != nil {
			return err
		}
		if rows == 0 {
			continue
		}
		swept++
		logx.WarnfContext(ctx, "[agentops-run] swept expired-lease run: runId=%d, runNo=%s, runnerInstance=%s, startedAt=%s",
			run.Id, run.RunNo, run.RunnerInstance, startedAt.Format("2006-01-02 15:04:05"))
	}
	if swept > 0 {
		logx.WarnfContext(ctx, "[agentops-run] lease sweep done: swept=%d", swept)
	}
	return nil
}

// runLeaseSeconds 该运行的租约基线秒数：自身超时 × 「阶段数」，未配置超时则取服务默认超时。
//
// 为什么必须乘阶段数：编排父运行与评测回放运行的 TimeoutSeconds 语义是**单步**超时
// （见 orchestrator 入参注释），而这两类运行直接以 running 落库、脱离工作池，由专属
// goroutine 串完 planner + N 步 + reviewer（评测是重放 + 评审两段）。按单步判租约，
// 跑到第 2 个阶段就会被清扫器腰斩：CAS 抢输真实执行者的收尾，编排报告与 token 汇总全部
// 丢失（编排步数上限 10，总耗时上限是单步的 12 倍）。现有测试抬不住是因为
// orchestrator_run_test 明确不启动工作池，sweeper 从不参与那个用例。
func runLeaseSeconds(run *entity.Run) int {
	base := run.TimeoutSeconds
	if base <= 0 {
		base = int(defaultRunTimeout / time.Second)
	}
	return base * runStageCount(run)
}

// evalTriggerRefPrefix 评测回放运行的 trigger_ref 前缀（同时也是其幂等键前缀）。
//
// 收成常量是因为清扫器要靠它识别「这类运行会串多个带超时的阶段」，而前缀字面量
// 之前在 eval_service 里写了两次：任何一处改动都会让租约基线静默退回单步口径
const evalTriggerRefPrefix = "eval:"

// runStageCount 该运行最多串多少个「各自带超时」的阶段（普通单步运行为 1）
func runStageCount(run *entity.Run) int {
	switch {
	case run.TriggerType == entity.TriggerTypeOrchestration:
		return 2 + maxOrchestrationSteps // planner + 最多 N 步 + reviewer
	case strings.HasPrefix(run.TriggerRef, evalTriggerRefPrefix):
		return 2 // 重放 + 评审打分
	default:
		return 1
	}
}

// sweeper 周期收敛租约过期的 running 运行（随工作池启停）。
//
// 不能只靠启动时扫一次：优雅停机钩子只在 SIGTERM/SIGINT 下生效，容器强杀（SIGKILL）、
// OOM 与滚动发布超时后的 kill -9 都会留下残留；周期清扫是这些场景下唯一的自愈路径
func (r *runServiceAppImpl) sweeper() {
	defer r.wg.Done()

	ticker := time.NewTicker(runSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			ctx := contextx.WithTraceId(context.Background())
			if err := r.SweepInterruptedRuns(ctx); err != nil {
				logx.ErrorfContext(ctx, "[agentops-run] periodic lease sweep failed: %v", err)
			}
		}
	}
}

// worker 工作池成员：drain 可认领的运行 → 无则等待信号/兜底轮询。
//
// recover 不能收在本函数出口（旧写法 `defer gox.Recover()` 就在此）：panic 沿栈上抛到函数级 recover 后，
// recover 只能让 goroutine 正常退出，回不到 for 循环——而 Start 只在启动时按固定数量拉起 worker，
// 于是池会只减不增地静默缩水；默认 4 个 worker 跑完 4 次 panic 后，所有 queued 运行再无人认领
// （任务保存成功、触发命中、执行历史永远停在 Queued），且只剩已被吞掉的 panic 栈可查。
// 收口方式：把恢复粒度降到「单次认领」（claimAndRun 内），worker 本体永不因一次运行失败而退出。
func (r *runServiceAppImpl) worker() {
	defer r.wg.Done()

	ticker := time.NewTicker(runWorkerPollInterval)
	defer ticker.Stop()

	for {
		// 持续认领并执行，直到无可认领（每次 claimAndRun 会阻塞至该运行结束）
		for {
			select {
			case <-r.stopCh:
				return
			default:
			}
			ran, err := r.claimAndRun()
			if err != nil {
				logx.Warnf("[agentops-run] claim/run error: %v", err)
			}
			if !ran {
				break
			}
		}

		select {
		case <-r.stopCh:
			return
		case <-r.signalCh: // 入队唤醒
		case <-ticker.C: // 兜底轮询（信号丢失/重启残留 queued）
		}
	}
}

// claimAndRun 认领并执行一个运行。返回是否认领到运行（供 worker 决定是否继续 drain）。
// panic 在本方法内收敛：已认领到的运行会被写成 failed 终态，worker 继续服务
func (r *runServiceAppImpl) claimAndRun() (ran bool, err error) {
	// ai 运行时尚未就绪（headless 未注册）时不认领，等待下次 tick，避免启动竞态误判失败
	if !headless.Registered() {
		return false, nil
	}
	run, err := r.GetRepo().ClaimNextQueued(context.Background(), time.Now(), runInstanceId)
	if err != nil || run == nil {
		return false, err
	}
	// 认领已生效：这一段一旦 panic，运行会永久停在 running（无人收尾），所以必须接住并补写终态
	var panicErr error
	func() {
		defer gox.Recover(func(e error) { panicErr = e })
		r.execute(run)
	}()
	if panicErr != nil {
		baseCtx := contextx.WithTraceId(context.Background())
		// 与正常收尾走同一个 finish：它的 CAS 保证「已被 Cancel/清扫收尾」时不覆写、也不误发通知
		r.finish(baseCtx, run, nil, errorx.NewBizI(baseCtx, imsg.ErrRunPanicked, "err", panicErr.Error()))
	}
	return true, nil
}

// execute 执行一个已认领（running）的运行：经 headless 无头运行 Agent，再写回终态。
// 返回本次运行的终态与「终态是否由本次写入」（同步调用方据此判断能否拿内存结果）
func (r *runServiceAppImpl) execute(run *entity.Run) (string, bool) {
	// 运行级链路 ctx（parent，不被 cancel/timeout 影响，供终态写库使用）
	baseCtx := contextx.WithTraceId(context.Background())
	// runCtx 可被 Cancel 取消（急停/人工取消），并向内传播到 headless 的 LLM 请求与工具执行
	runCtx, cancel := context.WithCancel(baseCtx)
	r.trackInflight(run.Id, cancel)
	defer func() {
		r.untrackInflight(run.Id)
		cancel()
	}()

	// 注入业务关联标识：使运行期工具（如 propose_remediation）能把产物（变更提案）
	// 回链到本 Run；会话 id 由 headless 运行解析会话后回填（见 agent_runner）。
	runCtx = headless.WithRunAssoc(runCtx, &headless.RunAssoc{BizRunId: run.Id})

	// 影子模式审计自洽：实际以只读档执行，回写生效档位（详情/列表展示与真实执行一致，
	// 原始配置档已在触发时快照进 Instruction/Input 上下文，不丢失排错信息）
	effProfileId, effInstruction := applyDryRun(run)
	if run.DryRun && run.ProfileId != headless.ProfileDiagnostician {
		if _, uerr := r.GetRepo().UpdateColsByCond(baseCtx,
			base.NewPatch().Set("profileId", headless.ProfileDiagnostician),
			model.NewCond().EqStrict("id", run.Id)); uerr != nil {
			logx.WarnfContext(baseCtx, "[agentops-run] dry-run profile rewrite failed (non-fatal): runNo=%s, err=%v", run.RunNo, uerr)
		}
		run.ProfileId = headless.ProfileDiagnostician
	}

	timeout := time.Duration(run.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultRunTimeout
	}

	logx.InfofContext(baseCtx, "[agentops-run] executing: runNo=%s, profile=%s, trigger=%s/%s",
		run.RunNo, run.ProfileId, run.TriggerType, run.TriggerRef)

	// 重试循环：仅超时/连接类可重试（业务失败/取消不重试，避免重复消耗 LLM 与误重试）。
	// 每次尝试经 headless.Run 内部各自施加 Timeout，故每次都是新鲜超时窗口；
	// runCtx 被 Cancel 后后续尝试快速失败，isRetryableRunResult 也会因 ctx.Err() 非空而停止重试。
	attempts := int(run.RetryTimes) + 1
	if attempts < 1 {
		attempts = 1
	}
	var res *headless.RunResult
	var runErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		res, runErr = headless.Run(runCtx, &headless.RunReq{
			ProfileId:      effProfileId,
			Instruction:    effInstruction,
			Input:          run.Input,
			Title:          run.Title,
			OwnerAccountId: run.OwnerAccountId,
			OwnerUsername:  run.OwnerUsername,
			Timeout:        timeout,
			MaxStep:        run.MaxStep,
			MaxTotalTokens: run.TokenBudget,
			Persist:        true, // 运行默认落库会话/TurnItem，获得审计与回放
			RunKind:        triggerTypeToRunKind(run.TriggerType),
			// 系统任务触发的会话标记 system：不进用户 AI 助手会话列表，
			// 与用户发起的 chat 区分；回放经执行历史按 convId 直达
			ConversationSource: aientity.ConversationSourceSystem,
			// 一次性工具许可（从运行行自身读回，同步/异步两条路径同源）：受控执行只能调清单内工具
			AllowedToolNames: run.AllowedToolNames(),
			ApprovedTools:    run.ApprovedActions(),
		})
		if runErr == nil && (res == nil || res.Status == headless.RunStatusSucceeded) {
			break
		}
		if attempt == attempts || !isRetryableRunResult(runCtx, res, runErr) {
			break
		}
		logx.WarnfContext(baseCtx, "[agentops-run] attempt %d/%d failed (retryable): runNo=%s, err=%v",
			attempt, attempts, run.RunNo, runErr)
		time.Sleep(time.Duration(attempt) * runRetryBackoffBase)
	}

	// 终态写库用 baseCtx（非取消/非超时），保证超时/取消后仍能落库结论
	return r.finish(baseCtx, run, res, runErr)
}

// finish 依据无头运行结果写入终态，返回（终态, 是否抢到写入）。
//
// 仅当仍为 running 才写（CAS），避免覆盖 Cancel 写入的 canceled。抢输时必须**不发通知**：
// 内存里算出的 status 不是库中真值（库已停在 canceled/failed），而通知不可回滚——
// 旧写法丢弃影响行数、无条件外发，会把「已被取消的运行」推成「运行成功 + 结论」，
// NotifyType=仅失败 时又会被推成一条假告警（本函数注释自己立的「通知不得先于状态写入」规矩，
// 不判赢就等于没生效）
func (r *runServiceAppImpl) finish(ctx context.Context, run *entity.Run, res *headless.RunResult, runErr error) (string, bool) {
	now := time.Now()
	patch := base.NewPatch().Set("finishedAt", now)
	if run.StartedAt != nil {
		patch = patch.Set("costMs", now.Sub(*run.StartedAt).Milliseconds())
	}

	status := entity.RunStatusSucceeded
	errMsg := ""
	if res != nil {
		if res.ConversationId != 0 {
			patch = patch.Set("conversationId", res.ConversationId)
		}
		if res.TurnId != "" {
			patch = patch.Set("turnId", res.TurnId)
		}
		patch = patch.Set("summary", res.Output)
		if res.Usage != nil {
			patch = patch.
				Set("promptTokens", res.Usage.PromptTokens).
				Set("completionTokens", res.Usage.CompletionTokens).
				Set("totalTokens", res.Usage.TotalTokens)
		}
		status = mapHeadlessStatus(res.Status)
	}
	if runErr != nil {
		errMsg = runErr.Error()
		if status == entity.RunStatusSucceeded {
			status = entity.RunStatusFailed
		}
	}
	if errMsg != "" {
		patch = patch.Set("err", truncateRunErr(errMsg))
	}
	// 工具级失败随终态一并落库（进 Extra，非查询维度不单独立列）：run 本身可以成功，
	// 但「变更有没有真发生」必须能被下游读到（否则提案会被记成已执行）
	if res != nil && len(res.ToolErrors) > 0 {
		run.SetToolErrors(res.ToolErrors)
		patch = patch.Set("extra", run.Extra)
		logx.WarnfContext(ctx, "[agentops-run] tool-level failures recorded: runNo=%s, errors=%d, first=%s",
			run.RunNo, len(res.ToolErrors), res.ToolErrors[0])
	}
	patch = patch.Set("status", status)

	rows, err := r.GetRepo().UpdateColsByCond(ctx, patch,
		model.NewCond().EqStrict("id", run.Id).EqStrict("status", entity.RunStatusRunning))
	if err != nil {
		logx.ErrorfContext(ctx, "[agentops-run] finish update failed: runNo=%s, err=%v", run.RunNo, err)
		return status, false
	}
	if rows != 1 {
		// 没抢到终态写入：运行已被取消或已被启动清扫收尾，库中状态才是真值，只记日志不外发通知
		logx.WarnfContext(ctx, "[agentops-run] finish skipped, terminal state written elsewhere: runNo=%s, computedStatus=%s",
			run.RunNo, status)
		return status, false
	}
	logx.InfofContext(ctx, "[agentops-run] finished: runNo=%s, status=%s, totalTokens=%d",
		run.RunNo, status, totalTokensOf(res))

	// 终态落库成功（抢到写入）才通知（通知不可回滚，不得先于状态写入）；失败不影响运行终态
	r.notifyResult(ctx, run, status, res, errMsg)
	return status, true
}

// notifyResult 按运行的通知方式与绑定模板推送结果（与告警/机器计划任务同一发送器）。
// msgTmplApp 为 nil（单测未注入）或未配模板/不通知时直接返回。
func (r *runServiceAppImpl) notifyResult(ctx context.Context, run *entity.Run, status string, res *headless.RunResult, errMsg string) {
	if r.msgTmplApp == nil || run.NotifyType == entity.TaskNotifyNone || run.NotifyTmplCode == "" {
		return
	}
	failed := status == entity.RunStatusFailed
	if run.NotifyType == entity.TaskNotifyFail && !failed {
		return
	}
	summary := ""
	if res != nil {
		summary = res.Output
	}
	// 参数键与内置「运维诊断结果通知」模板占位符同源（见 entity.DiagnosisNotifyTmpl），
	// 任一侧改名/增删都会被 entity 的模板覆盖测试拦下，不会再退化成 <no value>
	params := entity.NewDiagnosisNotifyParams(run.Title, run.RunNo, run.TriggerType, status, summary, errMsg, totalTokensOf(res))
	if err := r.msgTmplApp.Send(ctx, run.NotifyTmplCode, params); err != nil {
		logx.ErrorfContext(ctx, "[agentops-run] result notify failed: runNo=%s, tmpl=%s, err=%v",
			run.RunNo, run.NotifyTmplCode, err)
	}
}

// signal 唤醒工作池（合并多次入队为一次唤醒；未启动时忽略，由兜底轮询驱动）
func (r *runServiceAppImpl) signal() {
	if r.signalCh == nil {
		return
	}
	select {
	case r.signalCh <- struct{}{}:
	default:
	}
}

func (r *runServiceAppImpl) trackInflight(runId uint64, cancel context.CancelFunc) {
	r.inflightMu.Lock()
	if r.inflight == nil {
		r.inflight = make(map[uint64]context.CancelFunc)
	}
	if _, pending := r.cancelReq[runId]; pending {
		// 取消已先于注册到达：消费标记并 born-canceled，避免运行悬挂
		delete(r.cancelReq, runId)
		r.inflightMu.Unlock()
		cancel()
		return
	}
	r.inflight[runId] = cancel
	r.inflightMu.Unlock()
}

func (r *runServiceAppImpl) untrackInflight(runId uint64) {
	r.inflightMu.Lock()
	delete(r.inflight, runId)
	delete(r.cancelReq, runId)
	r.inflightMu.Unlock()
}

// runKindByTriggerType 触发类型 → headless 运行来源的表驱动映射：
// 新增触发类型只需在此登记，不会遗漏 switch 分支（未知类型保守回退 manual）
var runKindByTriggerType = map[string]string{
	entity.TriggerTypeCron:          headless.RunKindCron,
	entity.TriggerTypeEvent:         headless.RunKindEvent,
	entity.TriggerTypeOrchestration: headless.RunKindOrchestration,
}

// triggerTypeToRunKind 触发类型 → headless 运行来源（词汇一致但独立定义，显式映射防漂移）
func triggerTypeToRunKind(triggerType string) string {
	if kind, ok := runKindByTriggerType[triggerType]; ok {
		return kind
	}
	return headless.RunKindManual
}

// headlessStatusMap headless 终态 → Run 状态机终态映射（表驱动）
var headlessStatusMap = map[string]string{
	headless.RunStatusSucceeded: entity.RunStatusSucceeded,
	headless.RunStatusCanceled:  entity.RunStatusCanceled,
	// 中断挂起在无头运行里等价于「没有执行者」：必须落 failed。
	// 否则提案受控执行会把「一次写操作都没发生」判成 executed（proposal_gateway 的
	// execFailed 判定包含 status != succeeded，这里定下来后那条链自动恢复可信）
	headless.RunStatusInterrupted: entity.RunStatusFailed,
}

// mapHeadlessStatus headless 终态 → Run 状态机终态（超时归入 failed，语义保留在 err）。
// 未知/空值保守归 failed：无法确认结果时不能报成功
func mapHeadlessStatus(headlessStatus string) string {
	if status, ok := headlessStatusMap[headlessStatus]; ok {
		return status
	}
	return entity.RunStatusFailed
}

// applyDryRun 影子模式（演练）生效变换：强制降为只读诊断档（CapabilityGuard 保证任何
// write/dangerous 工具调用被拦截），并在指令前置「仅产出将执行什么、不得落地变更」约束，
// 用于安全验证任务指令/编排逻辑。非影子模式原样返回。
func applyDryRun(run *entity.Run) (profileId, instruction string) {
	profileId, instruction = run.ProfileId, run.Instruction
	if !run.DryRun {
		return
	}
	profileId = headless.ProfileDiagnostician
	instruction = dryRunInstructionPrefix + "\n\n" + instruction
	return
}

// dryRunInstructionPrefix 影子模式的强制约束前缀（与 Profile 静态指令叠加，随消息下发）
const dryRunInstructionPrefix = `【影子模式 / DRY-RUN】本次运行是演练，硬性约束：
1. 严禁执行任何会改变目标资源状态的写操作；
2. 对需要变更的环节，只输出「将执行什么」（动作、工具、参数、预期影响面），不实际执行；
3. 结论中明确标注这是影子模式的演练产出。`

// isRetryableRunResult 判定一次运行结果是否可重试（仅超时/连接类）。
//
// 不重试的情形：外层 ctx 已取消/超时（急停/人工取消，重试无意义）、业务失败
// （Agent 已产出结果但逻辑失败，重试只会重复消耗）、 canceled 终态。
func isRetryableRunResult(ctx context.Context, res *headless.RunResult, runErr error) bool {
	if ctx.Err() != nil {
		return false // 已被取消/外层超时，不重试
	}
	if res != nil && res.Status == headless.RunStatusTimeout {
		return true // 模型/工具超时：典型瞬时可重试
	}
	var netErr net.Error
	if runErr != nil && errors.As(runErr, &netErr) {
		return true // 网络类错误（模型服务不可达等）：保守仅覆盖明确的 net.Error
	}
	return false
}

func totalTokensOf(res *headless.RunResult) int64 {
	if res == nil || res.Usage == nil {
		return 0
	}
	return int64(res.Usage.TotalTokens)
}

// promptTokensOf / completionTokensOf 取无头运行的输入/输出用量（nil 安全，与 totalTokensOf 同一族）：
// 三者必须同取，否则回写后「输入 + 输出 ≠ 总数」，看板的成本归因指标直接失真
func promptTokensOf(res *headless.RunResult) int64 {
	if res == nil || res.Usage == nil {
		return 0
	}
	return int64(res.Usage.PromptTokens)
}

func completionTokensOf(res *headless.RunResult) int64 {
	if res == nil || res.Usage == nil {
		return 0
	}
	return int64(res.Usage.CompletionTokens)
}

// truncateRunErr 截断错误信息至列宽，避免写库溢出
func truncateRunErr(err string) string {
	rs := []rune(err)
	if len(rs) <= maxRunErrLen {
		return err
	}
	return string(rs[:maxRunErrLen])
}
