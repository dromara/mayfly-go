package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"mayfly-go/internal/flow/application/dto"
	"mayfly-go/internal/flow/domain/entity"
	"mayfly-go/internal/flow/domain/repository"
	"mayfly-go/internal/flow/imsg"
	"mayfly-go/internal/pkg/consts"
	sysapp "mayfly-go/internal/sys/application"
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

	// CanViewProcinst 读接口可见性（详情/操作历史）：发起人 ∨ 管理员 ∨ 某任务的办理人/候选人
	// 写接口已有归属校验，读接口此前是唯一例外（任意登录账号可遍历自增 id 读全量工单）
	CanViewProcinst(ctx context.Context, procinstId uint64) error

	// RecordBizHandleResult 异步业务在终态完成后回写工单的业务处理结果与处理状态。
	//
	// 同步业务（SQL/命令回放）的结果由 bizHandle 在回调返回时记账；异步业务（如 AI 提案受控执行）
	// 回调只能给出「已受理」回执，终态成功与否只有业务体自己知道——没有这条通道时工单永远停在
	// 回执，审批人回到工单看不出变更到底执行成了没有
	RecordBizHandleResult(ctx context.Context, procinstId uint64, bizStatus entity.ProcinstBizStatus, handleRes string) error

	// CompletedProc 完成流程
	CompletedProc(ctx context.Context, procinstId uint64) error
}

type procinstAppImpl struct {
	base.AppImpl[*entity.Procinst, repository.Procinst]

	procdefApp Procdef `inject:"T"`
	// 读接口可见性判定需要「我是不是候选人」：候选人以账号 id / role:<角色 id> 两种形式存储
	procinstTaskCandidateRepo repository.ProcinstTaskCandidate `inject:"T"`
	roleApp                   sysapp.Role                      `inject:"T"`
}

var _ Procinst = (*procinstAppImpl)(nil)

var _ (Procinst) = (*procinstAppImpl)(nil)

func (p *procinstAppImpl) GetPageList(condition *entity.ProcinstQuery, orderBy ...string) (*model.PageResult[*entity.Procinst], error) {
	return p.Repo.GetPageList(condition, orderBy...)
}

func (p *procinstAppImpl) StartProc(ctx context.Context, procdefId uint64, reqParam *dto.StarProc) (*entity.Procinst, error) {
	procdef, err := p.procdefApp.GetById(procdefId)
	if err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrProcdefNotExist)
	}

	// 业务类型必须已注册回调：拼错或前端配置漂移会生成一条「能批、批完什么也不发生」的工单
	if !HasBizHandler(reqParam.BizType) {
		return nil, errorx.NewBizI(ctx, imsg.ErrBizTypeUnregistered, "bizType", reqParam.BizType)
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
	reuseBack := false
	if bizKey == "" {
		bizKey = stringx.RandUUID()
	} else {
		// 若业务key已存在，则为修改重新提交流程。
		// 只有「确实不存在」才走新建：读失败必须上抛——否则一次超时就会生成同 (biz_type,biz_key) 的
		// 第二条活实例，两条审批流各自通过后业务被回放两次（该列没有唯一索引兜底）
		existProcinst := &entity.Procinst{BizType: reqParam.BizType, BizKey: bizKey}
		if err := p.GetByCond(existProcinst); err != nil {
			if !errors.Is(err, model.ErrNotFound) {
				return nil, err
			}
		} else {
			if existProcinst.Status != entity.ProcinstStatusBack {
				return nil, errorx.NewBizI(ctx, imsg.ErrProcinstNotBackStatus)
			}
			if existProcinst.CreatorId != contextx.GetLoginAccount(ctx).Id {
				return nil, errorx.NewBizI(ctx, imsg.ErrProcinstNotCreator)
			}
			procinst.Id = existProcinst.Id
			reuseBack = true
		}
	}
	procinst.BizKey = bizKey

	// 新增实例必须是启用定义；退回重提是存量单，定义停用只表示「不接新单」，不能否决重提
	//（否则用户只剩取消一条路，已填的 bizForm 再也无法走审批）
	if !reuseBack && procdef.Status != entity.ProcdefStatusEnable {
		return nil, errorx.NewBizI(ctx, imsg.ErrProcdefNotEnable)
	}
	if flowdef := procdef.GetFlowDef(); flowdef == nil || len(flowdef.Nodes) == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrProcdefFlowNotExist)
	}

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

// CanViewProcinst 实例可见性：发起人、管理员、或该实例任一任务的处理人/候选人（含角色命中）。
//
// 与写侧同一口径：列表接口本来就按 creator 收窄、取消/回执按「属主或管理员」；
// 详情与操作历史此前只按自增 id 取数，任意登录账号即可读到 bizForm（SQL 全文 / 变更提案）与全部审批意见
func (p *procinstAppImpl) CanViewProcinst(ctx context.Context, procinstId uint64) error {
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBizI(ctx, imsg.ErrOperationForbidden)
	}
	procinst, err := p.GetById(procinstId)
	if err != nil {
		return err
	}
	if la.Id == consts.AdminId || procinst.CreatorId == la.Id {
		return nil
	}

	candidates, err := p.procinstTaskCandidateRepo.SelectByCond(&entity.ProcinstTaskCandidate{ProcinstId: procinstId})
	if err != nil {
		return err
	}
	selfCandidates := map[string]struct{}{fmt.Sprintf("%d", la.Id): {}}
	if roles, roleErr := p.roleApp.GetAccountRoles(la.Id); roleErr == nil {
		for _, r := range roles {
			selfCandidates[fmt.Sprintf("role:%d", r.RoleId)] = struct{}{}
		}
	}
	for _, c := range candidates {
		// Handler 存的是账号名（见 dto.UserTaskOp.Handler）
		if c.Handler != nil && *c.Handler == la.Username {
			return nil
		}
		if _, ok := selfCandidates[c.Candidate]; ok {
			return nil
		}
	}
	return errorx.NewBizI(ctx, imsg.ErrOperationForbidden)
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
		return errorx.NewBizI(ctx, imsg.ErrProcinstNotExist)
	}

	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBizI(ctx, imsg.ErrNotLogin)
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
		return errorx.NewBizI(ctx, imsg.ErrProcinstNotExist)
	}
	la := contextx.GetLoginAccount(ctx)
	if la == nil {
		return errorx.NewBizI(ctx, imsg.ErrNotLogin)
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
	if procinst.Status == entity.ProcinstStatusCompleted ||
		procinst.Status == entity.ProcinstStatusTerminated ||
		procinst.Status == entity.ProcinstStatusCancelled {
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
	// 与 CancelProc 同一口径的 CAS：把前置状态写进更新条件并按行数判赢。
	// 无条件全量 Save 会把并发的「取消」覆盖回「已完成」，随后 AfterCommit 还会执行一次业务回放
	// ——审计上取消凭空消失，且一条本不该执行的变更被执行
	rows, err := p.GetRepo().UpdateColsByCond(ctx, base.NewPatch().
		Set("status", procinst.Status).
		Set("end_time", procinst.EndTime).
		Set("duration", procinst.Duration).
		Set("biz_status", procinst.BizStatus).
		Set("biz_handle_res", procinst.BizHandleRes),
		model.NewCond().Eq(model.IdColumn, procinstId).
			EqStrict("status", entity.ProcinstStatusActive))
	if err != nil {
		return err
	}
	if rows != 1 {
		return errorx.NewBizI(ctx, imsg.ErrProcinstCancelled)
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
	// panic 也要留下可对账的失败原因：只停在 Doing 时，30 分钟后只有一条聚合告警，
	// 看不到是哪条工单、什么原因
	defer func() {
		if r := recover(); r != nil {
			procinst.BizStatus = entity.ProcinstBizStatusFail
			procinst.BizHandleRes = fmt.Sprintf("panic: %v", r)
			logx.ErrorfContext(ctx, "flow biz handle panic for procinst %d: %v", procinst.Id, r)
		}
		p.recordBizHandleResult(ctx, procinst)
	}()

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

// recordBizHandleResult 回写业务回放结果（同步链路记账，降级为只记日志不中断收尾）。
//
// 委托公开实现而非自己再写一份 patch：两列的记账形态只允许一处书写，
// 将来加第三列（如回写耗时）不会只改一处漏另一处
func (p *procinstAppImpl) recordBizHandleResult(ctx context.Context, procinst *entity.Procinst) {
	if err := p.RecordBizHandleResult(ctx, procinst.Id, procinst.BizStatus, procinst.BizHandleRes); err != nil {
		logx.ErrorfContext(ctx, "failed to record biz handle result for procinst %d: %s", procinst.Id, err.Error())
	}
}

// RecordBizHandleResult 异步业务终态回写（接口注释见声明处）。
//
// 按 id 条件更新而不先读实体：业务体完成时工单早已离开流程事务，这里只钉住结果两列，
// 不碰其余字段（实体全量更新会把并发路径写的列刷回旧值）
func (p *procinstAppImpl) RecordBizHandleResult(ctx context.Context, procinstId uint64, bizStatus entity.ProcinstBizStatus, handleRes string) error {
	patch := base.NewPatch().
		Set("biz_status", bizStatus).
		Set("biz_handle_res", handleRes)
	// 加状态围栏：异步业务（agentops）晚到的终态不得覆盖用户已置的「不处理」，
	// 也不得二次改写已经成功的记录（回执只允许从 Wait/Doing 落到终态一次）
	return p.UpdateColsByCond(ctx, patch, model.NewCond().Eq(model.IdColumn, procinstId).
		NotIn("biz_status", []entity.ProcinstBizStatus{
			entity.ProcinstBizStatusNo, entity.ProcinstBizStatusSuccess, entity.ProcinstBizStatusFail,
		}))
}
