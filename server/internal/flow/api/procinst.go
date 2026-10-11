package api

import (
	"time"

	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/structx"

	"github.com/spf13/cast"
)

// plannedExecTime 把入参统一时间结构体（前端约定 "2006-01-02 15:04:05" 本地时区）转成业务层的
// *time.Time；nil 或零值表示未设置（= 批准后立即执行）
func plannedExecTime(jt *model.JsonTime) *time.Time {
	if jt == nil || jt.IsZero() {
		return nil
	}
	t := jt.Time
	return &t
}

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

		// 定时执行：到点前允许改期/取消（仅「已批准、待执行」可操作）
		//
		// 两条动作必须各自一个日志文案：共用「流程-取消」后审计列表里两种动作长得一模一样，
		// 既看不出是谁改的期，也看不出改到了什么时候
		req.NewPost("/:id/reschedule-exec", p.ProcinstRescheduleExec).Log(req.NewLogSaveI(imsg.LogProcinstReschedule)),
		req.NewPost("/:id/cancel-exec", p.ProcinstCancelExec).Log(req.NewLogSaveI(imsg.LogProcinstCancel)),
	}

	return req.NewConfs("/flow/procinsts", reqs[:]...)
}

func (p *Procinst) GetProcinstPage(rc *req.Ctx) {
	cond := rc.BindQuery[entity.ProcinstQuery]()
	// 非管理员只能获取自己申请的流程
	if laId := rc.GetLoginAccount().Id; laId != consts.AdminId {
		cond.CreatorId = laId
	}

	res, err := p.procinstApp.GetPageList(cond)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (p *Procinst) ProcinstStart(rc *req.Ctx) {
	startForm := rc.BindJson[form.ProcinstStart]()
	_, err := p.procinstApp.StartProc(rc.MetaCtx, startForm.ProcdefId, &dto.StarProc{
		BizType:         startForm.BizType,
		BizKey:          startForm.BizKey,
		BizForm:         jsonx.ToStr(startForm.BizForm),
		Remark:          startForm.Remark,
		PlannedExecTime: plannedExecTime(startForm.PlannedExecTime),
	})
	biz.ErrIsNil(err)
}

func (p *Procinst) ProcinstCancel(rc *req.Ctx) {
	instId := cast.ToUint64(rc.PathParam("id"))
	rc.ReqParam = instId
	biz.ErrIsNil(p.procinstApp.CancelProc(rc.MetaCtx, instId))
}

// ProcinstRescheduleExec 改期：修改已批准但未执行工单的计划执行时间
func (p *Procinst) ProcinstRescheduleExec(rc *req.Ctx) {
	reschedForm := rc.BindJson[form.ProcinstReschedule]()
	instId := cast.ToUint64(rc.PathParam("id"))
	at := plannedExecTime(reschedForm.PlannedExecTime)
	// 目标时间一并入日志：`null` 即「改成立即执行」（与 plannedExecTime 的 nil 语义同源），
	// 只记实例 id 就无法事后回答「把它改到了什么时候」
	rc.ReqParam = collx.Kvs("id", instId, "plannedExecTime", at)
	biz.ErrIsNil(p.procinstApp.RescheduleBizHandle(rc.MetaCtx, instId, at))
}

// ProcinstCancelExec 取消执行：已批准但未执行的工单不再回放业务
func (p *Procinst) ProcinstCancelExec(rc *req.Ctx) {
	instId := cast.ToUint64(rc.PathParam("id"))
	rc.ReqParam = instId
	biz.ErrIsNil(p.procinstApp.CancelBizHandle(rc.MetaCtx, instId))
}

func (p *Procinst) GetProcinstDetail(rc *req.Ctx) {
	pi, err := p.procinstApp.GetById(cast.ToUint64(rc.PathParam("id")))
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
