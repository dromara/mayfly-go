package api

import (
	"mayfly-go/internal/db/application"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree `inject:""`
	DbApp      application.Db `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	tagCodePaths := m.TagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{Types: collx.AsArray(tagentity.TagTypeDb)}).GetCodePaths()

	rc.ResData = collx.M{
		"dbNum": len(tagCodePaths),
	}
}
