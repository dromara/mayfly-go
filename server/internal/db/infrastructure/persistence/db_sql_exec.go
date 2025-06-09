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
	return &dbSqlExecRepoImpl{}
}

// 分页获取
func (d *dbSqlExecRepoImpl) GetPageList(condition *entity.DbSqlExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSqlExec], error) {
	qd := model.NewCond().
		Eq("db_id", condition.DbId).
		Eq("`table`", condition.Table).
		Eq("type", condition.Type).
		Eq("creator_id", condition.CreatorId).
		Eq("flow_biz_key", condition.FlowBizKey).
		In("status", condition.Status).
		Like("sql", condition.Keyword).
		Ge("create_time", condition.StartTime).
		Le("create_time", condition.EndTime).
		RLike("db", condition.Db).OrderBy(orderBy...)
	return d.PageByCond(qd, condition.PageParam)
}
