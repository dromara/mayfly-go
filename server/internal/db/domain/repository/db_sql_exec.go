package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/model"
)

type DbSqlExec interface {
	Insert(d *entity.DbSqlExec)

	DeleteBy(condition *entity.DbSqlExec)

	// 分页获取
	GetPageList(condition *entity.DbSqlExec, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]
}
