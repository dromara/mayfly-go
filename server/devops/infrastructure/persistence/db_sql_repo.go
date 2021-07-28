package persistence

import (
	"mayfly-go/base/biz"
	"mayfly-go/base/model"
	"mayfly-go/server/devops/domain/entity"
	"mayfly-go/server/devops/domain/repository"
)

type dbSqlRepo struct{}

var DbSqlDao repository.DbSql = &dbSqlRepo{}

// 分页获取数据库信息列表
func (d *dbSqlRepo) DeleteBy(condition *entity.DbSql) {
	biz.ErrIsNil(model.DeleteByCondition(condition), "删除sql失败")
}
