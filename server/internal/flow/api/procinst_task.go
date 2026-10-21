package api

import (
	"fmt"
	"mayfly-go/internal/flow/api/form"
	"mayfly-go/internal/flow/api/vo"
	"mayfly-go/internal/flow/application"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/imsg"
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
	instTaskQuery := rc.BindQuery[entity.ProcinstTaskQuery]()
	ctx := rc.MetaCtx
	// 代办任务
	if instTaskQuery.Status == entity.ProcinstTaskStatusProcess {
		candidates, err := p.GetLaCandidates(contextx.GetLoginAccount(ctx))
		biz.ErrIsNil(err)
		instTaskQuery.Candidates = candidates
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
	auditForm := rc.BindJson[form.ProcinstTaskAudit]()
	rc.ReqParam = auditForm

	la := rc.GetLoginAccount()
	op := dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark, Handler: la.Username}
	candidates, err := p.GetLaCandidates(la)
	biz.ErrIsNil(err)
	op.Candidate = candidates
	biz.ErrIsNil(p.procinstTaskApp.PassTask(rc.MetaCtx, op))
}

func (p *ProcinstTask) RejectTask(rc *req.Ctx) {
	auditForm := rc.BindJson[form.ProcinstTaskAudit]()
	rc.ReqParam = auditForm

	la := rc.GetLoginAccount()
	op := dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark, Handler: la.Username}
	candidates, err := p.GetLaCandidates(la)
	biz.ErrIsNil(err)
	op.Candidate = candidates
	biz.ErrIsNil(p.procinstTaskApp.RejectTask(rc.MetaCtx, op))
}

func (p *ProcinstTask) BackTask(rc *req.Ctx) {
	auditForm := rc.BindJson[form.ProcinstTaskAudit]()
	rc.ReqParam = auditForm

	la := rc.GetLoginAccount()
	op := dto.UserTaskOp{TaskId: auditForm.Id, Remark: auditForm.Remark, Handler: la.Username}
	candidates, err := p.GetLaCandidates(la)
	biz.ErrIsNil(err)
	op.Candidate = candidates
	biz.ErrIsNil(p.procinstTaskApp.BackTask(rc.MetaCtx, op))
}

// GetLaCandidates 当前账号可作为办理人的候选人标识（自身 id + 所属角色）。
//
// 角色查询失败必须上抛：降级成「只有自己」会让待办列表少一半、办理时报「非办理人」——
// fail-closed 不越权，但把一次 DB 抖动伪装成权限问题，排查方向会被带偏
func (p *ProcinstTask) GetLaCandidates(la *model.LoginAccount) ([]string, error) {
	candidates := []string{fmt.Sprintf("%d", la.Id)}

	roles, err := p.roleApp.GetAccountRoles(la.Id)
	if err != nil {
		return nil, err
	}
	candidates = append(candidates, collx.ArrayMap(roles, func(val *sysentity.AccountRole) string { return fmt.Sprintf("role:%d", val.RoleId) })...)
	return candidates, nil
}
