package application

import (
	"context"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"time"
)

type SaveMachineCronJobParam struct {
	CronJob   *entity.MachineCronJob
	CodePaths []string
}

type MachineCronJob interface {
	base.App[*entity.MachineCronJob]

	// 分页获取机器任务列表信息
	GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 获取分页执行结果列表
	GetExecPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	SaveMachineCronJob(ctx context.Context, param *SaveMachineCronJobParam) error

	Delete(ctx context.Context, id uint64)

	// 初始化计划任务
	InitCronJob()

	// 执行cron job
	// @param key cron job key
	RunCronJob(key string)
}

type machineCronJobAppImpl struct {
	base.AppImpl[*entity.MachineCronJob, repository.MachineCronJob]

	machineCronJobExecRepo repository.MachineCronJobExec `inject:"MachineCronJobExecRepo"`
	machineApp             Machine                       `inject:"MachineApp"`

	tagTreeApp       tagapp.TagTree       `inject:"TagTreeApp"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"TagTreeRelateApp"`
}

var _ (MachineCronJob) = (*machineCronJobAppImpl)(nil)

// 注入MachineCronJobRepo
func (m *machineCronJobAppImpl) InjectMachineCronJobRepo(repo repository.MachineCronJob) {
	m.Repo = repo
}

// 分页获取机器脚本任务列表
func (m *machineCronJobAppImpl) GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.GetRepo().GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 获取分页执行结果列表
func (m *machineCronJobAppImpl) GetExecPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return m.machineCronJobExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 保存机器任务信息
func (m *machineCronJobAppImpl) SaveMachineCronJob(ctx context.Context, param *SaveMachineCronJobParam) error {
	mcj := param.CronJob

	// 赋值cron job key
	if mcj.Id == 0 {
		mcj.Key = stringx.Rand(16)
	} else {
		oldMcj, err := m.GetById(mcj.Id)
		if err != nil {
			return errorx.NewBiz("该计划任务不存在")
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
			logx.ErrorTrace("机器计划任务初始化失败: %s", err.(error))
		}
	}()

	pageParam := &model.PageParam{
		PageSize: 100,
		PageNum:  1,
	}

	var mcjs []*entity.MachineCronJob
	cond := &entity.MachineCronJob{Status: entity.MachineCronJobStatusEnable}
	pr, _ := m.GetPageList(cond, pageParam, &mcjs)
	total := pr.Total
	add := 0

	for {
		for _, mcj := range mcjs {
			m.addCronJob(mcj)
			add++
		}
		if add >= int(total) {
			return
		}

		pageParam.PageNum = pageParam.PageNum + 1
		m.GetPageList(cond, pageParam, mcjs)
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
	m.tagTreeApp.ListByQuery(&tagentity.TagTreeQuery{CodePathLikes: relateCodePaths, Type: tagentity.TagTypeMachine}, &machineTags)
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

	scheduler.AddFunByKey(key, mcj.Cron, func() {
		m.RunCronJob(key)
	})
}

func (m *machineCronJobAppImpl) runCronJob0(mid uint64, cronJob *entity.MachineCronJob) {
	defer func() {
		if err := recover(); err != nil {
			res := anyx.ToString(err)
			m.machineCronJobExecRepo.Insert(context.TODO(), &entity.MachineCronJobExec{
				MachineId: mid,
				CronJobId: cronJob.Id,
				ExecTime:  time.Now(),
				Status:    entity.MachineCronJobExecStatusError,
				Res:       res,
			})
			logx.Errorf("机器:[%d]执行[%s]计划任务失败: %s", mid, cronJob.Name, res)
		}
	}()

	machineCli, err := m.machineApp.GetCli(uint64(mid))
	biz.ErrIsNilAppendErr(err, "获取客户端连接失败: %s")
	res, err := machineCli.Run(cronJob.Script)
	if err != nil {
		if res == "" {
			res = err.Error()
		}
		logx.Errorf("机器:[%d]执行[%s]计划任务失败: %s", mid, cronJob.Name, res)
	} else {
		logx.Debugf("机器:[%d]执行[%s]计划任务成功, 执行结果: %s", mid, cronJob.Name, res)
	}

	if cronJob.SaveExecResType == entity.SaveExecResTypeNo ||
		(cronJob.SaveExecResType == entity.SaveExecResTypeOnError && err == nil) {
		return
	}

	execRes := &entity.MachineCronJobExec{
		MachineId: mid,
		CronJobId: cronJob.Id,
		ExecTime:  time.Now(),
		Res:       res,
	}
	if err == nil {
		execRes.Status = entity.MachineCronJobExecStatusSuccess
	} else {
		execRes.Status = entity.MachineCronJobExecStatusError
	}
	// 保存执行记录
	m.machineCronJobExecRepo.Insert(context.TODO(), execRes)
}
