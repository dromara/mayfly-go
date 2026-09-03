package api

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"

	"mayfly-go/internal/ai/api/form"
	"mayfly-go/internal/ai/application"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/pkg/biz"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/model"
	"mayfly-go/pkg/req"
)

// maxSkillZipBytes zip 上传大小上限（与 application.maxZipBytes 一致，用于上传预检）
const maxSkillZipBytes = 10 << 20

// AiPlugin 插件管理 API（技能 + MCP 服务器）
type AiPlugin struct {
	skillPlugin application.SkillPlugin `inject:"T"`
	mcpPlugin   application.McpPlugin   `inject:"T"`
}

// ReqConfs 获取插件管理相关的请求配置
func (a *AiPlugin) ReqConfs() *req.Confs {
	reqs := [...]*req.Conf{
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
		// MCP 服务器管理
		req.NewGet("/plugin/mcp/servers", a.ListMcpServers),
		req.NewPost("/plugin/mcp/servers", a.CreateMcpServer),
		req.NewGet("/plugin/mcp/servers/:id", a.GetMcpServer),
		req.NewPut("/plugin/mcp/servers/:id", a.UpdateMcpServer),
		req.NewDelete("/plugin/mcp/servers/:id", a.DeleteMcpServer),
		req.NewPost("/plugin/mcp/servers/:id/discover", a.DiscoverMcpTools),
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

// ============== MCP 服务器管理 ==============

func (a *AiPlugin) ListMcpServers(rc *req.Ctx) {
	servers, err := a.mcpPlugin.ListServers(rc.MetaCtx)
	biz.ErrIsNil(err)
	rc.ResData = servers
}

func (a *AiPlugin) GetMcpServer(rc *req.Ctx) {
	server, err := a.mcpPlugin.GetById(pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = server
}

func (a *AiPlugin) CreateMcpServer(rc *req.Ctx) {
	formBody := rc.BindJson[form.McpServerSaveRequest]()
	validateMcpServerForm(formBody)
	biz.ErrIsNil(a.mcpPlugin.Insert(rc.MetaCtx, &entity.McpServer{
		Code:        formBody.Code,
		Name:        formBody.Name,
		Description: formBody.Description,
		Url:         formBody.Url,
		Headers:     formBody.Headers,
		TimeoutSec:  defaultMcpTimeout(formBody.TimeoutSec),
		Enabled:     defaultMcpEnabled(formBody.Enabled),
	}))
}

func (a *AiPlugin) UpdateMcpServer(rc *req.Ctx) {
	formBody := rc.BindJson[form.McpServerSaveRequest]()
	validateMcpServerForm(formBody)
	server, err := a.mcpPlugin.GetById(pathParamId(rc))
	biz.ErrIsNil(err)
	// map 更新使零值可写（GORM 结构体更新默认跳过零值）：enabled=0 为显式停用，
	// 不能经 defaultMcpEnabled 强转；code 为业务标识不在更新列
	biz.ErrIsNil(a.mcpPlugin.UpdateByCond(rc.MetaCtx, map[string]any{
		"name":        formBody.Name,
		"description": formBody.Description,
		"url":         formBody.Url,
		"headers":     formBody.Headers,
		"timeout_sec": defaultMcpTimeout(formBody.TimeoutSec),
		"enabled":     formBody.Enabled,
	}, model.NewCond().Eq("id", server.Id)))
}

func (a *AiPlugin) DeleteMcpServer(rc *req.Ctx) {
	biz.ErrIsNil(a.mcpPlugin.DeleteById(rc.MetaCtx, pathParamId(rc)))
}

// DiscoverMcpTools 连接测试 + 工具发现（实时连接，不落库）
func (a *AiPlugin) DiscoverMcpTools(rc *req.Ctx) {
	tools, err := a.mcpPlugin.DiscoverTools(rc.MetaCtx, pathParamId(rc))
	biz.ErrIsNil(err)
	rc.ResData = tools
}

// ============== 工具函数 ==============

// pathParamId 解析路径参数 :id
func pathParamId(rc *req.Ctx) uint64 {
	id, err := strconv.ParseUint(rc.PathParam("id"), 10, 64)
	biz.ErrIsNilAppendErr(err, "invalid id param: %s")
	return id
}

// validateMcpServerForm MCP 服务器表单校验（headers 须为合法 JSON 对象）
func validateMcpServerForm(formBody *form.McpServerSaveRequest) {
	if formBody.Headers != "" {
		parsed := map[string]string{}
		if err := json.Unmarshal([]byte(formBody.Headers), &parsed); err != nil {
			biz.ErrIsNil(errorx.NewBiz("headers must be a valid JSON object"))
		}
	}
}

// defaultMcpTimeout 超时默认值（<=0 时 30s）
func defaultMcpTimeout(timeoutSec int) int {
	if timeoutSec <= 0 {
		return 30
	}
	return timeoutSec
}

// defaultMcpEnabled 创建场景的启用默认值（未传时默认启用；更新场景不适用，0 为显式停用）
func defaultMcpEnabled(enabled int) int {
	if enabled == 0 {
		return 1
	}
	return enabled
}
