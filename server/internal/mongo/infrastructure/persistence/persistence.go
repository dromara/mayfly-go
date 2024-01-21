package persistence

import (
	"mayfly-go/internal/mongo/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newMongoRepo(), ioc.WithComponentName("MongoRepo"))
}

func GetMongoRepo() repository.Mongo {
	return ioc.Get[repository.Mongo]("MongoRepo")
}
