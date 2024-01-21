package persistence

import (
	"mayfly-go/internal/sys/domain/repository"
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

func GetAccountRepo() repository.Account {
	return ioc.Get[repository.Account]("AccountRepo")
}

func GetConfigRepo() repository.Config {
	return ioc.Get[repository.Config]("ConfigRepo")
}

func GetResourceRepo() repository.Resource {
	return ioc.Get[repository.Resource]("ResourceRepo")
}

func GetRoleRepo() repository.Role {
	return ioc.Get[repository.Role]("RoleRepo")
}

func GetAccountRoleRepo() repository.AccountRole {
	return ioc.Get[repository.AccountRole]("AccountRoleRepo")
}

func GetSyslogRepo() repository.Syslog {
	return ioc.Get[repository.Syslog]("SyslogRepo")
}
