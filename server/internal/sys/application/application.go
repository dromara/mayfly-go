package application

import (
	"mayfly-go/internal/sys/infrastructure/persistence"
)

var (
	accountApp  = newAccountApp(persistence.GetAccountRepo())
	configApp   = newConfigApp(persistence.GetConfigRepo())
	msgApp      = newMsgApp(persistence.GetMsgRepo())
	resourceApp = newResourceApp(persistence.GetResourceRepo())
	roleApp     = newRoleApp(persistence.GetRoleRepo())
	syslogApp   = newSyslogApp(persistence.GetSyslogRepo())
)

func GetAccountApp() Account {
	return accountApp
}

func GetConfigApp() Config {
	return configApp
}

func GetMsgApp() Msg {
	return msgApp
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
