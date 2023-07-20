package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineCropJobRepoImpl struct{}

func newMachineCronJobRepo() repository.MachineCronJob {
	return new(machineCropJobRepoImpl)
}

// 分页获取机器信息列表
func (m *machineCropJobRepoImpl) GetPageList(condition *entity.MachineCronJob, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).Like("name", condition.Name).Eq("status", condition.Status).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *machineCropJobRepoImpl) GetBy(cond *entity.MachineCronJob, cols ...string) error {
	return gormx.GetBy(cond, cols...)
}

func (m *machineCropJobRepoImpl) GetById(id uint64, cols ...string) *entity.MachineCronJob {
	res := new(entity.MachineCronJob)
	if err := gormx.GetById(res, id, cols...); err == nil {
		return res
	} else {
		return nil
	}
}

func (m *machineCropJobRepoImpl) Delete(id uint64) {
	biz.ErrIsNil(gormx.DeleteById(new(entity.MachineCronJob), id), "删除失败")
}

func (m *machineCropJobRepoImpl) Insert(entity *entity.MachineCronJob) {
	gormx.Insert(entity)
}

func (m *machineCropJobRepoImpl) UpdateById(entity *entity.MachineCronJob) {
	gormx.UpdateById(entity)
}
