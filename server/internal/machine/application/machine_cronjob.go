package application

import (
	"context"
	"errors"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/internal/machine/imsg"
	"mayfly-go/internal/machine/mcm"
	msgapp "mayfly-go/internal/msg/application"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"sync"
	"time"
)

// cronJobDefaultTimeout 计划任务单次执行的默认超时（cronJob.TimeoutSeconds<=0 时使用）
const cronJobDefaultTimeout = 120 * time.Second

type MachineCronJob interface {
	base.App[*entity.MachineCronJob]

	// 分页获取机器任务列表信息
	GetPageList(condition *entity.MachineCronJob, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJob], error)

	// 获取分页执行结果列表
	GetExecPageList(condition *entity.MachineCronJobExec, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJobExec], error)

	SaveMachineCronJob(ctx context.Context, param *dto.SaveMachineCronJob) error

	Delete(ctx context.Context, id uint64)

	// 初始化计划任务
	InitCronJob()

	// 执行cron job
	//  -  key cron job key
	RunCronJob(key string)
}

type machineCronJobAppImpl struct {
	base.AppImpl[*entity.MachineCronJob, repository.MachineCronJob]

	machineCronJobExecRepo repository.MachineCronJobExec `inject:"T"`
	machineApp             Machine                       `inject:"T"`

	tagTreeApp       tagapp.TagTreeReader `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`

	// msgTmplApp 结果通知发送器：与告警通知同一发送器，渠道由消息模板关联决定
	msgTmplApp msgapp.MsgTmpl `inject:"T"`

	// runGuard 分布式运行守卫：按 cron job key 互斥，防止多实例重复执行
	runGuard taskx.RunGuard[string]

	// runCancels 本实例在跑的实例级取消表（key = cron job key）：
	// Delete 时直接取消，避免任务删除后执行体继续跑完并补插执行记录
	runCancelsMu sync.Mutex
	runCancels   map[string]map[*runCancelEntry]struct{}
}

// runCancelEntry 一次在跑执行的取消登记项：用指针身份区分同一 key 的多轮/多机执行
// （同一计划任务可能已在多台机器上并发执行，删除时必须全部取消）
type runCancelEntry struct {
	cancel context.CancelFunc
}

var _ MachineCronJob = (*machineCronJobAppImpl)(nil)

var _ (MachineCronJob) = (*machineCronJobAppImpl)(nil)

// 分页获取机器脚本任务列表
func (m *machineCronJobAppImpl) GetPageList(condition *entity.MachineCronJob, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJob], error) {
	return m.GetRepo().GetPageList(condition, pageParam, orderBy...)
}

// 获取分页执行结果列表
func (m *machineCronJobAppImpl) GetExecPageList(condition *entity.MachineCronJobExec, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJobExec], error) {
	return m.machineCronJobExecRepo.GetPageList(condition, pageParam, orderBy...)
}

// 保存机器任务信息
func (m *machineCronJobAppImpl) SaveMachineCronJob(ctx context.Context, param *dto.SaveMachineCronJob) error {
	mcj := param.CronJob

	// 会被调度的任务先校验表达式：注册失败只写服务端日志，不拦下就是「保存成功但永不执行」
	if cronJobScheduled(mcj) {
		if err := scheduler.ValidateSpec(mcj.Cron); err != nil {
			return errorx.NewBizI(ctx, imsg.ErrCronJobSpecInvalid, "cron", mcj.Cron, "reason", err.Error())
		}
	}

	// 要通知就必须有模板：缺模板时发送逻辑是直接 return，用户以为配了通知却永远收不到
	if mcj.NotifyType != entity.CronJobNotifyNone && mcj.NotifyTmplCode == "" {
		return errorx.NewBizI(ctx, imsg.ErrCronJobNotifyTmplRequired)
	}
	// 模板存在也还不够：已停用或未关联任何渠道同样发不出去，而 sendNotify 只把失败写成一条
	// 服务端日志——界面上早已「操作成功」，配错的人看不到任何信号。与 agentops 的任务/触发器
	// 保存校验同一取向：能在保存期拦住的配置错误，不留到执行期只写日志
	//（msgTmplApp 单测未注入时为 nil，跳过可发性校验）
	if mcj.NotifyType != entity.CronJobNotifyNone && mcj.NotifyTmplCode != "" && m.msgTmplApp != nil {
		if err := m.msgTmplApp.CheckSendable(ctx, mcj.NotifyTmplCode); err != nil {
			return err
		}
	}

	// 赋值cron job key
	if mcj.Id == 0 {
		mcj.Key = stringx.Rand(16)
	} else {
		oldMcj, err := m.GetById(mcj.Id)
		if err != nil {
			return errorx.NewBiz("cronjob not found")
		}
		mcj.Key = oldMcj.Key
	}

	err := m.Tx(ctx, func(ctx context.Context) error {
		return m.Save(ctx, mcj)
	}, func(ctx context.Context) error {
		return m.tagTreeRelateApp.RelateTag(ctx, tagentity.TagRelateTypeMachineCronJob, mcj.Id, param.CodePaths...)
	})
	if err != nil {
		return err
	}

	m.addCronJob(mcj)
	return nil
}

func (m *machineCronJobAppImpl) Delete(ctx context.Context, id uint64) {
	// 先停调度与在跑执行：仅软删任务时，执行体还会在数秒后补插一条执行记录（实测删除后 14s 仍插入），
	// 先取消才能让「删除」对已在跑的机器也立即生效
	if cronJob, err := m.GetById(id); err == nil && cronJob != nil {
		taskx.UnbindCronTask(cronJob.Key)
		m.cancelRuns(cronJob.Key)
		// 跨实例：有 Redis 时写停止标记让其他实例上的执行也尽快收敛；无 Redis 时仅本地 Release
		m.runGuard.RequestStop(cronJob.Key)
	}

	m.DeleteById(ctx, id)
	m.machineCronJobExecRepo.DeleteByCond(ctx, &entity.MachineCronJobExec{CronJobId: id})
}

// registerRun 登记一轮在跑执行，返回登记项供收尾摘除
func (m *machineCronJobAppImpl) registerRun(key string, cancel context.CancelFunc) *runCancelEntry {
	entry := &runCancelEntry{cancel: cancel}
	m.runCancelsMu.Lock()
	defer m.runCancelsMu.Unlock()

	if m.runCancels == nil {
		m.runCancels = make(map[string]map[*runCancelEntry]struct{})
	}
	entries := m.runCancels[key]
	if entries == nil {
		entries = make(map[*runCancelEntry]struct{})
		m.runCancels[key] = entries
	}
	entries[entry] = struct{}{}
	return entry
}

func (m *machineCronJobAppImpl) unregisterRun(key string, entry *runCancelEntry) {
	m.runCancelsMu.Lock()
	defer m.runCancelsMu.Unlock()

	entries := m.runCancels[key]
	delete(entries, entry)
	if len(entries) == 0 {
		delete(m.runCancels, key)
	}
}

// cancelRuns 取消指定任务当前所有在跑执行
func (m *machineCronJobAppImpl) cancelRuns(key string) {
	m.runCancelsMu.Lock()
	entries := make([]*runCancelEntry, 0, len(m.runCancels[key]))
	for entry := range m.runCancels[key] {
		entries = append(entries, entry)
	}
	m.runCancelsMu.Unlock()

	for _, entry := range entries {
		entry.cancel()
	}
}

// runStopped 本轮执行是否已应停止：本地取消信号 + 跨实例停止标记
func (m *machineCronJobAppImpl) runStopped(ctx context.Context, key string) bool {
	return ctx.Err() != nil || m.runGuard.IsStopRequested(key)
}

func (m *machineCronJobAppImpl) InitCronJob() {
	defer func() {
		if err := recover(); err != nil {
			logx.ErrorTrace("the machine cronjob failed to initialize: %v", err)
		}
	}()

	if err := m.CursorByCond(&entity.MachineCronJob{Status: entity.MachineCronJobStatusEnable}, func(mcj *entity.MachineCronJob) error {
		m.addCronJob(mcj)
		return nil
	}); err != nil {
		logx.ErrorTrace("the machine cronjob failed to initialize: %v", err)
	}
}

func (m *machineCronJobAppImpl) RunCronJob(key string) {
	// 定时任务入口没有请求上下文：显式造一个带 traceId 的任务级 ctx 往下传，
	// 这一轮的执行记录既能与日志串联，也不会以「无上下文」写库而看不出是谁在什么时候跑的
	ctx := contextx.WithTraceId(context.Background())

	// 分布式互斥：多实例部署时只有一个实例执行同一 cron job
	if !m.runGuard.Acquire(key) {
		return
	}
	defer m.runGuard.Release(key)

	cronJob := new(entity.MachineCronJob)
	cronJob.Key = key
	err := m.GetByCond(cronJob)
	// 不存在或禁用，则移除该任务
	if err != nil || cronJob.Status == entity.MachineCronJobStatusDisable {
		taskx.UnbindCronTask(key)
		return
	}

	relateCodePaths, err := m.tagTreeRelateApp.GetTagPathsByRelate(tagentity.TagRelateTypeMachineCronJob, cronJob.Id)
	if err != nil {
		logx.Errorf("failed to get tag paths for cronjob %s: %v", key, err)
		return
	}
	var machineTags []tagentity.TagTree
	if err := m.tagTreeApp.ListByQuery(&tagentity.TagTreeQuery{CodePathLikes: relateCodePaths, Types: []tagentity.TagType{tagentity.TagTypeMachine}}, &machineTags); err != nil {
		logx.Errorf("failed to list machine tags for cronjob: %v", err)
		return
	}
	machines, err := m.machineApp.ListByCond(model.NewCond().In("code", collx.ArrayMap(machineTags, func(tag tagentity.TagTree) string {
		return tag.Code
	})), "id")
	if err != nil {
		logx.Errorf("failed to list machines for cronjob: %v", err)
		return
	}

	for _, machine := range machines {
		gox.Go(func() {
			m.runCronJob0(ctx, machine.Id, cronJob)
		})
	}
}

// cronJobScheduled 计划任务是否会被注册为定时任务：禁用态只解绑不注册。
// 保存前的表达式校验与 addCronJob 的绑定判定共用此函数，避免两处口径漂移。
func cronJobScheduled(mcj *entity.MachineCronJob) bool {
	return mcj.Status != entity.MachineCronJobStatusDisable
}

func (m *machineCronJobAppImpl) addCronJob(mcj *entity.MachineCronJob) {
	key := mcj.Key
	if !cronJobScheduled(mcj) {
		taskx.UnbindCronTask(key)
		return
	}

	if err := taskx.BindCronTask(key, mcj.Cron, true, func() {
		defer gox.Recover()
		m.RunCronJob(key)
	}); err != nil {
		logx.ErrorTrace("add machine cron job failed", err)
	}
}

func (m *machineCronJobAppImpl) runCronJob0(ctx context.Context, mid uint64, cronJob *entity.MachineCronJob) {
	timeout := time.Duration(cronJob.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = cronJobDefaultTimeout
	}

	machineCode := ""
	if machine, err := m.machineApp.GetById(mid); err == nil {
		machineCode = machine.Code
	}

	// RunCronJob 的 ctx 已脱离调度回调（Background+trace），这里保留 traceId 与上游取消信号；
	// 同时登记本轮执行，任务被删除时 Delete 可直接取消，不必等命令自然结束
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()
	runEntry := m.registerRun(cronJob.Key, cancelFunc)
	defer m.unregisterRun(cronJob.Key, runEntry)

	res := ""
	var err error
	attempts := int(cronJob.RetryTimes) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		if m.runStopped(ctx, cronJob.Key) {
			logx.Warnf("machine[%d] cronjob[%s] run canceled, stop remaining attempts", mid, cronJob.Name)
			break
		}
		var retryable bool
		res, err, retryable = m.runOnce(ctx, mid, cronJob.Script, timeout)
		if err == nil {
			logx.Debugf("machine[%d] successfully executed cronjob[%s], execution result: %s", mid, cronJob.Name, res)
			break
		}
		// 仅对连接失败/超时重试；命令非 0 退出属业务失败，不重试
		if !retryable || attempt == attempts {
			logx.Errorf("machine[%d] failed to execute cronjob[%s]: %s", mid, cronJob.Name, err.Error())
			break
		}
		logx.Warnf("machine[%d] cronjob[%s] attempt %d/%d failed (retryable): %s", mid, cronJob.Name, attempt, attempts, err.Error())
		// 退避等待也要能被删除动作打断，否则取消后仍要多等一轮重试间隔
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	// 任务在执行期间被删除/停止：不再落执行记录与通知，
	// 否则删除动作清理完执行记录后，执行体还会补插一条孤儿记录
	if m.runStopped(ctx, cronJob.Key) {
		logx.Warnf("machine[%d] cronjob[%s] execution canceled, skip saving exec record", mid, cronJob.Name)
		return
	}

	execRes := &entity.MachineCronJobExec{
		CronJobId:   cronJob.Id,
		MachineCode: machineCode,
		ExecTime:    time.Now(),
	}
	if res == "" && err != nil {
		res = err.Error()
	}
	execRes.Res = res

	if cronJob.SaveExecResType != entity.SaveExecResTypeNo &&
		(cronJob.SaveExecResType != entity.SaveExecResTypeOnError || err != nil) {
		if err == nil {
			execRes.Status = entity.MachineCronJobExecStatusSuccess
		} else {
			execRes.Status = entity.MachineCronJobExecStatusError
		}
		// 兜底：取消登记与 Delete 之间存在空窗（例如任务恰在取号前被删），任务不在了就不写
		if _, gerr := m.GetById(cronJob.Id); gerr != nil {
			logx.Warnf("cronjob[%s] was deleted during execution, skip saving exec record", cronJob.Name)
			return
		}
		// 保存执行记录
		if ierr := m.machineCronJobExecRepo.Insert(ctx, execRes); ierr != nil {
			logx.Errorf("failed to save cronjob[%s] exec record: %s", cronJob.Name, ierr.Error())
		}
	}

	m.notifyCronJobResult(ctx, cronJob, machineCode, res, err)
}

// runOnce 单次执行：取连接（失败=可重试）→ RunWithTimeout（超时=可重试；命令非 0 退出=业务失败不可重试）
func (m *machineCronJobAppImpl) runOnce(ctx context.Context, mid uint64, script string, timeout time.Duration) (string, error, bool) {
	cli, err := m.machineApp.GetCli(ctx, mid)
	if err != nil {
		return "", err, true
	}
	out, runErr := cli.RunWithTimeout(timeout, script)
	if runErr != nil {
		return out, runErr, errors.Is(runErr, mcm.ErrRunTimeout)
	}
	return out, nil, false
}

// notifyCronJobResult 按计划任务的通知方式与绑定的消息模板推送执行结果（与告警同一发送器）
func (m *machineCronJobAppImpl) notifyCronJobResult(ctx context.Context, cronJob *entity.MachineCronJob, machineCode, res string, runErr error) {
	if cronJob.NotifyType == entity.CronJobNotifyNone || cronJob.NotifyTmplCode == "" {
		return
	}
	failed := runErr != nil
	if cronJob.NotifyType == entity.CronJobNotifyFail && !failed {
		return
	}

	status := "success"
	if failed {
		status = "failed"
	}
	errMsg := ""
	if runErr != nil {
		errMsg = runErr.Error()
	}
	params := map[string]any{
		"jobName": cronJob.Name,
		"machine": machineCode,
		"status":  status,
		"result":  res,
		"error":   errMsg,
	}
	if sendErr := m.msgTmplApp.Send(ctx, cronJob.NotifyTmplCode, params); sendErr != nil {
		logx.Errorf("failed to send cronjob[%s] result notify: %s", cronJob.Name, sendErr.Error())
	}
}
