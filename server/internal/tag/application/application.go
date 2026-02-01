package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(tagTreeAppImpl))
	ioc.Register(new(teamAppImpl))
	ioc.Register(new(resourceAuthCertAppImpl))
	ioc.Register(new(resourceOpLogAppImpl))
	ioc.Register(new(tagTreeRelateAppImpl))
}

func GetResourceOpLogApp() ResourceOpLog {
	return ioc.Get[ResourceOpLog]()
}
