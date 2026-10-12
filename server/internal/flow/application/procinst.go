package application

import (
	"context"
	"time"

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
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
)

type Procinst interface {
	base.App[*entity.Procinst]

	GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error)

	// ExecuteBizHandle 领取并执行该流程实例的业务回放（对目标实例执行 SQL、下发命令等）。
	//
	// 领取是「待处理 → 执行中」的 CAS 判赢：审批请求的快速路径与清扫任务同时撞上时只有一方会执行，
	// 否则同一条 SQL 会在目标库上跑两遍。未抢到领取权时直接返回，什么都不做
	ExecuteBizHandle(ctx context.Context, procinstId uint64) error

	// SweepPendingBizHandles 重驱动崩溃遗留的「已完成但业务待处理」实例，返回本轮实际重驱动条数
	SweepPendingBizHandles(ctx context.Context) (int, error)

	// StartProc 根据流程定义启动一个流程实例
	StartProc(ctx context.Context, procdefId uint64, reqParam *dto.StarProc) (*entity.Procinst, error)

	// CancelProc 取消流程
	CancelProc(ctx context.Context, procinstId uint64) error

	// RescheduleBizHandle 改期：仅「已完成且业务待处理(未领取)」的工单可改计划执行时间；非属主/状态不符则拒绝
	RescheduleBizHandle(ctx context.Context, procinstId uint64, at *time.Time) error

	// CancelBizHandle 取消执行：把「已完成且业务待处理」的工单业务置为不处理（不改流程状态，清扫不再领取）
	CancelBizHandle(ctx context.Context, procinstId uint64) error

	// CompletedProc 完成流程
	CompletedProc(ctx context.Context, procinstId uint64) error
}

type procinstAppImpl struct {
	base.AppImpl[*entity.Procinst, repository.Procinst]

	procdefApp Procdef `inject:"T"`
}

var _ Procinst = (*procinstAppImpl)(nil)

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

	procinst := &entity.Procinst{
		BizType:         reqParam.BizType,
		BizForm:         reqParam.BizForm,
		BizStatus:       entity.ProcinstBizStatusWait,
		ProcdefId:       procdef.Id,
		ProcdefName:     procdef.Name,
		Remark:          reqParam.Remark,
		Status:          entity.ProcinstStatusActive,
		FlowDef:         procdef.FlowDef,
		PlannedExecTime: reqParam.PlannedExecTime,
	}

	bizKey := reqParam.BizKey
	if bizKey == "" {
		bizKey = stringx.RandUUID()
	} else {
		// 若业务key已存在，则为修改重新提交流程
		existProcinst := &entity.Procinst{BizKey: bizKey}
		if err := p.GetByCond(existProcinst); err == nil {
			if existProcinst.Status != entity.ProcinstStatusBack {
				return nil, errorx.NewBizI(ctx, imsg.ErrProcinstNotBackStatus)
			}
			if existProcinst.CreatorId != contextx.GetLoginAccount(ctx).Id {
				return nil, errorx.NewBizI(ctx, imsg.ErrProcinstNotCreator)
			}
			procinst.Id = existProcinst.Id
		}
	}
	procinst.BizKey = bizKey

	return procinst, p.Tx(ctx, func(ctx context.Context) error {
		if err := p.Save(ctx, procinst); err != nil {
			return err
		}
		// 退回重提交走 UpdateById 只写非零字段，nil 计划时间不会落库、会残留旧值；
		// 显式覆盖该列（Patch.Set 对 nil 写 NULL），让「去掉定时改回立即执行」真正生效
		if procinst.Id != 0 {
			if _, err := p.GetRepo().UpdateColsByCond(ctx, base.NewPatch().Set("planned_exec_time", procinst.PlannedExecTime),
				model.NewCond().Eq(model.IdColumn, procinst.Id)); err != nil {
				return err
			}
		}
		return flowEventBus.PublishSync(ctx, EventTopicFlowProcinstCreate, procinst)
	})
}

// CancelProc 取消流程：仅「审批中/挂起/退回」的实例可取消。
//
// 写入走 CAS 条件更新而非按实体全量保存：无条件保存能把已完成（甚至已执行完）的实例
// 改写为「已取消」——审计上留下假记录，还会把「已批准、待执行」的工单从清扫链路里吞掉
// （那是 CancelBizHandle 的管辖范围）；与审批完成并发时也无人判赢，后写者覆盖前者。
// CAS 下只有一方判赢，输的一方拿到明确的业务拒绝，而不是「成功但什么也没发生」
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
		rows, err := p.GetRepo().UpdateColsByCond(ctx, base.NewPatch().
			Set("status", procinst.Status).
			Set("biz_status", procinst.BizStatus).
			Set("end_time", procinst.EndTime).
			Set("duration", procinst.Duration),
			model.NewCond().Eq(model.IdColumn, procinstId).
				In("status", collx.AsArray(entity.ProcinstStatusActive, entity.ProcinstStatusSuspended, entity.ProcinstStatusBack)))
		if err != nil {
			return err
		}
		if rows != 1 {
			return errorx.NewBizI(ctx, imsg.ErrProcinstStatusNotCancelable)
		}

		return flowEventBus.PublishSync(ctx, EventTopicFlowProcinstCancel, procinstId)
	})
}

// assertBizHandleOwnedPending 校验工单处于「已完成+待处理」且当前登录者为属主或管理员。
// 供改期/取消共用；真正的写入另用 CAS 兜住「校验后被清扫抢先领取」的窗口。
func (p *procinstAppImpl) assertBizHandleOwnedPending(ctx context.Context, procinstId uint64) error {
	procinst, err := p.GetById(procinstId)
	if err != nil || procinst == nil {
		return errorx.NewBiz("procinst not found")
	}
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBiz("no login")
	}
	if la.Id != consts.AdminId && procinst.CreatorId != la.Id {
		return errorx.NewBizI(ctx, imsg.ErrProcinstCancelSelf)
	}
	if procinst.Status != entity.ProcinstStatusCompleted || procinst.BizStatus != entity.ProcinstBizStatusWait {
		return errorx.NewBizI(ctx, imsg.ErrProcinstBizNotPending)
	}
	return nil
}

func (p *procinstAppImpl) RescheduleBizHandle(ctx context.Context, procinstId uint64, at *time.Time) error {
	if err := p.assertBizHandleOwnedPending(ctx, procinstId); err != nil {
		return err
	}
	// CAS：仅当仍是「已完成+待处理」才改期，避免与清扫领取竞态
	rows, err := p.GetRepo().UpdateColsByCond(ctx, base.NewPatch().Set("planned_exec_time", at),
		model.NewCond().Eq(model.IdColumn, procinstId).
			EqStrict("status", entity.ProcinstStatusCompleted).
			EqStrict("biz_status", entity.ProcinstBizStatusWait))
	if err != nil {
		return err
	}
	if rows != 1 {
		return errorx.NewBizI(ctx, imsg.ErrProcinstBizNotPending)
	}
	// 改到已到期或立即（nil/过去）时同步触发一次回放，让「立即执行」名副其实、不必等下一轮清扫；
	// ExecuteBizHandle 的领取是 Wait→Doing 的 CAS 判赢，与清扫并发天然互斥，不会重复执行
	//
	// 必须剥掉请求的取消信号再跑（与 CompletedProc 走 AfterCommit 时 runAfterCommit 内部做的那一步同源）：
	// 这里传进来的是 rc.MetaCtx，操作者点完「立即执行」就关页面会立刻 cancel 它，而回放是对目标库执行 SQL。
	// 带着已取消的 ctx 跑，领取已生效（行停在 Doing）、回放被拦腰打断、回写结果的同一条 UPDATE 也一起失败，
	// 于是这条工单永久卡在「执行中」且清扫只重驱动 Wait 的行，界面上既看不到结果也等不到重试
	if at == nil || !at.After(time.Now()) {
		if err := p.ExecuteBizHandle(contextx.Detach(ctx), procinstId); err != nil {
			logx.ErrorfContext(ctx, "failed to trigger biz handle after reschedule for procinst %d: %v", procinstId, err)
		}
	}
	return nil
}

func (p *procinstAppImpl) CancelBizHandle(ctx context.Context, procinstId uint64) error {
	if err := p.assertBizHandleOwnedPending(ctx, procinstId); err != nil {
		return err
	}
	// 置为「不处理」而非改流程状态：工单已批准完成，只是不再回放业务；
	// 同时清空计划执行时间，避免一个「不处理」的工单还挂着未来的计划时间造成 UI 歧义
	rows, err := p.GetRepo().UpdateColsByCond(ctx, base.NewPatch().Set("biz_status", entity.ProcinstBizStatusNo).Set("planned_exec_time", nil),
		model.NewCond().Eq(model.IdColumn, procinstId).
			EqStrict("status", entity.ProcinstStatusCompleted).
			EqStrict("biz_status", entity.ProcinstBizStatusWait))
	if err != nil {
		return err
	}
	if rows != 1 {
		return errorx.NewBizI(ctx, imsg.ErrProcinstBizNotPending)
	}
	return nil
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

	// 业务回放先置为「待处理」，与「流程已完成」一起在同一事务内落库。
	//
	// 这样即便进程在提交之后、执行之前崩溃，留下的也是一条「已完成但业务待处理」的可发现记录
	// （可被重驱动），而不是无痕丢失
	procinst.BizStatus = entity.ProcinstBizStatusWait
	procinst.BizHandleRes = ""
	if err := p.Save(ctx, procinst); err != nil {
		return err
	}

	// 提交成功后立即回放一次，让审批人在本次请求内就拿到执行结果。
	//
	// 只保证「不提前做」：进程若在提交后、执行前退出，这条记录仍停在「已完成 + 待处理」，
	// 由清扫任务按同一状态重新领取执行，不会无痕丢失；领取本身的互斥见 claimBizHandle
	return base.AfterCommit(ctx, func(runCtx context.Context) error {
		return runBizHandleAfterCommit(runCtx, procinst)
	})
}

// bizHandle 在事务之外执行业务回放，并把执行结果单独记账落库。
//
// 结果落库走补丁显式列写入：成功时 BizHandleRes 可能是空串，按「非零字段」更新的语义会把它
// 漏掉，于是上一次的失败原因会留在记录上被当成当次结果
func (p *procinstAppImpl) bizHandle(ctx context.Context, procinst *entity.Procinst) {
	// 无论成功还是失败，结果都必须落库：它是「流程已完成」与「业务是否真的执行成功」之间唯一的
	// 对账依据。用 defer 统一回写，避免任何一条提前 return 的分支把结果留在内存里丢掉
	defer p.recordBizHandleResult(ctx, procinst)

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
		}
		logx.ErrorfContext(ctx, "process business [%s] processing failed: %v", procinst.BizKey, err.Error())
		return
	}

	procinst.BizStatus = entity.ProcinstBizStatusSuccess
	if procinst.BizHandleRes == "" {
		procinst.BizHandleRes = "success"
	}
}

// recordBizHandleResult 回写业务回放结果。
//
// 用 Patch 而不是实体更新：结果列需要能被真实覆盖（含清空），且这次写入独立于主流程事务
func (p *procinstAppImpl) recordBizHandleResult(ctx context.Context, procinst *entity.Procinst) {
	patch := base.NewPatch().
		Set("biz_status", procinst.BizStatus).
		Set("biz_handle_res", procinst.BizHandleRes)
	if err := p.UpdateColsByCond(ctx, patch, model.NewCond().Eq(model.IdColumn, procinst.Id)); err != nil {
		logx.ErrorfContext(ctx, "failed to record biz handle result for procinst %d: %s", procinst.Id, err.Error())
	}
}
