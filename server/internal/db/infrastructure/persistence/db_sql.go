package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type dbSqlRepoImpl struct{}

func newDbSqlRepo() repository.DbSql {
	return new(dbSqlRepoImpl)
}

// 分页获取数据库信息列表
func (d *dbSqlRepoImpl) DeleteBy(condition *entity.DbSql) {
	biz.ErrIsNil(model.DeleteByCondition(condition), "删除sql失败")
}
