package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// HisProcinstOp 流程实例关联的节点操作记录历史表
type HisProcinstOp struct {
	model.Model
	model.ExtraData

	ProcinstId  uint64 `json:"procinstId" gorm:"not null;index:idx_hpo_procinst_id;comment:流程实例id"` // 流程实例id
	ExecutionId uint64 `json:"executionId" gorm:"not null;index:idx_execution_id;comment:执行流id"`

	NodeKey  string       `json:"nodeKey" gorm:"not null;size:64;comment:节点key"` // 当前任务key
	NodeName string       `json:"nodeName" gorm:"size:64;comment:节点名称"`
	NodeType FlowNodeType `json:"nodeType" gorm:"comment:节点类型"`

	State    ProcinstOpState `json:"state" ` // 状态
	Remark   string          `json:"remark" gorm:"size:255;"`
	EndTime  *time.Time      `json:"endTime" gorm:"comment:结束时间"`
	Duration int64           `json:"duration" gorm:"comment:任务持续时间（开始到结束）"` // 持续时间（开始到结束）
}

func (a *HisProcinstOp) TableName() string {
	return "t_flow_his_procinst_op"
}

type ProcinstOpState int8

const (
	ProcinstOpStatePending   ProcinstOpState = 1  // 待处理
	ProcinstOpStateCompleted ProcinstOpState = 2  // 正常完成
	ProcinstOpStateFailed    ProcinstOpState = -1 // 执行失败
)
