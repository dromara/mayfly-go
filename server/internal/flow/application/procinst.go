package application

import (
	"context"
	"fmt"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
)

type Procinst interface {
	base.App[*entity.Procinst]

	GetPageList(condition *entity.ProcinstQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// 获取流程实例审批节点任务
	GetProcinstTasks(condition *entity.ProcinstTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error)

	// StartProc 根据流程定义启动一个流程实例
	StartProc(ctx context.Context, procdefId uint64, reqParam *StarProcParam) (*entity.Procinst, error)

	// 取消流程
	CancelProc(ctx context.Context, procinstId uint64) error

	// 完成任务
	CompleteTask(ctx context.Context, taskId uint64, remark string) error

	// 拒绝任务
	RejectTask(ctx context.Context, taskId uint64, remark string) error

	// 驳回任务（允许重新提交）
	BackTask(ctx context.Context, taskId uint64, remark string) error
}

type procinstAppImpl struct {
	base.AppImpl[*entity.Procinst, repository.Procinst]

	procinstTaskRepo repository.ProcinstTask `inject:"ProcinstTaskRepo"`
	procdefApp       Procdef                 `inject:"ProcdefApp"`
}

var _ (Procinst) = (*procinstAppImpl)(nil)

// 注入repo
func (p *procinstAppImpl) InjectProcinstRepo(procinstRepo repository.Procinst) {
	p.Repo = procinstRepo
}

func (p *procinstAppImpl) GetPageList(condition *entity.ProcinstQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.Repo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *procinstAppImpl) GetProcinstTasks(condition *entity.ProcinstTaskQuery, pageParam *model.PageParam, toEntity any, orderBy ...string) (*model.PageResult[any], error) {
	return p.procinstTaskRepo.GetPageList(condition, pageParam, toEntity, orderBy...)
}

func (p *procinstAppImpl) StartProc(ctx context.Context, procdefId uint64, reqParam *StarProcParam) (*entity.Procinst, error) {
	procdef, err := p.procdefApp.GetById(procdefId)
	if err != nil {
		return nil, errorx.NewBiz("流程实例[%d]不存在", procdefId)
	}

	if procdef.Status != entity.ProcdefStatusEnable {
		return nil, errorx.NewBiz("该流程定义非启用状态")
	}

	procinst := &entity.Procinst{
		BizType:     reqParam.BizType,
		BizKey:      reqParam.BizKey,
		BizForm:     reqParam.BizForm,
		BizStatus:   entity.ProcinstBizStatusWait,
		ProcdefId:   procdef.Id,
		ProcdefName: procdef.Name,
		Remark:      reqParam.Remark,
		Status:      entity.ProcinstStatusActive,
	}

	task := p.getNextTask(procdef, "")
	procinst.TaskKey = task.TaskKey

	if err := p.Save(ctx, procinst); err != nil {
		return nil, err
	}
	return procinst, p.createProcinstTask(ctx, procinst, task)
}

func (p *procinstAppImpl) CancelProc(ctx context.Context, procinstId uint64) error {
	procinst, err := p.GetById(procinstId)
	if err != nil {
		return errorx.NewBiz("流程不存在")
	}

	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBiz("未登录")
	}
	if procinst.CreatorId != la.Id {
		return errorx.NewBiz("只能取消自己创建的流程")
	}
	procinst.Status = entity.ProcinstStatusCancelled
	procinst.BizStatus = entity.ProcinstBizStatusNo
	procinst.SetEnd()

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.cancelInstTasks(ctx, procinstId, "流程已取消")
	}, func(ctx context.Context) error {
		return p.Save(ctx, procinst)
	}, func(ctx context.Context) error {
		return p.triggerProcinstStatusChangeEvent(ctx, procinst)
	})
}

func (p *procinstAppImpl) CompleteTask(ctx context.Context, instTaskId uint64, remark string) error {
	instTask, err := p.getAndValidInstTask(ctx, instTaskId)
	if err != nil {
		return err
	}

	// 赋值状态和备注
	instTask.Status = entity.ProcinstTaskStatusPass
	instTask.Remark = remark
	instTask.SetEnd()

	procinst, _ := p.GetById(instTask.ProcinstId)
	procdef, _ := p.procdefApp.GetById(procinst.ProcdefId)

	// 获取下一实例审批任务
	task := p.getNextTask(procdef, instTask.TaskKey)
	if task == nil {
		procinst.Status = entity.ProcinstStatusCompleted
		procinst.SetEnd()
	} else {
		procinst.TaskKey = task.TaskKey
	}

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.UpdateById(ctx, procinst)
	}, func(ctx context.Context) error {
		return p.procinstTaskRepo.UpdateById(ctx, instTask)
	}, func(ctx context.Context) error {
		return p.createProcinstTask(ctx, procinst, task)
	}, func(ctx context.Context) error {
		// 下一审批节点任务不存在，说明该流程已结束
		if task == nil {
			return p.triggerProcinstStatusChangeEvent(ctx, procinst)
		}
		return nil
	})
}

func (p *procinstAppImpl) RejectTask(ctx context.Context, instTaskId uint64, remark string) error {
	instTask, err := p.getAndValidInstTask(ctx, instTaskId)
	if err != nil {
		return err
	}

	// 赋值状态和备注
	instTask.Status = entity.ProcinstTaskStatusReject
	instTask.Remark = remark
	instTask.SetEnd()

	procinst, _ := p.GetById(instTask.ProcinstId)
	// 更新流程实例为终止状态，无法重新提交
	procinst.Status = entity.ProcinstStatusTerminated
	procinst.BizStatus = entity.ProcinstBizStatusNo
	procinst.SetEnd()

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.UpdateById(ctx, procinst)
	}, func(ctx context.Context) error {
		return p.procinstTaskRepo.UpdateById(ctx, instTask)
	}, func(ctx context.Context) error {
		return p.triggerProcinstStatusChangeEvent(ctx, procinst)
	})
}

func (p *procinstAppImpl) BackTask(ctx context.Context, instTaskId uint64, remark string) error {
	instTask, err := p.getAndValidInstTask(ctx, instTaskId)
	if err != nil {
		return err
	}

	// 赋值状态和备注
	instTask.Status = entity.ProcinstTaskStatusBack
	instTask.Remark = remark

	procinst, _ := p.GetById(instTask.ProcinstId)

	// 更新流程实例为挂起状态，等待重新提交
	procinst.Status = entity.ProcinstStatusSuspended

	return p.Tx(ctx, func(ctx context.Context) error {
		return p.UpdateById(ctx, procinst)
	}, func(ctx context.Context) error {
		return p.procinstTaskRepo.UpdateById(ctx, instTask)
	}, func(ctx context.Context) error {
		return p.triggerProcinstStatusChangeEvent(ctx, procinst)
	})
}

// 取消处理中的流程实例任务
func (p *procinstAppImpl) cancelInstTasks(ctx context.Context, procinstId uint64, cancelReason string) error {
	// 流程实例任务信息
	instTasks, _ := p.procinstTaskRepo.SelectByCond(&entity.ProcinstTask{ProcinstId: procinstId, Status: entity.ProcinstTaskStatusProcess})
	for _, instTask := range instTasks {
		instTask.Status = entity.ProcinstTaskStatusCanceled
		instTask.Remark = cancelReason
		instTask.SetEnd()
		p.procinstTaskRepo.UpdateById(ctx, instTask)
	}
	return nil
}

// 触发流程实例状态改变事件
func (p *procinstAppImpl) triggerProcinstStatusChangeEvent(ctx context.Context, procinst *entity.Procinst) error {
	err := FlowBizHandle(ctx, &BizHandleParam{
		BizType:        procinst.BizType,
		BizKey:         procinst.BizKey,
		BizForm:        procinst.BizForm,
		ProcinstStatus: procinst.Status,
	})

	if err != nil {
		// 业务处理错误，非完成状态则终止流程
		if procinst.Status != entity.ProcinstStatusCompleted {
			procinst.Status = entity.ProcinstStatusTerminated
			procinst.SetEnd()
			p.cancelInstTasks(ctx, procinst.Id, "业务处理失败")
		}
		procinst.BizStatus = entity.ProcinstBizStatusFail
		procinst.BizHandleRes = err.Error()
		return p.UpdateById(ctx, procinst)
	}

	// 处理成功，并且状态为完成，则更新业务状态为成功
	if procinst.Status == entity.ProcinstStatusCompleted {
		procinst.BizStatus = entity.ProcinstBizStatusSuccess
		procinst.BizHandleRes = "success"
		return p.UpdateById(ctx, procinst)
	}
	return err
}

// 获取并校验实例任务
func (p *procinstAppImpl) getAndValidInstTask(ctx context.Context, instTaskId uint64) (*entity.ProcinstTask, error) {
	instTask, err := p.procinstTaskRepo.GetById(instTaskId)
	if err != nil {
		return nil, errorx.NewBiz("流程实例任务不存在")
	}

	la := contextx.GetLoginAccount(ctx)
	if instTask.Assignee != fmt.Sprintf("%d", la.Id) {
		return nil, errorx.NewBiz("当前用户不是任务处理人，无法完成任务")
	}

	return instTask, nil
}

// 创建流程实例节点任务
func (p *procinstAppImpl) createProcinstTask(ctx context.Context, procinst *entity.Procinst, task *entity.ProcdefTask) error {
	if task == nil {
		return nil
	}

	procinstTask := &entity.ProcinstTask{
		ProcinstId: procinst.Id,
		Status:     entity.ProcinstTaskStatusProcess,

		TaskKey:  task.TaskKey,
		TaskName: task.Name,
		Assignee: task.UserId,
	}
	return p.procinstTaskRepo.Insert(ctx, procinstTask)
}

// 获取下一审批节点任务
func (p *procinstAppImpl) getNextTask(procdef *entity.Procdef, nowTaskKey string) *entity.ProcdefTask {
	tasks := procdef.GetTasks()
	if len(tasks) == 0 {
		return nil
	}

	if nowTaskKey == "" {
		// nowTaskKey为空，则说明为刚启动该流程实例
		return tasks[0]
	}

	for index, t := range tasks {
		if (t.TaskKey == nowTaskKey) && (index < len(tasks)-1) {
			return tasks[index+1]
		}
	}

	return nil
}
