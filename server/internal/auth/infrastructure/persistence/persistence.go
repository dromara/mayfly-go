package persistence

import (
	"mayfly-go/internal/auth/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newAuthAccountRepo(), ioc.WithComponentName("Oauth2AccountRepo"))
}

func GetOauthAccountRepo() repository.Oauth2Account {
	return ioc.Get[repository.Oauth2Account]("Oauth2AccountRepo")
}
