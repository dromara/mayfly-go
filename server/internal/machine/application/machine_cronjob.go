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
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/scheduler"
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
	// @param key cron job key
	RunCronJob(key string)
}

type machineCronJobAppImpl struct {
	base.AppImpl[*entity.MachineCronJob, repository.MachineCronJob]

	machineCronJobExecRepo repository.MachineCronJobExec `inject:"T"`
	machineApp             Machine                       `inject:"T"`

	tagTreeApp       tagapp.TagTree       `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
}

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
			logx.ErrorTrace("the machine cronjob failed to initialize: %v", err.(error))
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
	// 简单使用redis分布式锁防止多实例同一时刻重复执行
	if lock := rediscli.NewLock(key, 30*time.Second); lock != nil {
		if !lock.Lock() {
			return
		}
		defer lock.UnLock()
	}

	cronJob := new(entity.MachineCronJob)
	cronJob.Key = key
	err := m.GetByCond(cronJob)
	// 不存在或禁用，则移除该任务
	if err != nil || cronJob.Status == entity.MachineCronJobStatusDisable {
		scheduler.RemoveByKey(key)
		return
	}

	relateCodePaths := m.tagTreeRelateApp.GetTagPathsByRelate(tagentity.TagRelateTypeMachineCronJob, cronJob.Id)
	var machineTags []tagentity.TagTree
	m.tagTreeApp.ListByQuery(&tagentity.TagTreeQuery{CodePathLikes: relateCodePaths, Types: []tagentity.TagType{tagentity.TagTypeMachine}}, &machineTags)
	machines, _ := m.machineApp.ListByCond(model.NewCond().In("code", collx.ArrayMap(machineTags, func(tag tagentity.TagTree) string {
		return tag.Code
	})), "id")

	for _, machine := range machines {
		go m.runCronJob0(machine.Id, cronJob)
	}
}

func (m *machineCronJobAppImpl) addCronJob(mcj *entity.MachineCronJob) {
	key := mcj.Key
	isDisable := mcj.Status == entity.MachineCronJobStatusDisable

	if isDisable {
		scheduler.RemoveByKey(key)
		return
	}

	if err := scheduler.AddFunByKey(key, mcj.Cron, func() {
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
