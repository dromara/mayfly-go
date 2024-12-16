package application

import (
	"context"
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/model"
)

type Oauth2 interface {
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	BindOAuthAccount(e *entity.Oauth2Account) error

	Unbind(accountId uint64)
}

type oauth2AppImpl struct {
	oauth2AccountRepo repository.Oauth2Account `inject:"T"`
}

func (a *oauth2AppImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	return a.oauth2AccountRepo.GetByCond(model.NewModelCond(condition).Columns(cols...))
}

func (a *oauth2AppImpl) BindOAuthAccount(e *entity.Oauth2Account) error {
	if e.Id == 0 {
		return a.oauth2AccountRepo.Insert(context.Background(), e)
	}
	return a.oauth2AccountRepo.UpdateById(context.Background(), e)
}

func (a *oauth2AppImpl) Unbind(accountId uint64) {
	a.oauth2AccountRepo.DeleteByCond(context.Background(), &entity.Oauth2Account{AccountId: accountId})
}
