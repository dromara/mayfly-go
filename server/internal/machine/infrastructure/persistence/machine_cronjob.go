package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type machineCronJobRepoImpl struct {
	base.RepoImpl[*entity.MachineCronJob]
}

func newMachineCronJobRepo() repository.MachineCronJob {
	return &machineCronJobRepoImpl{base.RepoImpl[*entity.MachineCronJob]{M: new(entity.MachineCronJob)}}
}

// 分页获取机器信息列表
func (m *machineCronJobRepoImpl) GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().Like("name", condition.Name).Eq("status", condition.Status).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam, toEntity)
}
