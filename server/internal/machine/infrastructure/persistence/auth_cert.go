package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type authCertRepoImpl struct {
	base.RepoImpl[*entity.AuthCert]
}

func newAuthCertRepo() repository.AuthCert {
	return &authCertRepoImpl{base.RepoImpl[*entity.AuthCert]{M: new(entity.AuthCert)}}
}

func (m *authCertRepoImpl) GetPageList(condition *entity.AuthCertQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	qd := gormx.NewQuery(new(entity.AuthCert)).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

// func (m *authCertRepoImpl) GetByIds(ids ...uint64) []*entity.AuthCert {
// 	acs := new([]*entity.AuthCert)
// 	gormx.GetByIdIn(new(entity.AuthCert), acs, ids)
// 	return *acs
// }
