package api

import (
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree      `inject:""`
	MachineApp application.Machine `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id
	machienNum := len(m.TagTreeApp.GetAccountResourceCodes(accountId, consts.TagResourceTypeMachine, ""))

	rc.ResData = collx.M{
		"machineNum": machienNum,
	}
}
