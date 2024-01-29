package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newAccountRepo(), ioc.WithComponentName("AccountRepo"))
	ioc.Register(newRoleRepo(), ioc.WithComponentName("RoleRepo"))
	ioc.Register(newAccountRoleRepo(), ioc.WithComponentName("AccountRoleRepo"))
	ioc.Register(newResourceRepo(), ioc.WithComponentName("ResourceRepo"))
	ioc.Register(newConfigRepo(), ioc.WithComponentName("ConfigRepo"))
	ioc.Register(newSyslogRepo(), ioc.WithComponentName("SyslogRepo"))
}
