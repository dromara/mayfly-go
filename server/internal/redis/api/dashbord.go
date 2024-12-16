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
		req.NewGet("/redis/dashbord", d.Dashbord),
	}

	return req.NewConfs("", reqs[:]...)
}

func (d *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	redisNum := len(d.tagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{
		Types: collx.AsArray(tagentity.TagTypeRedis),
	}))

	rc.ResData = collx.M{
		"redisNum": redisNum,
	}
}
