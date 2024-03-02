package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(oauth2AppImpl), ioc.WithComponentName("Oauth2App"))
}
