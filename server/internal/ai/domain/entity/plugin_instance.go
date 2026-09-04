package entity

import (
	"mayfly-go/pkg/model"
)

// 插件实例类型常量（对齐 tokhub plugin_type，代码级类型注册表维护，见 application/plugin_type.go）
const (
	// PluginTypeSkill 技能插件（config 引用 t_ai_skill，Managed 模式）
	PluginTypeSkill = "skill"
	// PluginTypeMcp MCP 服务器插件（连接配置内联 config）
	PluginTypeMcp = "mcp"
)

// 插件实例健康状态常量（对齐 tokhub PluginStatus）
const (
	// PluginStatusUnknown 未知（未做过连接验证）
	PluginStatusUnknown = 0
	// PluginStatusHealthy 健康（最近一次工具发现成功）
	PluginStatusHealthy = 1
	// PluginStatusError 异常（最近一次工具发现失败）
	PluginStatusError = 2
)

// PluginInstance 插件实例（对齐 tokhub t_plugin_instance，剪裁租户与 definition 关联维度）
//
// 统一插件视图的唯一事实源：技能/MCP/未来插件类型均注册为一行实例，
// 列表分页、启停开关、Agent 装配均只读本表（单表查询，无跨表聚合）。
// 各插件类型的专业数据由 config JSON 按类型 schema 约定：
//   - skill: {"skillCode":"xxx"} 引用 t_ai_skill（Managed 模式）
//   - mcp:   {"url":"...","headers":"...","timeoutSec":30} 连接配置内联
//
// 用 ModelNLD（物理删除）：code 唯一索引，软删行会阻塞同 code 重建。
type PluginInstance struct {
	model.ModelNLD

	// Code 实例唯一标识（技能实例=技能 code，MCP 实例=服务器 code）
	Code string `gorm:"column:code;size:64;not null;uniqueIndex:uk_ai_plugin_instance_code;comment:实例唯一标识" json:"code"`
	// PluginType 插件类型标识（skill / mcp / ...）
	PluginType string `gorm:"column:plugin_type;size:32;not null;index:idx_ai_plugin_instance_type;comment:插件类型" json:"pluginType"`
	// Name 实例名称（默认随源实体，可独立修改）
	Name string `gorm:"column:name;size:128;not null;comment:实例名称" json:"name"`
	// Description 实例描述
	Description string `gorm:"column:description;size:512;comment:实例描述" json:"description"`
	// Config 插件类型配置 JSON（schema 由类型注册表约定）
	Config string `gorm:"column:config;size:2000;comment:插件配置JSON" json:"config"`
	// Enabled 是否启用：1=启用（Agent 装配期注入）
	Enabled int `gorm:"column:enabled;not null;default:1;comment:是否启用" json:"enabled"`
	// Status 健康状态：0=未知 1=健康 2=异常（工具发现时回写）
	Status int `gorm:"column:status;not null;default:0;comment:健康状态" json:"status"`
}

func (p *PluginInstance) TableName() string {
	return "t_ai_plugin_instance"
}
