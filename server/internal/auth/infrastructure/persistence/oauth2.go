package persistence

import (
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/gormx"
)

type oauth2AccountRepoImpl struct{}

func newAuthAccountRepo() repository.Oauth2Account {
	return new(oauth2AccountRepoImpl)
}

func (a *oauth2AccountRepoImpl) GetOAuthAccount(condition *entity.Oauth2Account, cols ...string) error {
	return gormx.GetBy(condition, cols...)
}

func (a *oauth2AccountRepoImpl) SaveOAuthAccount(e *entity.Oauth2Account) error {
	if e.Id == 0 {
		return gormx.Insert(e)
	}
	return gormx.UpdateById(e)
}

func (a *oauth2AccountRepoImpl) DeleteBy(e *entity.Oauth2Account) {
	gormx.DeleteByCondition(e)
}
