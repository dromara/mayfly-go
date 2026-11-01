package application

// 导出工单：把「导出被策略拦下」变成一条可审批的通道。
//
// 边界（这是本功能最重要的约定）：**表级/列级权限命中一律拒绝，不允许通过工单绕过**。
// 权限是硬边界（谁都不该看到那行数据），工单解决的是「这次导出范围太大/太敏感，
// 需要有人点头」这类策略问题。两类拦截混在一起做，工单就变成了绕权的正规通道。
//
// 实现上不新建业务表：工单本身就是凭据——审批完成后 24 小时内，申请人可凭该工单
// 下载「它申请的那一次导出」。工单里记着库、表清单与导出类型，下载时以工单为准，
// 不接受请求方改参数（否则就是拿一张小范围工单导出整库）。

import (
	"context"
	"encoding/json"
	"time"

	"mayfly-go/internal/db/imsg"
	flowapp "mayfly-go/internal/flow/application"
	flowentity "mayfly-go/internal/flow/domain/entity"
	"mayfly-go/pkg/contextx"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/ioc"
)

// DbExportFlowBizType 导出工单的业务类型
const DbExportFlowBizType = "db_export_flow"

// exportTicketTtl 审批完成后允许下载的时长。
//
// 不设成「永久」：工单是被审批的一次快照，越久越可能与当下的权限/数据状态脱节；
// 也不设太短：审批人多半在工作时间处理，24 小时覆盖一个跨天的工作节奏
const exportTicketTtl = 24 * time.Hour

// DbExportTicketForm 工单业务表单（由前端写入 bizForm，下载时以它为准）
type DbExportTicketForm struct {
	DbId     uint64   `json:"dbId"`
	DbName   string   `json:"dbName"`
	Tables   []string `json:"tables"`
	DumpType string   `json:"dumpType"` // 1 结构 2 数据 3 结构与数据
	Reason   string   `json:"reason"`
}

// DbExportTicketApp 导出工单。
//
// 刻意不继承 base.App：本功能不落业务表——工单本身就是凭据（t_flow_procinst），
// 为了凑一个通用的增删查基类去建一张空表，是把「表驱动」套在不需要表的地方。
type DbExportTicketApp interface {
	// ValidateForDownload 校验该工单是否允许当前账号下载它授权的这次导出
	ValidateForDownload(ctx context.Context, procinstId uint64) (*DbExportTicketForm, error)
}

type dbExportTicketAppImpl struct {
	procinstApp flowapp.Procinst `inject:"T"`
}

// exportTicketApp 单例：既给 IOC 注入，也给工单回调注册用（两者必须是同一个实例）
var exportTicketApp = &dbExportTicketAppImpl{}

// NewDbExportTicketApp 供父包 IOC 注册
func NewDbExportTicketApp() DbExportTicketApp {
	return exportTicketApp
}

// init 注册工单业务处理器。
//
// 不注册的后果不是报错而是「审批通过后什么都不会发生」：flow 侧靠 HasBizHandler
// 判断该场景的「需审批」能否落地，缺回调时策略只能配成不阻断
func init() {
	flowapp.RegisterBizHandler(DbExportFlowBizType, exportTicketApp)
}

func GetDbExportTicketApp() DbExportTicketApp {
	return ioc.Get[DbExportTicketApp]()
}

// ValidateForDownload 工单可用性判定：类型对、已通过、未过期、申请人是当前账号
func (a *dbExportTicketAppImpl) ValidateForDownload(ctx context.Context, procinstId uint64) (*DbExportTicketForm, error) {
	if procinstId == 0 {
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketNotFound)
	}
	procinst, err := a.procinstApp.GetById(procinstId)
	if err != nil || procinst == nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketNotFound)
	}
	if procinst.BizType != DbExportFlowBizType {
		// 用别的工单（比如执行 SQL 的工单）来换一次导出，是典型的越权尝试
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketMismatch)
	}
	if procinst.Status != flowentity.ProcinstStatusCompleted {
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketNotApproved)
	}
	if procinst.EndTime == nil || time.Since(*procinst.EndTime) > exportTicketTtl {
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketExpired)
	}

	la := contextx.GetLoginAccount(ctx)
	if la == nil || la.Id != procinst.CreatorId {
		// 工单只对申请人本人有效：同事之间「借一张已批的工单」不该成立
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketNotOwner)
	}

	form := &DbExportTicketForm{}
	if err := json.Unmarshal([]byte(procinst.BizForm), form); err != nil {
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketInvalid)
	}
	if form.DbId == 0 || len(form.Tables) == 0 {
		// 没写清导出范围的工单不构成授权：放行等于开了一张空白支票
		return nil, errorx.NewBizI(ctx, imsg.ErrExportTicketInvalid)
	}
	return form, nil
}

// FlowBizHandle 审批通过后的业务回放：本工单不执行任何数据库操作，
// 它的「执行」就是解锁一次下载，因此这里只回报授权已生效
func (a *dbExportTicketAppImpl) FlowBizHandle(ctx context.Context, param *flowapp.BizHandleParam) (any, error) {
	if param.Procinst.Status != flowentity.ProcinstStatusCompleted {
		// 被终止/驳回的工单不解锁任何东西
		return nil, nil
	}
	return map[string]any{"downloadable": true, "expireHours": int(exportTicketTtl.Hours())}, nil
}
