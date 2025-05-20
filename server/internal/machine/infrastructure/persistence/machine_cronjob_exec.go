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
	return &machineCropJobExecRepoImpl{}
}

// 分页获取机器信息列表
func (m *machineCropJobExecRepoImpl) GetPageList(condition *entity.MachineCronJobExec, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJobExec], error) {
	qd := model.NewModelCond(condition).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam)
}
