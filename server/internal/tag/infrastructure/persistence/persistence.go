package persistence

import "mayfly-go/internal/tag/domain/repository"

var (
	tagTreeRepo     repository.TagTree     = newTagTreeRepo()
	tagTreeTeamRepo repository.TagTreeTeam = newTagTreeTeamRepo()
	teamRepo        repository.Team        = newTeamRepo()
	teamMemberRepo  repository.TeamMember  = newTeamMemberRepo()
)

func GetTagTreeRepo() repository.TagTree {
	return tagTreeRepo
}

func GetTagTreeTeamRepo() repository.TagTreeTeam {
	return tagTreeTeamRepo
}

func GetTeamRepo() repository.Team {
	return teamRepo
}

func GetTeamMemberRepo() repository.TeamMember {
	return teamMemberRepo
}
