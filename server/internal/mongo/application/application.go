package application

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(new(mongoAppImpl), ioc.WithComponentName("MongoApp"))
}
