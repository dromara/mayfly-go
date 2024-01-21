package application

import (
	"context"
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
)

type Oauth2 interface {
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	BindOAuthAccount(e *entity.Oauth2Account) error

	Unbind(accountId uint64)
}

type oauth2AppImpl struct {
	Oauth2AccountRepo repository.Oauth2Account `inject:""`
}

func (a *oauth2AppImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	return a.Oauth2AccountRepo.GetBy(condition, cols...)
}

func (a *oauth2AppImpl) BindOAuthAccount(e *entity.Oauth2Account) error {
	if e.Id == 0 {
		return a.Oauth2AccountRepo.Insert(context.Background(), e)
	}
	return a.Oauth2AccountRepo.UpdateById(context.Background(), e)
}

func (a *oauth2AppImpl) Unbind(accountId uint64) {
	a.Oauth2AccountRepo.DeleteByCond(context.Background(), &entity.Oauth2Account{AccountId: accountId})
}
