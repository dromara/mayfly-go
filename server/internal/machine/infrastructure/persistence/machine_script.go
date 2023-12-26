package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineScriptRepoImpl struct {
	base.RepoImpl[*entity.MachineScript]
}

func newMachineScriptRepo() repository.MachineScript {
	return &machineScriptRepoImpl{base.RepoImpl[*entity.MachineScript]{M: new(entity.MachineScript)}}
}

// 分页获取机器信息列表
func (m *machineScriptRepoImpl) GetPageList(condition *entity.MachineScript, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
