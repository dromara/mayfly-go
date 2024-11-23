package api

import (
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	mongoNum := len(m.TagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{Types: []tagentity.TagType{
		tagentity.TagTypeMongo,
	}}))

	rc.ResData = collx.M{
		"mongoNum": mongoNum,
	}
}
