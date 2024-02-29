package entity

import (
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/model"
	"time"
)

// 流程实例信息 -> 根据流程定义信息启动一个流程实例
type Procinst struct {
	model.Model

	ProcdefId   uint64 `json:"procdefId"`   // 流程定义id
	ProcdefName string `json:"procdefName"` // 流程定义名称

	BizType      string            `json:"bizType"`      // 业务类型
	BizKey       string            `json:"bizKey"`       // 业务key
	BizStatus    ProcinstBizStatus `json:"bizStatus"`    // 业务状态
	BizHandleRes string            `json:"bizHandleRes"` // 业务处理结果
	TaskKey      string            `json:"taskKey"`      // 当前任务key
	Status       ProcinstStatus    `json:"status"`       // 状态
	Remark       string            `json:"remark"`
	EndTime      *time.Time        `json:"endTime"`
	Duration     int64             `json:"duration"` // 持续时间（开始到结束）
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
	ProcinstActive     ProcinstStatus = 1  // 流程实例正在执行中，当前有活动任务等待执行或者正在运行的流程节点
	ProcinstCompleted  ProcinstStatus = 2  // 流程实例已经成功执行完成，没有剩余任务或者等待事件
	ProcinstSuspended  ProcinstStatus = -1 // 流程实例被挂起，暂停执行，可能被驳回等待修改重新提交
	ProcinstTerminated ProcinstStatus = -2 // 流程实例被终止，可能是由于某种原因如被拒绝等导致流程无法正常执行
	ProcinstCancelled  ProcinstStatus = -3 // 流程实例被取消，通常是用户手动操作取消了流程的执行
)

var ProcinstStatusEnum = enumx.NewEnum[ProcinstStatus]("流程状态").
	Add(ProcinstActive, "执行中").
	Add(ProcinstCompleted, "完成").
	Add(ProcinstSuspended, "挂起").
	Add(ProcinstTerminated, "终止").
	Add(ProcinstCancelled, "取消")

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

	ProcinstId uint64 `json:"procinstId"` // 流程实例id
	TaskKey    string `json:"taskKey"`    // 当前任务key
	TaskName   string `json:"taskName"`   // 当前任务名称
	Assignee   string `json:"assignee"`   // 分配到该任务的用户

	Status   ProcinstTaskStatus `json:"status"` // 状态
	Remark   string             `json:"remark"`
	EndTime  *time.Time         `json:"endTime"`
	Duration int64              `json:"duration"` // 持续时间（开始到结束）
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
