package persistence

import "mayfly-go/internal/sys/domain/repository"

var (
	accountRepo     = newAccountRepo()
	configRepo      = newConfigRepo()
	resourceRepo    = newResourceRepo()
	roleRepo        = newRoleRepo()
	accountRoleRepo = newAccountRoleRepo()
	syslogRepo      = newSyslogRepo()
)

func GetAccountRepo() repository.Account {
	return accountRepo
}

func GetConfigRepo() repository.Config {
	return configRepo
}

func GetResourceRepo() repository.Resource {
	return resourceRepo
}

func GetRoleRepo() repository.Role {
	return roleRepo
}

func GetAccountRoleRepo() repository.AccountRole {
	return accountRoleRepo
}

func GetSyslogRepo() repository.Syslog {
	return syslogRepo
}
