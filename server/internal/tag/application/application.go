package application

import (
	"mayfly-go/internal/tag/infrastructure/persistence"
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	persistence.Init()

	ioc.Register(new(tagTreeAppImpl), ioc.WithComponentName("TagTreeApp"))
	ioc.Register(new(teamAppImpl), ioc.WithComponentName("TeamApp"))
	ioc.Register(new(tagResourceAppImpl), ioc.WithComponentName("TagResourceApp"))
}

func GetTagTreeApp() TagTree {
	return ioc.Get[TagTree]("TagTreeApp")
}

func GetTeamApp() Team {
	return ioc.Get[Team]("TeamApp")
}

func GetTagResourceApp() TagResource {
	return ioc.Get[TagResource]("TagResourceApp")
}
