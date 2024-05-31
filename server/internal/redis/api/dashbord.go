package api

import (
	"mayfly-go/internal/common/consts"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	redisNum := len(m.TagTreeApp.GetAccountTagCodes(accountId, consts.ResourceTypeRedis, ""))

	rc.ResData = collx.M{
		"redisNum": redisNum,
	}
}
