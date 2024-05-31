package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	tagapp "mayfly-go/internal/tag/application"
	tagentity "mayfly-go/internal/tag/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"strconv"
	"strings"
)

type Procdef struct {
	ProcdefApp       application.Procdef  `inject:""`
	TagTreeRelateApp tagapp.TagTreeRelate `inject:"TagTreeRelateApp"`
}

func (p *Procdef) GetProcdefPage(rc *req.Ctx) {
	cond, page := req.BindQueryAndPage(rc, new(entity.Procdef))
	var procdefs []*vo.Procdef
	res, err := p.ProcdefApp.GetPageList(cond, page, &procdefs)
	biz.ErrIsNil(err)

	p.TagTreeRelateApp.FillTagInfo(tagentity.TagRelateTypeFlowDef, collx.ArrayMap(procdefs, func(mvo *vo.Procdef) tagentity.IRelateTag {
		return mvo
	})...)

	rc.ResData = res
}

func (p *Procdef) GetProcdef(rc *req.Ctx) {
	resourceType := rc.PathParamInt("resourceType")
	resourceCode := rc.PathParam("resourceCode")
	rc.ResData = p.ProcdefApp.GetProcdefByResource(rc.MetaCtx, int8(resourceType), resourceCode)
}

func (a *Procdef) Save(rc *req.Ctx) {
	form := &form.Procdef{}
	procdef := req.BindJsonAndCopyTo(rc, form, new(entity.Procdef))
	rc.ReqParam = form
	biz.ErrIsNil(a.ProcdefApp.SaveProcdef(rc.MetaCtx, &dto.SaveProcdef{
		Procdef:   procdef,
		CodePaths: form.CodePaths,
	}))
}

func (p *Procdef) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("id")
	rc.ReqParam = idsStr
	ids := strings.Split(idsStr, ",")

	for _, v := range ids {
		value, err := strconv.Atoi(v)
		biz.ErrIsNilAppendErr(err, "string类型转换为int异常: %s")
		biz.ErrIsNilAppendErr(p.ProcdefApp.DeleteProcdef(rc.MetaCtx, uint64(value)), "删除失败：%s")
	}
}
