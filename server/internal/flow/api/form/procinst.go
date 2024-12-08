package form

import (
	"mayfly-go/pkg/utils/collx"
)

type ProcinstStart struct {
	ProcdefId uint64  `json:"procdefId" binding:"required"` // 流程定义id
	BizType   string  `json:"bizType" binding:"required"`   // 业务类型
	Remark    string  `json:"remark"`                       // 流程备注
	BizForm   collx.M `json:"bizForm" binding:"required"`   // 业务表单
}

type ProcinstTaskAudit struct {
	Id     uint64 `json:"id" binding:"required"`
	Remark string `json:"remark"`
}
