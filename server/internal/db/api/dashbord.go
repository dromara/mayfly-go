package api

import (
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	tagTreeApp tagapp.TagTree `inject:"T"`
}

func (d *Dashbord) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("dashbord", d.Dashbord),
	}

	return req.NewConfs("/dbs", reqs[:]...)
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	tagCodePaths := m.tagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{Types: collx.AsArray(tagentity.TagTypeDb)}).GetCodePaths()

	rc.ResData = collx.M{
		"dbNum": len(tagCodePaths),
	}
}
