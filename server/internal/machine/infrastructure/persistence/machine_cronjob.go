package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineCropJobRepoImpl struct {
	base.RepoImpl[*entity.MachineCronJob]
}

func newMachineCronJobRepo() repository.MachineCronJob {
	return &machineCropJobRepoImpl{base.RepoImpl[*entity.MachineCronJob]{M: new(entity.MachineCronJob)}}
}

// 分页获取机器信息列表
func (m *machineCropJobRepoImpl) GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(condition).Like("name", condition.Name).Eq("status", condition.Status).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}
