package application

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/global"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/rediscli"
	"mayfly-go/pkg/scheduler"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"time"
)

type MachineCronJob interface {
	// 分页获取机器任务列表信息
	GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	// 获取分页执行结果列表
	GetExecPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineCronJob

	Save(entity *entity.MachineCronJob) uint64

	Delete(id uint64)

	// 获取管理的机器列表id
	GetRelateMachineIds(cronJobId uint64) []uint64

	// 获取机器关联的计划任务列表
	GetRelateCronJobIds(machineId uint64) []uint64

	// 计划任务关联机器
	CronJobRelateMachines(cronJobId uint64, machineIds []uint64, la *model.LoginAccount)

	// 机器关联计划任务
	MachineRelateCronJobs(machineId uint64, cronJobs []uint64, la *model.LoginAccount)

	// 初始化计划任务
	InitCronJob()
}

type machineCropJobAppImpl struct {
	machineCropJobRepo       repository.MachineCronJob
	machineCropJobRelateRepo repository.MachineCronJobRelate
	machineCropJobExecRepo   repository.MachineCronJobExec
	machineApp               Machine
}

func newMachineCronJobApp(
	machineCropJobRepo repository.MachineCronJob,
	machineCropJobRelateRepo repository.MachineCronJobRelate,
	machineCropJobExecRepo repository.MachineCronJobExec,
	machineApp Machine,
) MachineCronJob {
	return &machineCropJobAppImpl{
		machineCropJobRepo:       machineCropJobRepo,
		machineCropJobRelateRepo: machineCropJobRelateRepo,
		machineCropJobExecRepo:   machineCropJobExecRepo,
		machineApp:               machineApp,
	}
}

// 分页获取机器脚本任务列表
func (m *machineCropJobAppImpl) GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return m.machineCropJobRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 获取分页执行结果列表
func (m *machineCropJobAppImpl) GetExecPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return m.machineCropJobExecRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据id获取
func (m *machineCropJobAppImpl) GetById(id uint64, cols ...string) *entity.MachineCronJob {
	return m.machineCropJobRepo.GetById(id, cols...)
}

// 保存机器任务信息
func (m *machineCropJobAppImpl) Save(mcj *entity.MachineCronJob) uint64 {
	// 更新操作
	if mcj.Id != 0 {
		m.machineCropJobRepo.UpdateById(mcj)
		// 处理最新的计划任务
		m.addCronJob(m.GetById(mcj.Id))
		return mcj.Id
	}

	m.addCronJob(mcj)
	m.machineCropJobRepo.Insert(mcj)
	return mcj.Id
}

func (m *machineCropJobAppImpl) Delete(id uint64) {
	m.machineCropJobRepo.Delete(id)
	m.machineCropJobExecRepo.Delete(&entity.MachineCronJobExec{CronJobId: id})
	m.machineCropJobRelateRepo.Delete(&entity.MachineCronJobRelate{CronJobId: id})
}

func (m *machineCropJobAppImpl) GetRelateMachineIds(cronJobId uint64) []uint64 {
	return m.machineCropJobRelateRepo.GetMachineIds(cronJobId)
}

func (m *machineCropJobAppImpl) GetRelateCronJobIds(machineId uint64) []uint64 {
	return m.machineCropJobRelateRepo.GetCronJobIds(machineId)
}

func (m *machineCropJobAppImpl) CronJobRelateMachines(cronJobId uint64, machineIds []uint64, la *model.LoginAccount) {
	oldMachineIds := m.machineCropJobRelateRepo.GetMachineIds(cronJobId)
	addIds, delIds, _ := collx.ArrayCompare[uint64](machineIds, oldMachineIds, func(u1, u2 uint64) bool { return u1 == u2 })
	addVals := make([]*entity.MachineCronJobRelate, 0)

	now := time.Now()
	for _, addId := range addIds {
		addVals = append(addVals, &entity.MachineCronJobRelate{
			MachineId:  addId,
			CronJobId:  cronJobId,
			Creator:    la.Username,
			CreatorId:  la.Id,
			CreateTime: &now,
		})
	}
	m.machineCropJobRelateRepo.BatchInsert(addVals)

	for _, delId := range delIds {
		m.machineCropJobRelateRepo.Delete(&entity.MachineCronJobRelate{CronJobId: cronJobId, MachineId: delId})
	}
}

func (m *machineCropJobAppImpl) MachineRelateCronJobs(machineId uint64, cronJobs []uint64, la *model.LoginAccount) {
	oldCronIds := m.machineCropJobRelateRepo.GetCronJobIds(machineId)
	addIds, delIds, _ := collx.ArrayCompare[uint64](cronJobs, oldCronIds, func(u1, u2 uint64) bool { return u1 == u2 })
	addVals := make([]*entity.MachineCronJobRelate, 0)

	now := time.Now()
	for _, addId := range addIds {
		addVals = append(addVals, &entity.MachineCronJobRelate{
			MachineId:  machineId,
			CronJobId:  addId,
			Creator:    la.Username,
			CreatorId:  la.Id,
			CreateTime: &now,
		})
	}
	m.machineCropJobRelateRepo.BatchInsert(addVals)

	for _, delId := range delIds {
		m.machineCropJobRelateRepo.Delete(&entity.MachineCronJobRelate{CronJobId: delId, MachineId: machineId})
	}
}

func (m *machineCropJobAppImpl) InitCronJob() {
	defer func() {
		if err := recover(); err != nil {
			global.Log.Errorf("机器计划任务初始化失败: %s", err.(error).Error())
		}
	}()

	pageParam := &model.PageParam{
		PageSize: 100,
		PageNum:  1,
	}
	cond := new(entity.MachineCronJob)
	cond.Status = entity.MachineCronJobStatusEnable
	mcjs := new([]entity.MachineCronJob)

	pr := m.GetPageList(cond, pageParam, mcjs)
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

func (m *machineCropJobAppImpl) addCronJob(mcj *entity.MachineCronJob) {
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
		go m.runCronJob(key)
	})
}

func (m *machineCropJobAppImpl) runCronJob(key string) {
	// 简单使用redis分布式锁防止多实例同一时刻重复执行
	if lock := rediscli.NewLock(key, 30*time.Second); lock != nil {
		if !lock.Lock() {
			return
		}
		defer lock.UnLock()
	}

	cronJob := new(entity.MachineCronJob)
	cronJob.Key = key
	err := m.machineCropJobRepo.GetBy(cronJob)
	// 不存在或禁用，则移除该任务
	if err != nil || cronJob.Status == entity.MachineCronJobStatusDisable {
		scheduler.RemoveByKey(key)
	}

	machienIds := m.machineCropJobRelateRepo.GetMachineIds(cronJob.Id)
	for _, machineId := range machienIds {
		go m.runCronJob0(machineId, cronJob)
	}
}

func (m *machineCropJobAppImpl) runCronJob0(mid uint64, cronJob *entity.MachineCronJob) {
	defer func() {
		if err := recover(); err != nil {
			res := err.(error).Error()
			m.machineCropJobExecRepo.Insert(&entity.MachineCronJobExec{
				MachineId: mid,
				CronJobId: cronJob.Id,
				ExecTime:  time.Now(),
				Status:    entity.MachineCronJobExecStatusError,
				Res:       res,
			})
			global.Log.Errorf("机器:[%d]执行[%s]计划任务失败: %s", mid, cronJob.Name, res)
		}
	}()

	res, err := m.machineApp.GetCli(uint64(mid)).Run(cronJob.Script)
	if err != nil {
		if res == "" {
			res = err.Error()
		}
		global.Log.Errorf("机器:[%d]执行[%s]计划任务失败: %s", mid, cronJob.Name, res)
	} else {
		global.Log.Debugf("机器:[%d]执行[%s]计划任务成功, 执行结果: %s", mid, cronJob.Name, res)
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
	m.machineCropJobExecRepo.Insert(execRes)
}
