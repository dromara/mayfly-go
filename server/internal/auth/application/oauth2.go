package application

import (
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
)

type Oauth2 interface {
	GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error

	BindOAuthAccount(e *entity.Oauth2Account) error

	Unbind(accountId uint64)
}

func newAuthApp(oauthAccountRepo repository.Oauth2Account) Oauth2 {
	return &oauth2AppImpl{
		oauthAccountRepo: oauthAccountRepo,
	}
}

type oauth2AppImpl struct {
	oauthAccountRepo repository.Oauth2Account
}

func (a *oauth2AppImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	return a.oauthAccountRepo.GetBy(condition, cols...)
}

func (a *oauth2AppImpl) BindOAuthAccount(e *entity.Oauth2Account) error {
	if e.Id == 0 {
		return a.oauthAccountRepo.Insert(e)
	}
	return a.oauthAccountRepo.UpdateById(e)
}

func (a *oauth2AppImpl) Unbind(accountId uint64) {
	a.oauthAccountRepo.DeleteByCond(&entity.Oauth2Account{AccountId: accountId})
}
