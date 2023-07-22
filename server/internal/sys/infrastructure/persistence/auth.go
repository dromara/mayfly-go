package persistence

import (
	"mayfly-go/internal/sys/domain/entity"
	"mayfly-go/internal/sys/domain/repository"
	"mayfly-go/pkg/gormx"
)

type authAccountRepoImpl struct{}

func newAuthAccountRepo() repository.OAuthAccount {
	return new(authAccountRepoImpl)
}

func (a *authAccountRepoImpl) GetOAuthAccount(condition *entity.OAuthAccount, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (a *authAccountRepoImpl) SaveOAuthAccount(e *entity.OAuthAccount) error {
	if e.Id == 0 {
		return gormx.Insert(e)
	}
	return gormx.UpdateById(e)
}
