package api

import (
	"mayfly-go/internal/machine/api/form"
	"mayfly-go/internal/machine/api/vo"
	"mayfly-go/internal/machine/application"
	"mayfly-go/internal/machine/application/dto"
	"mayfly-go/internal/machine/domain/entity"

	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type MachineCmdConf struct {
	MachineCmdConfApp application.MachineCmdConf `inject:""`
	TagTreeRelateApp  tagapp.TagTreeRelate       `inject:"TagTreeRelateApp"`
}

func (m *MachineCmdConf) MachineCmdConfs(rc *req.Ctx) {
	cond := req.BindQuery(rc, new(entity.MachineCmdConf))

	var vos []*vo.MachineCmdConfVO
	err := m.MachineCmdConfApp.ListByCondToAny(cond, &vos)
	biz.ErrIsNil(err)

	m.TagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeMachineCmd, collx.ArrayMap(vos, func(mvo *vo.MachineCmdConfVO) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = vos
}

func (m *MachineCmdConf) Save(rc *req.Ctx) {
	cmdForm := new(form.MachineCmdConfForm)
	mcj := req.BindJsonAndCopyTo[*entity.MachineCmdConf](rc, cmdForm, new(entity.MachineCmdConf))
	rc.ReqParam = cmdForm

	err := m.MachineCmdConfApp.SaveCmdConf(rc.MetaCtx, &dto.SaveMachineCmdConf{
		CmdConf:   mcj,
		CodePaths: cmdForm.CodePaths,
	})
	biz.ErrIsNil(err)
}

func (m *MachineCmdConf) Delete(rc *req.Ctx) {
	m.MachineCmdConfApp.DeleteCmdConf(rc.MetaCtx, uint64(rc.PathParamInt("id")))
}
