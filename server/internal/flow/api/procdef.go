package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"strconv"
	"strings"
)

type Procdef struct {
	ProcdefApp application.Procdef `inject:""`
}

func (p *Procdef) GetProcdefPage(rc *req.Ctx) {
	cond, page := req.BindQueryAndPage(rc, new(entity.Procdef))
	res, err := p.ProcdefApp.GetPageList(cond, page, new([]entity.Procdef))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (p *Procdef) GetProcdef(rc *req.Ctx) {
	defkey := rc.PathParam("key")
	biz.NotEmpty(defkey, "流程定义key不能为空")

	procdef := &entity.Procdef{DefKey: defkey}
	biz.ErrIsNil(p.ProcdefApp.GetByCond(procdef), "该流程定义不存在")
	rc.ResData = procdef
}

func (a *Procdef) Save(rc *req.Ctx) {
	form := &form.Procdef{}
	procdef := req.BindJsonAndCopyTo(rc, form, new(entity.Procdef))
	rc.ReqParam = form
	biz.ErrIsNil(a.ProcdefApp.Save(rc.MetaCtx, procdef))
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
