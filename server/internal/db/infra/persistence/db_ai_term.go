package persistence

import (
	"mayfly-go/internal/db/domain/entity"
	"mayfly-go/internal/db/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/model"
)

type dbAiTermRepoImpl struct {
	base.RepoImpl[*entity.DbAiTerm]
}

var _ repository.DbAiTerm = (*dbAiTermRepoImpl)(nil)

func newDbAiTermRepo() repository.DbAiTerm {
	return &dbAiTermRepoImpl{}
}

// ListByDb 取某库资源的全部词典条目
func (r *dbAiTermRepoImpl) ListByDb(dbId uint64) ([]*entity.DbAiTerm, error) {
	return r.SelectByCond(model.NewCond().Eq("db_id", dbId).OrderByAsc("term"))
}
