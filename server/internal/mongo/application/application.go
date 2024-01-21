package application

import (
	"mayfly-go/internal/mongo/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func init() {
	persistence.Init()

	ioc.Register(new(mongoAppImpl), ioc.WithComponentName("MongoApp"))
}

func GetMongoApp() Mongo {
	return ioc.Get[Mongo]("MongoApp")
}
