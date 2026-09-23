package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbSQLExecRepoImpl struct {
	base.RepoImpl[*entity.DbSQLExec]
}

var _ repository.DbSQLExec = (*dbSQLExecRepoImpl)(nil)

func newDbSQLExecRepo() repository.DbSQLExec {
	return &dbSQLExecRepoImpl{}
}

// 分页获取
func (d *dbSQLExecRepoImpl) GetPageList(condition *entity.DbSQLExecQuery, orderBy ...string) (*model.PageResult[*entity.DbSQLExec], error) {
	qd := model.NewCond().
		Eq("db_id", condition.DbId).
		Eq("`table`", condition.Table).
		Eq("type", condition.Type).
		Eq("creator_id", condition.CreatorId).
		Eq("flow_biz_key", condition.FlowBizKey).
		In("status", condition.Status).
		Like("`sql`", condition.Keyword).
		Ge("create_time", condition.StartTime).
		Le("create_time", condition.EndTime).
		RLike("db", condition.Db).OrderBy(orderBy...)
	return d.PageByCond(qd, condition.PageParam)
}
