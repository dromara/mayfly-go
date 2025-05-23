package form

import "mayfly-go/internal/flow/domain/entity"

type Procdef struct {
	Id        uint64               `json:"id"`
	Name      string               `json:"name" binding:"required"` // 名称
	DefKey    string               `json:"defKey" binding:"required"`
	Status    entity.ProcdefStatus `json:"status" binding:"required"`
	Condition string               `json:"condition"`
	Remark    string               `json:"remark"`
	MsgTmplId uint64               `json:"msgTmplId"`

	CodePaths []string `json:"codePaths"`
}

type ProcdefFlow struct {
	Id   uint64          `json:"id" binding:"required"`
	Flow *entity.FlowDef `json:"flow" binding:"required"`
}
