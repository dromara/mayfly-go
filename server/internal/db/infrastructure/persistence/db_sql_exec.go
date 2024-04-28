package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbSqlExecRepoImpl struct {
	base.RepoImpl[*entity.DbSqlExec]
}

func newDbSqlExecRepo() repository.DbSqlExec {
	return &dbSqlExecRepoImpl{base.RepoImpl[*entity.DbSqlExec]{M: new(entity.DbSqlExec)}}
}

// 分页获取
func (d *dbSqlExecRepoImpl) GetPageList(condition *entity.DbSqlExecQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := model.NewCond().
		Eq("db_id", condition.DbId).
		Eq("`table`", condition.Table).
		Eq("type", condition.Type).
		Eq("creator_id", condition.CreatorId).
		Eq("flow_biz_key", condition.FlowBizKey).
		In("status", condition.Status).
		RLike("db", condition.Db).OrderBy(orderBy...)
	return d.PageByCond(qd, pageParam, toEntity)
}
