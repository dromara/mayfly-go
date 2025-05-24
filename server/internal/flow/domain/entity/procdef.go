package entity

import (
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/utils/collx"
	"mayfly-go/pkg/utils/jsonx"
	"mayfly-go/pkg/utils/stringx"
	"strings"

	"github.com/may-fly/cast"
)

// 流程定义信息
type Procdef struct {
	model.Model

	Name      string        `json:"name" form:"name" gorm:"size:150;comment:流程名称"`                 // 名称
	DefKey    string        `json:"defKey" form:"defKey" gorm:"not null;size:100;comment:流程定义key"` //
	FlowDef   string        `json:"flowDef" gorm:"type:text;comment:流程定义信息"`                       // 流程定义信息
	Status    ProcdefStatus `json:"status" gorm:"comment:状态"`                                      // 状态
	Condition *string       `json:"condition" gorm:"type:text;comment:触发审批的条件（计算结果返回1则需要启用该流程）"`   // 触发审批的条件（计算结果返回1则需要启用该流程）
	Remark    *string       `json:"remark" gorm:"size:255;"`
}

func (p *Procdef) TableName() string {
	return "t_flow_procdef"
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
	flow, err := jsonx.To[*FlowDef](p.FlowDef)
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

func (p *FlowDef) GetNextNodes(key string, vars collx.M) []*FlowNode {
	edges := p.GetEdgeBySourceNode(key)
	targetNodeKeys := make([]string, 0)

	for _, edge := range edges {
		if edge.MatchCondition(vars) {
			targetNodeKeys = append(targetNodeKeys, edge.TargetNodeKey)
		}
	}

	return p.GetNodes(targetNodeKeys...)
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

	Condition string `json:"condition"` // 跳转条件
}

// MatchCondition 匹配条件
func (p *FlowEdge) MatchCondition(vars collx.M) bool {
	if p.Condition == "" {
		return true
	}

	// 解析条件
	res, err := stringx.TemplateParse(p.Condition, vars)
	if err != nil {
		logx.Warnf("parse condition error, edge: %s, err: %s", p.Condition, err.Error())
		return false
	}

	return cast.ToBool(res)
}
