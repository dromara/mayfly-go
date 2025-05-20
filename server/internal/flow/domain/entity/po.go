package entity

import (
	"mayfly-go/pkg/model"
	"time"
)

type ProcdefPagePO struct {
	model.Model

	Name      string        `json:"name" form:"name" gorm:"size:150;comment:流程名称"`                 // 名称
	DefKey    string        `json:"defKey" form:"defKey" gorm:"not null;size:100;comment:流程定义key"` //
	Status    ProcdefStatus `json:"status" gorm:"comment:状态"`                                      // 状态
	Condition *string       `json:"condition" gorm:"type:text;comment:触发审批的条件（计算结果返回1则需要启用该流程）"`   // 触发审批的条件（计算结果返回1则需要启用该流程）
	Remark    *string       `json:"remark" gorm:"size:255;"`
}

type ProcinstTaskPO struct {
	Id         uint64 `json:"id"`
	ProcinstId uint64 `json:"procinstId"` // 流程实例id
	NodeKey    string `json:"nodeKey"`    // 当前任务key
	NodeName   string `json:"nodeName"`   // 当前任务名称

	BizKey string `json:"bizKey"`

	Candidate string `json:"candidate"` // 处理候选人
	Handler   string `json:"handler"`   // 处理人

	Status     ProcinstTaskStatus `json:"status"`   // 状态
	Remark     string             `json:"remark"`   // 审批意见
	Duration   int64              `json:"duration"` // 持续时间（开始到结束）
	CreateTime *time.Time         `json:"createTime"`
	EndTime    *time.Time         `json:"endTime"`
}
