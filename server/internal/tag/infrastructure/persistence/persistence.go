package persistence

import (
	"mayfly-go/internal/tag/domain/repository"
	"mayfly-go/pkg/ioc"
)

func Init() {
	ioc.Register(newTagTreeRepo(), ioc.WithComponentName("TagTreeRepo"))
	ioc.Register(newTagTreeTeamRepo(), ioc.WithComponentName("TagTreeTeamRepo"))
	ioc.Register(newTagResourceRepo(), ioc.WithComponentName("TagResourceRepo"))
	ioc.Register(newTeamRepo(), ioc.WithComponentName("TeamRepo"))
	ioc.Register(newTeamMemberRepo(), ioc.WithComponentName("TeamMemberRepo"))
}

func GetTagTreeRepo() repository.TagTree {
	return ioc.Get[repository.TagTree]("TagTreeRepo")
}

func GetTagTreeTeamRepo() repository.TagTreeTeam {
	return ioc.Get[repository.TagTreeTeam]("TagTreeTeamRepo")
}

func GetTagResourceRepo() repository.TagResource {
	return ioc.Get[repository.TagResource]("TagResourceRepo")
}

func GetTeamRepo() repository.Team {
	return ioc.Get[repository.Team]("TeamRepo")
}

func GetTeamMemberRepo() repository.TeamMember {
	return ioc.Get[repository.TeamMember]("TeamMemberRepo")
}
