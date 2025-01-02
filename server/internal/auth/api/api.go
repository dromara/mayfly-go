package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(AccountLogin))
	ioc.Register(new(LdapLogin))
	ioc.Register(new(Oauth2Login))
	ioc.Register(new(Captcha))
}
