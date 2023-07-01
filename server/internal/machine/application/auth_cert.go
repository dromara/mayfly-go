package application

import (
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
)

type AuthCert interface {
	GetPageList(condition *entity.AuthCert, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any]

	Save(ac *entity.AuthCert)

	GetById(id uint64) *entity.AuthCert

	GetByIds(ids ...uint64) []*entity.AuthCert

	DeleteById(id uint64)
}

func newAuthCertApp(authCertRepo repository.AuthCert) AuthCert {
	return &authCertAppImpl{
		authCertRepo: authCertRepo,
	}
}

type authCertAppImpl struct {
	authCertRepo repository.AuthCert
}

func (a *authCertAppImpl) GetPageList(condition *entity.AuthCert, pageParam *model.PageParam, toEntity any, orderBy ...string) *model.PageResult[any] {
	return a.authCertRepo.GetPageList(condition, pageParam, toEntity)
}

func (a *authCertAppImpl) Save(ac *entity.AuthCert) {
	oldAc := &entity.AuthCert{Name: ac.Name}
	err := a.authCertRepo.GetByCondition(oldAc, "Id", "Name")

	ac.PwdEncrypt()
	if ac.Id == 0 {
		biz.IsTrue(err != nil, "该凭证名已存在")
		a.authCertRepo.Insert(ac)
		return
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil {
		biz.IsTrue(oldAc.Id == ac.Id, "该凭证名已存在")
	}
	a.authCertRepo.Update(ac)
}

func (a *authCertAppImpl) GetById(id uint64) *entity.AuthCert {
	return a.authCertRepo.GetById(id)
}

func (a *authCertAppImpl) GetByIds(ids ...uint64) []*entity.AuthCert {
	return a.authCertRepo.GetByIds(ids...)
}

func (a *authCertAppImpl) DeleteById(id uint64) {
	a.authCertRepo.DeleteById(id)
}
