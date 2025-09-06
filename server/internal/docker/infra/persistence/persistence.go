package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newContainerRepo(), ioc.WithComponentName("ContainerRepo"))
}
