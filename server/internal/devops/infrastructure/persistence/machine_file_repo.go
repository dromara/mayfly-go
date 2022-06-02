package persistence

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type machineFileRepo struct{}

var MachineFileDao repository.MachineFile = &machineFileRepo{}

// 分页获取机器文件信息列表
func (m *machineFileRepo) GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

// 根据条件获取账号信息
func (m *machineFileRepo) GetMachineFile(condition *entity.MachineFile, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineFileRepo) GetById(id uint64, cols ...string) *entity.MachineFile {
	ms := new(entity.MachineFile)
	if err := model.GetById(ms, id, cols...); err != nil {
		return nil

	}
	return ms
}

// 根据id获取
func (m *machineFileRepo) Delete(id uint64) {
	biz.ErrIsNil(model.DeleteById(new(entity.MachineFile), id), "删除失败")
}

func (m *machineFileRepo) Create(entity *entity.MachineFile) {
	model.Insert(entity)
}

func (m *machineFileRepo) UpdateById(entity *entity.MachineFile) {
	model.UpdateById(entity)
}
