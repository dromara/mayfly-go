package vo

import (
	"mayfly-go/internal/flow/domain/entity"
	"time"
)

type ProcinstVO struct {
	Id          uint64 `json:"id"`
	ProcdefId   uint64 `json:"procdefId"`   // 流程定义id
	ProcdefName string `json:"procdefName"` // 流程定义名称

	BizType      string     `json:"bizType"`      // 业务类型
	BizKey       string     `json:"bizKey"`       // 业务key
	BizStatus    int8       `json:"bizStatus"`    // 业务状态
	BizHandleRes string     `json:"bizHandleRes"` // 业务处理结果
	TaskKey      string     `json:"taskKey"`      // 当前任务key
	Remark       string     `json:"remark"`
	Status       int8       `json:"status"`
	EndTime      *time.Time `json:"endTime"`
	Duration     int64      `json:"duration"` // 持续时间（开始到结束）

	Creator    string     `json:"creator"`
	CreateTime *time.Time `json:"createTime"`
	UpdateTime *time.Time `json:"updateTime"`

	Procdef       *entity.Procdef        `json:"procdef"`
	ProcinstTasks []*entity.ProcinstTask `json:"procinstTasks"`
}

type ProcinstTask struct {
	Id         uint64 `json:"id"`
	ProcinstId uint64 `json:"procinstId"` // 流程实例id
	TaskKey    string `json:"taskKey"`    // 当前任务key
	TaskName   string `json:"taskName"`   // 当前任务名称
	Assignee   string `json:"assignee"`   // 分配到该任务的用户

	Status     entity.ProcinstTaskStatus `json:"status"` // 状态
	Remark     string                    `json:"remark"`
	Duration   int64                     `json:"duration"` // 持续时间（开始到结束）
	CreateTime *time.Time                `json:"createTime"`
	EndTime    *time.Time                `json:"endTime"`

	Procinst *entity.Procinst `json:"procinst"`
}
