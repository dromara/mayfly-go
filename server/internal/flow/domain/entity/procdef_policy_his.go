package entity

import (
	"mayfly-go/pkg/enumx"
	"mayfly-go/pkg/model"
)

// ProcdefPolicyAction 触发策略的变更动作
type ProcdefPolicyAction int8

const (
	// ProcdefPolicyActionCreate 首次配置：没有「变更前」，所以它与「修改」必须区分开，
	// 否则时间线第一条会显示成「从空策略改成本策略」，读起来像是有人删过规则
	ProcdefPolicyActionCreate ProcdefPolicyAction = 1
	// ProcdefPolicyActionUpdate 修改已存在的策略
	ProcdefPolicyActionUpdate ProcdefPolicyAction = 2
	// ProcdefPolicyActionDelete 随流程定义一起删除
	ProcdefPolicyActionDelete ProcdefPolicyAction = 3
)

var ProcdefPolicyActionEnum = enumx.NewEnum[ProcdefPolicyAction]("策略变更动作").
	Add(ProcdefPolicyActionCreate, "新建").
	Add(ProcdefPolicyActionUpdate, "修改").
	Add(ProcdefPolicyActionDelete, "删除")

// ProcdefPolicyHis 流程定义触发策略的变更记录。
//
// 只存变更事实（前后两份策略快照），不存算好的 diff：
// 规则的展示文案随字段字典与 i18n 演进，把 diff 落库等于把文案焊死在历史里，
// 将来加了一个检查项，历史条目的描述就会开始说不通
type ProcdefPolicyHis struct {
	model.CreateModel

	ProcdefId uint64 `json:"procdefId" gorm:"not null;index:idx_pph_procdef_id;comment:流程定义id"`

	// ProcdefName 变更时的流程名称快照：流程后来改名或删掉后，时间线仍然能说明当时改的是哪条
	ProcdefName string `json:"procdefName" gorm:"size:150;comment:流程名称快照"`

	Action ProcdefPolicyAction `json:"action" gorm:"comment:变更动作"`

	// Before 变更前的策略，新建时为空
	Before *TriggerPolicy `json:"before" gorm:"type:text;serializer:json;comment:变更前策略"`

	// After 变更后的策略，删除时为空
	After *TriggerPolicy `json:"after" gorm:"type:text;serializer:json;comment:变更后策略"`
}

func (p *ProcdefPolicyHis) TableName() string {
	return "t_flow_procdef_policy_his"
}
