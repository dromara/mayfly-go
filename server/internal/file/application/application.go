package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(fileAppImpl), ioc.WithComponentName("FileApp"))
}
