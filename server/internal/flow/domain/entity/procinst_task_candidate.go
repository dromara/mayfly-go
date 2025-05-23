package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

// ProcinstTaskCandidate 流程实例任务处理候选人
type ProcinstTaskCandidate struct {
	model.Model
	model.ExtraData

	ProcinstId uint64 `json:"procinstId" gorm:"index:idx_ptc_procinst_id;comment:流程实例id"`
	TaskId     uint64 `json:"taskId" gorm:"index:idx_ptc_task_id;comment:流程实例任务id"` // 流程实例任务id
	Candidate  string `json:"condidate" gorm:"size:64;comment:处理候选人"`               // 处理候选人

	Handler  *string            `json:"handler" gorm:"size:60;"` // 处理人
	Status   ProcinstTaskStatus `json:"status" `                 // 状态
	Remark   string             `json:"remark" gorm:"size:255;"`
	EndTime  *time.Time         `json:"endTime" gorm:"comment:结束时间"`
	Duration int64              `json:"duration" gorm:"comment:任务持续时间（开始到结束）"` // 持续时间（开始到结束）
}

func (a *ProcinstTaskCandidate) TableName() string {
	return "t_flow_procinst_task_candidate"
}

// 设置流程任务终止结束的一些信息
func (p *ProcinstTaskCandidate) SetEnd() {
	nowTime := time.Now()
	p.EndTime = &nowTime
	p.Duration = int64(time.Since(*p.CreateTime).Seconds())
}
