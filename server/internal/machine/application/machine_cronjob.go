package application

import (
	"context"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/gox"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/taskx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"time"
)

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

func (m *machineCronJobAppImpl) addCronJob(mcj *entity.MachineCronJob) {
	key := mcj.Key
	isDisable := mcj.Status == entity.MachineCronJobStatusDisable

	if isDisable {
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
	execRes := &entity.MachineCronJobExec{
		CronJobId: cronJob.Id,
		ExecTime:  time.Now(),
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	machineCli, err := m.machineApp.GetCli(ctx, mid)
	res := ""
	if err != nil {
		machine, _ := m.machineApp.GetById(mid)
		execRes.MachineCode = machine.Code
	} else {
		execRes.MachineCode = machineCli.Info.Code
		res, err = machineCli.Run(cronJob.Script)
		if err != nil {
			if res == "" {
				res = err.Error()
			}
			logx.Errorf("machine[%d] failed to execute cronjob[%s]: %s", mid, cronJob.Name, res)
		} else {
			logx.Debugf("machine[%d] successfully executed cronjob[%s], execution result: %s", mid, cronJob.Name, res)
		}
	}
	execRes.Res = res

	if cronJob.SaveExecResType == entity.SaveExecResTypeNo ||
		(cronJob.SaveExecResType == entity.SaveExecResTypeOnError && err == nil) {
		return
	}

	if err == nil {
		execRes.Status = entity.MachineCronJobExecStatusSuccess
	} else {
		execRes.Status = entity.MachineCronJobExecStatusError
	}
	// 保存执行记录
	m.machineCronJobExecRepo.Insert(context.TODO(), execRes)
}
