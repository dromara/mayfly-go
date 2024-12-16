package api

import "mayfly-go/pkg/ioc"

func InitIoc() {
	ioc.Register(new(Account))
	ioc.Register(new(Config))
	ioc.Register(new(Resource))
	ioc.Register(new(Role))
	ioc.Register(new(Syslog))
	ioc.Register(new(System))
}
