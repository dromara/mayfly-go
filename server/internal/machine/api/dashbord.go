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
		req.NewGet("/machines/dashbord", d.Dashbord),
	}

	return req.NewConfs("", reqs[:]...)
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	tagCodePaths := m.tagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{TypePaths: collx.AsArray(tagentity.NewTypePaths(tagentity.TagTypeMachine, tagentity.TagTypeAuthCert))}).GetCodePaths()
	machineCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeMachine, tagCodePaths...)

	rc.ResData = collx.M{
		"machineNum": len(machineCodes),
	}
}
