package application

import (
	"mayfly-go/internal/mongo/infrastructure/persistence"
	tagapp "mayfly-go/internal/tag/application"
)

var (
	mongoApp Mongo = newMongoAppImpl(persistence.GetMongoRepo(), tagapp.GetTagTreeApp())
)

func GetMongoApp() Mongo {
	return mongoApp
}
