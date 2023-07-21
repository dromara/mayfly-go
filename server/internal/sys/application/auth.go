package application

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
)

type Auth interface {
	GetOAuthAccount(condition *entity.OAuthAccount, cols ...string) error
	BindOAuthAccount(e *entity.OAuthAccount) error
}

func newAuthApp(oauthAccountRepo repository.OAuthAccount) Auth {
	return &authAppImpl{
		oauthAccountRepo: oauthAccountRepo,
	}
}

type authAppImpl struct {
	oauthAccountRepo repository.OAuthAccount
}

func (a *authAppImpl) GetOAuthAccount(condition *entity.OAuthAccount, cols ...string) error {
	return a.oauthAccountRepo.GetOAuthAccount(condition, cols...)
}

func (a *authAppImpl) BindOAuthAccount(e *entity.OAuthAccount) error {
	return a.oauthAccountRepo.SaveOAuthAccount(e)
}
