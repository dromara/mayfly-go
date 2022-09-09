package application

import (
	dbapp "mayfly-go/internal/db/application"
	machineapp "mayfly-go/internal/machine/application"
	mongoapp "mayfly-go/internal/mongo/application"
	"mayfly-go/internal/project/infrastructure/persistence"
	redisapp "mayfly-go/internal/redis/application"
)

var (
	projectApp Project = newProjectApp(
		persistence.GetProjectRepo(),
		persistence.GetProjectEnvRepo(),
		persistence.GetProjectMemberRepo(),
		machineapp.GetMachineApp(),
		redisapp.GetRedisApp(),
		dbapp.GetDbApp(),
		mongoapp.GetMongoApp(),
	)
)

func GetProjectApp() Project {
	return projectApp
}
