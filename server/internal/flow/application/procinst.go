package application

import (
	"context"
	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/anyx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
)

type Procinst interface {
	base.App[*entity.Procinst]

	GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error)

	// StartProc 根据流程定义启动一个流程实例
	StartProc(ctx context.Context, procdefId uint64, reqParam *dto.StarProc) (*entity.Procinst, error)

	// CancelProc 取消流程
	CancelProc(ctx context.Context, procinstId uint64) error

	// CompletedProc 完成流程
	CompletedProc(ctx context.Context, procinstId uint64) error
}

type procinstAppImpl struct {
	base.AppImpl[*entity.Procinst, repository.Procinst]

	procdefApp Procdef `inject:"T"`
}

var _ (Procinst) = (*procinstAppImpl)(nil)

func (p *procinstAppImpl) GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error) {
	return p.Repo.GetPageList(condition, orderBy...)
}

func (p *procinstAppImpl) StartProc(ctx context.Context, procdefId uint64, reqParam *dto.StarProc) (*entity.Procinst, error) {
	procdef, err := p.procdefApp.GetById(procdefId)
	if err != nil {
		return nil, errorx.NewBiz("procdef not found")
	}

	if procdef.Status != entity.ProcdefStatusEnable {
		return nil, errorx.NewBizI(ctx, imsg.ErrProcdefNotEnable)
	}
	if flowdef := procdef.GetFlowDef(); flowdef == nil || len(flowdef.Nodes) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrProcdefFlowNotExist)
	}

	bizKey := reqParam.BizKey
	if bizKey == "" {
		bizKey = stringx.RandUUID()
	}
	procinst := &entity.Procinst{
		BizType:     reqParam.BizType,
		BizKey:      bizKey,
		BizForm:     reqParam.BizForm,
		BizStatus:   entity.ProcinstBizStatusWait,
		ProcdefId:   procdef.Id,
		ProcdefName: procdef.Name,
		Remark:      reqParam.Remark,
		Status:      entity.ProcinstStatusActive,
		FlowDef:     procdef.FlowDef,
	}

	return procinst, p.Tx(ctx, func(ctx context.Context) error {
		if err := p.Save(ctx, procinst); err != nil {
			return err
		}
		return flowEventBus.PublishSync(ctx, EventTopicFlowProcinstCreate, procinst)
	})
}

func (p *procinstAppImpl) CancelProc(ctx context.Context, procinstId uint64) error {
	procinst, err := p.GetById(procinstId)
	if err != nil {
		return errorx.NewBiz("procinst not found")
	}

	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBiz("no login")
	}
	if la.Id != consts.AdminId && procinst.CreatorId != la.Id {
		return errorx.NewBizI(ctx, imsg.ErrProcinstCancelSelf)
	}
	procinst.Status = entity.ProcinstStatusCancelled
	procinst.BizStatus = entity.ProcinstBizStatusNo
	procinst.SetEnd()

	return p.Tx(ctx, func(ctx context.Context) error {
		if err := p.Save(ctx, procinst); err != nil {
			return err
		}

		return flowEventBus.PublishSync(ctx, EventTopicFlowProcinstCancel, procinstId)
	})
}

func (p *procinstAppImpl) CompletedProc(ctx context.Context, procinstId uint64) error {
	procinst, err := p.GetById(procinstId)
	if err != nil {
		return err
	}
	// 已完成或已终止，则不进行后续处理
	if procinst.Status == entity.ProcinstStatusCompleted || procinst.Status == entity.ProcinstStatusTerminated {
		return nil
	}

	procinst.Status = entity.ProcinstStatusCompleted
	procinst.SetEnd()

	// 业务处理
	p.bizHandle(ctx, procinst)

	return p.Save(ctx, procinst)
}

// 业务处理
func (p *procinstAppImpl) bizHandle(ctx context.Context, procinst *entity.Procinst) {
	handleRes, err := FlowBizHandle(ctx, &BizHandleParam{
		Procinst: *procinst,
	})

	if !anyx.IsBlank(handleRes) {
		procinst.BizHandleRes = jsonx.ToStr(handleRes)
	}

	if err != nil {
		procinst.BizStatus = entity.ProcinstBizStatusFail
		if procinst.BizHandleRes == "" {
			procinst.BizHandleRes = err.Error()
		} else {
			logx.Errorf("process business [%s] processing failed: %v", procinst.BizKey, err.Error())
		}
		return
	}

	procinst.BizStatus = entity.ProcinstBizStatusSuccess
	if procinst.BizHandleRes == "" {
		procinst.BizHandleRes = "success"
	}
}
