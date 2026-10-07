package form

import (
	"mayfly-go/internal/flow/domain/entity"
)

type Procdef struct {
	Id        uint64               `json:"id"`
	Name      string               `json:"name" binding:"required"` // 名称
	DefKey    string               `json:"defKey" binding:"required"`
	Status    entity.ProcdefStatus `json:"status" binding:"required"`
	Remark    string               `json:"remark"`
	MsgTmplId uint64               `json:"msgTmplId"`

	// TriggerPolicy 触发策略，结构与引擎注册表对应
	TriggerPolicy *entity.TriggerPolicy `json:"triggerPolicy"`

	CodePaths []string `json:"codePaths"`
}

// ProcdefSimulate 策略试算入参。只接受草稿策略：试算不落库也不回读已保存策略
type ProcdefSimulate struct {
	BizType string                `json:"bizType" binding:"required"`
	Policy  *entity.TriggerPolicy `json:"policy"`
	Raw     map[string]string     `json:"raw"`
}

type ProcdefFlow struct {
	Id   uint64          `json:"id" binding:"required"`
	Flow *entity.FlowDef `json:"flow" binding:"required"`
}

// RuleSegment 可复用条件组表单。
// 条件树直接收下 entity.RuleNode：与触发策略里用的是同一个结构，前端提交什么就存什么
type RuleSegment struct {
	Id      uint64 `json:"id"`
	Ref     string `json:"ref" binding:"required"`
	Name    string `json:"name" binding:"required"`
	BizType string `json:"bizType" binding:"required"`
	Remark  string `json:"remark"`

	RuleNode *entity.RuleNode `json:"ruleNode"`
}
