package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newAccountRepo())
	ioc.Register(newRoleRepo())
	ioc.Register(newRoleResourceRepo())
	ioc.Register(newAccountRoleRepo())
	ioc.Register(newResourceRepo())
	ioc.Register(newConfigRepo())
	ioc.Register(newSyslogRepo())
}
