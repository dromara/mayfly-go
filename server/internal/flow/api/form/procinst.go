package form

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

type ProcinstStart struct {
	ProcdefId uint64  `json:"procdefId" binding:"required"` // 流程定义id
	BizKey    string  `json:"bizKey"`
	BizType   string  `json:"bizType" binding:"required"` // 业务类型
	Remark    string  `json:"remark"`                     // 流程备注
	BizForm   collx.M `json:"bizForm" binding:"required"` // 业务表单
	// PlannedExecTime 计划执行时间（可选）：审批通过后到点才执行；不传/零值=批准后立即执行。
	// 用 model.JsonTime 而非裸 time.Time：前后端约定时间格式为 "2006-01-02 15:04:05"（本地时区），
	// 裸 time.Time 只接受 RFC3339，日期选择器提交的格式会导致 JSON 解析失败
	PlannedExecTime *model.JsonTime `json:"plannedExecTime"`
}

// ProcinstReschedule 改期入参：将已批准未执行工单的计划执行时间改为新时间（nil/零值=改成立即执行）
type ProcinstReschedule struct {
	PlannedExecTime *model.JsonTime `json:"plannedExecTime"`
}

type ProcinstTaskAudit struct {
	Id uint64 `json:"id" binding:"required"`
	// max=255 与候选人表 remark 列一致：超长会让整次审批以数据库原始错误失败（用户看到 500）
	Remark string `json:"remark" binding:"max=255"`
}
