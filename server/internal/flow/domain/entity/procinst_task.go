package entity

import (
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"time"
)

//----------流程实例关联任务-----------

// 流程实例关联的节点任务
type ProcinstTask struct {
	model.Model
	model.ExtraData

	ProcinstId  uint64       `json:"procinstId" gorm:"not null;index:idx_pt_procinst_id;comment:流程实例id"` // 流程实例id
	ExecutionId uint64       `json:"executionId" gorm:"comment:执行流id"`
	NodeKey     string       `json:"nodeKey" gorm:"size:64;comment:节点key"`
	NodeName    string       `json:"nodeName" gorm:"size:64;comment:节点名称"`
	NodeType    FlowNodeType `json:"nodeType" gorm:"comment:节点类型"`
	Vars        collx.M      `json:"vars" gorm:"type:text;comment:任务变量"`

	Status   ProcinstTaskStatus `json:"status" ` // 状态
	Remark   string             `json:"remark" gorm:"size:255;"`
	EndTime  *time.Time         `json:"endTime" gorm:"comment:结束时间"`
	Duration int64              `json:"duration" gorm:"comment:任务持续时间（开始到结束）"` // 持续时间（开始到结束）
}

func (a *ProcinstTask) TableName() string {
	return "t_flow_procinst_task"
}

// 设置流程任务终止结束的一些信息
func (p *ProcinstTask) SetEnd() {
	nowTime := time.Now()
	p.EndTime = &nowTime
	p.Duration = int64(time.Since(*p.CreateTime).Seconds())
}

type ProcinstTaskStatus int8

const (
	ProcinstTaskStatusProcess   ProcinstTaskStatus = 1  // 审批中
	ProcinstTaskStatusCompleted ProcinstTaskStatus = 2  // 完成
	ProcinstTaskStatusReject    ProcinstTaskStatus = -1 // 拒绝
	ProcinstTaskStatusBack      ProcinstTaskStatus = -2 // 驳回
	ProcinstTaskStatusCanceled  ProcinstTaskStatus = -3 // 取消
)
