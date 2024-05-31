package entity

import (
	"encoding/json"
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// 流程定义信息
type Procdef struct {
	model.Model

	Name   string        `json:"name" form:"name"`     // 名称
	DefKey string        `json:"defKey" form:"defKey"` //
	Tasks  string        `json:"tasks"`                // 审批节点任务信息
	Status ProcdefStatus `json:"status"`               // 状态
	Remark string        `json:"remark"`
}

func (p *Procdef) TableName() string {
	return "t_flow_procdef"
}

// 获取审批节点任务列表
func (p *Procdef) GetTasks() []*ProcdefTask {
	var tasks []*ProcdefTask
	err := json.Unmarshal([]byte(p.Tasks), &tasks)
	if err != nil {
		logx.ErrorTrace("解析procdef tasks失败", err)
		return tasks
	}

	return tasks
}

type ProcdefTask struct {
	Name    string `json:"name" form:"name"`       // 审批节点任务名称
	TaskKey string `json:"taskKey" form:"taskKey"` // 任务key
	UserId  string `json:"userId"`                 // 审批人
}

type ProcdefStatus int8

const (
	ProcdefStatusEnable  ProcdefStatus = 1
	ProcdefStatusDisable ProcdefStatus = -1
)

var ProcdefStatusEnum = enumx.NewEnum[ProcdefStatus]("流程定义状态").
	Add(ProcdefStatusEnable, "启用").
	Add(ProcdefStatusDisable, "禁用")
