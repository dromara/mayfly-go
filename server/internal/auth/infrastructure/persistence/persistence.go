package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newAuthAccountRepo(), ioc.WithComponentName("Oauth2AccountRepo"))
}
