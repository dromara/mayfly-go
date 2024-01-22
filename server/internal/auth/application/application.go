package application

import (
	"mayfly-go/internal/auth/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(oauth2AppImpl), ioc.WithComponentName("Oauth2App"))
}

func GetAuthApp() Oauth2 {
	return ioc.Get[Oauth2]("Oauth2App")
}
