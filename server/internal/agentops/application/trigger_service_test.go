package application

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/agentops/trigger"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/internal/pkg/event"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTriggerSvc 打开 sqlite 内存库（Run + AgentTask + Trigger 三表），装配一个绑定真实仓储、
// 且已注入 RunService / AgentTaskService 的 TriggerService。RunService 工作池「不启动」：本组用例
// 只验证「事件 → 匹配/冷却/幂等 → 入队产出 queued Run + 配置快照」，运行保持 queued 便于确定性断言。
// 内置触发源安装到 trigger.Default（生产由 initTriggerWiring 装配）。
func setupTriggerSvc(t *testing.T) (*triggerServiceAppImpl, *agentTaskAppImpl, *runServiceAppImpl) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		NamingStrategy:       gormx.MetaNamingStrategy(),
		DisableAutomaticPing: true,
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&entity.Run{}, &entity.AgentTask{}, &entity.Trigger{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerRepoOnce.Do(persistence.InitIoc)
	trigger.InstallAlertSource(trigger.Default)
	trigger.InstallWebhookSource(trigger.Default)

	runSvc := &runServiceAppImpl{PoolSize: 1}
	runSvc.Repo = ioc.Get[repository.Run]()
	runSvc.signalCh = make(chan struct{}, 1)
	runSvc.stopCh = make(chan struct{})
	runSvc.inflight = make(map[uint64]context.CancelFunc)
	runSvc.cancelReq = make(map[uint64]struct{})
	t.Cleanup(runSvc.Stop)

	taskSvc := &agentTaskAppImpl{}
	taskSvc.Repo = ioc.Get[repository.AgentTask]()
	taskSvc.runService = runSvc

	triggerSvc := &triggerServiceAppImpl{}
	triggerSvc.Repo = ioc.Get[repository.Trigger]()
	triggerSvc.runService = runSvc
	triggerSvc.agentTaskApp = taskSvc
	return triggerSvc, taskSvc, runSvc
}

// alertEvent 构造一个告警来源的中立事件
func alertEvent(sourceRef, resType, severity string, resId uint64, occurredAt int64) *event.OpsTriggerEvent {
	return &event.OpsTriggerEvent{
		Source: trigger.SourceTypeAlert, SourceRef: sourceRef,
		ResourceType: resType, ResourceId: resId, ResourceName: "res-" + sourceRef,
		Severity: severity, Metric: "cpu", Value: 99, OccurredAt: occurredAt,
	}
}

// --- SaveTrigger 校验 + webhook 令牌生命周期 ---

func TestSaveTriggerValidation(t *testing.T) {
	svc, _, _ := setupTriggerSvc(t)
	ctx := runTestCtx()

	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1, Instruction: "x"}), "缺名称须拒绝")
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: "unknown", OwnerAccountId: 1, Instruction: "x"}), "未注册来源须拒绝")
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, Instruction: "x"}), "内联缺 owner 须 fail-closed 拒绝")
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1}), "内联缺指令须拒绝")
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1, Instruction: "x", MatchCond: "{not json"}), "非法匹配条件 JSON 须拒绝")
	// 契约外键（键名拼错是真实高频失误）：lenient Unmarshal 会忽略它并解成零值条件，
	// 而零值=全部维度不限定=全命中，等于把一条定向触发器存成配额放大器，必须在写侧就拒
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1, Instruction: "x", MatchCond: `{"resourceTyp":"machine"}`}), "拼错的键名须拒绝（不得静默解成零值全命中）")
	require.Error(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1, Instruction: "x", MatchCond: `{"labels":{"env":"prod"}}`}), "契约外扩展键须拒绝")
	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 1, Instruction: "x", MatchCond: "{}"}), "空对象是合法条件（等价于不限定）")

	// 合法内联：默认 diagnostician 档
	ok := &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, OwnerUsername: "ops", Instruction: "诊断", MatchCond: `{"resourceType":"machine"}`}
	require.NoError(t, svc.SaveTrigger(ctx, ok))
	require.NotZero(t, ok.Id)
	require.Equal(t, headless.ProfileDiagnostician, ok.ProfileId)

	// 展示字段（负责人账号名/备注）收敛进 extra：保存后读回必须还原（BeforeSave/AfterFind 成对生效）
	ok.Remark = "告警自动诊断"
	require.NoError(t, svc.SaveTrigger(ctx, ok))
	okReloaded, err := svc.GetById(ok.Id)
	require.NoError(t, err)
	require.Equal(t, "ops", okReloaded.OwnerUsername, "负责人账号名必须随 extra 落库并读回")
	require.Equal(t, "告警自动诊断", okReloaded.Remark)

	// webhook 来源：令牌只由服务端生成
	wh := &entity.Trigger{Name: "wh", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7, Instruction: "诊断"}
	require.NoError(t, svc.SaveTrigger(ctx, wh))
	require.Len(t, wh.WebhookToken, 32, "webhook 来源新建须服务端生成令牌（hex 16 字节）")

	// 客户端传来的令牌不被采信（否则低权账号可钉住一个已知令牌，或猜中他人令牌）
	pinned := &entity.Trigger{Name: "wh2", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7, Instruction: "诊断", WebhookToken: "attacker-known-token"}
	require.NoError(t, svc.SaveTrigger(ctx, pinned))
	require.NotEqual(t, "attacker-known-token", pinned.WebhookToken, "新建不接受客户端传来的令牌")
	require.NotEqual(t, wh.WebhookToken, pinned.WebhookToken, "两个 webhook 触发器令牌应不同")

	// 更新一律保留旧令牌：不能「空即重生成」（表单不携带令牌，那样会让任何编辑静默换掉外部已配地址）
	origToken := wh.WebhookToken
	wh.Name = "wh-renamed"
	wh.WebhookToken = ""
	require.NoError(t, svc.SaveTrigger(ctx, wh))
	reloaded, err := svc.GetById(wh.Id)
	require.NoError(t, err)
	require.Equal(t, "wh-renamed", reloaded.Name)
	require.Equal(t, origToken, wh.WebhookToken, "保留的是库中现值而非传入的空值")
	require.Equal(t, wh.WebhookToken, reloaded.WebhookToken, "更新不得轮换令牌")

	// 轮换必须显式，且轮换后落库值已变
	newToken, err := svc.RotateWebhookToken(ctx, wh.Id)
	require.NoError(t, err)
	require.NotEmpty(t, newToken)
	require.NotEqual(t, origToken, newToken, "轮换必须得到新令牌")
	afterRotate, err := svc.GetById(wh.Id)
	require.NoError(t, err)
	require.Equal(t, newToken, afterRotate.WebhookToken, "轮换后的库中值即新令牌（旧地址随之失效）")

	// 非 webhook 来源：令牌被清空
	nonWh := &entity.Trigger{Name: "a2", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "诊断", WebhookToken: "should-clear"}
	require.NoError(t, svc.SaveTrigger(ctx, nonWh))
	require.Empty(t, nonWh.WebhookToken, "非 webhook 来源令牌应被清空")

	// 可编辑列零值必须真实写入：冷却改回 0、解除关联任务（否则「改了但不生效」且无任何提示）
	wh.CooldownSeconds = 300
	require.NoError(t, svc.SaveTrigger(ctx, wh))
	wh.CooldownSeconds = 0
	require.NoError(t, svc.SaveTrigger(ctx, wh))
	persisted, err := svc.GetById(wh.Id)
	require.NoError(t, err)
	require.Zero(t, persisted.CooldownSeconds, "冷却改回 0 必须落库（旧写法结构体更新会跳过零值）")
	require.NotNil(t, persisted.UpdateTime, "显式列更新仍必须维护修改审计列")
}

// --- Dispatch：匹配启用触发器 → 入队 event Run + 配置快照 + 记录触发时间 ---

func TestDispatchMatchesAndEnqueues(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	tr := &entity.Trigger{
		Name: "机器告警诊断", SourceType: trigger.SourceTypeAlert,
		OwnerAccountId: 7, OwnerUsername: "ops", Instruction: "只读诊断该机器",
		MatchCond: `{"resourceType":"machine","severities":["P0"]}`, Enabled: entity.TriggerEnabled,
	}
	require.NoError(t, svc.SaveTrigger(ctx, tr))

	evt := alertEvent("100", "machine", "P0", 3, 1700000000)
	require.NoError(t, svc.Dispatch(ctx, evt))

	var run entity.Run
	require.NoError(t, runSvc.GetByCond(model.NewCond().EqStrict("triggerRef", "100").Dest(&run)))
	require.Equal(t, entity.TriggerTypeEvent, run.TriggerType)
	require.Equal(t, entity.RunStatusQueued, run.Status, "工作池未启动，运行应保持 queued")
	require.Equal(t, uint64(7), run.OwnerAccountId)
	require.Equal(t, "只读诊断该机器", run.Instruction)
	require.Equal(t, headless.ProfileDiagnostician, run.ProfileId)
	require.Contains(t, run.Input, "res-100", "事件资源信息须并入运行输入")
	require.Contains(t, run.Input, "machine")
	// 幂等键按 trigger.Id + dedupKey 细分
	require.Contains(t, run.IdempotencyKey, "event:")
	require.Contains(t, run.IdempotencyKey, evt.DedupKey())

	reloaded, err := svc.GetById(tr.Id)
	require.NoError(t, err)
	require.NotNil(t, reloaded.LastFireTime, "派发后须记录触发时间（冷却基线）")
}

// --- Dispatch：不匹配 / 禁用触发器不入队 ---

func TestDispatchNoMatchOrDisabledSkips(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "db only", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "x", MatchCond: `{"resourceType":"db"}`, Enabled: entity.TriggerEnabled}))
	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "disabled", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "x", Enabled: entity.TriggerDisabled}))

	// machine 事件不命中 db 触发器；禁用触发器不参与匹配
	require.NoError(t, svc.Dispatch(ctx, alertEvent("1", "machine", "P0", 3, 1000)))
	cnt, err := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, err)
	require.Equal(t, int64(0), cnt, "不匹配/禁用的触发器不得入队")
}

// --- Dispatch：存量脏条件（修复前写入的键名拼错）必须 fail-closed 跳过，不能放大成全命中 ---

func TestDispatchUnusableMatchConditionFailsClosed(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	// 绕过 SaveTrigger 直接落库，模拟「严格校验上线前已存在的脏数据」：
	// 这类行如果仍按零值条件参与匹配，就会对本来源每一条事件派生一次运行
	dirty := &entity.Trigger{
		Name: "脏条件", SourceType: trigger.SourceTypeAlert,
		OwnerAccountId: 7, Instruction: "x", MatchCond: `{"resourceTyp":"machine"}`,
		Enabled: entity.TriggerEnabled,
	}
	require.NoError(t, svc.Repo.Insert(ctx, dirty))

	require.NoError(t, svc.Dispatch(ctx, alertEvent("1", "machine", "P0", 3, 1000)))
	require.NoError(t, svc.Dispatch(ctx, alertEvent("2", "db", "P2", 4, 1001)))

	cnt, err := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, err)
	require.Equal(t, int64(0), cnt, "条件不可用的触发器必须被跳过：按零值匹配会把每条事件都派生成运行")
}

// --- Dispatch：同一事件重复投递被幂等键去重 ---

func TestDispatchIdempotency(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "x", Enabled: entity.TriggerEnabled}))

	evt := alertEvent("55", "machine", "P0", 3, 1700000000)
	require.NoError(t, svc.Dispatch(ctx, evt))
	require.NoError(t, svc.Dispatch(ctx, evt), "重复投递不得报错")

	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", "55"))
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "同一事件重复投递须被幂等键去重")
}

// --- Dispatch：冷却窗口内抑制后续不同事件 ---

func TestDispatchCooldown(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "t", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "x", Enabled: entity.TriggerEnabled, CooldownSeconds: 3600}))

	require.NoError(t, svc.Dispatch(ctx, alertEvent("1", "machine", "P0", 3, 1000)))
	require.NoError(t, svc.Dispatch(ctx, alertEvent("2", "machine", "P0", 3, 2000)), "不同事件（不同 dedupKey）")

	cnt, err := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "冷却窗口内的第二个事件应被抑制")
}

// --- Dispatch：同一事件命中多个触发器 → 各产出独立 Run（幂等键按 trigger.Id 细分） ---

func TestDispatchMultipleTriggersEachFire(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "broad", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "a", Enabled: entity.TriggerEnabled, MatchCond: `{"severities":["P0"]}`}))
	require.NoError(t, svc.SaveTrigger(ctx, &entity.Trigger{Name: "machine", SourceType: trigger.SourceTypeAlert, OwnerAccountId: 7, Instruction: "b", Enabled: entity.TriggerEnabled, MatchCond: `{"resourceType":"machine"}`}))

	require.NoError(t, svc.Dispatch(ctx, alertEvent("9", "machine", "P0", 3, 1000)))

	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", "9"))
	require.NoError(t, err)
	require.Equal(t, int64(2), cnt, "同一事件命中两个触发器应各产出一个 Run")
}

// --- Dispatch：关联任务模式复用任务全套运行配置 ---

func TestDispatchBoundTaskReusesConfig(t *testing.T) {
	svc, taskSvc, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	tk := &entity.AgentTask{
		Name: "DB巡检", OwnerAccountId: 9, OwnerUsername: "dba", Instruction: "检查慢查询",
		ProfileId:      headless.ProfileReviewer,
		TimeoutSeconds: 120, MaxStep: 20, RetryTimes: 3,
		NotifyType: entity.TaskNotifyAlways, NotifyTmplCode: "tmpl",
		Status: entity.AgentTaskStatusDisable, // 不自动调度，仅被触发器复用配置
	}
	require.NoError(t, taskSvc.SaveTask(ctx, tk))

	tr := &entity.Trigger{Name: "db告警→巡检", SourceType: trigger.SourceTypeAlert, TaskId: tk.Id, Enabled: entity.TriggerEnabled, MatchCond: `{"resourceType":"db"}`}
	require.NoError(t, svc.SaveTrigger(ctx, tr))

	require.NoError(t, svc.Dispatch(ctx, alertEvent("77", "db", "P1", 1, 1000)))

	var run entity.Run
	require.NoError(t, runSvc.GetByCond(model.NewCond().EqStrict("triggerRef", "77").Dest(&run)))
	require.Equal(t, tk.Name, run.Title)
	require.Equal(t, "检查慢查询", run.Instruction)
	require.Equal(t, headless.ProfileReviewer, run.ProfileId)
	require.Equal(t, uint64(9), run.OwnerAccountId, "owner 取自关联任务")
	require.Equal(t, 120, run.TimeoutSeconds)
	require.Equal(t, 20, run.MaxStep)
	require.Equal(t, int8(3), run.RetryTimes)
	require.Equal(t, entity.TaskNotifyAlways, run.NotifyType)
	require.Equal(t, "tmpl", run.NotifyTmplCode)
	require.Contains(t, run.Input, "\"sourceRef\":\"77\"", "事件快照须并入运行输入")
}

// --- DispatchWebhook：令牌鉴权 + 归一化 + 入队；无效/禁用/非 webhook 拒绝 ---

func TestDispatchWebhook(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	tr := &entity.Trigger{Name: "wh", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7, OwnerUsername: "ops", Instruction: "诊断外部事件", Enabled: entity.TriggerEnabled}
	require.NoError(t, svc.SaveTrigger(ctx, tr))
	token := tr.WebhookToken // 令牌由服务端生成，调用方不指定
	require.NotEmpty(t, token)

	// 无效令牌拒绝
	_, err := svc.DispatchWebhook(ctx, "bad-token", map[string]any{})
	require.Error(t, err)
	// 空令牌拒绝
	_, err = svc.DispatchWebhook(ctx, "  ", map[string]any{})
	require.Error(t, err)

	// 合法推送：归一化原始 JSON → 入队 event Run
	run, err := svc.DispatchWebhook(ctx, token, map[string]any{
		"resourceType": "db", "resourceId": float64(5), "severity": "P1", "sourceRef": "ext-1",
	})
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, entity.TriggerTypeEvent, run.TriggerType)
	require.Equal(t, "ext-1", run.TriggerRef)
	require.Equal(t, "诊断外部事件", run.Instruction)
	require.Equal(t, uint64(7), run.OwnerAccountId)

	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", "ext-1"))
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt)

	// 禁用后拒绝
	tr.Enabled = entity.TriggerDisabled
	require.NoError(t, svc.SaveTrigger(ctx, tr))
	_, err = svc.DispatchWebhook(ctx, token, map[string]any{"sourceRef": "ext-2"})
	require.Error(t, err, "禁用的 webhook 触发器须拒绝")
}

// --- DispatchWebhook：无 sourceRef 时用触发器主键兜底（绝不用令牌）；无 occurredAt 时按投递窗口合成（修「一生只跑一次」）---

func TestDispatchWebhookFallbackSourceRef(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	tr := &entity.Trigger{Name: "wh", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7, Instruction: "诊断", Enabled: entity.TriggerEnabled}
	require.NoError(t, svc.SaveTrigger(ctx, tr))
	token := tr.WebhookToken
	fallbackRef := "webhook:" + strconv.FormatUint(tr.Id, 10)

	run, err := svc.DispatchWebhook(ctx, token, map[string]any{"resourceType": "machine"})
	require.NoError(t, err)
	require.Equal(t, fallbackRef, run.TriggerRef, "无显式 sourceRef 时须以触发器主键兜底")
	// 安全守卫：TriggerRef 会随运行列表接口外泄（列表 GET 按平台惯例不带权限码），而令牌是
	// 免鉴权 webhook 路由的唯一凭据；整条运行记录（含输入快照/Extra）都不得出现它
	require.NotContains(t, run.TriggerRef, token, "运行引用不得携带 webhook 令牌")
	require.NotContains(t, fmt.Sprintf("%+v", run), token, "运行任意字段不得泄漏 webhook 令牌")

	// 同窗口内重复投递（无 sourceRef、无 occurredAt）被同一幂等键去重
	_, err = svc.DispatchWebhook(ctx, token, map[string]any{"resourceType": "machine"})
	require.NoError(t, err)
	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", fallbackRef))
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "同投递窗口内的重复推送须去重")

	// 显式带不同 sourceRef 的两次推送各自成运行（不能因同一地址而被永久去重）
	_, err = svc.DispatchWebhook(ctx, token, map[string]any{"sourceRef": "evt-a"})
	require.NoError(t, err)
	_, err = svc.DispatchWebhook(ctx, token, map[string]any{"sourceRef": "evt-b"})
	require.NoError(t, err)
	cnt, err = runSvc.CountByCond(model.NewCond().EqStrict("triggerType", entity.TriggerTypeEvent))
	require.NoError(t, err)
	require.Equal(t, int64(3), cnt, "不同事件必须各自派生运行")
}

// 不配冷却时不得拿「影响行数」当闸门：MySQL 不上报 clientFoundRows 且 datetime 秒精度，
// 同一秒内同值重写是 0 行，会被误判成「冷却中」而丢推（升级库实测踩到）
func TestDispatchWebhookNoCooldownStillDispatchesOnSameSecond(t *testing.T) {
	svc, _, runSvc := setupTriggerSvc(t)
	ctx := runTestCtx()

	tr := &entity.Trigger{Name: "wh0", SourceType: trigger.SourceTypeWebhook, OwnerAccountId: 7, Instruction: "诊断", Enabled: entity.TriggerEnabled, CooldownSeconds: 0}
	require.NoError(t, svc.SaveTrigger(ctx, tr))

	for _, ref := range []string{"evt-1", "evt-2", "evt-3"} {
		run, err := svc.DispatchWebhook(ctx, tr.WebhookToken, map[string]any{"sourceRef": ref, "resourceType": "machine"})
		require.NoError(t, err, "同秒内连续不同事件均应派发")
		require.NotNil(t, run)
		require.Equal(t, ref, run.TriggerRef)
	}
	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerType", entity.TriggerTypeEvent))
	require.NoError(t, err)
	require.Equal(t, int64(3), cnt)
}

// webhookOccurrenceAt 投递窗口取整：同窗口重试同键、跨窗口新键
func TestWebhookOccurrenceWindow(t *testing.T) {
	base := time.Unix(1_700_000_045, 0) // 1_700_000_040 是 60 的整数倍，故 60s 窗口为 [...040, ...100)
	require.Equal(t, int64(1_700_000_040), webhookOccurrenceAt(0, base), "未配冷却时按 60s 窗口取整到窗口起点")
	require.Equal(t, int64(1_700_000_040), webhookOccurrenceAt(0, base.Add(5*time.Second)), "同窗口内重试得到同一时刻")
	require.Equal(t, int64(1_700_000_040), webhookOccurrenceAt(0, base.Add(54*time.Second)), "窗口最后一刻仍属同一窗口")
	require.Equal(t, int64(1_700_000_100), webhookOccurrenceAt(0, base.Add(55*time.Second)), "跳窗口得到新的窗口起点")
	require.Equal(t, int64(1_700_000_000), webhookOccurrenceAt(1000, base), "配了冷却时按冷却秒数分桶")
}

// --- 纯函数助手 ---

// 匹配条件解析契约：留空=不限定（产品语义），非法/契约外键=报错。
// 旧断言把「非法 JSON → nil → 全命中」钉成期望值，等于把配额放大器写进回归基线，已推翻。
func TestTriggerHelpers(t *testing.T) {
	// 留空（含纯空白）：无约束，命中该来源全部事件
	cond, err := trigger.ParseMatchCondition(context.Background(), "")
	require.NoError(t, err)
	require.Nil(t, cond, "空条件应为 nil")
	cond, err = trigger.ParseMatchCondition(context.Background(), "   ")
	require.NoError(t, err)
	require.Nil(t, cond, "纯空白等同留空")

	// 非法 JSON / 契约外键：必须报错，不能静默降级成零值全命中
	for _, bad := range []string{"{invalid", `[{"resourceType":"machine"}]`, `"machine"`, `{"resourceTyp":"machine"}`, `{"labels":{"env":"prod"}}`} {
		got, err := trigger.ParseMatchCondition(context.Background(), bad)
		require.Errorf(t, err, "不可用条件必须报错: %s", bad)
		require.Nil(t, got, "报错时不得返回半解析的条件")
	}

	// 尾随内容：Decoder 默认不报错，必须判出来（否则后半截条件被静默丢弃）
	_, err = trigger.ParseMatchCondition(context.Background(), `{"resourceType":"machine"}{"resourceType":"db"}`)
	require.Error(t, err, "尾随第二个对象不得被静默丢弃")

	// 合法条件逐维度取值
	cond, err = trigger.ParseMatchCondition(context.Background(), `{"resourceType":"machine","severities":["P0"]}`)
	require.NoError(t, err)
	require.NotNil(t, cond)
	require.Equal(t, "machine", cond.ResourceType)
	require.Equal(t, []string{"P0"}, cond.Severities)

	evt := alertEvent("1", "machine", "P0", 3, 1000)
	in := buildEventInput(evt)
	require.Contains(t, in, "res-1", "事件快照须并入运行输入，诊断才有依据")
	require.Contains(t, in, "只读根因诊断")
}
