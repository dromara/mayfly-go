package persistence

import "mayfly-go/internal/sys/domain/repository"

var (
	accountRepo     = newAccountRepo()
	authAccountRepo = newAuthAccountRepo()
	configRepo      = newConfigRepo()
	resourceRepo    = newResourceRepo()
	roleRepo        = newRoleRepo()
	syslogRepo      = newSyslogRepo()
)

func GetAccountRepo() repository.Account {
	return accountRepo
}

func GetOAuthAccountRepo() repository.OAuthAccount {
	return authAccountRepo
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

func GetSyslogRepo() repository.Syslog {
	return syslogRepo
}
