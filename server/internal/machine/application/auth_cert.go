package application

import (
	"context"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type AuthCert interface {
	base.App[*entity.AuthCert]

	GetPageList(condition *entity.AuthCertQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	Save(ctx context.Context, ac *entity.AuthCert) error

	GetByIds(ids ...uint64) []*entity.AuthCert
}

type authCertAppImpl struct {
	base.AppImpl[*entity.AuthCert, repository.AuthCert]
}

// 注入AuthCertRepo
func (a *authCertAppImpl) InjectAuthCertRepo(repo repository.AuthCert) {
	a.Repo = repo
}

func (a *authCertAppImpl) GetPageList(condition *entity.AuthCertQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return a.GetRepo().GetPageList(condition, pageParam, toEntity)
}

func (a *authCertAppImpl) Save(ctx context.Context, ac *entity.AuthCert) error {
	oldAc := &entity.AuthCert{Name: ac.Name}
	err := a.GetBy(oldAc, "Id", "Name")

	ac.PwdEncrypt()
	if ac.Id == 0 {
		if err == nil {
			return errorx.NewBiz("该凭证名已存在")
		}
		return a.Insert(ctx, ac)
	}

	// 如果存在该库，则校验修改的库是否为该库
	if err == nil && oldAc.Id != ac.Id {
		return errorx.NewBiz("该凭证名已存在")
	}
	return a.UpdateById(ctx, ac)
}

func (a *authCertAppImpl) GetByIds(ids ...uint64) []*entity.AuthCert {
	acs := new([]*entity.AuthCert)
	a.GetByIdIn(acs, ids)
	return *acs
}
