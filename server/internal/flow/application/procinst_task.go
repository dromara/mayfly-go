package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/eventbus"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/stringx"

	"github.com/may-fly/cast"
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
			if err := p.procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate); err != nil {
				return err
			}
		}

		if err := p.procinstApp.Save(ctx, procinst); err != nil {
			return err
		}
		if err := p.Save(ctx, instTask); err != nil {
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
	// instTask, err := p.getAndValidInstTask(ctx, instTaskId)
	// if err != nil {
	// 	return err
	// }

	// // 赋值状态和备注
	// instTask.Status = entity.ProcinstTaskStatusBack
	// instTask.Remark = remark

	// procinst, _ := p.GetById(instTask.ProcinstId)

	// // 更新流程实例为挂起状态，等待重新提交
	// procinst.Status = entity.ProcinstStatusSuspended

	// return p.Tx(ctx, func(ctx context.Context) error {
	// 	return p.UpdateById(ctx, procinst)
	// }, func(ctx context.Context) error {
	// 	return p.procinstTaskRepo.UpdateById(ctx, instTask)
	// }, func(ctx context.Context) error {
	// 	return p.triggerProcinstStatusChangeEvent(ctx, procinst)
	// })
	return nil
}

func (p *procinstTaskAppImpl) CompleteTask(ctx context.Context, taskOp dto.UserTaskOp) error {
	taskId := taskOp.TaskId
	instTask, taskCandidates, procinst, execution, err := p.getAndValidInstTask(ctx, taskId, taskOp.Candidate)
	if err != nil {
		return err
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		executionCtx := NewExecutionCtx(ctx, procinst, execution)
		usertaskNode := ToUserTaskNode(executionCtx.GetFlowNode())

		for _, taskCandidate := range taskCandidates {
			taskCandidate.Status = entity.ProcinstTaskStatusCompleted
			taskCandidate.SetEnd()
			taskCandidate.Handler = &taskOp.Handler
			if err := p.procinstTaskCandidateRepo.UpdateById(ctx, taskCandidate); err != nil {
				return err
			}
		}

		executionCtx.parent = ctx

		vars := instTask.Vars
		// map[string]any整数会被解析为float64，故统一转为float64
		nrOfCompleted := cast.ToFloat64(vars.GetInt(NrOfCompleted) + len(taskCandidates))
		vars.Set(NrOfCompleted, nrOfCompleted)
		// 完成比例
		vars.Set("nrOfCompletedRate", float32(nrOfCompleted)/float32(vars.GetInt(NrOfAll)))

		isCompleteRes, err := stringx.TemplateParse(usertaskNode.CompletionCondition, vars)
		if err != nil {
			return err
		}
		// 不满足通过条件则保存更新任务完成数等变量即可
		if !cast.ToBool(isCompleteRes) {
			return p.Save(ctx, instTask)
		}

		// 赋值状态和备注
		instTask.Status = entity.ProcinstTaskStatusCompleted
		instTask.Remark = taskOp.Remark
		instTask.SetEnd()
		if err := p.Save(ctx, instTask); err != nil {
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
