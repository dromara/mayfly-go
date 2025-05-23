package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
)

// Execution 流程执行流信息
type Execution struct {
	model.Model

	ProcinstId uint64 `json:"procinstId" gorm:"not null;index:idx_exe_procinst_id;comment:流程实例id"`
	ParentId   uint64 `json:"parentId" gorm:"default:0;comment:父级执行id"` // 父执行流ID（并行网关分支时指向网关的Execution ID）

	Vars     collx.M      `json:"vars" gorm:"type:text;comment:执行流变量"`
	NodeKey  string       `json:"nodeKey" gorm:"not null;size:64;comment:节点key"`
	NodeName string       `json:"nodeName" gorm:"size:64;comment:节点名称"`
	NodeType FlowNodeType `json:"nodeType" gorm:"comment:节点类型"`

	State        ExectionState `json:"state" gorm:"comment:执行状态"`
	IsConcurrent int8          `json:"isConcurrent" gorm:"comment:是否并行"`
}

func (m *Execution) TableName() string {
	return "t_flow_execution"
}

type ExectionState int8

const (
	ExectionStateActive     ExectionState = 1  // 运行中
	ExectionStateSuspended  ExectionState = -1 // 挂起
	ExectionStateTerminated ExectionState = -2 // 已终止
	ExectionStateCompleted  ExectionState = 2  // 完成
)
