package dto

import "mayfly-go/internal/flow/domain/entity"

type SaveProcdef struct {
	Procdef   *entity.Procdef
	MsgTmplId uint64 // 消息模板id
	CodePaths []string
}

type SaveFlowDef struct {
	Id      uint64
	FlowDef *entity.FlowDef // 消息模板id
}

// 启动流程实例请求入参
type StarProc struct {
	BizType string // 业务类型
	BizKey  string // 业务key
	Remark  string // 备注
	BizForm string // 业务表单信息
}
