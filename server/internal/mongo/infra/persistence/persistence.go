package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newMongoRepo(), ioc.WithComponentName("MongoRepo"))
}
