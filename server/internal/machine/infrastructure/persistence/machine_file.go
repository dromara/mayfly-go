package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineFileRepoImpl struct{}

func newMachineFileRepo() repository.MachineFile {
	return new(machineFileRepoImpl)
}

// 分页获取机器文件信息列表
func (m *machineFileRepoImpl) GetPageList(condition *entity.MachineFile, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

// 根据条件获取账号信息
func (m *machineFileRepoImpl) GetMachineFile(condition *entity.MachineFile, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取
func (m *machineFileRepoImpl) GetById(id uint64, cols ...string) *entity.MachineFile {
	ms := new(entity.MachineFile)
	if err := gormx.GetById(ms, id, cols...); err != nil {
		return nil
	}
	return ms
}

// 根据id获取
func (m *machineFileRepoImpl) Delete(id uint64) error {
	return gormx.DeleteById(new(entity.MachineFile), id)
}

func (m *machineFileRepoImpl) Create(entity *entity.MachineFile) error {
	return gormx.Insert(entity)
}

func (m *machineFileRepoImpl) UpdateById(entity *entity.MachineFile) error {
	return gormx.UpdateById(entity)
}
