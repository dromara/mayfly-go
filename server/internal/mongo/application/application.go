package application

import "mayfly-go/internal/mongo/infrastructure/persistence"

var (
	mongoApp Mongo = newMongoAppImpl(persistence.GetMongoRepo())
)

func GetMongoApp() Mongo {
	return mongoApp
}
