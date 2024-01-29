package application

import (
	"mayfly-go/internal/mongo/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(mongoAppImpl), ioc.WithComponentName("MongoApp"))
}
