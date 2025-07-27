package persistence

import (
	"mayfly-go/internal/auth/domain/entity"
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/base"
)

type oauth2AccountRepoImpl struct {
	base.RepoImpl[*entity.Oauth2Account]
}

func newAuthAccountRepo() repository.Oauth2Account {
	return &oauth2AccountRepoImpl{}
}
