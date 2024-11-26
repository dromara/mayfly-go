package api

import (
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	"mayfly-go/internal/tag/domain/entity"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	TagTreeApp tagapp.TagTree      `inject:""`
	MachineApp application.Machine `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	tagCodePaths := m.TagTreeApp.GetAccountTags(accountId, &tagentity.TagTreeQuery{TypePaths: collx.AsArray(entity.NewTypePaths(tagentity.TagTypeMachine, tagentity.TagTypeAuthCert))}).GetCodePaths()
	machineCodes := tagentity.GetCodesByCodePaths(tagentity.TagTypeMachine, tagCodePaths...)

	rc.ResData = collx.M{
		"machineNum": len(machineCodes),
	}
}
