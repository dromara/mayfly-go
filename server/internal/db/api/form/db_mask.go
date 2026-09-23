package form

import "mayfly-go/internal/db/domain/entity"

// DbMaskRuleForm 脱敏规则表单
type DbMaskRuleForm struct {
	Id        uint64                `json:"id"`
	Name      string                `binding:"required" json:"name"`
	MatchType entity.MaskMatchType  `binding:"required" json:"matchType"`
	Pattern   string                `binding:"required" json:"pattern"`
	Algorithm string                `binding:"required" json:"algorithm"`
	Params    string                `json:"params"`
	Status    entity.MaskRuleStatus `json:"status"`
	Weight    int                   `json:"weight"`
	Remark    string                `json:"remark"`
}

// DbMaskColumnForm 脱敏列标签表单
type DbMaskColumnForm struct {
	Id         uint64                  `json:"id"`
	InstanceId uint64                  `binding:"required" json:"instanceId"`
	DbName     string                  `json:"dbName"`
	TableName  string                  `json:"tableName"`
	ColumnName string                  `json:"columnName"`
	Action     entity.MaskColumnAction `binding:"required" json:"action"`
	RuleId     uint64                  `json:"ruleId"`
	Algorithm  string                  `json:"algorithm"`
	Params     string                  `json:"params"`
	Remark     string                  `json:"remark"`
}
