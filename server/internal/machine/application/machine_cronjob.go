package application

import (
	"context"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
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

type MachineCronJob interface {
	base.App[*entity.MachineCronJob]

	// 分页获取机器任务列表信息
	GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 获取分页执行结果列表
	GetExecPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	SaveMachineCronJob(ctx context.Context, entity *entity.MachineCronJob) (uint64, error)

	Delete(ctx context.Context, id uint64)

	// 获取计划任务关联的机器列表id
	GetRelateMachineIds(cronJobId uint64) []uint64

	// 获取机器关联的计划任务列表
	GetRelateCronJobIds(machineId uint64) []uint64

	// 计划任务关联机器
	CronJobRelateMachines(ctx context.Context, cronJobId uint64, machineIds []uint64)

	// 机器关联计划任务
	MachineRelateCronJobs(ctx context.Context, machineId uint64, cronJobs []uint64)

	// 初始化计划任务
	InitCronJob()

	// 执行cron job
	// @param key cron job key
	RunCronJob(key string)
}

type machineCronJobAppImpl struct {
	base.AppImpl[*entity.MachineCronJob, repository.MachineCronJob]

	machineCronJobRelateRepo repository.MachineCronJobRelate `inject:"MachineCronJobRelateRepo"`
	machineCronJobExecRepo   repository.MachineCronJobExec   `inject:"MachineCronJobExecRepo"`
	machineApp               Machine                         `inject:"MachineApp"`
}

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
func (m *machineCronJobAppImpl) SaveMachineCronJob(ctx context.Context, mcj *entity.MachineCronJob) (uint64, error) {
	// 更新操作
	if mcj.Id != 0 {
		m.UpdateById(ctx, mcj)
		cj, err := m.GetById(new(entity.MachineCronJob), mcj.Id)
		if err != nil {
			return 0, errorx.NewBiz("该任务不存在")
		}
		// 处理最新的计划任务
		m.addCronJob(cj)
		return mcj.Id, nil
	}

	m.addCronJob(mcj)
	if err := m.Insert(ctx, mcj); err != nil {
		return 0, err
	}
	return mcj.Id, nil
}

func (m *machineCronJobAppImpl) Delete(ctx context.Context, id uint64) {
	m.DeleteById(ctx, id)
	m.machineCronJobExecRepo.DeleteByCond(ctx, &entity.MachineCronJobExec{CronJobId: id})
	m.machineCronJobRelateRepo.DeleteByCond(ctx, &entity.MachineCronJobRelate{CronJobId: id})
}

func (m *machineCronJobAppImpl) GetRelateMachineIds(cronJobId uint64) []uint64 {
	return m.machineCronJobRelateRepo.GetMachineIds(cronJobId)
}

func (m *machineCronJobAppImpl) GetRelateCronJobIds(machineId uint64) []uint64 {
	return m.machineCronJobRelateRepo.GetCronJobIds(machineId)
}

func (m *machineCronJobAppImpl) CronJobRelateMachines(ctx context.Context, cronJobId uint64, machineIds []uint64) {
	oldMachineIds := m.machineCronJobRelateRepo.GetMachineIds(cronJobId)
	addIds, delIds, _ := collx.ArrayCompare[uint64](machineIds, oldMachineIds)
	addVals := make([]*entity.MachineCronJobRelate, 0)

	for _, addId := range addIds {
		addVals = append(addVals, &entity.MachineCronJobRelate{
			MachineId: addId,
			CronJobId: cronJobId,
		})
	}
	m.machineCronJobRelateRepo.BatchInsert(ctx, addVals)

	for _, delId := range delIds {
		m.machineCronJobRelateRepo.DeleteByCond(ctx, &entity.MachineCronJobRelate{CronJobId: cronJobId, MachineId: delId})
	}
}

func (m *machineCronJobAppImpl) MachineRelateCronJobs(ctx context.Context, machineId uint64, cronJobs []uint64) {
	if len(cronJobs) == 0 {
		m.machineCronJobRelateRepo.DeleteByCond(ctx, &entity.MachineCronJobRelate{MachineId: machineId})
		return
	}

	oldCronIds := m.machineCronJobRelateRepo.GetCronJobIds(machineId)
	addIds, delIds, _ := collx.ArrayCompare[uint64](cronJobs, oldCronIds)
	addVals := make([]*entity.MachineCronJobRelate, 0)

	for _, addId := range addIds {
		addVals = append(addVals, &entity.MachineCronJobRelate{
			MachineId: machineId,
			CronJobId: addId,
		})
	}
	m.machineCronJobRelateRepo.BatchInsert(ctx, addVals)

	for _, delId := range delIds {
		m.machineCronJobRelateRepo.DeleteByCond(ctx, &entity.MachineCronJobRelate{CronJobId: delId, MachineId: machineId})
	}
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
	cond := new(entity.MachineCronJob)
	cond.Status = entity.MachineCronJobStatusEnable
	mcjs := new([]entity.MachineCronJob)

	pr, _ := m.GetPageList(cond, pageParam, mcjs)
	total := pr.Total
	add := 0

	for {
		for _, mcj := range *mcjs {
			m.addCronJob(&mcj)
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
	err := m.GetBy(cronJob)
	// 不存在或禁用，则移除该任务
	if err != nil || cronJob.Status == entity.MachineCronJobStatusDisable {
		scheduler.RemoveByKey(key)
	}

	machienIds := m.machineCronJobRelateRepo.GetMachineIds(cronJob.Id)
	for _, machineId := range machienIds {
		go m.runCronJob0(machineId, cronJob)
	}
}

func (m *machineCronJobAppImpl) addCronJob(mcj *entity.MachineCronJob) {
	var key string
	isDisable := mcj.Status == entity.MachineCronJobStatusDisable
	if mcj.Id == 0 {
		key = stringx.Rand(16)
		mcj.Key = key
		if isDisable {
			return
		}
	} else {
		key = mcj.Key
	}

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
