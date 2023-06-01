package persistence

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type authCertRepoImpl struct{}

func newAuthCertRepo() repository.AuthCert {
	return new(authCertRepoImpl)
}

func (m *authCertRepoImpl) GetPageList(condition *entity.AuthCert, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult {
	return model.GetPage(pageParam, condition, condition, toEntity)
}

func (m *authCertRepoImpl) Insert(ac *entity.AuthCert) {
	biz.ErrIsNil(model.Insert(ac), "新增授权凭证失败")
}

func (m *authCertRepoImpl) Update(ac *entity.AuthCert) {
	biz.ErrIsNil(model.UpdateById(ac), "更新授权凭证失败")
}

func (m *authCertRepoImpl) GetById(id uint64) *entity.AuthCert {
	ac := new(entity.AuthCert)
	err := model.GetById(ac, id)
	if err != nil {
		return nil
	}
	return ac
}

func (m *authCertRepoImpl) GetByIds(ids ...uint64) []*entity.AuthCert {
	acs := new([]*entity.AuthCert)
	model.GetByIdIn(new(entity.AuthCert), acs, ids)
	return *acs
}

func (m *authCertRepoImpl) GetByCondition(condition *entity.AuthCert, cols ...string) error {
	return model.GetBy(condition, cols...)
}

func (m *authCertRepoImpl) DeleteById(id uint64) {
	model.DeleteById(new(entity.AuthCert), id)
}
