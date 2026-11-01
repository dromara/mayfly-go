package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbColumnAclRepoImpl struct {
	base.RepoImpl[*entity.DbColumnAcl]
}

var _ repository.DbColumnAcl = (*dbColumnAclRepoImpl)(nil)

func newDbColumnAclRepo() repository.DbColumnAcl {
	return &dbColumnAclRepoImpl{}
}

// ListByDb 取某库配置的全部列级规则（稳定排序，界面顺序不随插入顺序漂移）
func (r *dbColumnAclRepoImpl) ListByDb(dbId uint64) ([]*entity.DbColumnAcl, error) {
	return r.SelectByCond(model.NewCond().Eq("db_id", dbId).OrderByAsc("table_name").OrderByAsc("column_name"))
}
