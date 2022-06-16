package repository

import (
	"mayfly-go/internal/devops/domain/entity"
	"mayfly-go/pkg/model"
)

type DbSqlExec interface {
	Insert(d *entity.DbSqlExec)

	DeleteBy(condition *entity.DbSqlExec)

	// 分页获取
	GetPageList(condition *entity.DbSqlExec, pageParam *model.PageParam, toEntity interface{}, orderBy ...string) *model.PageResult
}
