package api

import (
	"fmt"
	"mayfly-go/internal/common/consts"
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/structx"
)

type Procinst struct {
	ProcinstApp application.Procinst `inject:""`
	ProcdefApp  application.Procdef  `inject:""`

	ProcinstTaskRepo repository.ProcinstTask `inject:""`
}

func (p *Procinst) GetProcinstPage(rc *req.Ctx) {
	cond, page := req.BindQueryAndPage(rc, new(entity.ProcinstQuery))
	// 非管理员只能获取自己申请的流程
	if laId := rc.GetLoginAccount().Id; laId != consts.AdminId {
		cond.CreatorId = laId
	}

	res, err := p.ProcinstApp.GetPageList(cond, page, new([]entity.Procinst))
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (p *Procinst) ProcinstStart(rc *req.Ctx) {
	startForm := new(form.ProcinstStart)
	req.BindJsonAndValid(rc, startForm)
	_, err := p.ProcinstApp.StartProc(rc.MetaCtx, startForm.ProcdefId, &dto.StarProc{
		BizType: startForm.BizType,
		BizForm: jsonx.ToStr(startForm.BizForm),
		Remark:  startForm.Remark,
	})
	biz.ErrIsNil(err)
}

func (p *Procinst) ProcinstCancel(rc *req.Ctx) {
	instId := uint64(rc.PathParamInt("id"))
	rc.ReqParam = instId
	biz.ErrIsNil(p.ProcinstApp.CancelProc(rc.MetaCtx, instId))
}

func (p *Procinst) GetProcinstDetail(rc *req.Ctx) {
	pi, err := p.ProcinstApp.GetById(uint64(rc.PathParamInt("id")))
	biz.ErrIsNil(err, "流程实例不存在")
	pivo := new(vo.ProcinstVO)
	structx.Copy(pivo, pi)

	// 流程定义信息
	procdef, _ := p.ProcdefApp.GetById(pi.ProcdefId)
	pivo.Procdef = procdef

	// 流程实例任务信息
	instTasks, err := p.ProcinstTaskRepo.SelectByCond(&entity.ProcinstTask{ProcinstId: pi.Id})
	biz.ErrIsNil(err)
	pivo.ProcinstTasks = instTasks

	rc.ResData = pivo
}

func (p *Procinst) GetTasks(rc *req.Ctx) {
	instTaskQuery, page := req.BindQueryAndPage(rc, new(entity.ProcinstTaskQuery))
	if laId := rc.GetLoginAccount().Id; laId != consts.AdminId {
		// 赋值操作人为当前登录账号
		instTaskQuery.Assignee = fmt.Sprintf("%d", rc.GetLoginAccount().Id)
	}

	taskVos := new([]*vo.ProcinstTask)
	res, err := p.ProcinstApp.GetProcinstTasks(instTaskQuery, page, taskVos)
	biz.ErrIsNil(err)

	instIds := collx.ArrayMap[*vo.ProcinstTask, uint64](*taskVos, func(val *vo.ProcinstTask) uint64 { return val.ProcinstId })
	insts, _ := p.ProcinstApp.GetByIds(instIds)
	instId2Inst := collx.ArrayToMap[*entity.Procinst, uint64](insts, func(val *entity.Procinst) uint64 { return val.Id })

	// 赋值任务对应的流程实例
	for _, task := range *taskVos {
		task.Procinst = instId2Inst[task.ProcinstId]
	}
	rc.ResData = res
}

func (p *Procinst) CompleteTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid(rc, new(form.ProcinstTaskAudit))
	rc.ReqParam = auditForm
	biz.ErrIsNil(p.ProcinstApp.CompleteTask(rc.MetaCtx, auditForm.Id, auditForm.Remark))
}

func (p *Procinst) RejectTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid(rc, new(form.ProcinstTaskAudit))
	rc.ReqParam = auditForm
	biz.ErrIsNil(p.ProcinstApp.RejectTask(rc.MetaCtx, auditForm.Id, auditForm.Remark))
}

func (p *Procinst) BackTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid(rc, new(form.ProcinstTaskAudit))
	rc.ReqParam = auditForm
	biz.ErrIsNil(p.ProcinstApp.BackTask(rc.MetaCtx, auditForm.Id, auditForm.Remark))
}
