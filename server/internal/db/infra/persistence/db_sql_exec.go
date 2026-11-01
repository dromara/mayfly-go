package persistence

import (
	"context"
	"time"

	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
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
	// 可见库范围必须由调用方注入；空集合直接返回空页（fail-closed）。
	// 不能把「未注入」当成「不过滤」：执行记录里有行级旧值/新值，那等于把全平台业务数据摊开
	if len(condition.ScopeDbIds) == 0 {
		return model.NewEmptyPageResult[*entity.DbSQLExec](), nil
	}
	qd := model.NewCond().
		In("db_id", condition.ScopeDbIds).
		Eq("db_id", condition.DbId).
		// 表名用包含匹配：执行记录里的表名带前缀/大小写差异（如 t_order、order_info），
		// 精确等值要求用户先猜中全名，作为筛选条件几乎不可用
		Like("`table`", condition.Table).
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

// DeleteBefore 物理删除早于 cutoff 的执行记录。
//
// 用 Unscoped：审计保留期要的是「表不再增长」，软删只是把行标记为不可见；
// 若改成软删，清理任务会永远"成功"却一行都没真正释放。
func (d *dbSQLExecRepoImpl) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	res := gormx.DB(ctx).Unscoped().Where("create_time < ?", cutoff).Delete(&entity.DbSQLExec{})
	return res.RowsAffected, res.Error
}
