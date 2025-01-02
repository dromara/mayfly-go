package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strings"

	"github.com/may-fly/cast"
)

type Procdef struct {
	procdefApp       application.Procdef  `inject:"T"`
	tagTreeRelateApp tagapp.TagTreeRelate `inject:"T"`
}

func (p *Procdef) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", p.GetProcdefPage),

		req.NewGet("/:resourceType/:resourceCode", p.GetProcdef),

		req.NewPost("", p.Save).Log(req.NewLogSaveI(imsg.LogProcdefSave)).RequiredPermissionCode("flow:procdef:save"),

		req.NewDelete(":id", p.Delete).Log(req.NewLogSaveI(imsg.LogProcdefDelete)).RequiredPermissionCode("flow:procdef:del"),
	}

	return req.NewConfs("/flow/procdefs", reqs[:]...)
}

func (p *Procdef) GetProcdefPage(rc *req.Ctx) {
	cond, page := req.BindQueryAndPage(rc, new(entity.Procdef))
	var procdefs []*vo.Procdef
	res, err := p.procdefApp.GetPageList(cond, page, &procdefs)
	biz.ErrIsNil(err)

	p.tagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeFlowDef, collx.ArrayMap(procdefs, func(mvo *vo.Procdef) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = res
}

func (p *Procdef) GetProcdef(rc *req.Ctx) {
	resourceType := rc.PathParamInt("resourceType")
	resourceCode := rc.PathParam("resourceCode")
	rc.ResData = p.procdefApp.GetProcdefByResource(rc.MetaCtx, int8(resourceType), resourceCode)
}

func (a *Procdef) Save(rc *req.Ctx) {
	form := &form.Procdef{}
	procdef := req.BindJsonAndCopyTo(rc, form, new(entity.Procdef))
	rc.ReqParam = form
	biz.ErrIsNil(a.procdefApp.SaveProcdef(rc.MetaCtx, &dto.SaveProcdef{
		Procdef:   procdef,
		CodePaths: form.CodePaths,
	}))
}

func (p *Procdef) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		biz.ErrIsNilAppendErr(p.procdefApp.DeleteProcdef(rc.MetaCtx, cast.ToUint64(v)), "delete error: %s")
	}
}
