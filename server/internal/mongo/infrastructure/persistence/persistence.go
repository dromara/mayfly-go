package persistence

import (
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newMongoRepo(), ioc.WithComponentName("MongoRepo"))
}
