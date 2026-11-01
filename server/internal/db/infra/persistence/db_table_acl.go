package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbTableAclRepoImpl struct {
	base.RepoImpl[*entity.DbTableAcl]
}

var _ repository.DbTableAcl = (*dbTableAclRepoImpl)(nil)

func newDbTableAclRepo() repository.DbTableAcl {
	return &dbTableAclRepoImpl{}
}

// ListByDb 取某库配置的全部表访问规则。
//
// 按表名升序：同一库的规则要在界面上稳定展示，map 遍历顺序会让每次刷新都换位置
func (r *dbTableAclRepoImpl) ListByDb(dbId uint64) ([]*entity.DbTableAcl, error) {
	return r.SelectByCond(model.NewCond().Eq("db_id", dbId).OrderByAsc("table_name"))
}
