package persistence

import (
	"mayfly-go/pkg/ioc"
)

func InitIoc() {
	ioc.Register(newTagTreeRepo())
	ioc.Register(newTeamRepo())
	ioc.Register(newTeamMemberRepo())
	ioc.Register(newResourceAuthCertRepoImpl())
	ioc.Register(newResourceOpLogRepo())
	ioc.Register(newTagTreeRelateRepo())
}
