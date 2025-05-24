package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	msgapp "mayfly-go/internal/msg/application"
	msgentity "mayfly-go/internal/msg/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/structx"
	"strings"

	"github.com/may-fly/cast"
)

type Procdef struct {
	procdefApp       application.Procdef  `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
	msgTmplBizApp    msgapp.MsgTmplBiz    `inject:"T"`
}

func (p *Procdef) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", p.GetProcdefPage),

		req.NewGet("/detail/:id", p.GetProcdefDetail),

		req.NewGet("/:resourceType/:resourceCode", p.GetProcdef),

		req.NewPost("", p.Save).Log(req.NewLogSaveI(imsg.LogProcdefSave)).RequiredPermissionCode("flow:procdef:save"),

		req.NewPost("/flowdef", p.SaveFlowDef).Log(req.NewLogSaveI(imsg.LogProcdefSave)).RequiredPermissionCode("flow:procdef:save"),

		req.NewGet("/flowdef/:id", p.GetFlowDef),

		req.NewDelete(":id", p.Delete).Log(req.NewLogSaveI(imsg.LogProcdefDelete)).RequiredPermissionCode("flow:procdef:del"),
	}

	return req.NewConfs("/flow/procdefs", reqs[:]...)
}

func (p *Procdef) GetProcdefPage(rc *req.Ctx) {
	cond, page := req.BindQueryAndPage[*entity.Procdef](rc)

	res, err := p.procdefApp.GetPageList(cond, page)
	biz.ErrIsNil(err)
	resVo := model.PageResultConv[*entity.ProcdefPagePO, *vo.Procdef](res)

	p.tagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeFlowDef, collx.ArrayMap(resVo.List, func(mvo *vo.Procdef) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = resVo
}

func (p *Procdef) GetProcdefDetail(rc *req.Ctx) {
	def, err := p.procdefApp.GetById(cast.ToUint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err)
	res := new(vo.Procdef)
	biz.ErrIsNil(structx.Copy(res, def))

	p.tagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeFlowDef, res)

	bizMsgTmpl := &msgentity.MsgTmplBiz{
		BizId:   res.Id,
		BizType: application.FlowTaskNotifyBizKey,
	}
	if p.msgTmplBizApp.GetByCond(bizMsgTmpl) == nil {
		res.MsgTmplId = &bizMsgTmpl.TmplId
	}

	rc.ResData = res
}

func (p *Procdef) GetProcdef(rc *req.Ctx) {
	resourceType := rc.PathParamInt("resourceType")
	resourceCode := rc.PathParam("resourceCode")
	rc.ResData = p.procdefApp.GetProcdefByResource(rc.MetaCtx, int8(resourceType), resourceCode)
}

func (a *Procdef) Save(rc *req.Ctx) {
	form, procdef := req.BindJsonAndCopyTo[*form.Procdef, *entity.Procdef](rc)
	rc.ReqParam = form
	biz.ErrIsNil(a.procdefApp.SaveProcdef(rc.MetaCtx, &dto.SaveProcdef{
		Procdef:   procdef,
		MsgTmplId: form.MsgTmplId,
		CodePaths: form.CodePaths,
	}))
}

func (a *Procdef) SaveFlowDef(rc *req.Ctx) {
	form := req.BindJsonAndValid[*form.ProcdefFlow](rc)
	rc.ReqParam = form

	biz.ErrIsNil(a.procdefApp.SaveFlowDef(rc.MetaCtx, &dto.SaveFlowDef{
		Id:      form.Id,
		FlowDef: form.Flow,
	}))
}

func (a *Procdef) GetFlowDef(rc *req.Ctx) {
	defId := rc.PathParamInt("id")
	procdef, err := a.procdefApp.GetById(uint64(defId))
	biz.ErrIsNil(err)
	rc.ResData = procdef.GetFlowDef()
}

func (p *Procdef) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNilAppendErr(p.procdefApp.DeleteProcdef(rc.MetaCtx, cast.ToUint64(v)), "delete error: %s")
	}
}
