package persistence

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/internal/devops/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type dbSqlExecRepo struct{}

var DbSqlExecDao repository.DbSqlExec = &dbSqlExecRepo{}

func (d *dbSqlExecRepo) Insert(dse *entity.DbSqlExec) {
	model.Insert(dse)
}

func (d *dbSqlExecRepo) DeleteBy(condition *entity.DbSqlExec) {
	biz.ErrIsNil(model.DeleteByCondition(condition), "删除sql执行记录失败")
}

// 分页获取
func (d *dbSqlExecRepo) GetPageList(condition *entity.DbSqlExec, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, toEntity, orderBy...)
}
