package form

import "mayfly-go/internal/flow/domain/entity"

type Procdef struct {
	Id     uint64               `json:"id"`
	Name   string               `json:"name" binding:"required"` // 名称
	DefKey string               `json:"defKey" binding:"required"`
	Tasks  string               `json:"tasks" binding:"required"` // 审批节点任务信息
	Status entity.ProcdefStatus `json:"status" binding:"required"`
	Remark string               `json:"remark"`
}
