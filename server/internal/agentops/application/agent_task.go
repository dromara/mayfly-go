package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/ai/pkg/headless"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/stringx"
)

const (
	// agentTaskCronLockDuration cron 触发的分布式锁 TTL：RunByKey 仅做「查重 + 入队」毫秒级返回，
	// 30s 足以覆盖并留足余量；多实例部署时同一任务同一触发点仅一个实例入队（去重）
	agentTaskCronLockDuration = 30 * time.Second
)

// AgentTaskService 定时计划任务应用服务（对标 machine_cronjob，但驱动无头 Agent 运行）。
//
// 职责：任务 CRUD + cron 绑定/解绑 + 到点触发（RunByKey）经 RunService.Enqueue 产出 Run +
// 手动触发一次（RunOnce）+ 重启 misfire 处理。触发后的重试/通知在 RunService 生命周期内完成
// （配置已快照到 Run），本服务只负责「何时、以何配置入队」。
type AgentTaskService interface {
	base.App[*entity.AgentTask]

	// GetPageList 分页查询任务
	GetPageList(condition *entity.AgentTask, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.AgentTask], error)

	// SaveTask 保存任务（校验 + 落库 + 重新绑定 cron）
	SaveTask(ctx context.Context, task *entity.AgentTask) error

	// Delete 删除任务（解绑 cron）
	Delete(ctx context.Context, id uint64) error

	// Init 启动扫描：绑定全部启用任务的 cron，并对重启期间错过的窗口按策略处理（misfire）
	Init()

	// RunByKey cron 触发入口（按任务 key）：重叠防护 → 入队（trigger=cron）→ 记录触发时间
	RunByKey(key string)

	// RunOnce 手动触发一次（按任务 id，trigger=manual）：不做重叠防护，用户显式触发即入队
	RunOnce(ctx context.Context, id uint64) (*entity.Run, error)
}

type agentTaskAppImpl struct {
	base.AppImpl[*entity.AgentTask, repository.AgentTask]

	// runService 运行聚合服务：任务触发的运行统一经其 Enqueue 入队（有界工作池异步执行）
	runService RunService `inject:"T"`

	// governance 治理服务：急停守护（单测未装配治理时为 nil，守护点已做 nil 守卫跳过）
	governance Governance `inject:"T"`

	// triggerRepo 绑定关系查询：删任务时要把「引用了这个任务」的触发器一并处理
	triggerRepo repository.Trigger `inject:"T"`

	// msgTmplApp 结果通知模板校验：保存期确认模板真的发得出去
	//（单测绕过 IOC 注入时为 nil，checkNotifyTmplSendable 已做 nil 守卫跳过）
	msgTmplApp msgapp.MsgTmpl `inject:"T"`
}

var _ AgentTaskService = (*agentTaskAppImpl)(nil)

// GetPageList 分页查询任务
func (a *agentTaskAppImpl) GetPageList(condition *entity.AgentTask, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.AgentTask], error) {
	return a.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

// SaveTask 保存任务：先校验（会被调度的任务表达式必须合法、要通知必须有可用模板、必须有运行身份），
// 再落库，最后按最新配置重新绑定/解绑 cron。
func (a *agentTaskAppImpl) SaveTask(ctx context.Context, task *entity.AgentTask) error {
	if task == nil {
		return errorx.NewBizI(ctx, imsg.ErrTaskNil)
	}
	// fail-closed：无运行身份拒绝保存（触发的运行按 owner 收敛工具权限，缺省会越权/被 Enqueue 拒绝）
	if task.OwnerAccountId == 0 {
		return errorx.NewBizI(ctx, imsg.ErrTaskOwnerRequired)
	}
	if strings.TrimSpace(task.Instruction) == "" {
		return errorx.NewBizI(ctx, imsg.ErrTaskInstructionRequired)
	}
	// 会被调度的任务先校验表达式：绑定失败只写服务端日志，不拦下就是「保存成功但永不执行」
	if taskScheduled(task) {
		if err := scheduler.ValidateSpec(task.Cron); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrTaskCronInvalid, "err", err.Error())
		}
	}
	// 要通知就必须有模板：缺模板时发送逻辑直接 return，用户以为配了通知却永远收不到
	if task.NotifyType != entity.TaskNotifyNone && task.NotifyTmplCode == "" {
		return errorx.NewBizI(ctx, imsg.ErrTaskNotifyTmplRequired)
	}
	// 模板存在也还不够：已停用或未关联渠道同样发不出去，而那只在运行收尾留一条服务端日志
	//（见 checkNotifyTmplSendable 的成因说明）
	if err := checkNotifyTmplSendable(ctx, a.msgTmplApp, task.NotifyType, task.NotifyTmplCode); err != nil {
		return err
	}
	if task.ProfileId == "" {
		task.ProfileId = headless.ProfileDiagnostician
	}
	// 档位必须是已注册且可用于无人值守派发的档（见 profile_guard.go 的成因说明）
	if err := validateDispatchProfile(ctx, task.ProfileId); err != nil {
		return err
	}

	// key：创建时生成、更新时沿用旧值（key 是 cron 绑定与 Run.trigger_ref 的稳定标识，不可变）
	if task.Id == 0 {
		task.Key = stringx.Rand(16)
	} else {
		old, err := a.GetById(task.Id)
		if err != nil {
			return errorx.NewBizI(ctx, imsg.ErrTaskNotFound)
		}
		task.Key = old.Key
		// 继承库中现值 Extra：展示字段（owner 账号名/备注）现在随 extra 整列写回，
		// 不先合并会把本次表单不携带的键一并抹掉
		mergeExtra(&task.ExtraData, old.ExtraData)
	}

	// 落库：新建走 Insert；更新按表单可编辑列整列写入（零值真实写入）。
	// 不可改走 App.Save 的结构体更新：那会跳过 dry_run=false、notify_type=0、misfire_policy=0、
	// target_resources=''、cron='' 这类「改回零值」的编辑，表现为保存成功但库里仍旧值
	if err := a.saveEntity(ctx, task); err != nil {
		return err
	}
	a.bindCronTask(task)
	return nil
}

// saveEntity 落库任务配置：Id 为 0 走新增，否则按 entity.TaskEditableColumns 整列更新
func (a *agentTaskAppImpl) saveEntity(ctx context.Context, task *entity.AgentTask) error {
	if task.Id == 0 {
		return a.Insert(ctx, task)
	}
	return a.GetRepo().UpdateById(ctx, task, saveColumns(entity.TaskEditableColumns)...)
}

// Delete 删除任务并解绑 cron
func (a *agentTaskAppImpl) Delete(ctx context.Context, id uint64) error {
	task, err := a.GetById(id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrTaskNotFound)
	}
	if err := a.DeleteById(ctx, id); err != nil {
		return err
	}
	taskx.UnbindCronTask(task.Key)

	// 级联处理绑定该任务的触发器：置为禁用，而不是静默把任务配置复制进触发器。
	//
	// 为什么选禁用：复制快照会让触发器此后以「看不见的配置」运行（用户以为还在用任务，
	// 实际用的是复制品，改任务不再影响它）；禁用则让下一次派发明确失败并在日志里
	// 指名道姓，运维能立刻看到并自己决定改成内联还是换绑任务。
	// 失败不阻断删任务（任务已删是事实），但必须留痕
	if a.triggerRepo == nil {
		// 单测里手工构造的实例不注入仓储（与 governance 同一约定）：无仓可查时不做级联
		return nil
	}
	triggers, err := a.triggerRepo.SelectByCond(model.NewCond().Eq("task_id", id))
	if err != nil {
		logx.WarnfContext(ctx, "[agentops-task] list triggers bound to deleted task[%d] failed: %v", id, err)
		return nil
	}
	for _, trigger := range triggers {
		if trigger.Enabled != entity.TriggerEnabled {
			continue
		}
		trigger.Enabled = entity.TriggerDisabled
		if uerr := a.triggerRepo.UpdateById(ctx, trigger); uerr != nil {
			logx.WarnfContext(ctx, "[agentops-task] disable trigger[%d] bound to deleted task[%d] failed: %v", trigger.Id, id, uerr)
			continue
		}
		logx.InfofContext(ctx, "[agentops-task] disabled trigger[%d] because bound task[%d] was deleted", trigger.Id, id)
	}
	return nil
}

// Init 启动扫描：绑定全部启用任务的 cron + 处理重启期间错过的触发窗口。
//
// 由 starter 在 DB 迁移建表之后调用；单条任务处理失败不影响其余任务（逐条 recover 由
// CursorByCond 的 handler 返回值控制，这里对每条独立兜底日志）。
func (a *agentTaskAppImpl) Init() {
	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTrace("the agent task failed to initialize: %v", err)
		}
	}()

	ctx := contextx.WithTraceId(context.Background())
	if err := a.CursorByCond(&entity.AgentTask{Status: entity.AgentTaskStatusEnable}, func(task *entity.AgentTask) error {
		a.bindCronTask(task)
		a.handleMisfire(ctx, task)
		return nil
	}); err != nil {
		logx.ErrorTrace("the agent task failed to initialize: %v", err)
	}
}

// RunByKey cron 触发入口：定位任务 → 经统一守护序列派发（急停/重叠）→ 记录触发时间。
//
// 定时触发没有请求上下文：显式造带 traceId 的任务级 ctx 往下传，使本轮运行记录能与日志串联。
// 多实例同一触发点的去重由 cron 绑定时加的分布式锁（AddFunByKeyWithLock）保证；本方法内的重叠防护
// 针对「同一任务上一次运行尚未收尾」的堆叠（异步模型下 Enqueue 即返回，不能靠持锁覆盖运行全程）。
func (a *agentTaskAppImpl) RunByKey(key string) {
	ctx := contextx.WithTraceId(context.Background())

	task := new(entity.AgentTask)
	task.Key = key
	// 不存在或禁用：解绑该 key 的 cron（任务可能已被删除/停用）
	if err := a.GetByCond(task); err != nil || task.Status == entity.AgentTaskStatusDisable {
		taskx.UnbindCronTask(key)
		return
	}

	if _, err := a.dispatchTask(ctx, task, taskDispatch{TriggerType: entity.TriggerTypeCron}); err != nil {
		// 急停拦住是人的显式意图，不是故障：dispatchTask 已经按 Info 记了一次「rejected by guard」，
		// 这里再按 Error 记一次会让急停期间每个触发点都刷一条 ERROR，「看错误数」从此不可用
		// （重叠跳过的 SkipOverlapGuard/nil 回 nil，不会走到这里）
		if errors.Is(err, ErrKillSwitchEngaged) {
			return
		}
		logx.ErrorfContext(ctx, "[agentops-task] dispatch failed: key=%s, err=%v", key, err)
	}
}

// RunOnce 手动触发一次（trigger=manual）：跳过重叠防护（用户显式触发即入队），仍受急停守护
func (a *agentTaskAppImpl) RunOnce(ctx context.Context, id uint64) (*entity.Run, error) {
	task, err := a.GetById(id)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrTaskNotFound)
	}
	return a.dispatchTask(ctx, task, taskDispatch{TriggerType: entity.TriggerTypeManual, SkipOverlapGuard: true})
}

// taskDispatch 任务派发入参（cron/manual/misfire 共用 dispatchTask 这一条守护序列）
type taskDispatch struct {
	// TriggerType 写入 Run 的触发类型（cron/manual）
	TriggerType string
	// IdempotencyKey 幂等键；空则由 RunService 生成天然唯一键（每次触发都是新运行，不参与去重）
	IdempotencyKey string
	// SkipOverlapGuard 跳过「同任务上一次运行尚未收尾」防护（仅人工显式触发）
	SkipOverlapGuard bool
}

// dispatchTask 任务派发的唯一守护序列：急停 → 重叠防护 → 入队 → 记录触发时间。
//
// 为什么必须收成一个序列：这三道判定曾分散在 RunByKey/RunOnce/事件派发/webhook/编排各入口手写，
// 启动期的 misfire 补跑因此三道全漏（「刚出事故、刚重启」恰恰是最需要急停的时刻）。新增派发源
// 只要走本方法就天然带全套守护，不可能漏项；急停的谓词定义与结构收口分别见
// Governance.GuardDispatch 与 RunService.Enqueue（本处的判定是同一谓词的前置调用，非重复实现）。
func (a *agentTaskAppImpl) dispatchTask(ctx context.Context, task *entity.AgentTask, d taskDispatch) (*entity.Run, error) {
	if err := guardDispatch(a.governance); err != nil {
		logx.InfofContext(ctx, "[agentops-task] dispatch rejected by guard: key=%s, err=%v", task.Key, err)
		return nil, err
	}

	if !d.SkipOverlapGuard {
		active, err := a.runService.HasActiveRun(ctx, d.TriggerType, task.Key)
		if err != nil {
			return nil, err
		}
		if active {
			logx.InfofContext(ctx, "[agentops-task] skip fire, previous run still active: key=%s", task.Key)
			return nil, nil
		}
	}

	run, err := a.enqueueRun(ctx, task, d)
	if err != nil {
		return nil, err
	}
	a.markFired(ctx, task)
	return run, nil
}

// enqueueRun 由任务配置构造入队请求并经 RunService 入队（配置快照到 Run，使运行自持）。
func (a *agentTaskAppImpl) enqueueRun(ctx context.Context, task *entity.AgentTask, d taskDispatch) (*entity.Run, error) {
	profileId := task.ProfileId
	if profileId == "" {
		profileId = headless.ProfileDiagnostician
	}
	return a.runService.Enqueue(ctx, &EnqueueReq{
		TriggerType:    d.TriggerType,
		TriggerRef:     task.Key,
		ProfileId:      profileId,
		Title:          task.Name,
		Instruction:    instructionText(task.Instruction, task.InstructionSegments),
		Input:          buildTaskInput(task),
		OwnerAccountId: task.OwnerAccountId,
		OwnerUsername:  task.OwnerUsername,
		TimeoutSeconds: task.TimeoutSeconds,
		MaxStep:        task.MaxStep,
		TokenBudget:    task.TokenBudget,
		DryRun:         task.DryRun,
		RetryTimes:     task.RetryTimes,
		NotifyType:     task.NotifyType,
		NotifyTmplCode: task.NotifyTmplCode,
		IdempotencyKey: d.IdempotencyKey,
	})
}

// bindCronTask 按任务最新配置绑定/解绑 cron（带分布式锁，支持多实例部署）
func (a *agentTaskAppImpl) bindCronTask(task *entity.AgentTask) {
	key := task.Key
	if !taskScheduled(task) {
		taskx.UnbindCronTask(key)
		return
	}
	if err := taskx.BindCronTaskWithLock(key, task.Cron, agentTaskCronLockDuration, true, func() {
		defer gox.Recover()
		a.RunByKey(key)
	}); err != nil {
		logx.ErrorTrace("bind agent task cron failed", err)
	}
}

// handleMisfire 处理进程重启期间错过的触发窗口。
//
// in-memory 调度器不持久化下次触发点：以 LastFireTime 为基线用 cron 算出「本应触发的下一次时间」，
// 若该时间已过去（说明重启期间到点却没跑），按 MisfirePolicy 处理——CatchUpOnce 补跑一次，
// Skip（默认）不补跑。LastFireTime 为空表示从未触发过，无 misfire 可言。
//
// 补跑必须走 dispatchTask：急停/重叠守护与 cron 触发同源，不得因为「这是启动兜底路径」而豁免。
// 多实例去重靠确定性幂等键（misfire:<key>:<本应触发时间>）：各实例算出的本应触发时间一致，
// 滚动重启时同一次错过只产出一个补跑运行（Run 的幂等键唯一索引即裁决点，无需额外分布式锁）。
func (a *agentTaskAppImpl) handleMisfire(ctx context.Context, task *entity.AgentTask) {
	if task.MisfirePolicy != entity.MisfirePolicyCatchUpOnce || task.LastFireTime == nil {
		return
	}
	next, err := scheduler.NextFireTime(task.Cron, *task.LastFireTime)
	if err != nil {
		logx.WarnfContext(ctx, "[agentops-task] misfire check skipped, invalid cron: key=%s, err=%v", task.Key, err)
		return
	}
	now := time.Now()
	if !next.Before(now) {
		return // 下一次触发尚未到点，未错过
	}
	logx.InfofContext(ctx, "[agentops-task] misfire catch-up once: key=%s, missedFire=%s", task.Key, next.Format(time.DateTime))
	if _, err := a.dispatchTask(ctx, task, taskDispatch{
		TriggerType:    entity.TriggerTypeCron,
		IdempotencyKey: misfireIdempotencyKey(task.Key, next),
	}); err != nil {
		logx.ErrorfContext(ctx, "[agentops-task] misfire catch-up dispatch failed: key=%s, err=%v", task.Key, err)
	}
}

// misfireIdempotencyKey misfire 补跑的幂等键：以「本应触发时间」为组成部分，
// 使多实例/多次重启对同一次错过只派生一个运行
func misfireIdempotencyKey(key string, missedFire time.Time) string {
	return "misfire:" + key + ":" + missedFire.UTC().Format(time.RFC3339)
}

// markFired 记录本次触发时间（misfire 判定基线）；best-effort，失败仅记日志
func (a *agentTaskAppImpl) markFired(ctx context.Context, task *entity.AgentTask) {
	now := time.Now()
	if _, err := a.GetRepo().UpdateColsByCond(ctx,
		base.NewPatch().Set("lastFireTime", now),
		model.NewCond().EqStrict("id", task.Id)); err != nil {
		logx.ErrorfContext(ctx, "[agentops-task] mark fired failed: key=%s, err=%v", task.Key, err)
	}
}

// taskScheduled 任务是否会被注册为定时任务：禁用态或空表达式只解绑不注册。
// 保存前的表达式校验与 bindCronTask 的绑定判定共用此函数，避免两处口径漂移。
func taskScheduled(task *entity.AgentTask) bool {
	return task.Status != entity.AgentTaskStatusDisable && strings.TrimSpace(task.Cron) != ""
}

// buildTaskInput 构造运行输入（user 消息）：任务名 + 执行要求。
//
// 「对哪些资源执行」不再是独立字段：指令里的 @资源引用已随 system 指令展开为完整定位信息
// （见 instructionText），因此这里只负责一件事——指令没有引用任何资源时显式引导先盘点，
// 否则模型会凭空猜资源 id/名称、乱调工具浪费 token（历史实测踩过）。
func buildTaskInput(task *entity.AgentTask) string {
	var sb strings.Builder
	sb.WriteString("任务：" + task.Name + "\n\n")
	if !hasResourceRef(task.InstructionSegments) {
		sb.WriteString("任务指令未引用具体资源：请先调用 ListResources 盘点可用资源，再从中确定本次执行的目标（禁止凭空猜测资源 id/名称）。\n\n")
	}
	sb.WriteString("请严格按任务指令执行，并产出结构化结论。")
	return sb.String()
}
