package entity

type ProcinstQuery struct {
	ProcdefId   uint64 `json:"procdefId" form:"procdefId"` // 流程定义id
	ProcdefName string `json:"procdefName"`                // 流程定义名称

	BizType string         `json:"bizType" form:"bizType"` // 业务类型
	BizKey  string         `json:"bizKey" form:"bizKey"`   // 业务key
	Status  ProcinstStatus `json:"status" form:"status"`   // 状态

	CreatorId uint64
}

type ProcinstTaskQuery struct {
	ProcinstId   uint64             `json:"procinstId"`   // 流程实例id
	ProcinstName string             `json:"procinstName"` // 流程实例名称
	BizType      string             `json:"bizType" form:"bizType"`
	BizKey       string             `json:"bizKey" form:"bizKey"` // 业务key
	Assignee     string             `json:"assignee"`             // 分配到该任务的用户
	Status       ProcinstTaskStatus `json:"status" form:"status"` // 状态
}
