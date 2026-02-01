package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(accountAppImpl))
	ioc.Register(new(roleAppImpl))
	ioc.Register(new(configAppImpl))
	ioc.Register(new(resourceAppImpl))
	ioc.Register(new(syslogAppImpl))
}

func GetAccountApp() Account {
	return ioc.Get[Account]()
}

func GetConfigApp() Config {
	return ioc.Get[Config]()
}

func GetSyslogApp() Syslog {
	return ioc.Get[Syslog]()
}
