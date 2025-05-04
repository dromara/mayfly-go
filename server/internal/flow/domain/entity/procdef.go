package entity

import (
	"encoding/json"
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/stringx"
	"strings"
)

// 流程定义信息
type Procdef struct {
	model.Model

	Name      string        `json:"name" form:"name" gorm:"size:150;comment:流程名称"`                 // 名称
	DefKey    string        `json:"defKey" form:"defKey" gorm:"not null;size:100;comment:流程定义key"` //
	Tasks     string        `json:"tasks" gorm:"not null;size:3000;comment:审批节点任务信息"`              // 审批节点任务信息
	Status    ProcdefStatus `json:"status" gorm:"comment:状态"`                                      // 状态
	Condition *string       `json:"condition" gorm:"type:text;comment:触发审批的条件（计算结果返回1则需要启用该流程）"`   // 触发审批的条件（计算结果返回1则需要启用该流程）
	Remark    *string       `json:"remark" gorm:"size:255;"`
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

// MatchCondition 是否匹配审批条件，匹配则需要启用该流程
// @param bizType 业务类型
// @param param 业务参数
// Condition返回值为1，则表面该操作需要启用流程
func (p *Procdef) MatchCondition(bizType string, param map[string]any) bool {
	if p.Condition == nil || *p.Condition == "" {
		return true
	}

	res, err := stringx.TemplateResolve(*p.Condition, collx.Kvs("bizType", bizType, "param", param))
	if err != nil {
		logx.ErrorTrace("parse condition error", err.Error())
		return true
	}
	return strings.TrimSpace(res) == "1"
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
