// Package entity AI 插件域实体（对齐 tokhub plugin 体系，剪裁多租户维度）：
// 技能（Skill）+ 插件实例（PluginInstance，统一视图见 plugin_instance.go），
// 经「集成 → 插件管理」统一治理。
package entity

import (
	"mayfly-go/pkg/model"
)

// Skill 状态/来源常量（值域对齐 tokhub SkillStatus/SkillSource，字符串化存储）
const (
	// SkillStatusDraft 草稿（不注入 LLM）
	SkillStatusDraft = "draft"
	// SkillStatusPublished 已发布（注入 LLM）
	SkillStatusPublished = "published"

	// SkillSourceBuiltin 内置（随迁移落库的默认技能）
	SkillSourceBuiltin = "builtin"
	// SkillSourceImported zip 导入
	SkillSourceImported = "imported"
	// SkillSourceCustom 在线创建
	SkillSourceCustom = "custom"
)

// SkillMdPath 技能主文件路径（SKILL.md，L2 指令层）
const SkillMdPath = "SKILL.md"

// Skill 技能（对齐 tokhub t_skill，符合 Agent Skills 开放规范）
//
// 正文（SKILL.md）与附属资源存 t_ai_skill_resource，本表仅存 frontmatter
// 解析后的元数据（name/description/allowed_tools）与治理字段。
//
// 用 ModelNLD（物理删除）：code 有唯一索引，软删行会永久占坑导致同 code
// 重建/二次导入撞唯一键；且删除为不可恢复语义（前端确认文案一致）。
type Skill struct {
	model.ModelNLD

	// Code 业务唯一标识（暴露给 LLM，$code 显式提及与 skill_read 工具使用）
	Code string `gorm:"column:code;size:64;not null;uniqueIndex:uk_ai_skill_code;comment:技能唯一标识" json:"code"`
	// Name 技能名称（来自 frontmatter）
	Name string `gorm:"column:name;size:128;not null;comment:技能名称" json:"name"`
	// Description 技能描述（来自 frontmatter，L1 目录展示）
	Description string `gorm:"column:description;size:1024;comment:技能描述" json:"description"`
	// AllowedTools 预批准工具列表（空格分隔，来自 frontmatter）
	AllowedTools string `gorm:"column:allowed_tools;size:512;comment:预批准工具列表" json:"allowedTools"`
	// Version 语义化版本号（SemVer，导入同 code 时 patch 递增）
	Version string `gorm:"column:version;size:32;not null;default:1.0.0;comment:SemVer版本号" json:"version"`
	// Source 来源：builtin/imported/custom
	Source string `gorm:"column:source;size:16;not null;default:custom;comment:来源" json:"source"`
	// Status 状态：draft/published（仅 published 注入 LLM）
	Status string `gorm:"column:status;size:16;not null;default:draft;comment:状态" json:"status"`
}

func (s *Skill) TableName() string {
	return "t_ai_skill"
}

// SkillResource 技能资源（对齐 tokhub FsBackend /skills/{code}/... 的 DB 存储形态）
//
// SKILL.md 为主文件（path="SKILL.md"），其余为附属资源（references/、scripts/ 等）。
// 用 ModelNLD（物理删除）：(skill_id, path) 唯一索引，资源随技能存亡、无审计诉求，
// 软删行会阻塞同路径重建与 zip 覆盖导入。
type SkillResource struct {
	model.ModelNLD

	// SkillId 所属技能 ID
	SkillId uint64 `gorm:"column:skill_id;not null;uniqueIndex:uk_ai_skill_resource;comment:技能ID" json:"skillId"`
	// Path 技能内相对路径（禁 ../ 与绝对路径，SKILL.md 保留为主文件）
	Path string `gorm:"column:path;size:255;not null;uniqueIndex:uk_ai_skill_resource;comment:相对路径" json:"path"`
	// Content 文件文本内容
	Content string `gorm:"column:content;type:mediumtext;comment:文件内容" json:"content"`
	// Size 内容字节大小（对齐 tokhub UpsertResourceRequest.size）
	Size int `gorm:"column:size;not null;default:0;comment:内容字节大小" json:"size"`
}

func (r *SkillResource) TableName() string {
	return "t_ai_skill_resource"
}
