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
	return &machineCronJobRepoImpl{}
}

// 分页获取机器信息列表
func (m *machineCronJobRepoImpl) GetPageList(condition *entity.MachineCronJob, pageParam model.PageParam, orderBy ...string) (*model.PageResult[*entity.MachineCronJob], error) {
	qd := model.NewCond().Like("name", condition.Name).Eq("status", condition.Status).OrderBy(orderBy...)
	return m.PageByCond(qd, pageParam)
}
