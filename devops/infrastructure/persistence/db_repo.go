package persistence

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
)

type dbRepo struct{}

var DbDao repository.Db = &dbRepo{}

// 分页获取数据库信息列表
func (d *dbRepo) GetDbList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}

// 根据条件获取账号信息
func (d *dbRepo) GetDb(condition *entity.Db, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (d *dbRepo) GetById(id uint64, cols ...string) *entity.Db {
	db := new(entity.Db)
	if err := model.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}
