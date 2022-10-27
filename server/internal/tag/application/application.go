package application

import (
	dbapp "mayfly-go/internal/db/application"
	machineapp "mayfly-go/internal/machine/application"
	mongoapp "mayfly-go/internal/mongo/application"
	redisapp "mayfly-go/internal/redis/application"
	"mayfly-go/internal/tag/infrastructure/persistence"
)

var (
	tagTreeApp TagTree = newTagTreeApp(
		persistence.GetTagTreeRepo(),
		persistence.GetTagTreeTeamRepo(),
		machineapp.GetMachineApp(),
		redisapp.GetRedisApp(),
		dbapp.GetDbApp(),
		mongoapp.GetMongoApp(),
	)

	teamApp Team = newTeamApp(
		persistence.GetTeamRepo(),
		persistence.GetTeamMemberRepo(),
		persistence.GetTagTreeTeamRepo(),
	)
)

func GetTagTreeApp() TagTree {
	return tagTreeApp
}

func GetTeamApp() Team {
	return teamApp
}
