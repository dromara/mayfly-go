package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type machineCropJobExecRepoImpl struct{}

func newMachineCronJobExecRepo() repository.MachineCronJobExec {
	return new(machineCropJobExecRepoImpl)
}

// 分页获取机器信息列表
func (m *machineCropJobExecRepoImpl) GetPageList(condition *entity.MachineCronJobExec, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *machineCropJobExecRepoImpl) Insert(entity *entity.MachineCronJobExec) {
	gormx.Insert(entity)
}

func (m *machineCropJobExecRepoImpl) Delete(mcje *entity.MachineCronJobExec) {
	gormx.DeleteByCondition(mcje)
}
