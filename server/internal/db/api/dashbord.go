package api

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/db/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree `inject:""`
	DbApp      application.Db `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	dbNum := len(m.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeDb, ""))

	rc.ResData = collx.M{
		"dbNum": dbNum,
	}
}
