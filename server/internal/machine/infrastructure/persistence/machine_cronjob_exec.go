package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineCropJobExecRepoImpl struct {
	base.RepoImpl[*entity.MachineCronJobExec]
}

func newMachineCronJobExecRepo() repository.MachineCronJobExec {
	return &machineCropJobExecRepoImpl{base.RepoImpl[*entity.MachineCronJobExec]{M: new(entity.MachineCronJobExec)}}
}

// 分页获取机器信息列表
func (m *machineCropJobExecRepoImpl) GetPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam, toEntity)
}
