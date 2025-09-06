package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(containerAppImpl), ioc.WithComponentName("ContainerApp"))
}

func GetContainerApp() Container {
	return ioc.Get[Container]("ContainerApp")
}
