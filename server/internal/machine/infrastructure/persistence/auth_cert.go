package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/gormx"
	"mayfly-go/pkg/model"
)

type authCertRepoImpl struct{}

func newAuthCertRepo() repository.AuthCert {
	return new(authCertRepoImpl)
}

func (m *authCertRepoImpl) GetPageList(condition *entity.AuthCert, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	qd := gormx.NewQuery(condition).WithCondModel(condition).WithOrderBy(orderBy...)
	return gormx.PageQuery(qd, pageParam, toEntity)
}

func (m *authCertRepoImpl) Insert(ac *entity.AuthCert) {
	biz.ErrIsNil(gormx.Insert(ac), "新增授权凭证失败")
}

func (m *authCertRepoImpl) Update(ac *entity.AuthCert) {
	biz.ErrIsNil(gormx.UpdateById(ac), "更新授权凭证失败")
}

func (m *authCertRepoImpl) GetById(id uint64) *entity.AuthCert {
	ac := new(entity.AuthCert)
	err := gormx.GetById(ac, id)
	if err != nil {
		return nil
	}
	return ac
}

func (m *authCertRepoImpl) GetByIds(ids ...uint64) []*entity.AuthCert {
	acs := new([]*entity.AuthCert)
	gormx.GetByIdIn(new(entity.AuthCert), acs, ids)
	return *acs
}

func (m *authCertRepoImpl) GetByCondition(condition *entity.AuthCert, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (m *authCertRepoImpl) DeleteById(id uint64) {
	gormx.DeleteById(new(entity.AuthCert), id)
}
