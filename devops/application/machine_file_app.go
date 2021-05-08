package application

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
	"mayfly-go/devops/infrastructure/persistence"
)

type IMachineFile interface {
	// 分页获取机器文件信息列表
	GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	// 根据条件获取
	GetMachineFile(condition *entity.MachineFile, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.MachineFile

	Save(entity *entity.MachineFile)

	Delete(id uint64)
}

type machineFileApp struct {
	machineFileRepo repository.MachineFile
	machineRepo     repository.Machine
}

// 实现类单例
var MachineFile IMachineFile = &machineFileApp{
	machineRepo:     persistence.MachineDao,
	machineFileRepo: persistence.MachineFileDao}

// 分页获取机器脚本信息列表
func (m *machineFileApp) GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return m.machineFileRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (m *machineFileApp) GetMachineFile(condition *entity.MachineFile, cols ...string) error {
	return m.machineFileRepo.GetMachineFile(condition, cols...)
}

// 根据id获取
func (m *machineFileApp) GetById(id uint64, cols ...string) *entity.MachineFile {
	return m.machineFileRepo.GetById(id, cols...)
}

// 保存机器脚本
func (m *machineFileApp) Save(entity *entity.MachineFile) {
	biz.NotNil(m.machineFileRepo.GetById(entity.MachineId, "Name"), "该机器不存在")

	if entity.Id != 0 {
		model.UpdateById(entity)
	} else {
		model.Insert(entity)
	}
}

// 根据id删除
func (m *machineFileApp) Delete(id uint64) {
	m.machineFileRepo.Delete(id)
}
