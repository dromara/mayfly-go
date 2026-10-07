package entity

import (
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
)

// 流程定义信息
type Procdef struct {
	model.Model

	Name    string        `json:"name" form:"name" gorm:"size:150;comment:流程名称"`                 // 名称
	DefKey  string        `json:"defKey" form:"defKey" gorm:"not null;size:100;comment:流程定义key"` //
	FlowDef string        `json:"flowDef" gorm:"type:text;comment:流程定义信息"`                       // 流程定义信息
	Status  ProcdefStatus `json:"status" gorm:"comment:状态"`                                      // 状态
	Remark  *string       `json:"remark" gorm:"size:255;"`

	// TriggerPolicy 触发策略，决定哪些资源操作需要走该审批流程
	TriggerPolicy *TriggerPolicy `json:"triggerPolicy" gorm:"type:text;serializer:json;comment:触发策略json"`
}

func (p *Procdef) TableName() string {
	return "t_flow_procdef"
}

type ProcdefStatus int8

const (
	ProcdefStatusEnable  ProcdefStatus = 1
	ProcdefStatusDisable ProcdefStatus = -1
)

var ProcdefStatusEnum = enumx.NewEnum[ProcdefStatus]("流程定义状态").
	Add(ProcdefStatusEnable, "启用").
	Add(ProcdefStatusDisable, "禁用")

// GetFlowDef 获取流程定义信息
func (p *Procdef) GetFlowDef() *FlowDef {
	if p.FlowDef == "" {
		return nil
	}
	flow, err := jsonx.ToByStr[FlowDef](p.FlowDef)
	if err != nil {
		logx.ErrorTrace("parse flow def failed", err)
		return flow
	}

	return flow
}

// FlowDef 流程定义-流程内容
type FlowDef struct {
	Nodes []*FlowNode `json:"nodes" form:"nodes"`
	Edges []*FlowEdge `json:"edges" form:"edges"`
}

func (p *FlowDef) GetNodes(keys ...string) []*FlowNode {
	result := make([]*FlowNode, 0)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for _, node := range p.Nodes {
		if keySet[node.Key] {
			result = append(result, node)
		}
	}
	return result
}

func (p *FlowDef) GetNodeByType(nodeType FlowNodeType) []*FlowNode {
	return collx.ArrayFilter(p.Nodes, func(node *FlowNode) bool {
		return node.Type == nodeType
	})
}

func (p *FlowDef) GetEdgeBySourceNode(sourceNodeKey string) []*FlowEdge {
	return collx.ArrayFilter(p.Edges, func(edge *FlowEdge) bool {
		return edge.SourceNodeKey == sourceNodeKey
	})
}

// GetNextNodes 按跳转条件筛出该节点的后继节点。
//
// matched 由调用方提供：条件用哪些字段、怎么求值都登记在触发引擎侧，实体层只负责图结构，
// 因此新增条件语义（可复用条件组、新字段）不必改动这里。
// matched 返回 error 表示无法判定，调用方据此中断流转而不是猜一条分支
func (p *FlowDef) GetNextNodes(key string, matched func(*FlowEdge) (bool, error)) ([]*FlowNode, error) {
	edges := p.GetEdgeBySourceNode(key)
	targetNodeKeys := make([]string, 0, len(edges))

	for _, edge := range edges {
		matchedEdge, err := matched(edge)
		if err != nil {
			return nil, err
		}
		if matchedEdge {
			targetNodeKeys = append(targetNodeKeys, edge.TargetNodeKey)
		}
	}

	return p.GetNodes(targetNodeKeys...), nil
}

// FlowNode 流程定义-流程节点
type FlowNode struct {
	model.ExtraData

	Name string       `json:"name" form:"name"` // 审批节点任务名称
	Key  string       `json:"key" form:"key"`   // 任务key
	Type FlowNodeType `json:"type" form:"type"` // 任务节点类型
}

type FlowNodeType string

// FlowEdge 流程定义-流程节点-跳转
type FlowEdge struct {
	model.ExtraData

	Name          string `json:"name" form:"name"`                   // 跳转名
	Key           string `json:"key" form:"key"`                     // 跳转key
	SourceNodeKey string `json:"sourceNodeKey" form:"sourceNodeKey"` // 源节点key
	TargetNodeKey string `json:"targetNodeKey" form:"targetNodeKey"` // 目标节点key
}

// FlowEdgeConditionKey 跳转条件在连线 extra 中的键名。
// 流程设计器把节点与连线的属性统一写进 extra，条件树也存在这里，与其它属性同一取法
const FlowEdgeConditionKey = "condition"

// FlowNodeCompletionConditionKey 节点完成条件在节点 extra 中的键名
const FlowNodeCompletionConditionKey = "completionCondition"

// Condition 返回连线上的跳转条件，nil 表示未配置条件（默认流转）
func (p *FlowEdge) Condition() (*RuleNode, error) {
	return RuleNodeFromExtra(p.Extra, FlowEdgeConditionKey)
}
