package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"
	"mayfly-go/internal/machine/imsg"

	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type MachineCmdConf struct {
	machineCmdConfApp application.MachineCmdConf `inject:"T"`
	tagTreeRelateApp  tagapp.TagTreeRelate       `inject:"T"`
}

func (mcc *MachineCmdConf) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", mcc.MachineCmdConfs),

		req.NewPost("", mcc.Save).Log(req.NewLogSaveI(imsg.LogMachineSecurityCmdSave)).RequiredPermissionCode("cmdconf:save"),

		req.NewDelete(":id", mcc.Delete).Log(req.NewLogSaveI(imsg.LogMachineSecurityCmdDelete)).RequiredPermissionCode("cmdconf:del"),
	}

	return req.NewConfs("machine/security/cmd-confs", reqs[:]...)
}

func (m *MachineCmdConf) MachineCmdConfs(rc *req.Ctx) {
	cond := req.BindQuery[*entity.MachineCmdConf](rc)

	var vos []*vo.MachineCmdConfVO
	err := m.machineCmdConfApp.ListByCondToAny(cond, &vos)
	biz.ErrIsNil(err)

	m.tagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeMachineCmd, collx.ArrayMap(vos, func(mvo *vo.MachineCmdConfVO) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = vos
}

func (m *MachineCmdConf) Save(rc *req.Ctx) {
	cmdForm, mcj := req.BindJsonAndCopyTo[*form.MachineCmdConfForm, *entity.MachineCmdConf](rc)
	rc.ReqParam = cmdForm

	err := m.machineCmdConfApp.SaveCmdConf(rc.MetaCtx, &dto.SaveMachineCmdConf{
		CmdConf:   mcj,
		CodePaths: cmdForm.CodePaths,
	})
	biz.ErrIsNil(err)
}

func (m *MachineCmdConf) Delete(rc *req.Ctx) {
	m.machineCmdConfApp.DeleteCmdConf(rc.MetaCtx, uint64(rc.PathParamInt("id")))
}
