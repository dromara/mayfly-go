package api

import (
	"strings"

	"mayfly-go/internal/agentops/api/form"
	"mayfly-go/internal/agentops/api/vo"
	"mayfly-go/internal/agentops/application"
	"mayfly-go/internal/agentops/domain/entity"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
	"mayfly-go/pkg/scheduler"

	"github.com/spf13/cast"
	"mayfly-go/internal/agentops/imsg"
)

// AgentTask 定时/计划触发的 Agent 任务 API：任务 CRUD + 手动触发。
//
// 触发后的实际运行由 RunService 有界工作池异步执行，运行记录见 Run API（/agentops/runs）。
type AgentTask struct {
	agentTaskApp application.AgentTaskService `inject:"T"`
	// accountApp 用于把表单指定的 ownerAccountId 解析为账号名（运行身份展示/审计）。
	// 未指定 owner 时以当前登录账号为运行身份，无需查库（判定均在 resolveRunOwner 内）
	accountApp sysapp.Account `inject:"T"`
}

func (a *AgentTask) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 任务分页列表
		req.NewGet("", a.GetPageList),

		// 任务详情（编辑回填）
		req.NewGet("/:id", a.GetById),

		// 保存任务（新建/更新）
		req.NewPost("", a.Save).Log(req.NewLogSaveI(imsg.LogTaskSave)).RequiredPermissionCode("agentops:task:save"),

		// 删除任务（批量，逗号分隔）
		req.NewDelete(":ids", a.Delete).Log(req.NewLogSaveI(imsg.LogTaskDelete)).RequiredPermissionCode("agentops:task:del"),

		// 手动触发一次（trigger=manual）
		req.NewPost("/run/:id", a.RunOnce).Log(req.NewLogSaveI(imsg.LogTaskRun)).RequiredPermissionCode("agentops:task:run"),
	}

	return req.NewConfs("/agentops/tasks", reqs[:]...)
}

// GetPageList 分页查询任务，并回填「cron 是否已在调度器注册」的运行态
func (a *AgentTask) GetPageList(rc *req.Ctx) {
	cond, pageParam := rc.BindQueryAndPage[entity.AgentTask]()
	// 列表同样收窄：否则非全局可见者能读到他人任务的完整指令与调度配置
	// （指令里常含机器名/库名/业务口径，属于真实信息泄露面）；写侧的归属断言见 Save 更新分支
	scopeOwnerToCaller(rc, func(accountId uint64) { cond.OwnerAccountId = accountId })

	pageRes, err := a.agentTaskApp.GetPageList(cond, pageParam, "id desc")
	biz.ErrIsNil(err)

	resVo := model.PageResultConv[*entity.AgentTask, *vo.AgentTaskVO](pageRes)
	for _, t := range resVo.List {
		t.Running = scheduler.ExistKey(t.Key)
	}
	rc.ResData = resVo
}

// GetById 任务详情（含完整 instruction，供编辑表单回填）
func (a *AgentTask) GetById(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	task, err := a.agentTaskApp.GetById(id)
	biz.ErrIsNil(err)
	biz.ErrIsNil(assertResourceOwned(rc, task.OwnerAccountId))

	resVo := model.PageResultConv[*entity.AgentTask, *vo.AgentTaskVO](&model.PageResult[*entity.AgentTask]{List: []*entity.AgentTask{task}})
	voItem := resVo.List[0]
	voItem.Running = scheduler.ExistKey(task.Key)
	rc.ResData = voItem
}

// Save 保存任务：运行身份经 resolveRunOwner 解析（缺省当前登录账号，指定他人需超管或专门权限点）
func (a *AgentTask) Save(rc *req.Ctx) {
	taskForm := rc.BindJson[form.AgentTaskForm]()
	rc.ReqParam = taskForm

	owner, err := resolveRunOwner(rc, a.accountApp, taskForm.OwnerAccountId)
	biz.ErrIsNil(err)

	task := &entity.AgentTask{
		Name:        taskForm.Name,
		Instruction: taskForm.Instruction,
		// 芯片段随指令一并落库（Extra），派发时展开为模型可直接取参的定位信息
		InstructionSegments: taskForm.InstructionSegments,
		Cron:                taskForm.Cron,
		ProfileId:           taskForm.ProfileId,
		OwnerAccountId:      owner.AccountId,
		OwnerUsername:       owner.Username,
		Status:              taskForm.Status,
		TimeoutSeconds:      taskForm.TimeoutSeconds,
		RetryTimes:          taskForm.RetryTimes,
		MaxStep:             taskForm.MaxStep,
		TokenBudget:         taskForm.TokenBudget,
		DryRun:              taskForm.DryRun,
		NotifyType:          taskForm.NotifyType,
		NotifyTmplCode:      taskForm.NotifyTmplCode,
		MisfirePolicy:       taskForm.MisfirePolicy,
		Remark:              taskForm.Remark,
	}
	task.Id = taskForm.Id

	// 更新分支必须断言归属（与 GetById/Delete/RunOnce 同一口径）：此前写侧无闸，
	// 知道他人任务 id 即可整列覆盖其指令/档位/cron 并把 owner 改成自己（TaskEditableColumns
	// 含 owner_account_id，而 resolveRunOwner 只拦「直接指定他人为 owner」，拦不住这个）
	if task.Id != 0 {
		old, gerr := a.agentTaskApp.GetById(task.Id)
		biz.ErrIsNil(gerr)
		biz.ErrIsNil(assertResourceOwned(rc, old.OwnerAccountId))
	}

	biz.ErrIsNil(a.agentTaskApp.SaveTask(rc.MetaCtx, task))
	rc.ResData = task.Id
}

// Delete 批量删除任务（逗号分隔 id），逐个删除并解绑 cron
func (a *AgentTask) Delete(rc *req.Ctx) {
	idsStr := rc.PathParam("ids")
	rc.ReqParam = idsStr
	for _, id := range strings.Split(idsStr, ",") {
		// 跳过空段：批量删除末尾空串会变成 id=0 的无意义查询
		if id == "" {
			continue
		}
		taskId := cast.ToUint64(id)
		// 先取后断言：归属不明时不得静默删掉（app.Delete 本身不认调用者）
		task, err := a.agentTaskApp.GetById(taskId)
		biz.ErrIsNil(err)
		biz.ErrIsNil(assertResourceOwned(rc, task.OwnerAccountId))
		biz.ErrIsNil(a.agentTaskApp.Delete(rc.MetaCtx, taskId))
	}
}

// RunOnce 手动触发一次运行（trigger=manual），返回入队运行的编号供前端跳转执行历史
func (a *AgentTask) RunOnce(rc *req.Ctx) {
	id := cast.ToUint64(rc.PathParam("id"))
	rc.ReqParam = id

	// 越权派发是本案最危险的入口：dispatchTask 会以**任务 owner** 的身份入队运行，
	// 只持有 task:run 权限码就能猜 id 以 admin 身份读监控数据、并登记 owner=admin 的变更提案
	task, err := a.agentTaskApp.GetById(id)
	biz.ErrIsNil(err)
	biz.ErrIsNil(assertResourceOwned(rc, task.OwnerAccountId))

	run, err := a.agentTaskApp.RunOnce(rc.MetaCtx, id)
	biz.ErrIsNil(err)
	rc.ResData = map[string]any{"runId": run.Id, "runNo": run.RunNo}
}

var _ req.RouterApi = (*AgentTask)(nil)
