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
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
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
	m.DeleteById(ctx, id)
	m.machineCronJobExecRepo.DeleteByCond(ctx, &entity.MachineCronJobExec{CronJobId: id})
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

	relateCodePaths := m.tagTreeRelateApp.GetTagPathsByRelate(tagentity.TagRelateTypeMachineCronJob, cronJob.Id)
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
			m.runCronJob0(machine.Id, cronJob)
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

func (m *machineCronJobAppImpl) runCronJob0(mid uint64, cronJob *entity.MachineCronJob) {
	timeout := time.Duration(cronJob.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = cronJobDefaultTimeout
	}

	machineCode := ""
	if machine, err := m.machineApp.GetById(mid); err == nil {
		machineCode = machine.Code
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	res := ""
	var err error
	attempts := int(cronJob.RetryTimes) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
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
		time.Sleep(time.Duration(attempt) * time.Second)
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
		!(cronJob.SaveExecResType == entity.SaveExecResTypeOnError && err == nil) {
		if err == nil {
			execRes.Status = entity.MachineCronJobExecStatusSuccess
		} else {
			execRes.Status = entity.MachineCronJobExecStatusError
		}
		// 保存执行记录
		m.machineCronJobExecRepo.Insert(context.TODO(), execRes)
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
