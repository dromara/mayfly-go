package application

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"mayfly-go/internal/agentops/domain/entity"
	"mayfly-go/internal/agentops/domain/repository"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/cache"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"

	"mayfly-go/internal/agentops/imsg"
	"mayfly-go/pkg/i18n"

	"github.com/spf13/cast"
)

// governanceConfigKey 急停开关在系统配置（t_sys_config）中的 key。
// 复用平台既有配置存储（与 ldapEnable/MachineConfig 同源），值形如 {"killSwitch":false}
const governanceConfigKey = "AgentopsGovernance"

// killSwitchCacheTTL 急停开关读缓存 TTL：守护检查在每次 cron/事件派发都会读，
// 短缓存把 DB 读收敛到低频（多实例下开关变更至多延迟该时长收敛）
const killSwitchCacheTTL = 2 * time.Second

// defaultGovernanceWindowDays 治理统计默认窗口天数
const defaultGovernanceWindowDays = 30

// ErrKillSwitchEngaged 急停已拉起时拒绝派发的统一错误（同时是「该运行为何没跑」的识别信号）。
//
// 它必须是包级单例而不是每次新建：调用方靠 errors.Is 区分「被急停拦住」与「真的执行失败」
// （见 agent_task_test 与 Enqueue 的错误透传路径）。代价是文案在包初始化时就固定，只能取
// i18n.T（服务端默认语言），拿不到请求 ctx 的 Accept-Language——中文部署下英文界面会看到
// 这句中文。全仓仅这一条与 ErrRunNotCancelable 受此限制，其余业务错误均走 NewBizI(ctx, …) 按请求语言渲染。
var ErrKillSwitchEngaged = errorx.NewBiz(i18n.T(imsg.ErrKillSwitchEngaged))

// Governance 治理应用服务：急停开关（Kill Switch）+ 守护谓词单点 + 治理统计（成本/成功率/资源盘点）。
//
// 急停语义：所有「产出新 Run」的入队路径由 RunService.Enqueue 结构性收口拒绝（新派发源不可能漏），
// 需要在入队前就整批拒绝的入口（事件批量派发、编排发起）与需要任务语义定序的入口（cron/manual/misfire
// 共用序列）均调 GuardDispatch 做同一谓词的前置判定。运行中 Run 不自动中断，由一键取消
// （RunService.CancelAllActive）显式终止。审批回调的受控执行（RunSync）不经入队派发路径，
// 不受急停影响——已批准的变更应完成执行并回写终态。
type Governance interface {
	// KillSwitch 急停是否已拉起（true = 停止派发新运行）
	KillSwitch() bool

	// GuardDispatch 派发守护判定（唯一谓词定义点）：急停拉起时返回 ErrKillSwitchEngaged，否则 nil。
	// 判定「能不能产出新 Run」只在这里定义一次，调用点只决定何时拒（前置整批拒 / 入队时拒）
	GuardDispatch() error

	// SetKillSwitch 设置急停开关（写 sys 配置 + 同步本地缓存）
	SetKillSwitch(ctx context.Context, engaged bool) error

	// Stats 治理看板统计：窗口内运行 KPI（状态分布/token/耗时/成功率）+ 资源盘点（全量）
	// + 停滞提案不变量（审批已领取但受控执行迟迟未落库的提案数）
	Stats(ctx context.Context, windowDays int) (*GovernanceStats, error)
}

// guardDispatch 治理未装配（单测绕过 IOC）时的守护调用点助手：不拦截，保持用例可跑
func guardDispatch(g Governance) error {
	if g == nil {
		return nil
	}
	return g.GuardDispatch()
}

// GovernanceStats 治理看板统计（运行 KPI 限定 windowDays 窗口；资源盘点为全量）
type GovernanceStats struct {
	// KillSwitch 急停是否已拉起
	KillSwitch bool `json:"killSwitch"`
	// WindowDays 统计窗口天数
	WindowDays int `json:"windowDays"`

	// RunTotal 窗口内运行总数
	RunTotal int64 `json:"runTotal"`
	// RunByStatus 窗口内运行状态分布（见 entity.ActiveRunStatuses + 三个终态）
	RunByStatus map[string]int64 `json:"runByStatus"`
	// ActiveRuns 当前非终态运行数（entity.ActiveRunStatuses，全量）
	ActiveRuns int64 `json:"activeRuns"`
	// SuccessRate 窗口内终态运行成功率（succeeded / (succeeded+failed+canceled)；无终态运行为 0）
	SuccessRate float64 `json:"successRate"`

	// TotalTokens 窗口内消耗总 token
	TotalTokens int64 `json:"totalTokens"`
	// PromptTokens 窗口内输入 token
	PromptTokens int64 `json:"promptTokens"`
	// CompletionTokens 窗口内输出 token
	CompletionTokens int64 `json:"completionTokens"`
	// AvgCostMs 窗口内平均耗时（毫秒）
	AvgCostMs int64 `json:"avgCostMs"`

	// PendingProposals 待审批变更提案数（全量）
	PendingProposals int64 `json:"pendingProposals"`
	// StuckApproved 停滞提案数：审批已领取（approved）但受控执行超过静默期仍未落库（exec_run_id=0）。
	//
	// 判据与停滞清扫共用同一份（见 stalledProposalCond），所以这个数字恰好等于「下一轮清扫会去重投的那批」。
	// 它不是死态：受控执行走进程内队列，进程在领取后、写入 exec_run_id 前退出就会丢单，
	// 而工单在回调那一刻已被 flow 记为处理成功，flow 侧的重推入口对它全部不可达——故由 agentops
	// 自己的停滞清扫自动重投（幂等键防重复执行）。本计数因此是「尚未被清扫收走」的瞬时值，
	// 长期不归零才说明重投本身在失败（去提案详情看 err）
	StuckApproved int64 `json:"stuckApproved"`
	// StallAfterMinutes 停滞判定的静默期（分钟）。透给前端而不是让界面手抄常量：
	// 与 WindowDays 同一取向，后端调阈值时界面文案不会静默说谎
	StallAfterMinutes int `json:"stallAfterMinutes"`
	// EnabledTasks 启用中的定时任务数（全量）
	EnabledTasks int64 `json:"enabledTasks"`
	// EnabledTriggers 启用中的事件触发器数（全量）
	EnabledTriggers int64 `json:"enabledTriggers"`
}

// killSwitchStore 急停开关的持久化读写接缝（默认走 sys 配置；单测注入内存实现，
// 避免 mock 庞大的 sysapp.Config 全量接口——与 flow 窄接缝同范式）
type killSwitchStore interface {
	// Load 读取急停状态；存储缺失返回 false（未急停，fail-open 不误停自动化）
	Load() (bool, error)
	// Persist 写入急停状态
	Persist(ctx context.Context, engaged bool) error
}

// killSwitchStoreFn 接缝函数：默认返回 sys 配置后端，单测覆写注入内存实现
var killSwitchStoreFn = func() killSwitchStore { return sysConfigKillSwitchStore{} }

// sysConfigKillSwitchStore 急停开关的 sys 配置后端。
//
// 读走 GetConfig（缓存读，写后经 cache.Del 失效）；写走 UpdateById 直更（绕开 sys Save 的
// 用户名权限模型——本开关由 agentops:governance:killswitch 权限点在 API 层把守），
// 并手动失效 sys 配置缓存。多实例部署下其他实例的 sys 配置缓存至多在重启前保持旧值，
// 与平台其它配置开关（ldapEnable 等）语义一致。
type sysConfigKillSwitchStore struct{}

func (sysConfigKillSwitchStore) Load() (bool, error) {
	cfg := sysapp.GetConfigApp().GetConfig(governanceConfigKey)
	if cfg == nil || cfg.Id == 0 {
		// 未播种（跳过迁移的存量环境）：视为未急停
		return false, nil
	}
	return cast.ToBool(cfg.GetJsonM()["killSwitch"]), nil
}

func (sysConfigKillSwitchStore) Persist(ctx context.Context, engaged bool) error {
	cfgApp := sysapp.GetConfigApp()
	cfg := cfgApp.GetConfig(governanceConfigKey)
	value := fmt.Sprintf(`{"killSwitch":%t}`, engaged)
	if cfg == nil || cfg.Id == 0 {
		// 未播种：补建配置行（Save 对 Id=0 走 Insert，不触发用户名权限校验）
		return cfgApp.Save(ctx, &sysentity.Config{
			Name:       governanceConfigKey,
			Key:        governanceConfigKey,
			Value:      value,
			Permission: "all",
		})
	}
	cfg.Value = value
	// GetConfig 仅装载 Id/Key/Value/Permission 四列，UpdateById 走 GORM 结构体更新（零值字段跳过），
	// 不会清空未装载的 Name/Params/Remark
	if err := cfgApp.UpdateById(ctx, cfg); err != nil {
		return err
	}
	// App.UpdateById 不经 sys Save 的缓存失效逻辑，手动清缓存保证本实例读新
	cache.Del(sysapp.SysConfigKeyPrefix + governanceConfigKey)
	return nil
}

type governanceAppImpl struct {
	runRepo      repository.Run       `inject:"T"`
	taskRepo     repository.AgentTask `inject:"T"`
	triggerRepo  repository.Trigger   `inject:"T"`
	proposalRepo repository.Proposal  `inject:"T"`

	// 急停开关短缓存（2s）：守护检查高频读，避免每次派发都查库
	ksMu     sync.Mutex
	ksCache  bool
	ksLoaded time.Time
}

var _ Governance = (*governanceAppImpl)(nil)

// KillSwitch 急停是否已拉起（带 2s 短缓存；读失败 fail-open 沿用上次缓存）
func (g *governanceAppImpl) KillSwitch() bool {
	g.ksMu.Lock()
	defer g.ksMu.Unlock()
	if !g.ksLoaded.IsZero() && time.Since(g.ksLoaded) < killSwitchCacheTTL {
		return g.ksCache
	}
	engaged, err := killSwitchStoreFn().Load()
	if err != nil {
		// 读失败不误停自动化（fail-open），沿用上次缓存值，下轮重试
		logx.Errorf("[agentops-governance] load kill switch failed: %v", err)
		return g.ksCache
	}
	g.ksCache = engaged
	g.ksLoaded = time.Now()
	return engaged
}

// GuardDispatch 派发守护判定单点（见接口注释）
func (g *governanceAppImpl) GuardDispatch() error {
	if g.KillSwitch() {
		return ErrKillSwitchEngaged
	}
	return nil
}

// SetKillSwitch 设置急停开关并同步本地缓存
func (g *governanceAppImpl) SetKillSwitch(ctx context.Context, engaged bool) error {
	if err := killSwitchStoreFn().Persist(ctx, engaged); err != nil {
		return err
	}
	g.ksMu.Lock()
	g.ksCache = engaged
	g.ksLoaded = time.Now()
	g.ksMu.Unlock()
	logx.InfofContext(ctx, "[agentops-governance] kill switch set: engaged=%t", engaged)
	return nil
}

// Stats 治理看板统计。运行 KPI 限定窗口（create_time >= now-windowDays），资源盘点为全量：
// 聚合 SUM/AVG 走 Run 仓储 Aggregate（一条 SQL），状态分布与各资源计数走 CountByCond。
func (g *governanceAppImpl) Stats(ctx context.Context, windowDays int) (*GovernanceStats, error) {
	if windowDays <= 0 {
		windowDays = defaultGovernanceWindowDays
	}
	since := time.Now().AddDate(0, 0, -windowDays)

	stats := &GovernanceStats{
		WindowDays:  windowDays,
		KillSwitch:  g.KillSwitch(),
		RunByStatus: make(map[string]int64),
	}

	agg, err := g.runRepo.Aggregate(ctx, since)
	if err != nil {
		return nil, err
	}
	stats.RunTotal = agg.Total
	stats.PromptTokens = agg.PromptTokens
	stats.CompletionTokens = agg.CompletionTokens
	stats.TotalTokens = agg.TotalTokens
	stats.AvgCostMs = int64(math.Round(agg.AvgCostMs))

	// 状态分布与成功率分母均取「一次 GROUP BY 计数」，再按 domain 的两个集合逐个落键：
	// 不在这里逐状态写三个键（曾因此新增终态不进分布、不进分母、却进 RunTotal，
	// 看板 ΣRunByStatus 与 RunTotal 自相矛盾）。集合里没有的状态（脏数据）不进分布，
	// 但仍在 RunTotal 里——分布是「已知状态各有几条」，不是「总数的拆分」
	byStatus, err := g.runRepo.CountGroupedByStatus(ctx, since)
	if err != nil {
		return nil, err
	}
	var terminalTotal int64
	for _, s := range entity.TerminalRunStatuses {
		stats.RunByStatus[s] = byStatus[s]
		terminalTotal += byStatus[s]
	}
	for _, s := range entity.ActiveRunStatuses {
		stats.RunByStatus[s] = byStatus[s]
	}
	if terminalTotal > 0 {
		stats.SuccessRate = float64(byStatus[entity.RunStatusSucceeded]) / float64(terminalTotal)
	}

	// 资源盘点（全量，不受窗口约束）
	if stats.ActiveRuns, err = g.runRepo.CountByCond(model.NewCond().In("status", entity.ActiveRunStatuses)); err != nil {
		return nil, err
	}
	if stats.PendingProposals, err = g.proposalRepo.CountByCond(
		model.NewCond().EqStrict("status", entity.ProposalStatusPending)); err != nil {
		return nil, err
	}
	// 停滞不变量：审批已领取但受控执行超过静默期仍未落库。判据取自 stalledProposalCond（与清扫共用），
	// 故看板报红的就是清扫下一轮会去重投的那批；静默期不可省，否则正常在途的执行
	//（exec_run_id 要等执行档返回终态才回写）会被当成停滞打红，成了纯假警报
	if stats.StuckApproved, err = g.proposalRepo.CountByCond(stalledProposalCond()); err != nil {
		return nil, err
	}
	stats.StallAfterMinutes = int(proposalStallAfter / time.Minute)
	if stats.EnabledTasks, err = g.taskRepo.CountByCond(
		model.NewCond().EqStrict("status", entity.AgentTaskStatusEnable)); err != nil {
		return nil, err
	}
	if stats.EnabledTriggers, err = g.triggerRepo.CountByCond(
		model.NewCond().EqStrict("enabled", entity.TriggerEnabled)); err != nil {
		return nil, err
	}
	return stats, nil
}
