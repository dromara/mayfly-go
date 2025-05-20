package repository

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type DbSqlExec interface {
	base.Repo[*entity.DbSqlExec]

	// 分页获取
	GetPageList(condition *entity.DbSqlExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSqlExec], error)
}
