package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineFileRepoImpl struct {
	base.RepoImpl[*entity.MachineFile]
}

func newMachineFileRepo() repository.MachineFile {
	return &machineFileRepoImpl{}
}

// 分页获取机器文件信息列表
func (m *machineFileRepoImpl) GetPageList(condition *entity.MachineFile, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineFile], error) {
	qd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam)
}
