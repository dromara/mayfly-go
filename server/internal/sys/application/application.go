package application

import (
	"mayfly-go/internal/sys/infrastructure/persistence"
)

var (
	accountApp  = newAccountApp(persistence.GetAccountRepo())
	authApp     = newAuthApp(persistence.GetOAuthAccountRepo())
	configApp   = newConfigApp(persistence.GetConfigRepo())
	resourceApp = newResourceApp(persistence.GetResourceRepo())
	roleApp     = newRoleApp(persistence.GetRoleRepo())
	syslogApp   = newSyslogApp(persistence.GetSyslogRepo())
)

func GetAccountApp() Account {
	return accountApp
}

func GetAuthApp() Auth {
	return authApp
}

func GetConfigApp() Config {
	return configApp
}

func GetResourceApp() Resource {
	return resourceApp
}

func GetRoleApp() Role {
	return roleApp
}

func GetSyslogApp() Syslog {
	return syslogApp
}
