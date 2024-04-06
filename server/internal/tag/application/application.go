package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(tagTreeAppImpl), ioc.WithComponentName("TagTreeApp"))
	ioc.Register(new(teamAppImpl), ioc.WithComponentName("TeamApp"))
}
