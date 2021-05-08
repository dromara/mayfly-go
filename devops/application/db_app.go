package application

import (
	"mayfly-go/base/model"
	"mayfly-go/devops/domain/entity"
	"mayfly-go/devops/domain/repository"
	"mayfly-go/devops/infrastructure/db"
	"mayfly-go/devops/infrastructure/persistence"
)

type IDb interface {
	// 分页获取机器脚本信息列表
	GetPageList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult

	// 根据条件获取
	GetDbBy(condition *entity.Db, cols ...string) error

	// 根据id获取
	GetById(id uint64, cols ...string) *entity.Db

	Save(entity *entity.Db)

	GetDbInstance(id uint64) *db.DbInstance
}

type dbApp struct {
	dbRepo repository.Db
}

var Db IDb = &dbApp{dbRepo: persistence.DbDao}

// 分页获取数据库信息列表
func (d *dbApp) GetPageList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) model.PageResult {
	return d.dbRepo.GetDbList(condition, pageParam, toEntity, orderBy...)
}

// 根据条件获取
func (d *dbApp) GetDbBy(condition *entity.Db, cols ...string) error {
	return d.dbRepo.GetDb(condition, cols...)
}

// 根据id获取
func (d *dbApp) GetById(id uint64, cols ...string) *entity.Db {
	return d.dbRepo.GetById(id, cols...)
}

func (d *dbApp) Save(entity *entity.Db) {

}

func (d *dbApp) GetDbInstance(id uint64) *db.DbInstance {
	return db.GetDbInstance(id, func(id uint64) *entity.Db {
		return d.dbRepo.GetById(id)
	})
}
