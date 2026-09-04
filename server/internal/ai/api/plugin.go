package api

import (
	"bytes"
	"io"
	"strconv"

	"mayfly-go/internal/ai/api/form"
	"mayfly-go/internal/ai/application"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/req"
)

// maxSkillZipBytes zip 上传大小上限（与 application.maxZipBytes 一致，用于上传预检）
const maxSkillZipBytes = 10 << 20

// AiPlugin 插件管理 API（统一插件实例 + 技能）
type AiPlugin struct {
	skillPlugin    application.SkillPlugin       `inject:"T"`
	pluginInstance application.PluginInstanceApp `inject:"T"`
}

// ReqConfs 获取插件管理相关的请求配置
func (a *AiPlugin) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
		// 插件类型注册表（前端类型选择器数据源）
		req.NewGet("/plugin/types", a.ListPluginTypes),
		// 插件实例管理（统一插件视图，t_ai_plugin_instance 单表分页）
		req.NewGet("/plugin/instances", a.ListInstances),
		req.NewPost("/plugin/instances", a.CreateInstance),
		req.NewGet("/plugin/instances/:id", a.GetInstance),
		req.NewPut("/plugin/instances/:id", a.UpdateInstance),
		req.NewDelete("/plugin/instances/:id", a.DeleteInstance),
		req.NewPost("/plugin/instances/:id/discover", a.DiscoverInstanceTools),
		req.NewPost("/plugin/instances/:id/enabled", a.ToggleInstanceEnabled),
		// 技能管理
		req.NewGet("/plugin/skills", a.ListSkills),
		req.NewPost("/plugin/skills", a.CreateSkill),
		req.NewGet("/plugin/skills/:id", a.GetSkill),
		req.NewPut("/plugin/skills/:id", a.UpdateSkill),
		req.NewDelete("/plugin/skills/:id", a.DeleteSkill),
		req.NewGet("/plugin/skills/:id/instructions", a.GetSkillInstructions),
		req.NewGet("/plugin/skills/:id/resources", a.ListSkillResources),
		req.NewPost("/plugin/skills/:id/resources", a.UpsertSkillResource),
		req.NewDelete("/plugin/skills/:id/resources", a.DeleteSkillResource),
		req.NewPost("/plugin/skills/import", a.ImportSkillZip),
		req.NewGet("/plugin/skills/:id/export", a.ExportSkillZip),
		req.NewPost("/plugin/skills/:id/publish", a.PublishSkill),
		req.NewPost("/plugin/skills/:id/unpublish", a.UnpublishSkill),
	}
	return req.NewConfs("/ai", reqs[:]...)
}

// ============== 技能管理 ==============

func (a *AiPlugin) ListSkills(rc *req.Ctx) {
	skills, err := a.skillPlugin.ListSkills(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = skills
}

func (a *AiPlugin) GetSkill(rc *req.Ctx) {
	skill, err := a.skillPlugin.GetById(pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = skill
}

func (a *AiPlugin) CreateSkill(rc *req.Ctx) {
	formBody := rc.BindJson[form.SkillSaveRequest]()
	skill, err := a.skillPlugin.CreateSkill(rc.MetaCtx, &application.SkillSaveReq{
		Code:         formBody.Code,
		Description:  formBody.Description,
		AllowedTools: formBody.AllowedTools,
		Instructions: formBody.Instructions,
	})
	biz.ErrIsNil(err)
	rc.ResData = skill
}

func (a *AiPlugin) UpdateSkill(rc *req.Ctx) {
	formBody := rc.BindJson[form.SkillSaveRequest]()
	biz.ErrIsNil(a.skillPlugin.UpdateSkill(rc.MetaCtx, pathParamId(rc), &application.SkillSaveReq{
		Code:         formBody.Code,
		Description:  formBody.Description,
		AllowedTools: formBody.AllowedTools,
		Instructions: formBody.Instructions,
	}))
}

func (a *AiPlugin) DeleteSkill(rc *req.Ctx) {
	biz.ErrIsNil(a.skillPlugin.DeleteSkill(rc.MetaCtx, pathParamId(rc)))
}

// GetSkillInstructions 读取 SKILL.md 原文（frontmatter + 正文）
func (a *AiPlugin) GetSkillInstructions(rc *req.Ctx) {
	instructions, err := a.skillPlugin.GetInstructions(rc.MetaCtx, pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = map[string]string{"content": instructions}
}

func (a *AiPlugin) PublishSkill(rc *req.Ctx) {
	biz.ErrIsNil(a.skillPlugin.Publish(rc.MetaCtx, pathParamId(rc)))
}

func (a *AiPlugin) UnpublishSkill(rc *req.Ctx) {
	biz.ErrIsNil(a.skillPlugin.Unpublish(rc.MetaCtx, pathParamId(rc)))
}

// ============== 技能资源 ==============

func (a *AiPlugin) ListSkillResources(rc *req.Ctx) {
	resources, err := a.skillPlugin.ListResources(rc.MetaCtx, pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = resources
}

func (a *AiPlugin) UpsertSkillResource(rc *req.Ctx) {
	formBody := rc.BindJson[form.SkillResourceRequest]()
	res, err := a.skillPlugin.UpsertResource(rc.MetaCtx, pathParamId(rc), formBody.Path, formBody.Content)
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AiPlugin) DeleteSkillResource(rc *req.Ctx) {
	resPath := rc.Query("path")
	biz.ErrIsNil(a.skillPlugin.DeleteResource(rc.MetaCtx, pathParamId(rc), resPath))
}

// ============== 技能 zip 导入导出 ==============

// ImportSkillZip multipart 上传 zip 导入（同 code upsert 覆盖 + patch 递增）
func (a *AiPlugin) ImportSkillZip(rc *req.Ctx) {
	fileHeader, err := rc.FormFile("file")
	biz.ErrIsNil(err)
	// 先用声明大小预检，避免无界读入内存（声明可伪造，ImportZip 内还有二次校验兑底）
	if fileHeader.Size > maxSkillZipBytes {
		biz.ErrIsNil(errorx.NewBiz("zip file too large"))
	}
	file, err := fileHeader.Open()
	biz.ErrIsNil(err)
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSkillZipBytes+1))
	biz.ErrIsNil(err)
	skill, err := a.skillPlugin.ImportZip(rc.MetaCtx, data)
	biz.ErrIsNil(err)
	rc.ResData = skill
}

// ExportSkillZip 导出技能 zip（{code}.zip 下载）
func (a *AiPlugin) ExportSkillZip(rc *req.Ctx) {
	filename, data, err := a.skillPlugin.ExportZip(rc.MetaCtx, pathParamId(rc))
	biz.ErrIsNil(err)
	rc.Download(bytes.NewReader(data), filename)
}

// ============== 插件类型 / 实例 ==============

// ListPluginTypes 已注册插件类型列表（类型选择器数据源）
func (a *AiPlugin) ListPluginTypes(rc *req.Ctx) {
	rc.ResData = application.ListPluginTypes()
}

// ListInstances 插件实例分页列表（keyword 匹配 name/code/description，type 过滤）
func (a *AiPlugin) ListInstances(rc *req.Ctx) {
	res, err := a.pluginInstance.ListInstances(rc.MetaCtx, rc.Query("keyword"), rc.Query("type"), rc.GetPageParam())
	biz.ErrIsNil(err)
	rc.ResData = res
}

func (a *AiPlugin) GetInstance(rc *req.Ctx) {
	instance, err := a.pluginInstance.GetInstanceDetail(pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = instance
}

func (a *AiPlugin) CreateInstance(rc *req.Ctx) {
	formBody := rc.BindJson[form.PluginInstanceSaveRequest]()
	instance, err := a.pluginInstance.CreateInstance(rc.MetaCtx, toInstanceSaveReq(formBody))
	biz.ErrIsNil(err)
	rc.ResData = instance
}

func (a *AiPlugin) UpdateInstance(rc *req.Ctx) {
	formBody := rc.BindJson[form.PluginInstanceSaveRequest]()
	biz.ErrIsNil(a.pluginInstance.UpdateInstance(rc.MetaCtx, pathParamId(rc), toInstanceSaveReq(formBody)))
}

func (a *AiPlugin) DeleteInstance(rc *req.Ctx) {
	biz.ErrIsNil(a.pluginInstance.DeleteInstance(rc.MetaCtx, pathParamId(rc)))
}

// DiscoverInstanceTools 连接测试 + 工具发现（实时连接，不落库；健康状态回写）
func (a *AiPlugin) DiscoverInstanceTools(rc *req.Ctx) {
	tools, err := a.pluginInstance.DiscoverTools(rc.MetaCtx, pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = tools
}

func (a *AiPlugin) ToggleInstanceEnabled(rc *req.Ctx) {
	formBody := rc.BindJson[form.PluginInstanceToggleRequest]()
	biz.ErrIsNil(a.pluginInstance.ToggleEnabled(rc.MetaCtx, pathParamId(rc), formBody.Enabled))
}

// toInstanceSaveReq 表单转应用层请求（config 对象转 JSON 字符串）
func toInstanceSaveReq(formBody *form.PluginInstanceSaveRequest) *application.InstanceSaveReq {
	return &application.InstanceSaveReq{
		PluginType:  formBody.PluginType,
		Code:        formBody.Code,
		Name:        formBody.Name,
		Description: formBody.Description,
		Config:      string(formBody.Config),
	}
}

// ============== 工具函数 ==============

// pathParamId 解析路径参数 :id
func pathParamId(rc *req.Ctx) uint64 {
	id, err := strconv.ParseUint(rc.PathParam("id"), 10, 64)
	biz.ErrIsNilAppendErr(err, "invalid id param: %s")
	return id
}
