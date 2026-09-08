package entity

import (
	"mayfly-go/pkg/model"
)

// 脱敏规则状态
const (
	MaskRuleStatusEnabled  int8 = 1 // 启用
	MaskRuleStatusDisabled int8 = 0 // 停用
)

// 脱敏规则匹配类型
const (
	MaskMatchTypeRegex  int8 = 1 // 正则匹配
	MaskMatchTypeExact  int8 = 2 // 精确匹配
	MaskMatchTypePrefix int8 = 3 // 前缀匹配
)

// DbMaskRule 数据库脱敏规则（全局列名规则库）
type DbMaskRule struct {
	model.Model `orm:"-"`

	Name      string `json:"name" gorm:"size:64;not null;comment:规则名称"`
	MatchType int8   `json:"matchType" gorm:"not null;comment:匹配类型:1正则 2精确 3前缀"`
	Pattern   string `json:"pattern" gorm:"size:255;not null;comment:匹配模式"`
	Algorithm string `json:"algorithm" gorm:"size:32;not null;comment:脱敏算法"`
	Params    string `json:"params" gorm:"size:500;comment:算法参数json"`
	Status    int8   `json:"status" gorm:"not null;default:1;comment:状态:1启用 0停用"`
	Weight    int    `json:"weight" gorm:"default:0;comment:权重，越大越先匹配"`
	Remark    string `json:"remark" gorm:"size:255;comment:备注"`
}

func (DbMaskRule) TableName() string {
	return "t_db_mask_rule"
}

// 列标签动作
const (
	MaskColumnActionBind   int8 = 1 // 绑定规则：对该列执行指定算法
	MaskColumnActionExempt int8 = 2 // 豁免：该列不脱敏
)

// DbMaskColumn 脱敏列标签（实例级覆盖与豁免，优先级高于全局规则）
type DbMaskColumn struct {
	model.Model `orm:"-"`

	InstanceId uint64 `json:"instanceId" gorm:"not null;comment:数据库实例id"`
	DbName     string `json:"dbName" gorm:"size:128;not null;default:'';comment:库名，空为所有库"`
	// MatchTable 标签匹配的表名；不能直接命名为TableName，与gorm.TableName()表名方法冲突
	MatchTable string `json:"tableName" gorm:"column:table_name;size:128;not null;default:'';comment:表名，空为所有表"`
	ColumnName string `json:"columnName" gorm:"size:128;not null;default:'';comment:列名，空为所有列"`
	Action     int8   `json:"action" gorm:"not null;comment:动作:1绑定规则 2豁免"`
	RuleId     uint64 `json:"ruleId" gorm:"default:0;comment:绑定的规则id"`
	Algorithm  string `json:"algorithm" gorm:"size:32;comment:直接指定的算法，优先于规则"`
	Params     string `json:"params" gorm:"size:500;comment:算法参数json"`
	Remark     string `json:"remark" gorm:"size:255;comment:备注"`
}

func (DbMaskColumn) TableName() string {
	return "t_db_mask_column"
}

// MaskRuleQuery 脱敏规则查询条件
type MaskRuleQuery struct {
	model.PageParam

	Name    string `json:"name" form:"name"`
	Keyword string `json:"keyword" form:"keyword"`
	Status  *int8  `json:"status" form:"status"`
}

// MaskColumnQuery 列标签查询条件
type MaskColumnQuery struct {
	model.PageParam

	InstanceId uint64 `json:"instanceId" form:"instanceId"`
	DbName     string `json:"dbName" form:"dbName"`
	TableName  string `json:"tableName" form:"tableName"`
	ColumnName string `json:"columnName" form:"columnName"`
}
