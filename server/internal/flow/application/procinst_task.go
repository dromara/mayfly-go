package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/model"

	"github.com/spf13/cast"
)

type ProcinstTask interface {
	base.App[*entity.ProcinstTask]

	Init()

	// 获取代办任务
	GetTasks(ctx context.Context, condition *entity.ProcinstTaskQuery, orderBy ...string) (*model.PageResult[*entity.ProcinstTaskPO], error)

	// 任务审批通过
	PassTask(ctx context.Context, taskOp dto.UserTaskOp) error

	// 拒绝任务
	RejectTask(ctx context.Context, taskOp dto.UserTaskOp) error

	// 驳回任务（允许重新提交）
	BackTask(ctx context.Context, taskOp dto.UserTaskOp) error
}

type procinstTaskAppImpl struct {
	base.AppImpl[*entity.ProcinstTask, repository.ProcinstTask]

	procinstApp               Procinst                         `inject:"T"`
	executionApp              Execution                        `inject:"T"`
	procinstTaskCandidateRepo repository.ProcinstTaskCandidate `inject:"T"`
}

var _ ProcinstTask = (*procinstTaskAppImpl)(nil)

var _ (ProcinstTask) = (*procinstTaskAppImpl)(nil)

func (p *procinstTaskAppImpl) Init() {
	const subId = "ProcinstTaskApp"

	flowEventBus.Subscribe(EventTopicFlowProcinstCancel, subId, func(ctx context.Context, event *eventbus.Event[any]) error {
		procinstId := event.Val.(uint64)
		if err := p.UpdateByCond(ctx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusCanceled}, &entity.ProcinstTask{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess}); err != nil {
			return err
		}

		return p.procinstTaskCandidateRepo.DeleteByCond(ctx, &entity.ProcinstTaskCandidate{ProcinstId: procinstId})
	})

}

func (p *procinstTaskAppImpl) GetTasks(ctx context.Context, condition *entity.ProcinstTaskQuery, orderBy ...string) (*model.PageResult[*entity.ProcinstTaskPO], error) {
	return p.Repo.GetPageList(condition, orderBy...)
}

func (p *procinstTaskAppImpl) PassTask(ctx context.Context, taskOp dto.UserTaskOp) error {
	return p.CompleteTask(ctx, taskOp)
}

func (p *procinstTaskAppImpl) RejectTask(ctx context.Context, taskOp dto.UserTaskOp) error {
	taskId := taskOp.TaskId
	instTask, taskCandidates, procinst, execution, err := p.getAndValidInstTask(ctx, taskId, taskOp.Candidate)
	if err != nil {
		return err
	}

	// 赋值状态和备注
	instTask.Status = entity.ProcinstTaskStatusReject
	instTask.Remark = taskOp.Remark
	instTask.SetEnd()

	// 更新流程实例为终止状态，无法重新提交
	procinst.Status = entity.ProcinstStatusTerminated
	procinst.BizStatus = entity.ProcinstBizStatusNo
	procinst.SetEnd()

	procinstId := procinst.Id

	return p.Tx(ctx, func(ctx context.Context) error {
		executionCtx := NewExecutionCtx(ctx, procinst, execution)
		executionCtx.OpExtra.Set("approvalResult", instTask.Status)

		for _, taskCandidate := range taskCandidates {
			taskCandidate.Status = entity.ProcinstTaskStatusReject
			taskCandidate.SetEnd()
			taskCandidate.Handler = &taskOp.Handler
			// 同 CompleteTask：拒绝意见也属于本次操作的候选人，不能只留在任务级共享字段上
			taskCandidate.Remark = taskOp.Remark
			if err := p.procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate); err != nil {
				return err
			}
		}

		if err := p.procinstApp.Save(ctx, procinst); err != nil {
			return err
		}
		if err := p.completeInstTask(ctx, instTask); err != nil {
			return err
		}
		if err := p.UpdateByCond(ctx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusCanceled}, &entity.ProcinstTask{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess}); err != nil {
			return err
		}
		// 跳转至结束节点
		if err := p.executionApp.MoveTo(executionCtx, executionCtx.GetFlowDef().GetNodeByType(FlowNodeTypeEnd)[0]); err != nil {
			return err
		}

		// 删除待处理的其他候选人任务
		return p.procinstTaskCandidateRepo.DeleteByCond(ctx, &entity.ProcinstTaskCandidate{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess})
	})
}

func (p *procinstTaskAppImpl) BackTask(ctx context.Context, taskOp dto.UserTaskOp) error {
	taskId := taskOp.TaskId
	instTask, taskCandidates, procinst, execution, err := p.getAndValidInstTask(ctx, taskId, taskOp.Candidate)
	if err != nil {
		return err
	}

	// 赋值状态和备注
	instTask.Status = entity.ProcinstTaskStatusBack
	instTask.Remark = taskOp.Remark

	// 更新流程实例为退回状态，支持重新提交
	procinst.Status = entity.ProcinstStatusBack

	// 执行流挂起
	execution.State = entity.ExectionStateSuspended

	procinstId := procinst.Id

	return p.Tx(ctx, func(ctx context.Context) error {
		executionCtx := NewExecutionCtx(ctx, procinst, execution)
		executionCtx.OpExtra.Set("approvalResult", instTask.Status)

		for _, taskCandidate := range taskCandidates {
			taskCandidate.Status = entity.ProcinstTaskStatusBack
			taskCandidate.SetEnd()
			taskCandidate.Handler = &taskOp.Handler
			// 同 CompleteTask：退回意见属于本次操作的候选人
			taskCandidate.Remark = taskOp.Remark
			if err := p.procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate); err != nil {
				return err
			}
		}

		if err := p.procinstApp.Save(ctx, procinst); err != nil {
			return err
		}
		if err := p.completeInstTask(ctx, instTask); err != nil {
			return err
		}
		if err := p.UpdateByCond(ctx, &entity.ProcinstTask{Status: entity.ProcinstTaskStatusCanceled}, &entity.ProcinstTask{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess}); err != nil {
			return err
		}
		// 跳转至开始节点
		if err := p.executionApp.MoveTo(executionCtx, executionCtx.GetFlowDef().GetNodeByType(FlowNodeTypeStart)[0]); err != nil {
			return err
		}

		// 删除待处理的其他候选人任务
		return p.procinstTaskCandidateRepo.DeleteByCond(ctx, &entity.ProcinstTaskCandidate{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess})
	})
}

func (p *procinstTaskAppImpl) CompleteTask(ctx context.Context, taskOp dto.UserTaskOp) error {
	taskId := taskOp.TaskId
	instTask, taskCandidates, procinst, execution, err := p.getAndValidInstTask(ctx, taskId, taskOp.Candidate)
	if err != nil {
		return err
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		executionCtx := NewExecutionCtx(ctx, procinst, execution)
		usertaskNode, err := ToUserTaskNode(executionCtx.GetFlowNode())
		if err != nil {
			return err
		}

		for _, taskCandidate := range taskCandidates {
			taskCandidate.Status = entity.ProcinstTaskStatusCompleted
			taskCandidate.SetEnd()
			taskCandidate.Handler = &taskOp.Handler
			// 审批意见落在自己的候选人行上：任务级 remark 是整节点共享的，
			// 会签时后一个审批人会把前一个人的意见覆盖掉
			taskCandidate.Remark = taskOp.Remark
			if err := p.procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate); err != nil {
				return err
			}
		}

		executionCtx.parent = ctx

		vars := instTask.Vars
		// 审批结果必须在完成条件求值之前就可读：它原先只在推进执行流时写进 OpExtra，
		// 于是把「审批结果=通过」配成完成条件会永远判不出来 —— 节点静默卡在待审批，不报错也不提示
		// （下面 OpExtra 那份供连线条件与操作留痕使用，两者用途不同）
		vars.Set(flowFieldApprovalResult, ApprovalResultCompleted)
		// map[string]any整数会被解析为float64，故统一转为float64
		nrOfCompleted := cast.ToFloat64(vars.GetInt(NrOfCompleted) + len(taskCandidates))
		vars.Set(NrOfCompleted, nrOfCompleted)
		// 会签计数随执行流带走：后续连线条件也要能按「几人已通过」分流，
		// 这些计数原本只存在于任务变量里，不带过去就只能读到审批结果一个事实
		executionCtx.ExecutionVars.Set(NrOfAll, vars.GetInt(NrOfAll))
		executionCtx.ExecutionVars.Set(NrOfCompleted, nrOfCompleted)

		// 完成条件判不出来时必须报错中断：按「未完成」会把已通过的审批静默吞掉，
		// 按「已完成」又等于绕过剩下的审批人
		isComplete, err := IsUserTaskComplete(ctx, usertaskNode.CompletionCondition, vars)
		if err != nil {
			return err
		}
		// 不满足通过条件则保存更新任务完成数等变量即可
		if !isComplete {
			return p.Save(ctx, instTask)
		}

		// 赋值状态和备注
		instTask.Status = entity.ProcinstTaskStatusCompleted
		instTask.Remark = taskOp.Remark
		instTask.SetEnd()
		if err := p.completeInstTask(ctx, instTask); err != nil {
			return err
		}

		// 删除待处理的任务处理候选人
		if err := p.procinstTaskCandidateRepo.DeleteByCond(ctx, &entity.ProcinstTaskCandidate{TaskId: taskId, Status: entity.ProcinstTaskStatusProcess}); err != nil {
			return err
		}

		executionCtx.OpExtra.Set("approvalResult", instTask.Status)
		// 继续推进执行流
		return p.executionApp.ContinueExecution(executionCtx)
	})
}

// getAndValidInstTask 获取并校验实例任务
func (p *procinstTaskAppImpl) getAndValidInstTask(ctx context.Context, instTaskId uint64, candidates []string) (*entity.ProcinstTask, []*entity.ProcinstTaskCandidate, *entity.Procinst, *entity.Execution, error) {
	instTask, err := p.GetById(instTaskId)
	if err != nil {
		return nil, nil, nil, nil, errorx.NewBiz("procinst task not found")
	}

	taskCandidates, err := p.procinstTaskCandidateRepo.SelectByCond(model.NewCond().
		In("candidate", candidates).
		Eq("task_id", instTask.Id).
		Eq("status", entity.ProcinstTaskStatusProcess))
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if len(taskCandidates) == 0 {
		return nil, nil, nil, nil, errorx.NewBiz("the current candidates is not a task handler and cannot complete the task")
	}

	procinst, err := p.procinstApp.GetById(instTask.ProcinstId)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	execution, err := p.executionApp.GetById(instTask.ExecutionId)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if execution.NodeKey != instTask.NodeKey {
		return nil, nil, nil, nil, errorx.NewBiz("the current process instance node does not match and cannot process the task")
	}

	return instTask, taskCandidates, procinst, execution, nil
}

// completeInstTask 把审批任务从「审批中」原子地推到终态（通过 / 拒绝 / 退回）。
//
// 必须带 status=Process 条件并按是否真的改到行来判定成败：或签与会签都可能有两个审批人
// 同时点同一个任务，双方各自读到的都是「审批中」。少了这道 CAS，两边都会走完后续流程 ——
// 工单被批两次、批后回放的业务（一条 SQL、一条命令）被执行两次，且第二次不再查触发策略。
// MySQL 下后一个事务会阻塞在行锁上，前者提交后条件不再成立，这里拿到 0 行即拒绝
func (p *procinstTaskAppImpl) completeInstTask(ctx context.Context, instTask *entity.ProcinstTask) error {
	// 取仓储的行数入口：App 层的 UpdateByCond 不回报行数，而这里必须知道有没有改到行。
	// 条件里的 status=Process 与要写入的终态必然不同值，所以 0 行只可能是前置条件不成立
	// （别人已处理），不会是「值没变所以没改」
	rows, err := p.GetRepo().UpdateByCond(ctx, instTask, model.NewCond().Eq("id", instTask.Id).Eq("status", entity.ProcinstTaskStatusProcess))
	if err != nil {
		return err
	}
	if rows == 0 {
		return errorx.NewBizI(ctx, imsg.ErrProcinstTaskAlreadyHandled)
	}
	return nil
}
