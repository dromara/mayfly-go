package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newAuthAccountRepo(), ioc.WithComponentName("Oauth2AccountRepo"))
}
