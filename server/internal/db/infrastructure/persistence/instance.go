package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type instanceRepoImpl struct{}

func newInstanceRepo() repository.Instance {
	return new(instanceRepoImpl)
}

// 分页获取数据库信息列表
func (d *instanceRepoImpl) GetInstanceList(condition *entity.InstanceQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(new(entity.Instance)).
		Eq("id", condition.Id).
		Eq("host", condition.Host).
		Like("name", condition.Name)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (d *instanceRepoImpl) Count(condition *entity.InstanceQuery) int64 {
	where := make(map[string]any)
	return gormx.CountByCond(new(entity.Instance), where)
}

// 根据条件获数据库实例信息
func (d *instanceRepoImpl) GetInstance(condition *entity.Instance, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

// 根据id获取数据库实例
func (d *instanceRepoImpl) GetById(id uint64, cols ...string) *entity.Instance {
	instance := new(entity.Instance)
	if err := gormx.GetById(instance, id, cols...); err != nil {
		return nil
	}
	return instance
}

func (d *instanceRepoImpl) Insert(db *entity.Instance) {
	biz.ErrIsNil(gormx.Insert(db), "新增数据库实例失败")
}

func (d *instanceRepoImpl) Update(db *entity.Instance) {
	biz.ErrIsNil(gormx.UpdateById(db), "更新数据库实例失败")
}

func (d *instanceRepoImpl) Delete(id uint64) {
	gormx.DeleteById(new(entity.Instance), id)
}
