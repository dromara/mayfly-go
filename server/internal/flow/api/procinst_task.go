package api

import (
	"fmt"
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	sysapp "mayfly-go/internal/sys/application"
	sysentity "mayfly-go/internal/sys/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/utils/collx"
)

type ProcinstTask struct {
	procinstApp     application.Procinst     `inject:"T"`
	procinstTaskApp application.ProcinstTask `inject:"T"`

	roleApp sysapp.Role `inject:"T"`
}

func (p *ProcinstTask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{

		req.NewGet("/tasks", p.GetTasks),

		req.NewPost("/tasks/pass", p.PassTask).Log(req.NewLogSaveI(imsg.LogCompleteTask)),

		req.NewPost("/tasks/reject", p.RejectTask).Log(req.NewLogSaveI(imsg.LogRejectTask)),

		req.NewPost("/tasks/back", p.BackTask).Log(req.NewLogSaveI(imsg.LogBackTask)),
	}

	return req.NewConfs("/flow/procinsts", reqs[:]...)
}

func (p *ProcinstTask) GetTasks(rc *req.Ctx) {
	instTaskQuery := req.BindQuery[*entity.ProcinstTaskQuery](rc)
	if laId := rc.GetLoginAccount().Id; laId != consts.AdminId {
		// 赋值操作人为当前登录账号
		instTaskQuery.Assignee = fmt.Sprintf("%d", rc.GetLoginAccount().Id)
	}
	ctx := rc.MetaCtx
	// 代办任务
	if instTaskQuery.Status == entity.ProcinstTaskStatusProcess {
		instTaskQuery.Candidates = p.GetLaCandidates(contextx.GetLoginAccount(ctx))
	} else {
		// 我的任务
		instTaskQuery.Handler = contextx.GetLoginAccount(ctx).Username
	}

	res, err := p.procinstTaskApp.GetTasks(rc.MetaCtx, instTaskQuery)
	biz.ErrIsNil(err)

	resVos := model.PageResultConv[*entity.ProcinstTaskPO, *vo.ProcinstTask](res)
	taskVos := resVos.List

	instIds := collx.ArrayMap[*vo.ProcinstTask, uint64](taskVos, func(val *vo.ProcinstTask) uint64 { return val.ProcinstId })
	insts, _ := p.procinstApp.GetByIds(instIds)
	instId2Inst := collx.ArrayToMap[*entity.Procinst, uint64](insts, func(val *entity.Procinst) uint64 { return val.Id })

	// 赋值任务对应的流程实例
	for _, task := range taskVos {
		task.Procinst = instId2Inst[task.ProcinstId]
	}

	rc.ResData = resVos
}

func (p *ProcinstTask) PassTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid[*form.ProcinstTaskAudit](rc)
	rc.ReqParam = auditForm

	la := rc.GetLoginAccount()
	op := dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark, Handler: la.Username}
	op.Candidate = p.GetLaCandidates(la)
	biz.ErrIsNil(p.procinstTaskApp.PassTask(rc.MetaCtx, op))
}

func (p *ProcinstTask) RejectTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid[*form.ProcinstTaskAudit](rc)
	rc.ReqParam = auditForm

	la := rc.GetLoginAccount()
	op := dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark, Handler: la.Username}
	op.Candidate = p.GetLaCandidates(la)
	biz.ErrIsNil(p.procinstTaskApp.RejectTask(rc.MetaCtx, op))
}

func (p *ProcinstTask) BackTask(rc *req.Ctx) {
	auditForm := req.BindJsonAndValid[*form.ProcinstTaskAudit](rc)
	rc.ReqParam = auditForm
	biz.ErrIsNil(p.procinstTaskApp.BackTask(rc.MetaCtx, dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark}))
}

func (p *ProcinstTask) GetLaCandidates(la *model.LoginAccount) []string {
	candidates := []string{fmt.Sprintf("%d", la.Id)}

	roles, err := p.roleApp.GetAccountRoles(la.Id)
	if err == nil {
		candidates = append(candidates, collx.ArrayMap(roles, func(val *sysentity.AccountRole) string { return fmt.Sprintf("role:%d", val.RoleId) })...)
	}

	return candidates
}
