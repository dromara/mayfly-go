package application

import (
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/pkg/base"
	"time"
)

type HisProcinstOp interface {
	base.App[*entity.HisProcinstOp]

	RecordStart(ctx *ExecutionCtx) error

	RecordEnd(ctx *ExecutionCtx, remark string) error
}

type hisProcinstOpAppImpl struct {
	base.AppImpl[*entity.HisProcinstOp, repository.HisProcinstOp]
}

var _ (HisProcinstOp) = (*hisProcinstOpAppImpl)(nil)

func (h *hisProcinstOpAppImpl) RecordStart(ctx *ExecutionCtx) error {
	execution := ctx.Execution
	opLog := &entity.HisProcinstOp{
		ProcinstId:  execution.ProcinstId,
		ExecutionId: execution.Id,
		NodeKey:     execution.NodeKey,
		NodeType:    execution.NodeType,
		NodeName:    execution.NodeName,
		State:       entity.ProcinstOpStatePending,
	}
	ctx.HisProcinstOp = opLog

	return h.Save(ctx, opLog)
}

func (h *hisProcinstOpAppImpl) RecordEnd(ctx *ExecutionCtx, remark string) error {
	execution := ctx.Execution
	// 从执行流上下文获取当前节点的执行记录，若存在则使用该记录即可
	op := ctx.HisProcinstOp

	if op == nil || op.State != entity.ProcinstOpStatePending {
		op = &entity.HisProcinstOp{
			State:       entity.ProcinstOpStatePending,
			NodeKey:     execution.NodeKey,
			ExecutionId: execution.Id,
			ProcinstId:  execution.ProcinstId,
		}
		// 不存在未完成的节点类型
		if err := h.GetByCond(op); err != nil {
			return nil
		}
	}

	now := time.Now()
	updateOp := &entity.HisProcinstOp{
		State:    entity.ProcinstOpStateCompleted,
		EndTime:  &now,
		Duration: int64(now.Sub(*op.CreateTime).Seconds()),
		Remark:   remark,
	}
	updateOp.Extra = ctx.OpExtra
	updateOp.Id = op.Id

	return h.Save(ctx, updateOp)
}
