package persistence

import (
	"fmt"
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type dbRepoImpl struct{}

func newDbRepo() repository.Db {
	return new(dbRepoImpl)
}

// 分页获取数据库信息列表
func (d *dbRepoImpl) GetDbList(condition *entity.Db, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	sql := "SELECT d.* FROM t_db d JOIN t_project_member pm ON d.project_id = pm.project_id WHERE 1 = 1 "
	if condition.CreatorId != 0 {
		// 使用创建者id模拟项目成员id
		sql = fmt.Sprintf("%s AND pm.account_id = %d", sql, condition.CreatorId)
	}
	if condition.ProjectId != 0 {
		sql = fmt.Sprintf("%s AND d.project_id = %d", sql, condition.ProjectId)
	}
	if condition.EnvId != 0 {
		sql = fmt.Sprintf("%s AND d.env_id = %d", sql, condition.EnvId)
	}
	if condition.Host != "" {
		sql = sql + " AND d.host LIKE '%" + condition.Host + "%'"
	}
	if condition.Database != "" {
		sql = sql + " AND d.database LIKE '%" + condition.Database + "%'"
	}
	sql = sql + " ORDER BY d.create_time DESC"
	return model.GetPageBySql(sql, pageParam, toEntity)
}

func (d *dbRepoImpl) Count(condition *entity.Db) int64 {
	return model.CountBy(condition)
}

// 根据条件获取账号信息
func (d *dbRepoImpl) GetDb(condition *entity.Db, cols ...string) error {
	return model.GetBy(condition, cols...)
}

// 根据id获取
func (d *dbRepoImpl) GetById(id uint64, cols ...string) *entity.Db {
	db := new(entity.Db)
	if err := model.GetById(db, id, cols...); err != nil {
		return nil

	}
	return db
}

func (d *dbRepoImpl) Insert(db *entity.Db) {
	biz.ErrIsNil(model.Insert(db), "新增数据库信息失败")
}

func (d *dbRepoImpl) Update(db *entity.Db) {
	biz.ErrIsNil(model.UpdateById(db), "更新数据库信息失败")
}

func (d *dbRepoImpl) Delete(id uint64) {
	model.DeleteById(new(entity.Db), id)
}
