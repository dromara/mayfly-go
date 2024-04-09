package api

import (
	"mayfly-go/internal/machine/application"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type Dashbord struct {
	ResourceAuthCertApp tagapp.ResourceAuthCert `inject:""`
	MachineApp          application.Machine     `inject:""`
}

func (m *Dashbord) Dashbord(rc *req.Ctx) {
	accountId := rc.GetLoginAccount().Id

	machienAuthCerts := m.ResourceAuthCertApp.GetAccountAuthCert(accountId, tagentity.TagTypeMachineAuthCert)
	machineCodes := collx.ArrayMap(machienAuthCerts, func(ac *tagentity.ResourceAuthCert) string {
		return ac.ResourceCode
	})

	machienNum := len(collx.ArrayDeduplicate(machineCodes))

	rc.ResData = collx.M{
		"machineNum": machienNum,
	}
}
