package application

import (
	"mayfly-go/internal/tag/infrastructure/persistence"
)

var (
	tagTreeApp TagTree = newTagTreeApp(
		persistence.GetTagTreeRepo(),
		GetTagResourceApp(),
		persistence.GetTagTreeTeamRepo(),
	)

	teamApp Team = newTeamApp(
		persistence.GetTeamRepo(),
		persistence.GetTeamMemberRepo(),
		persistence.GetTagTreeTeamRepo(),
	)

	tagResourceApp TagResource = newTagResourceApp(persistence.GetTagResourceRepo())
)

func GetTagTreeApp() TagTree {
	return tagTreeApp
}

func GetTeamApp() Team {
	return teamApp
}

func GetTagResourceApp() TagResource {
	return tagResourceApp
}
