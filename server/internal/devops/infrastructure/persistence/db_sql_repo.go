package persistence

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type dbSqlRepo struct{}

var DbSqlDao repository.DbSql = &dbSqlRepo{}

// 分页获取数据库信息列表
func (d *dbSqlRepo) DeleteBy(condition *entity.DbSql) {
	biz.ErrIsNil(model.DeleteByCondition(condition), "删除sql失败")
}
