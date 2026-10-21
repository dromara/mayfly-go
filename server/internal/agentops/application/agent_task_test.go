package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	"mayfly-go/internal/agentops/infra/persistence"
	"mayfly-go/internal/ai/pkg/headless"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/ioc"
	"mayfly-go/pkg/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTaskSvc 打开 sqlite 内存库（Run + AgentTask 两表），装配一个绑定真实仓储、
// 且已注入 RunService 的 AgentTaskService。RunService 工作池「不启动」：本组用例只验证
// 「触发 → 入队产出 queued Run + 配置快照」，运行保持 queued 便于确定性断言。
func setupTaskSvc(t *testing.T) (*agentTaskAppImpl, *runServiceAppImpl) {
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
	require.NoError(t, db.AutoMigrate(&entity.Run{}, &entity.AgentTask{}))

	prev := global.Db
	global.Db = db
	t.Cleanup(func() { global.Db = prev })

	registerRepoOnce.Do(persistence.InitIoc)

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
	return taskSvc, runSvc
}

// yearlyCron 一年只触发一次的表达式：既满足「会被调度」的合法性校验，又保证测试期间
// 绑定到全局调度器后不会真的 fire（避免异步 RunByKey 干扰计数断言）。
const yearlyCron = "0 0 1 1 *"

func validTask() *entity.AgentTask {
	return &entity.AgentTask{
		Name:           "DB 慢查询巡检",
		OwnerAccountId: 7,
		OwnerUsername:  "ops",
		Instruction:    "检查慢查询并产出报告",
		Cron:           yearlyCron,
		Status:         entity.AgentTaskStatusEnable,
	}
}

// --- 治理测试替身 ---

// fakeGovernance 治理服务测试替身：只驱动急停谓词的两个分支（守护判定本身另有真实实现的用例）
type fakeGovernance struct {
	engaged bool
}

var _ Governance = (*fakeGovernance)(nil)

func (f *fakeGovernance) KillSwitch() bool { return f.engaged }

func (f *fakeGovernance) GuardDispatch() error {
	if f.engaged {
		return ErrKillSwitchEngaged
	}
	return nil
}

func (f *fakeGovernance) SetKillSwitch(_ context.Context, engaged bool) error {
	f.engaged = engaged
	return nil
}

func (f *fakeGovernance) Stats(_ context.Context, _ int) (*GovernanceStats, error) {
	return &GovernanceStats{KillSwitch: f.engaged}, nil
}

// --- SaveTask 校验 + key 生命周期 ---

func TestSaveTaskValidation(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	noOwner := validTask()
	noOwner.OwnerAccountId = 0
	require.Error(t, taskSvc.SaveTask(ctx, noOwner), "缺运行身份必须 fail-closed 拒绝")

	noInstr := validTask()
	noInstr.Instruction = "  "
	require.Error(t, taskSvc.SaveTask(ctx, noInstr), "缺指令必须拒绝")

	badCron := validTask()
	badCron.Cron = "not a cron"
	require.Error(t, taskSvc.SaveTask(ctx, badCron), "启用态非法 cron 必须拒绝")

	badNotify := validTask()
	badNotify.NotifyType = entity.TaskNotifyAlways // 配了通知却没模板
	require.Error(t, taskSvc.SaveTask(ctx, badNotify), "配通知但缺模板必须拒绝")

	// 合法创建：生成 key、默认 diagnostician 档
	ok := validTask()
	require.NoError(t, taskSvc.SaveTask(ctx, ok))
	require.NotEmpty(t, ok.Key)
	require.Equal(t, headless.ProfileDiagnostician, ok.ProfileId)
	key := ok.Key

	// 更新：沿用原 key（key 是 cron 绑定与 Run.trigger_ref 的稳定标识，不可变）
	ok.Name = "DB 慢查询巡检 v2"
	require.NoError(t, taskSvc.SaveTask(ctx, ok))
	require.Equal(t, key, ok.Key, "更新必须沿用原 key")

	// 禁用态 + 空 cron 也允许保存（只解绑不调度，不校验表达式）
	disabled := validTask()
	disabled.Status = entity.AgentTaskStatusDisable
	disabled.Cron = ""
	require.NoError(t, taskSvc.SaveTask(ctx, disabled))
}

// --- RunByKey：触发入队 + 配置快照 + 记录触发时间 ---

func TestRunByKeyEnqueues(t *testing.T) {
	taskSvc, runSvc := setupTaskSvc(t)
	ctx := runTestCtx()

	tk := validTask()
	tk.InstructionSegments = []entity.InstructionSegment{
		{Type: "input_text", Text: "巡检该机器负载\n"},
		{Type: "resource", Text: "aliyun", Extra: map[string]any{"resourceType": "machine", "id": "11", "code": "m11", "ip": "10.0.0.11", "port": "22"}},
	}
	tk.TimeoutSeconds = 60
	tk.MaxStep = 12
	tk.RetryTimes = 2
	tk.NotifyType = entity.TaskNotifyFail
	tk.NotifyTmplCode = "tmpl-x"
	require.NoError(t, taskSvc.SaveTask(ctx, tk))

	taskSvc.RunByKey(tk.Key)

	var run entity.Run
	require.NoError(t, runSvc.GetByCond(model.NewCond().EqStrict("triggerRef", tk.Key).Dest(&run)))
	require.Equal(t, entity.TriggerTypeCron, run.TriggerType)
	require.Equal(t, entity.RunStatusQueued, run.Status, "工作池未启动，运行应保持 queued")
	require.Equal(t, tk.Name, run.Title)
	// 指令里的 @资源引用在派发时展开为完整定位信息（模型不必猜 machineId/ip/port）
	require.Contains(t, run.Instruction, "检查慢查询并产出报告")
	require.Contains(t, run.Instruction, "[引用资源] 机器: aliyun")
	require.Contains(t, run.Instruction, "ip=10.0.0.11")
	// 目标资源不再是独立字段：引用已在 system 指令里，input 不再注入 JSON，也不再引导盘点
	require.NotContains(t, run.Input, "ListResources")
	require.Equal(t, uint64(7), run.OwnerAccountId)
	require.Equal(t, 60, run.TimeoutSeconds)
	require.Equal(t, 12, run.MaxStep)
	require.Equal(t, int8(2), run.RetryTimes)
	require.Equal(t, entity.TaskNotifyFail, run.NotifyType)
	require.Equal(t, "tmpl-x", run.NotifyTmplCode)

	// 触发后记录 LastFireTime（misfire 判定基线）
	reloaded, err := taskSvc.GetById(tk.Id)
	require.NoError(t, err)
	require.NotNil(t, reloaded.LastFireTime)
}

// --- RunByKey：未知 key / 禁用任务不入队 ---

func TestRunByKeySkipsWhenMissingOrDisabled(t *testing.T) {
	taskSvc, runSvc := setupTaskSvc(t)
	ctx := runTestCtx()

	taskSvc.RunByKey("nonexistent-key") // 不得 panic
	cnt, err := runSvc.CountByCond(&entity.Run{})
	require.NoError(t, err)
	require.Equal(t, int64(0), cnt)

	disabled := validTask()
	disabled.Status = entity.AgentTaskStatusDisable
	require.NoError(t, taskSvc.SaveTask(ctx, disabled))
	taskSvc.RunByKey(disabled.Key)

	cnt, err = runSvc.CountByCond(&entity.Run{})
	require.NoError(t, err)
	require.Equal(t, int64(0), cnt, "禁用任务触发不得入队")
}

// --- RunByKey：重叠防护（上一次运行未收尾则跳过本次触发） ---

func TestRunByKeyOverlapProtection(t *testing.T) {
	taskSvc, runSvc := setupTaskSvc(t)
	ctx := runTestCtx()

	tk := validTask()
	require.NoError(t, taskSvc.SaveTask(ctx, tk))

	taskSvc.RunByKey(tk.Key) // 第一次入队（queued，非终态）
	taskSvc.RunByKey(tk.Key) // 第二次：上一次仍 queued → 跳过

	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", tk.Key))
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "重叠触发必须被跳过，只入队一次")
}

// --- RunOnce：手动触发（trigger=manual，不做重叠防护） ---

func TestRunOnce(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	tk := validTask()
	tk.Status = entity.AgentTaskStatusDisable // 禁用（不自动调度）仍可手动跑一次
	require.NoError(t, taskSvc.SaveTask(ctx, tk))

	run, err := taskSvc.RunOnce(ctx, tk.Id)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, entity.TriggerTypeManual, run.TriggerType)
	require.Equal(t, tk.Key, run.TriggerRef)

	_, err = taskSvc.RunOnce(ctx, 999999)
	require.Error(t, err, "不存在的任务手动触发必须报错")
}

// --- misfire（错过触发）处理 ---

func TestHandleMisfire(t *testing.T) {
	taskSvc, runSvc := setupTaskSvc(t)
	ctx := runTestCtx()

	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	now := time.Now()

	countRuns := func(key string) int64 {
		c, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", key))
		require.NoError(t, err)
		return c
	}

	// CatchUpOnce + 错过窗口（上次触发在 2 年前，年任务本应在去年触发）→ 补跑一次
	catchUp := validTask()
	catchUp.MisfirePolicy = entity.MisfirePolicyCatchUpOnce
	catchUp.LastFireTime = &twoYearsAgo
	require.NoError(t, taskSvc.SaveTask(ctx, catchUp))
	taskSvc.handleMisfire(ctx, catchUp)
	require.Equal(t, int64(1), countRuns(catchUp.Key), "错过窗口应补跑一次")

	// Skip 策略 → 不补跑
	skip := validTask()
	skip.MisfirePolicy = entity.MisfirePolicySkip
	skip.LastFireTime = &twoYearsAgo
	require.NoError(t, taskSvc.SaveTask(ctx, skip))
	taskSvc.handleMisfire(ctx, skip)
	require.Equal(t, int64(0), countRuns(skip.Key), "Skip 策略不得补跑")

	// CatchUpOnce 但从未触发过（LastFireTime 为空）→ 不补跑
	neverFired := validTask()
	neverFired.MisfirePolicy = entity.MisfirePolicyCatchUpOnce
	require.NoError(t, taskSvc.SaveTask(ctx, neverFired))
	taskSvc.handleMisfire(ctx, neverFired)
	require.Equal(t, int64(0), countRuns(neverFired.Key), "从未触发过无 misfire")

	// CatchUpOnce 但下次触发在未来（上次触发=现在，年任务下次在明年）→ 未错过，不补跑
	future := validTask()
	future.MisfirePolicy = entity.MisfirePolicyCatchUpOnce
	future.LastFireTime = &now
	require.NoError(t, taskSvc.SaveTask(ctx, future))
	taskSvc.handleMisfire(ctx, future)
	require.Equal(t, int64(0), countRuns(future.Key), "未错过窗口不补跑")
}

// 补跑必须与 cron 触发同受守护：急停拉起时不补跑，且同一次错过只补跑一个运行
// （旧写法把 misfire 当作「启动兜底路径」直连入队，三道守护全漏；多实例/多次重启还会重复补跑）
func TestHandleMisfireRespectsGuardAndDedups(t *testing.T) {
	taskSvc, runSvc := setupTaskSvc(t)
	ctx := runTestCtx()

	twoYearsAgo := time.Now().AddDate(-2, 0, 0)
	missed := validTask()
	missed.MisfirePolicy = entity.MisfirePolicyCatchUpOnce
	missed.LastFireTime = &twoYearsAgo
	require.NoError(t, taskSvc.SaveTask(ctx, missed))

	// 1) 急停拉起：不得补跑（治理契约明确承诺所有派发入口拒绝产出新 Run）
	taskSvc.governance = &fakeGovernance{engaged: true}
	taskSvc.handleMisfire(ctx, missed)
	cnt, err := runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", missed.Key))
	require.NoError(t, err)
	require.Equal(t, int64(0), cnt, "急停期间不得补跑")

	// 2) 解除急停后补跑一次，同一次错过再重跑（多实例/多次重启）仍只有一个运行
	taskSvc.governance = &fakeGovernance{engaged: false}
	taskSvc.handleMisfire(ctx, missed)
	missed.LastFireTime = &twoYearsAgo // 回推基线，模拟第二个实例算出同一个本应触发时间
	taskSvc.handleMisfire(ctx, missed)

	cnt, err = runSvc.CountByCond(model.NewCond().EqStrict("triggerRef", missed.Key))
	require.NoError(t, err)
	require.Equal(t, int64(1), cnt, "同一次错过跨实例/跨重启只补跑一次")

	runs, err := runSvc.ListByCond(model.NewCond().EqStrict("triggerRef", missed.Key))
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.True(t, strings.HasPrefix(runs[0].IdempotencyKey, "misfire:"), "补跑必须带确定性幂等键")
}

// 保存必须能写入零值：否则「关影子模式/取消通知/改回跳过/清空目标资源」全部静默失效，
// 表现为保存成功但行为不变（对以 owner 身份驱动真实资源的运维任务是行为失真）
func TestSaveTaskPersistsZeroValues(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	task := validTask()
	task.DryRun = true
	task.MisfirePolicy = entity.MisfirePolicyCatchUpOnce
	task.NotifyType = entity.TaskNotifyAlways
	task.NotifyTmplCode = "tmpl-a"
	task.InstructionSegments = []entity.InstructionSegment{{Type: "skill", Text: "db-ops-guide", Extra: map[string]any{"id": "db-ops-guide"}}}
	task.MaxStep = 9
	require.NoError(t, taskSvc.SaveTask(ctx, task))

	// 用户把这些全部改回零值/清空
	task.DryRun = false
	task.MisfirePolicy = entity.MisfirePolicySkip
	task.NotifyType = entity.TaskNotifyNone
	task.NotifyTmplCode = ""
	// 引用段清空：用户把指令改成纯文本时，旧芯片段必须真的不再展开
	task.InstructionSegments = nil
	task.MaxStep = 0
	require.NoError(t, taskSvc.SaveTask(ctx, task))

	got, err := taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.False(t, got.DryRun, "关影子模式必须落库（旧写法结构体更新会跳过 false）")
	require.Equal(t, entity.MisfirePolicySkip, got.MisfirePolicy, "改回跳过必须落库")
	require.Equal(t, entity.TaskNotifyNone, got.NotifyType, "取消通知必须落库")
	require.Empty(t, got.NotifyTmplCode)
	require.Empty(t, got.InstructionSegments, "引用段清空必须落库，否则模型继续按旧靶子定位资源")
	require.Zero(t, got.MaxStep)
	require.NotNil(t, got.UpdateTime, "显式列更新仍必须维护修改审计列")

	// 服务端独占列不被表单陈旧值覆盖：key 沿用旧值，触发基线不由保存路径写
	require.NotEmpty(t, got.Key)
	require.Nil(t, got.LastFireTime, "未经触发不得写入 last_fire_time")
}

// 显式列更新不得弄丢修改人审计：Select(列名) 只写列入的列，modifier/modifier_id 不列入就会
// 静默停在创建者（实测过），「谁改了这条运维配置」的审计链会断
func TestSaveTaskUpdatesAuditColumns(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)

	as := func(id uint64, name string) context.Context {
		return contextx.WithLoginAccount(context.Background(), &model.LoginAccount{Id: id, Username: name})
	}

	task := validTask()
	require.NoError(t, taskSvc.SaveTask(as(7, "alice"), task))
	created, err := taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.Equal(t, "alice", created.Modifier)

	task.Instruction = "changed instruction"
	require.NoError(t, taskSvc.SaveTask(as(9, "bob"), task))
	got, err := taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.Equal(t, "bob", got.Modifier, "更新必须记录修改人")
	require.Equal(t, uint64(9), got.ModifierId)
	require.Equal(t, "alice", got.Creator, "创建人不得被更新覆写")
}

// 展示字段（负责人账号名/备注）收敛进 extra 后的三条契约，违反任何一条都是无报错的界面丢值：
// ① 保存→读回不变（BeforeSave 写、AfterFind 还原，两边必须成对）；
// ② 备注改成空串必须真清掉（model.SetExtraValue 跳过空值，用它会留下旧值）；
// ③ extra 整列写回不得抹掉表单不携带的键（靠 mergeExtra 继承旧值）
func TestSaveTaskDisplayFieldsRoundTripThroughExtra(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	task := validTask()
	task.Remark = "内置模板"
	require.NoError(t, taskSvc.SaveTask(ctx, task))

	got, err := taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.Equal(t, "ops", got.OwnerUsername, "负责人账号名必须随 extra 落库并在读回时还原")
	require.Equal(t, "内置模板", got.Remark)

	// 造一个表单永不携带的 extra 键（等价于将来写在 extra 里的运行期约束）
	var foreign entity.AgentTask
	require.NoError(t, global.Db.First(&foreign, task.Id).Error)
	foreign.SetExtraValue("probeKey", "probeValue")
	require.NoError(t, global.Db.Model(&entity.AgentTask{}).Where("id = ?", task.Id).
		UpdateColumn("extra", foreign.Extra).Error)

	// 二次保存：改负责人 + 清空备注
	task.OwnerUsername = "dba"
	task.Remark = ""
	require.NoError(t, taskSvc.SaveTask(ctx, task))

	got, err = taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.Equal(t, "dba", got.OwnerUsername, "改后的负责人必须落库")
	require.Empty(t, got.Remark, "清空备注必须真落库（空值走删键，不是跳过）")
	require.Equal(t, "probeValue", got.GetExtraString("probeKey"), "extra 整列写回不得丢表单外的键")
}

// --- 纯函数助手 ---

func TestTaskHelpers(t *testing.T) {
	require.True(t, taskScheduled(&entity.AgentTask{Status: entity.AgentTaskStatusEnable, Cron: yearlyCron}))
	require.False(t, taskScheduled(&entity.AgentTask{Status: entity.AgentTaskStatusDisable, Cron: yearlyCron}), "禁用不调度")
	require.False(t, taskScheduled(&entity.AgentTask{Status: entity.AgentTaskStatusEnable, Cron: "  "}), "空表达式不调度")

	// 指令里已 @引用资源：定位信息随 system 指令下发，input 不必再注入 JSON 目标，也不再引导盘点
	withRes := buildTaskInput(&entity.AgentTask{
		Name:                "DB 慢查询巡检",
		InstructionSegments: []entity.InstructionSegment{{Type: "resource", Text: "mayfly_go", Extra: map[string]any{"resourceType": "db", "id": "1"}}},
	})
	require.Contains(t, withRes, "请严格按任务指令")
	require.NotContains(t, withRes, "ListResources", "指令已引用资源时无需再盘点")

	noRes := buildTaskInput(&entity.AgentTask{})
	require.Contains(t, noRes, "请严格按任务指令", "无目标资源也须有明确执行提示")
	require.Contains(t, noRes, "ListResources", "无目标资源时须显式引导先盘点资源，禁止猜测")
}

// 指令芯片段必须随保存落库、读回时类型与信息都不丢（收敛在 Extra 的 JSON 里，
// 序列化一旦退化，编辑回填就看不到原引用，下一次保存会把引用静默清掉）
func TestSaveTaskPersistsInstructionSegments(t *testing.T) {
	taskSvc, _ := setupTaskSvc(t)
	ctx := runTestCtx()

	task := validTask()
	task.Instruction = "巡检 @aliyun 的磁盘"
	task.InstructionSegments = []entity.InstructionSegment{
		{Type: "input_text", Text: "巡检 "},
		{Type: "resource", Text: "aliyun", Extra: map[string]any{"resourceType": "machine", "id": "11", "ip": "10.0.0.11", "port": "22"}},
		{Type: "input_text", Text: " 的磁盘"},
	}
	require.NoError(t, taskSvc.SaveTask(ctx, task))

	got, err := taskSvc.GetById(task.Id)
	require.NoError(t, err)
	require.Len(t, got.InstructionSegments, 3)
	resource := got.InstructionSegments[1]
	require.Equal(t, "resource", resource.Type)
	require.Equal(t, "aliyun", resource.Text)
	require.Equal(t, "10.0.0.11", resource.Extra["ip"], "定位信息必须原样读回")
	require.Equal(t, "巡检 @aliyun 的磁盘", got.Instruction, "指令文本与引用段各自保存，互不覆盖")
}

// instructionText 是「指令里的引用如何变成模型可取参的文本」的唯一出口：
// 无段时按原文下发（内置 Runbook 与历史任务都是纯文本），有段时按聊天同一渲染器就地展开
func TestInstructionText(t *testing.T) {
	// 无引用段：原文透出，且不得出现渲染痕迹
	require.Equal(t, "巡检慢查询", instructionText("巡检慢查询", nil))

	// 技能引用：展开为 [引用技能] 标记（模型据此 skill_read 加载全文）
	skill := instructionText("请先加载技能：", []entity.InstructionSegment{{Type: "skill", Text: "db-ops-guide"}})
	require.Contains(t, skill, "[引用技能] db-ops-guide")

	// 资源引用：展开为带完整定位标识的文本，工具入参不必靠模型猜
	resource := instructionText("巡检该机器：", []entity.InstructionSegment{
		{Type: "input_text", Text: "巡检该机器："},
		{Type: "resource", Text: "aliyun", Extra: map[string]any{"resourceType": "machine", "id": "11", "code": "m11", "ip": "10.0.0.11", "port": "22"}},
	})
	require.Contains(t, resource, "[引用资源] 机器: aliyun")
	require.Contains(t, resource, "ip=10.0.0.11")
	require.Contains(t, resource, "port=22")
	// 任务指令没有会话历史，不能追加聊天消息那句「以上引用资源为本次任务的唯一目标」
	require.NotContains(t, resource, "唯一目标")
}
