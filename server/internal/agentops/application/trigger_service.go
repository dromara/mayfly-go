package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/internal/ai/pkg/headless"
	msgapp "mayfly-go/internal/msg/application"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

const (
	// webhookTokenBytes 推送令牌的随机字节数（hex 编码后 32 字符）。
	// 该令牌是免鉴权 webhook 入口的唯一鉴权因子（拿到即可驱动 LLM 运行、以绑定 owner 身份读监控数据），
	// 必须用 CSPRNG 生成，见 newWebhookToken
	webhookTokenBytes = 16
	// webhookTokenRotateAttempts 轮换时的取号重试上限（唯一性碰撞概率极低，仅兜底）
	webhookTokenRotateAttempts = 3
	// webhookDefaultOccurrenceWindowSec webhook 载荷未带 occurredAt 时的默认「一次投递窗口」（秒）：
	// 同窗口内重试共用一个幂等键，跨窗口即视为新事件
	webhookDefaultOccurrenceWindowSec = 60
)

// newWebhookToken 生成推送令牌。
//
// 为什么不用 stringx.Rand：它走 math/rand。对「名字随机串」够用，对鉴权凭据不够用：
// 伪随机序列的内部状态可被观测到的若干令牌反推，而本令牌对应路由 .DontNeedToken()（路由级不校
// token），拿到就等于拿到一条「无人值守、以 owner 身份跑诊断」的入口。随机性在此是安全边界，不是防碰撞。
func newWebhookToken(ctx context.Context) (string, error) {
	buf := make([]byte, webhookTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		// 不向模型/外部透递密码学层错误细节，统一转为可读的业务错误
		return "", errorx.NewBizI(ctx, imsg.ErrTriggerTokenGenerateFailed)
	}
	return hex.EncodeToString(buf), nil
}

// TriggerService 事件驱动触发器应用服务（告警/webhook → 自动诊断 Run 的派发中枢）。
//
// 职责：触发器 CRUD + 事件派发（Dispatch/DispatchWebhook）。派发链路统一为
// 「匹配启用触发器 → 冷却裁决 → 解析运行配置（关联任务 或 内联）→ 幂等键 → RunService.Enqueue
// （trigger=event）」，产出只读诊断 Run。急停守护不在这里重复实现：它收在 Enqueue 结构入口。
// 触发源的可扩展性由 trigger.Registry 承载（开闭原则），本服务只消费注册表，新增来源零改动。
type TriggerService interface {
	base.App[*entity.Trigger]

	// GetPageList 分页查询触发器
	GetPageList(condition *entity.Trigger, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Trigger], error)

	// SaveTrigger 保存触发器（校验来源/运行配置/匹配条件 + webhook 令牌服务端治理 + 落库）
	SaveTrigger(ctx context.Context, t *entity.Trigger) error

	// RotateWebhookToken 显式轮换指定 webhook 触发器的推送令牌，返回新令牌。
	// 轮换只有一个显式入口：保存路径一律保留库中现值，避免不带令牌的编辑静默换掉外部已配地址
	RotateWebhookToken(ctx context.Context, id uint64) (string, error)

	// Delete 删除触发器
	Delete(ctx context.Context, id uint64) error

	// Dispatch 派发一个「已归一化」的中立事件（EventBus 订阅入口，如告警联动）：
	// 匹配该来源全部启用触发器，逐个冷却去抖后派生诊断 Run。单条失败不影响其余。
	Dispatch(ctx context.Context, evt *event.OpsTriggerEvent) error

	// DispatchWebhook 处理 webhook 推送：按令牌定位触发器 → 归一化原始载荷 → 匹配/冷却 → 派生 Run。
	// 令牌即鉴权：无效/禁用/非 webhook 来源一律拒绝。
	DispatchWebhook(ctx context.Context, token string, raw any) (*entity.Run, error)
}

type triggerServiceAppImpl struct {
	base.AppImpl[*entity.Trigger, repository.Trigger]

	// runService 运行聚合服务：触发器派生的运行统一经其 Enqueue 入队（有界工作池异步执行）
	runService RunService `inject:"T"`
	// agentTaskApp 关联任务模式（task_id>0）时复用任务全套运行配置
	agentTaskApp AgentTaskService `inject:"T"`

	// governance 治理服务：急停守护（单测未装配治理时为 nil，守护点已做 nil 守卫跳过）
	governance Governance `inject:"T"`

	// msgTmplApp 结果通知模板校验：保存期确认内联通知配置真的发得出去
	//（单测绕过 IOC 注入时为 nil，checkNotifyTmplSendable 已做 nil 守卫跳过）
	msgTmplApp msgapp.MsgTmpl `inject:"T"`
}

var _ TriggerService = (*triggerServiceAppImpl)(nil)

// GetPageList 分页查询触发器
func (t *triggerServiceAppImpl) GetPageList(condition *entity.Trigger, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.Trigger], error) {
	return t.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

// SaveTrigger 保存触发器：先校验（来源已注册、运行配置二选一齐备、内联结果通知可用、匹配条件合法），
// 再治理令牌，最后落库。
func (t *triggerServiceAppImpl) SaveTrigger(ctx context.Context, tr *entity.Trigger) error {
	if tr == nil {
		return errorx.NewBizI(ctx, imsg.ErrTriggerNil)
	}
	if strings.TrimSpace(tr.Name) == "" {
		return errorx.NewBizI(ctx, imsg.ErrTriggerNameRequired)
	}
	if tr.SourceType == "" {
		return errorx.NewBizI(ctx, imsg.ErrTriggerSourceRequired)
	}
	// 来源必须已注册：否则事件到达时无 source 可匹配，触发器永远不生效（配置即失效）
	if _, ok := trigger.Default.Get(tr.SourceType); !ok {
		return errorx.NewBizI(ctx, imsg.ErrTriggerSourceUnknown, "type", tr.SourceType)
	}
	// 运行配置二选一：关联任务（task_id>0，复用任务配置）或 内联（instruction+owner 必填）
	if tr.TaskId == 0 {
		if tr.OwnerAccountId == 0 {
			return errorx.NewBizI(ctx, imsg.ErrTriggerOwnerRequired)
		}
		if strings.TrimSpace(tr.Instruction) == "" {
			return errorx.NewBizI(ctx, imsg.ErrTriggerInstructionRequired)
		}
		// 内联模式的结果通知由触发器自带配置承担（关联任务时以任务配置为准，见 resolveRunConfig，
		// 任务侧已在 SaveTask 校验过），故只在这一支校验：要通知必须有模板，且模板当前发得出去。
		// 不拦的后果是「保存成功、告警来了 Run 也跑成了，结果通知却静默丢失」，
		// 只在服务端日志里留一条 ERROR，配置人在界面上得不到任何信号
		if tr.NotifyType != entity.TaskNotifyNone && tr.NotifyTmplCode == "" {
			return errorx.NewBizI(ctx, imsg.ErrTriggerNotifyTmplRequired)
		}
		if err := checkNotifyTmplSendable(ctx, t.msgTmplApp, tr.NotifyType, tr.NotifyTmplCode); err != nil {
			return err
		}
	} else {
		// 保存期就校验绑定任务存在：否则只有派发时才报 `bound task not found`，而那时
		// 每次事件命中都会先吃掉一个冷却窗口再失败，日志里只留下「in cooldown」，运维查不到根因
		task, err := t.agentTaskApp.GetById(tr.TaskId)
		if err != nil {
			return errorx.NewBizI(ctx, imsg.ErrTriggerTaskNotFound, "id", tr.TaskId)
		}
		// 关联任务时运行身份以任务 owner 为准（与 resolveRunConfig 一致）：保存期就写入，
		// 不让触发器行里留一个与派发时不同的 owner 误导审计与权限归属判定
		tr.OwnerAccountId = task.OwnerAccountId
		tr.OwnerUsername = task.OwnerUsername
	}
	if tr.ProfileId == "" {
		tr.ProfileId = headless.ProfileDiagnostician
	}
	// 保存期就拦住不可用于事件/定时派发的档位（派发期再报只能从执行历史里看到）
	if err := validateDispatchProfile(ctx, tr.ProfileId); err != nil {
		return err
	}
	// 匹配条件非空时必须是「可用」的条件：不只验 JSON 语法，还须拒契约外的键（见
	// trigger.ParseMatchCondition）——键名拼错会解出零值条件，而零值在 MatchByFields 里
	// 意味着「全部维度不限定」，于是该触发器静默变成「本来源每条事件都跑一次 LLM 诊断」
	if _, err := trigger.ParseMatchCondition(ctx, tr.MatchCond); err != nil {
		// 错误文案已由 ParseMatchCondition 按同一请求 ctx 的语言生成，这里透传即可，
		// 再包一层「trigger rejected: <英文>」等于把译文重新污染成中英混排
		return err
	}
	// webhook 令牌治理（非 webhook 来源置空）
	if err := t.resolveWebhookToken(ctx, tr); err != nil {
		return err
	}
	return t.saveEntity(ctx, tr)
}

// resolveWebhookToken 解析「本次保存后应生效的推送令牌」：
//   - 非 webhook 来源：置空（无令牌可言）
//   - 新建：服务端生成（表单不接受客户端传来的令牌值，防止低权用户钉住一个已知令牌）
//   - 更新：一律保留库中现值；轮换必须走 RotateWebhookToken 显式动作
//
// 为什么不能沿用「空即重生成」：表单不携带令牌（列表 VO 也不再回传），任何一次不带令牌的编辑
// 都会换掉外部监控系统里已配置好的推送地址，且返回体还告诉你新令牌——外部持续往旧地址推，
// 表现为「告警发了但 Agent 不再跑」，排查成本极高。
func (t *triggerServiceAppImpl) resolveWebhookToken(ctx context.Context, tr *entity.Trigger) error {
	if tr.SourceType != trigger.SourceTypeWebhook {
		tr.WebhookToken = ""
		return nil
	}
	if tr.Id == 0 {
		token, err := newWebhookToken(ctx)
		if err != nil {
			return err
		}
		tr.WebhookToken = token
		return t.ensureWebhookTokenUnique(ctx, tr.WebhookToken, tr.Id)
	}
	old, err := t.GetById(tr.Id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrTriggerNotFound)
	}
	tr.WebhookToken = old.WebhookToken
	// 存量行无令牌时补分一个：保证 webhook 来路始终有可用地址
	if tr.WebhookToken == "" {
		token, err := newWebhookToken(ctx)
		if err != nil {
			return err
		}
		tr.WebhookToken = token
		return t.ensureWebhookTokenUnique(ctx, tr.WebhookToken, tr.Id)
	}
	return nil
}

// ensureWebhookTokenUnique 令牌唯一性查重（应用层）。
//
// 为什么没把这一列建成唯一索引：非 webhook 来源的行共享空串，唯一索引会直接拦下第二条普通
// 触发器（见 entity.Trigger 注释）。令牌值由服务端随机生成且不采信客户端输入，碰撞概率可忽略。
func (t *triggerServiceAppImpl) ensureWebhookTokenUnique(ctx context.Context, token string, selfId uint64) error {
	if token == "" {
		return nil
	}
	// Ne("id", selfId) 在新建（Id=0）时因零值被忽略 → 全表查重；更新时排除自身
	exists, err := t.GetRepo().Exists(ctx, model.NewCond().EqStrict("webhookToken", token).Ne("id", selfId))
	if err != nil {
		return err
	}
	if exists {
		return errorx.NewBizI(ctx, imsg.ErrTriggerTokenInUse)
	}
	return nil
}

// saveEntity 落库触发器配置：Id 为 0 走新增，否则按 entity.TriggerEditableColumns 整列更新
// （零值真实写入：否则 cooldown_seconds 改回 0、task_id 解除关联这类编辑会静默不生效）
func (t *triggerServiceAppImpl) saveEntity(ctx context.Context, tr *entity.Trigger) error {
	if tr.Id == 0 {
		return t.Insert(ctx, tr)
	}
	// 继承库中现值 Extra：展示字段（owner 账号名/备注）随 extra 整列写回，不先合并会丢掉
	// 本次表单不携带的键（与 AgentTask 同源问题）
	old, err := t.GetById(tr.Id)
	if err != nil {
		return errorx.NewBizI(ctx, imsg.ErrTriggerNotFound)
	}
	mergeExtra(&tr.ExtraData, old.ExtraData)
	return t.GetRepo().UpdateById(ctx, tr, saveColumns(entity.TriggerEditableColumns)...)
}

// RotateWebhookToken 显式轮换推送令牌（仅 webhook 来源）。只改令牌列，不触碰其余配置
func (t *triggerServiceAppImpl) RotateWebhookToken(ctx context.Context, id uint64) (string, error) {
	tr, err := t.GetById(id)
	if err != nil {
		return "", errorx.NewBizI(ctx, imsg.ErrTriggerNotFound)
	}
	if tr.SourceType != trigger.SourceTypeWebhook {
		return "", errorx.NewBizI(ctx, imsg.ErrTriggerNotWebhook)
	}
	for attempt := 0; attempt < webhookTokenRotateAttempts; attempt++ {
		token, err := newWebhookToken(ctx)
		if err != nil {
			return "", err
		}
		if err := t.ensureWebhookTokenUnique(ctx, token, id); err != nil {
			return "", err
		}
		rows, err := t.GetRepo().UpdateColsByCond(ctx,
			base.NewPatch().Set("webhookToken", token),
			model.NewCond().EqStrict("id", id).EqStrict("sourceType", trigger.SourceTypeWebhook))
		if err != nil {
			return "", err
		}
		if rows != 1 {
			return "", errorx.NewBizI(ctx, imsg.ErrTriggerRotateLost)
		}
		// 不打印令牌本身（它会进日志文件与采集链路）；只记「已轮换」事实供审计
		logx.InfofContext(ctx, "[agentops-trigger] webhook token rotated: id=%d", id)
		return token, nil
	}
	return "", errorx.NewBizI(ctx, imsg.ErrTriggerTokenAllocateFailed)
}

// Delete 删除触发器
func (t *triggerServiceAppImpl) Delete(ctx context.Context, id uint64) error {
	if _, err := t.GetById(id); err != nil {
		return errorx.NewBizI(ctx, imsg.ErrTriggerNotFound)
	}
	return t.DeleteById(ctx, id)
}

// Dispatch 派发已归一化事件（EventBus 订阅入口）。
//
// 事件由发布侧（如 alert 引擎）归一化为中立 OpsTriggerEvent，此处不再 Normalize（避免与来源
// 原始载荷形态耦合），仅用对应 TriggerSource 做匹配。匹配上的每个启用触发器各自派生一个 Run
// （幂等键按 trigger.Id 细分，故多触发器命中同一事件互不去重）。
func (t *triggerServiceAppImpl) Dispatch(ctx context.Context, evt *event.OpsTriggerEvent) error {
	if evt == nil {
		return errorx.NewBizI(ctx, imsg.ErrTriggerEventNil)
	}
	// 急停守护：开关拉起时整批跳过派发（事件照常消费不积压）。这是同一谓词的「前置整批拒」，
	// 为了避开风暴下无谓的查库与逐条失败日志；真正的结构收口点在 RunService.Enqueue
	if err := guardDispatch(t.governance); err != nil {
		logx.InfofContext(ctx, "[agentops-trigger] dispatch paused by guard: source=%s", evt.Source)
		return nil
	}
	source, ok := trigger.Default.Get(evt.Source)
	if !ok {
		logx.WarnfContext(ctx, "[agentops-trigger] no source registered for type=%s, skip dispatch", evt.Source)
		return nil
	}
	triggers, err := t.GetRepo().ListEnabledBySource(evt.Source)
	if err != nil {
		return err
	}
	for _, tr := range triggers {
		cond, condErr := trigger.ParseMatchCondition(ctx, tr.MatchCond)
		if condErr != nil {
			// 条件不可用 → 跳过该触发器（fail-closed）：按零值条件继续判例会把它静默放大成
			// 「本来源每条事件都派生一次运行」，与「留空=全命中」是两种语义，不能归二为一
			logx.WarnfContext(ctx, "[agentops-trigger] trigger[%d] match condition unusable, dispatch skipped: %v", tr.Id, condErr)
			continue
		}
		if !source.Match(evt, cond) {
			continue
		}
		if t.inCooldown(tr) {
			logx.InfofContext(ctx, "[agentops-trigger] skip trigger[%d] in cooldown: %s", tr.Id, tr.Name)
			continue
		}
		if _, err := t.fire(ctx, tr, evt); err != nil {
			// 单条触发器派发失败不阻断其余（fail-open）：告警风暴下个别配置错误不应拖垮整批
			logx.ErrorfContext(ctx, "[agentops-trigger] fire trigger[%d] failed: %v", tr.Id, err)
			continue
		}
	}
	return nil
}

// DispatchWebhook 处理 webhook 推送：令牌定位触发器 → 归一化 → 匹配 → 派生 Run。
//
// 令牌即鉴权：无效/禁用/非 webhook 来源一律拒绝。急停不再在本方手写判定（由 Enqueue 统一拒绝并透传），
// 冷却也不再在这里「读一下算一下」（见 fire 的冷却裁决点）。
func (t *triggerServiceAppImpl) DispatchWebhook(ctx context.Context, token string, raw any) (*entity.Run, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrWebhookTokenRequired)
	}
	tr, err := t.GetRepo().GetByWebhookToken(token)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrWebhookTokenInvalid)
	}
	if tr.SourceType != trigger.SourceTypeWebhook {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerNotWebhook)
	}
	if tr.Enabled != entity.TriggerEnabled {
		return nil, errorx.NewBizI(ctx, imsg.ErrWebhookTriggerDisabled)
	}
	source, ok := trigger.Default.Get(trigger.SourceTypeWebhook)
	if !ok {
		return nil, errorx.NewBizI(ctx, imsg.ErrWebhookSourceNotRegistered)
	}
	evt, err := source.Normalize(ctx, raw)
	if err != nil {
		return nil, err
	}
	evt.Source = trigger.SourceTypeWebhook
	if strings.TrimSpace(evt.SourceRef) == "" {
		// 无显式引用时用触发器主键兜底，保证 TriggerRef 有稳定可读的来源标识。
		// 绝不能用令牌兜底：TriggerRef 会随运行列表接口外泄（列表 GET 按平台惯例不带权限码），
		// 而令牌是免鉴权 webhook 路由的唯一凭据——拿到它即可冒充该触发器的 owner 身份派生运行
		// （与 vo.TriggerVO 对 webhookToken「列表一律清空」同一口径）
		evt.SourceRef = "webhook:" + strconv.FormatUint(tr.Id, 10)
	}
	if evt.OccurredAt == 0 {
		// 外部载荷未声明发生时刻（Alertmanager 用 startsAt、自定义推送常见不带时刻）：
		// 按投递窗口合成，不能留 0 —— 零值会让幂等键恒定不变，同一 webhook 地址一生只跑一次运行
		evt.OccurredAt = webhookOccurrenceAt(tr.CooldownSeconds, time.Now())
	}
	cond, condErr := trigger.ParseMatchCondition(ctx, tr.MatchCond)
	if condErr != nil {
		// 同样 fail-closed：宁可拒一次推送也不能把该触发器变成全命中放大器（令牌属于它自己，不波及他触发器）
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerMatchCondUnusable, "id", tr.Id, "err", condErr.Error())
	}
	if !source.Match(evt, cond) {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerEventNotMatch)
	}
	return t.fire(ctx, tr, evt)
}

// webhookOccurrenceAt 为缺少显式发生时刻的 webhook 投递合成「本次投递时刻」：按投递窗口取整。
//
// 直接取 time.Now().Unix() 会让「重复投递去重」失效（每次重试的键都不同）；取整到窗口起点后，
// 同窗口内的重试共用一个键、跨窗口即新事件，与冷却闸门分工一致。窗口取冷却秒数（未配冷却时用默认值）。
func webhookOccurrenceAt(cooldownSeconds int, now time.Time) int64 {
	window := cooldownSeconds
	if window <= 0 {
		window = webhookDefaultOccurrenceWindowSec
	}
	sec := now.Unix()
	return sec - sec%int64(window)
}

// fire 由触发器 + 事件派生运行：先解析配置、再占冷却窗口，最后经 RunService 入队。
//
// 顺序必须是「先解析后占窗口」：旧写法先抢冷却再解析配置，而绑定任务已被删除时
// resolveRunConfig 会失败——结果是每次事件命中这个悬空触发器都吃掉一个冷却窗口然后失败，
// 日志里只留下「trigger is in cooldown」而永远看不到真正原因（运维查不到根因）。
func (t *triggerServiceAppImpl) fire(ctx context.Context, tr *entity.Trigger, evt *event.OpsTriggerEvent) (*entity.Run, error) {
	cfg, err := t.resolveRunConfig(ctx, tr)
	if err != nil {
		return nil, err
	}
	// 冷却裁决点先于入队：占不到窗口说明同触发器刚派生过（并发同源事件在此只会有一个胜出）
	won, err := t.claimFireWindow(ctx, tr)
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerInCooldown, "id", tr.Id)
	}
	// 幂等键按 trigger.Id 细分：同一事件被多个触发器命中时各产出独立 Run，
	// 仅「同事件重复投递到同一触发器」才去重（阻断 EventBus 重投/告警重复通知派生重复诊断）
	idemKey := "event:" + strconv.FormatUint(tr.Id, 10) + ":" + evt.DedupKey()
	run, err := t.runService.Enqueue(ctx, &EnqueueReq{
		TriggerType:    entity.TriggerTypeEvent,
		TriggerRef:     evt.SourceRef,
		ProfileId:      cfg.profileId,
		Title:          cfg.title,
		Instruction:    instructionText(cfg.instruction, cfg.instructionSegments),
		Input:          buildEventInput(evt),
		OwnerAccountId: cfg.ownerAccountId,
		OwnerUsername:  cfg.ownerUsername,
		TimeoutSeconds: cfg.timeoutSeconds,
		MaxStep:        cfg.maxStep,
		TokenBudget:    cfg.tokenBudget,
		DryRun:         cfg.dryRun,
		RetryTimes:     cfg.retryTimes,
		NotifyType:     cfg.notifyType,
		NotifyTmplCode: cfg.notifyTmplCode,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		return nil, err
	}
	return run, nil
}

// runConfig 触发器派生运行的快照配置（关联任务 或 内联，解析后统一形态）
type runConfig struct {
	profileId   string
	title       string
	instruction string
	// instructionSegments 指令内的芯片引用段（随 instruction 一并解析，派发时展开）
	instructionSegments []entity.InstructionSegment
	ownerAccountId      uint64
	ownerUsername       string
	timeoutSeconds      int
	maxStep             int
	tokenBudget         int64
	dryRun              bool
	retryTimes          int8
	notifyType          int8
	notifyTmplCode      string
}

// resolveRunConfig 解析触发器的运行配置：task_id>0 复用关联任务全套配置，否则用内联字段。
// fail-closed：无运行身份或无指令一律拒绝（Enqueue 亦会二次拦截）。
func (t *triggerServiceAppImpl) resolveRunConfig(ctx context.Context, tr *entity.Trigger) (*runConfig, error) {
	cfg := &runConfig{
		profileId:           tr.ProfileId,
		title:               tr.Name,
		instruction:         tr.Instruction,
		instructionSegments: tr.InstructionSegments,
		ownerAccountId:      tr.OwnerAccountId,
		ownerUsername:       tr.OwnerUsername,
		// 内联模式的结果通知配置：不取则「外部告警进来→自动诊断→通知到人」这条核心链路的
		// 最后一跳不通（关联任务分支会用任务配置覆盖，不冲突）
		notifyType:     tr.NotifyType,
		notifyTmplCode: tr.NotifyTmplCode,
	}
	if tr.TaskId > 0 {
		task, err := t.agentTaskApp.GetById(tr.TaskId)
		if err != nil {
			return nil, errorx.NewBizI(ctx, imsg.ErrTriggerTaskNotFound, "id", tr.TaskId)
		}
		cfg.profileId = task.ProfileId
		cfg.title = task.Name
		cfg.instruction = task.Instruction
		cfg.instructionSegments = task.InstructionSegments
		cfg.ownerAccountId = task.OwnerAccountId
		cfg.ownerUsername = task.OwnerUsername
		cfg.timeoutSeconds = task.TimeoutSeconds
		cfg.maxStep = task.MaxStep
		cfg.tokenBudget = task.TokenBudget
		cfg.dryRun = task.DryRun
		cfg.retryTimes = task.RetryTimes
		cfg.notifyType = task.NotifyType
		cfg.notifyTmplCode = task.NotifyTmplCode
	}
	if cfg.profileId == "" {
		cfg.profileId = headless.ProfileDiagnostician
	}
	// 再校验一次：任务/触发器可能是在档位被注销或被改成需许可之前保存的，
	// 拦住比「跑一次什么都没做」更有信号
	if err := validateDispatchProfile(ctx, cfg.profileId); err != nil {
		return nil, err
	}
	if cfg.ownerAccountId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerOwnerRequired)
	}
	if strings.TrimSpace(cfg.instruction) == "" {
		return nil, errorx.NewBizI(ctx, imsg.ErrTriggerInstructionRequired)
	}
	return cfg, nil
}

// inCooldown 冷却预筛（廉价读旧值）：只用于批量派发时少做无用功，**不是裁决点**（并发裁决见 claimFireWindow）
func (t *triggerServiceAppImpl) inCooldown(tr *entity.Trigger) bool {
	if tr.CooldownSeconds <= 0 || tr.LastFireTime == nil {
		return false
	}
	return time.Since(*tr.LastFireTime) < time.Duration(tr.CooldownSeconds)*time.Second
}

// claimFireWindow 冷却裁决点：以条件更新抢占「本触发器本次派生窗口」，返回是否抢到。
//
// 为什么不能沿用「先读后判再无条件写回」：并发到达的同源事件（告警多分组重复通知、webhook 重试
// 并发）都读到同一个旧值，双双通过冷却判定 → 各自派生一个 Run（Run 的幂等键只在 occurredAt 完全
// 相同时才去重）。冷却是「同一触发器的派生频率闸门」，闸门本身必须原子。
//
// 不配冷却（CooldownSeconds<=0）时不做裁决、只时间戳 best-effort：MySQL 默认不上报
// clientFoundRows 且 datetime 秒精度，同一秒内「同值重写」是 0 影响行，拿它当裁决会把
// 「不冷却」误判成「冷却中」而丢推（实测）。
func (t *triggerServiceAppImpl) claimFireWindow(ctx context.Context, tr *entity.Trigger) (bool, error) {
	now := time.Now()
	cond := model.NewCond().EqStrict("id", tr.Id)
	if tr.CooldownSeconds <= 0 {
		if _, err := t.GetRepo().UpdateColsByCond(ctx, base.NewPatch().Set("lastFireTime", now), cond); err != nil {
			return false, err
		}
		return true, nil
	}
	// 只有「上次派生早于 now-cooldown」（或从未派生）才占得住本次窗口
	cond = cond.And("(last_fire_time IS NULL OR last_fire_time < ?)", now.Add(-time.Duration(tr.CooldownSeconds)*time.Second))
	rows, err := t.GetRepo().UpdateColsByCond(ctx, base.NewPatch().Set("lastFireTime", now), cond)
	if err != nil {
		return false, err
	}
	return rows == 1, nil
}

// buildEventInput 由触发事件与目标资源构造运行输入（user 消息）。
// 事件快照以 JSON 下发，使 diagnostician 能直接读到资源/指标/严重级等诊断上下文；
// Instruction 作为 system 级指令单独下发，这里补齐「针对什么事件、做什么」。
func buildEventInput(evt *event.OpsTriggerEvent) string {
	var sb strings.Builder
	sb.WriteString("触发事件(JSON):\n")
	if b, err := json.Marshal(evt); err == nil {
		sb.Write(b)
	}
	sb.WriteString("\n\n")
	sb.WriteString("请针对上述事件所指资源做只读根因诊断，定位可能原因并给出处置建议，产出结构化结论。")
	return sb.String()
}
