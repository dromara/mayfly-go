package api

import (
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/structx"
)

type Procinst struct {
	procinstApp     application.Procinst     `inject:"T"`
	procdefApp      application.Procdef      `inject:"T"`
	procinstTaskApp application.ProcinstTask `inject:"T"`
}

func (p *Procinst) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		req.NewGet("", p.GetProcinstPage),

		req.NewGet("/:id", p.GetProcinstDetail),

		req.NewPost("/start", p.ProcinstStart).Log(req.NewLogSaveI(imsg.LogProcinstStart)),

		req.NewPost("/:id/cancel", p.ProcinstCancel).Log(req.NewLogSaveI(imsg.LogProcinstCancel)),
	}

	return req.NewConfs("/flow/procinsts", reqs[:]...)
}

func (p *Procinst) GetProcinstPage(rc *req.Ctx) {
	cond := req.BindQuery[*entity.ProcinstQuery](rc)
	// 非管理员只能获取自己申请的流程
	if laId := rc.GetLoginAccount().Id; laId != consts.AdminId {
		cond.CreatorId = laId
	}

	res, err := p.procinstApp.GetPageList(cond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (p *Procinst) ProcinstStart(rc *req.Ctx) {
	startForm := req.BindJsonAndValid[*form.ProcinstStart](rc)
	_, err := p.procinstApp.StartProc(rc.MetaCtx, startForm.ProcdefId, &dto.StarProc{
		BizType: startForm.BizType,
		BizForm: jsonx.ToStr(startForm.BizForm),
		Remark:  startForm.Remark,
	})
	biz.ErrIsNil(err)
}

func (p *Procinst) ProcinstCancel(rc *req.Ctx) {
	instId := uint64(rc.PathParamInt("id"))
	rc.ReqParam = instId
	biz.ErrIsNil(p.procinstApp.CancelProc(rc.MetaCtx, instId))
}

func (p *Procinst) GetProcinstDetail(rc *req.Ctx) {
	pi, err := p.procinstApp.GetById(uint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err, "procinst not found")
	pivo := new(vo.ProcinstVO)
	structx.Copy(pivo, pi)

	// 流程定义信息
	procdef, _ := p.procdefApp.GetById(pi.ProcdefId)
	procdef.FlowDef = "" // 流程定义信息不需要返回
	pivo.Procdef = procdef

	// 流程实例任务信息
	taskQuery := &entity.ProcinstTaskQuery{ProcinstId: pi.Id}
	tasks, err := p.procinstTaskApp.GetTasks(rc.MetaCtx, taskQuery)
	biz.ErrIsNil(err)
	pivo.ProcinstTasks = tasks.List

	rc.ResData = pivo
}
