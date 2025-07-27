package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineScriptRepoImpl struct {
	base.RepoImpl[*entity.MachineScript]
}

func newMachineScriptRepo() repository.MachineScript {
	return &machineScriptRepoImpl{}
}

// 分页获取机器信息列表
func (m *machineScriptRepoImpl) GetPageList(condition *entity.MachineScript, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineScript], error) {
	qd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam)
}
