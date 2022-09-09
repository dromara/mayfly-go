package persistence

import (
	"mayfly-go/internal/mongo/domain/repository"
)

var (
	mongoRepo repository.Mongo = newMongoRepo()
)

func GetMongoRepo() repository.Mongo {
	return mongoRepo
}
