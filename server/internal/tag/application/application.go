package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(tagTreeAppImpl), ioc.WithComponentName("TagTreeApp"))
	ioc.Register(new(teamAppImpl), ioc.WithComponentName("TeamApp"))
	ioc.Register(new(resourceAuthCertAppImpl), ioc.WithComponentName("ResourceAuthCertApp"))
	ioc.Register(new(resourceOpLogAppImpl), ioc.WithComponentName("ResourceOpLogApp"))
	ioc.Register(new(tagTreeRelateAppImpl), ioc.WithComponentName("TagTreeRelateApp"))
}

func GetResourceOpLogApp() ResourceOpLog {
	return ioc.Get[ResourceOpLog]("ResourceOpLogApp")
}
