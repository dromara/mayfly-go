package entity

import "mayfly-go/pkg/model"

// RuleSegment 可复用条件组：把一段判断（如「DBA 角色成员」「工作时段之外」）单独维护，
// 供触发策略与流程内的跳转/完成条件通过 segment 节点引用。
//
// 只存条件树本身、不存处置级别：同一个判断在不同规则里可以定不同级别，
// 级别属于「谁引用了它」，判断属于条件组，两者混在一起会让复用变成互相牵制
type RuleSegment struct {
	model.Model

	// Ref 条件组的稳定标识，条件树通过它引用。用可读标识而不是自增 id：
	// 导出策略 JSON、跨环境迁移时 id 会变，标识不会。
	//
	// 唯一性由应用层在未删除的行里校验，不建 DB 唯一索引：本表是软删除，
	// 唯一索引会把已删除的行一起算进来，删掉一个条件组后就再也建不出同名标识
	Ref string `json:"ref" gorm:"size:64;not null;index:idx_flow_segment_ref;comment:条件组标识"`

	// Name 条件组名称，管理员自拟的展示文本，不是 i18n key，原样展示
	Name string `json:"name" gorm:"size:100;not null;comment:条件组名称"`

	// BizType 条件组所属的字段字典（业务场景）。条件叶子只能引用本场景注册的字段，
	// 因此引用方必须与条件组同场景，否则保存时拒绝
	BizType string `json:"bizType" gorm:"size:64;not null;index:idx_flow_segment_biz_type;comment:业务场景"`

	Remark string `json:"remark" gorm:"size:255;comment:说明"`

	// RuleNode 条件树本体。字段名与物理列名保持一致（rule_node），
	// 不叫 Condition：那是 MySQL 保留字，本仓库已因它把 alert 的列改成 alert_condition
	RuleNode *RuleNode `json:"ruleNode" gorm:"type:text;serializer:json;comment:条件树json"`
}

func (r *RuleSegment) TableName() string {
	return "t_flow_rule_segment"
}
