package entity

import (
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/model"
	"time"
)

// 流程实例信息 -> 根据流程定义信息启动一个流程实例
type Procinst struct {
	model.Model

	ProcdefId   uint64 `json:"procdefId" gorm:"not null;index:idx_procdef_id;comment:流程定义id"` // 流程定义id
	ProcdefName string `json:"procdefName" gorm:"not null;size:64;comment:流程定义名称"`            // 流程定义名称

	BizType      string            `json:"bizType" gorm:"not null;size:64;comment:关联业务类型"`  // 业务类型
	BizKey       string            `json:"bizKey" gorm:"not null;size:64;comment:关联业务key"`  // 业务key
	BizForm      string            `json:"bizForm" gorm:"type:text;comment:业务form"`         // 业务表单
	BizStatus    ProcinstBizStatus `json:"bizStatus" gorm:"comment:业务状态"`                   // 业务状态
	BizHandleRes string            `json:"bizHandleRes" gorm:"size:4000;comment:关联的业务处理结果"` // 业务处理结果
	TaskKey      string            `json:"taskKey" gorm:"size:100;comment:当前任务key"`         // 当前任务key
	Status       ProcinstStatus    `json:"status" gorm:"comment:状态"`                        // 状态
	Remark       string            `json:"remark" gorm:"size:255;"`
	EndTime      *time.Time        `json:"endTime" gorm:"comment:结束时间"`
	Duration     int64             `json:"duration" gorm:"comment:流程持续时间（开始到结束）"` // 持续时间（开始到结束）
}

func (a *Procinst) TableName() string {
	return "t_flow_procinst"
}

// 设置流程终止结束的一些信息
func (a *Procinst) SetEnd() {
	nowTime := time.Now()
	a.EndTime = &nowTime
	a.Duration = int64(time.Since(*a.CreateTime).Seconds())
}

type ProcinstStatus int8

const (
	ProcinstStatusActive     ProcinstStatus = 1  // 流程实例正在执行中，当前有活动任务等待执行或者正在运行的流程节点
	ProcinstStatusCompleted  ProcinstStatus = 2  // 流程实例已经成功执行完成，没有剩余任务或者等待事件
	ProcinstStatusSuspended  ProcinstStatus = -1 // 流程实例被挂起，暂停执行，可能被驳回等待修改重新提交
	ProcinstStatusTerminated ProcinstStatus = -2 // 流程实例被终止，可能是由于某种原因如被拒绝等导致流程无法正常执行
	ProcinstStatusCancelled  ProcinstStatus = -3 // 流程实例被取消，通常是用户手动操作取消了流程的执行
)

var ProcinstStatusEnum = enumx.NewEnum[ProcinstStatus]("流程状态").
	Add(ProcinstStatusActive, "执行中").
	Add(ProcinstStatusCompleted, "完成").
	Add(ProcinstStatusSuspended, "挂起").
	Add(ProcinstStatusTerminated, "终止").
	Add(ProcinstStatusCancelled, "取消")

type ProcinstBizStatus int8

const (
	ProcinstBizStatusWait    ProcinstBizStatus = 1  // 待处理
	ProcinstBizStatusSuccess ProcinstBizStatus = 2  // 成功
	ProcinstBizStatusNo      ProcinstBizStatus = -1 // 不处理
	ProcinstBizStatusFail    ProcinstBizStatus = -2 // 失败
)

//----------流程实例关联任务-----------

// 流程实例关联的审批节点任务
type ProcinstTask struct {
	model.Model

	ProcinstId uint64 `json:"procinstId" gorm:"not null;index:idx_procinst_id;comment:流程实例id"` // 流程实例id
	TaskKey    string `json:"taskKey" gorm:"not null;size:64;comment:任务key"`                   // 当前任务key
	TaskName   string `json:"taskName" gorm:"size:64;comment:任务名称"`                            // 当前任务名称
	Assignee   string `json:"assignee" gorm:"size:64;comment:分配到该任务的用户"`                       // 分配到该任务的用户

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
	ProcinstTaskStatusProcess  ProcinstTaskStatus = 1  // 处理中
	ProcinstTaskStatusPass     ProcinstTaskStatus = 2  // 通过
	ProcinstTaskStatusReject   ProcinstTaskStatus = -1 // 拒绝
	ProcinstTaskStatusBack     ProcinstTaskStatus = -2 // 驳回
	ProcinstTaskStatusCanceled ProcinstTaskStatus = -3 // 取消
)
