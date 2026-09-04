package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/agent/ext/mcp/mcpext"
	"mayfly-go/internal/ai/domain/entity"
	"mayfly-go/internal/ai/domain/repository"
	"mayfly-go/internal/ai/infra/mcpclient"
	"mayfly-go/pkg/base"
	"mayfly-go/pkg/errorx"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/model"
)

// escapeLike 转义 LIKE 通配符，保证 keyword 按精确子串匹配（%/_/\ 均为字面量）
func escapeLike(kw string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(kw)
}

// PluginInstanceItem 插件实例分页视图（t_ai_plugin_instance 单表分页，无跨表合并）
type PluginInstanceItem struct {
	Id          uint64 `json:"id"`
	PluginType  string `json:"pluginType"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Config 类型化配置（原样 JSON 下发，前端按 pluginType 解读）
	Config json.RawMessage `json:"config"`
	// Enabled 启停（1/0）
	Enabled int `json:"enabled"`
	// Status 实例健康状态（0=unknown 1=healthy 2=error，discover 时回写）
	Status int `json:"status"`

	// SkillId 引用技能 ID（技能管理接口按技能 id 操作，实例 id 与之不同），仅 skill 类型填充
	SkillId uint64 `json:"skillId,omitempty"`
	// SkillStatus 引用技能的状态（draft/published），仅 skill 类型填充（展示字段，非聚合分页）
	SkillStatus string `json:"skillStatus,omitempty"`
	// SkillVersion 引用技能的 SemVer 版本，仅 skill 类型填充
	SkillVersion string `json:"skillVersion,omitempty"`
}

// InstanceSaveReq 插件实例创建/更新请求
type InstanceSaveReq struct {
	// PluginType 插件类型（须为已注册类型；更新时不可变更）
	PluginType string `json:"pluginType"`
	// Code 实例唯一标识（更新时技能类型不可变更，须与引用技能 code 一致）
	Code string `json:"code"`
	Name string `json:"name"`
	// Description 描述
	Description string `json:"description"`
	// Config 类型化配置 JSON 字符串（schema 由 PluginTypeHandler 约定）
	Config string `json:"config"`
}

// PluginInstanceApp 插件实例统一管理服务（对齐 tokhub plugin instance service）
//
// t_ai_plugin_instance 为统一插件视图唯一事实源：skill 实例 config 引用 t_ai_skill，
// mcp 实例 config 内联连接配置；列表/分页只查本表，技能展示字段由应用层按需填充。
type PluginInstanceApp interface {
	base.App[*entity.PluginInstance]

	// ListInstances 关键字 + 类型过滤的单表分页。
	// keyword 匹配 name/code/description；pluginType 为空查询全部已注册类型。
	ListInstances(ctx context.Context, keyword, pluginType string, pageParam model.PageParam) (*model.PageResult[*PluginInstanceItem], error)

	// GetInstanceDetail 实例详情（config 对象化，与列表 DTO 同构；出参统一为 Item，
	// 避免 entity.Config string 经 JSON 序列化后双重编码成字符串）
	GetInstanceDetail(id uint64) (*PluginInstanceItem, error)

	// CreateInstance 创建实例（校验类型注册过 + config 合法 + code 唯一；
	// 技能实例由技能接口联动产生，拒绝直建）
	CreateInstance(ctx context.Context, req *InstanceSaveReq) (*PluginInstanceItem, error)

	// UpdateInstance 更新实例（map 更新保零值可写；技能实例由技能接口管理，拒绝直改）
	UpdateInstance(ctx context.Context, id uint64, req *InstanceSaveReq) error

	// DeleteInstance 删除实例（技能实例由 DeleteSkill 联动删除，拒绝直删）
	DeleteInstance(ctx context.Context, id uint64) error

	// ToggleEnabled 启停实例（运行时总开关，技能注入与 MCP 工具装配均受其约束）
	ToggleEnabled(ctx context.Context, id uint64, enabled int) error

	// DiscoverTools 实时连接 MCP 服务器发现工具（连接测试 + 前端回显），成功/失败回写健康状态
	DiscoverTools(ctx context.Context, id uint64) ([]mcpclient.ToolInfo, error)

	// ResolveEnabledMcpServers 启用中的 MCP 实例转连接配置（mcpext loader 数据源，
	// config 解析失败的实例 fail-open 跳过并告警，不阻断 Agent 装配）
	ResolveEnabledMcpServers(ctx context.Context) ([]*mcpext.ServerConfig, error)
}

type pluginInstanceAppImpl struct {
	base.AppImpl[*entity.PluginInstance, repository.PluginInstance]

	skillRepo repository.Skill `inject:"T"`
}

var _ PluginInstanceApp = (*pluginInstanceAppImpl)(nil)

// ============== 查询 ==============

func (a *pluginInstanceAppImpl) ListInstances(ctx context.Context, keyword, pluginType string, pageParam model.PageParam) (*model.PageResult[*PluginInstanceItem], error) {
	if pluginType != "" {
		if _, ok := GetPluginType(pluginType); !ok {
			return nil, errorx.NewBizf("invalid plugin type: %s", pluginType)
		}
	}

	cond := model.NewCond().Eq("plugin_type", pluginType)
	if kw := escapeLike(strings.TrimSpace(keyword)); kw != "" {
		like := "%" + kw + "%"
		cond = cond.And("(name LIKE ? OR code LIKE ? OR description LIKE ?)", like, like, like)
	}
	cond = cond.OrderByAsc("plugin_type").OrderByAsc("id")

	page, err := a.PageByCond(cond, pageParam)
	if err != nil {
		return nil, err
	}

	// 当前页 skill 实例的 status/version 由应用层批量 IN 查 t_ai_skill 填充（展示字段，非聚合分页）
	items := make([]*PluginInstanceItem, 0, len(page.List))
	skillInfo := a.loadSkillInfo(page.List)
	for _, inst := range page.List {
		item := newInstanceItem(inst)
		if info, ok := skillInfo[inst.Code]; ok {
			item.SkillId = info.id
			item.SkillStatus = info.status
			item.SkillVersion = info.version
		}
		items = append(items, item)
	}
	return &model.PageResult[*PluginInstanceItem]{Total: page.Total, List: items}, nil
}

// newInstanceItem 实体 → 分页视图（config 对象化：空/非法 fail-open 为 {}，
// 避免 json.RawMessage marshal 整体报错）
func newInstanceItem(inst *entity.PluginInstance) *PluginInstanceItem {
	return &PluginInstanceItem{
		Id:          inst.Id,
		PluginType:  inst.PluginType,
		Code:        inst.Code,
		Name:        inst.Name,
		Description: inst.Description,
		Config:      rawInstanceConfig(inst.Config),
		Enabled:     inst.Enabled,
		Status:      inst.Status,
	}
}

// rawInstanceConfig 落库 config 字符串 → 原样 JSON 下发；空/非法回退 {}（fail-open）
func rawInstanceConfig(config string) json.RawMessage {
	trimmed := strings.TrimSpace(config)
	if trimmed == "" || !json.Valid([]byte(trimmed)) {
		return json.RawMessage("{}")
	}
	return json.RawMessage(trimmed)
}

type skillDisplayInfo struct {
	id      uint64
	status  string
	version string
}

// loadSkillInfo 批量填充当前页技能实例的展示字段（按 code IN 一次查询）
func (a *pluginInstanceAppImpl) loadSkillInfo(instances []*entity.PluginInstance) map[string]skillDisplayInfo {
	info := make(map[string]skillDisplayInfo)
	var skillCodes []string
	for _, inst := range instances {
		if inst.PluginType == entity.PluginTypeSkill {
			skillCodes = append(skillCodes, inst.Code)
		}
	}
	if len(skillCodes) == 0 {
		return info
	}
	skills, err := a.skillRepo.SelectByCond(model.NewCond().In("code", skillCodes))
	if err != nil {
		// 展示字段填充失败不阻断列表返回
		logx.Warnf("[plugin_instance] load skill display info failed: %v", err)
		return info
	}
	for _, s := range skills {
		info[s.Code] = skillDisplayInfo{id: s.Id, status: s.Status, version: s.Version}
	}
	return info
}

func (a *pluginInstanceAppImpl) GetInstanceDetail(id uint64) (*PluginInstanceItem, error) {
	inst, err := a.GetById(id)
	if err != nil {
		return nil, err
	}
	item := newInstanceItem(inst)
	if inst.PluginType == entity.PluginTypeSkill {
		if info, ok := a.loadSkillInfo([]*entity.PluginInstance{inst})[inst.Code]; ok {
			item.SkillId = info.id
			item.SkillStatus = info.status
			item.SkillVersion = info.version
		}
	}
	return item, nil
}

// ============== 写维护 ==============

func (a *pluginInstanceAppImpl) CreateInstance(ctx context.Context, req *InstanceSaveReq) (*PluginInstanceItem, error) {
	handler, ok := GetPluginType(req.PluginType)
	if !ok {
		return nil, errorx.NewBizf("unknown plugin type: %s", req.PluginType)
	}
	// 技能实例仅由技能创建/导入联动注册（upsertSkillInstance），直建会绕过技能表产生游离引用
	if req.PluginType == entity.PluginTypeSkill {
		return nil, errorx.NewBiz("skill instances are created with their skill, use the skill APIs")
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, errorx.NewBiz("instance code is required")
	}
	if err := handler.ValidateConfig(req.Config); err != nil {
		return nil, err
	}
	if a.CountByCond(&entity.PluginInstance{Code: code}) > 0 {
		return nil, errorx.NewBizf("instance code already exists: %s", code)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = code
	}
	inst := &entity.PluginInstance{
		Code:        code,
		PluginType:  req.PluginType,
		Name:        name,
		Description: req.Description,
		Config:      req.Config,
		Enabled:     1,
	}
	if err := a.Insert(ctx, inst); err != nil {
		return nil, err
	}
	a.invalidateAgent()
	return newInstanceItem(inst), nil
}

func (a *pluginInstanceAppImpl) UpdateInstance(ctx context.Context, id uint64, req *InstanceSaveReq) error {
	inst, err := a.GetById(id)
	if err != nil {
		return err
	}
	if inst.PluginType != req.PluginType {
		return errorx.NewBiz("plugin type of an existing instance cannot be changed")
	}
	handler, ok := GetPluginType(inst.PluginType)
	if !ok {
		return errorx.NewBizf("unknown plugin type: %s", inst.PluginType)
	}
	if err := handler.ValidateConfig(req.Config); err != nil {
		return err
	}
	// 技能实例由技能接口统一管理（name/description 随 UpsertForSkill 同步），拒绝直改避免双源失同步
	if inst.PluginType == entity.PluginTypeSkill {
		return errorx.NewBiz("skill instance is managed by skill APIs, edit the skill instead")
	}
	if inst.Code != req.Code && a.CountByCond(&entity.PluginInstance{Code: req.Code}) > 0 {
		return errorx.NewBizf("instance code already exists: %s", req.Code)
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Code
	}
	// map 更新保零值可写（GORM struct 更新跳过零值，既有教训）
	if err := a.UpdateByCond(ctx, map[string]any{
		"code":        req.Code,
		"name":        name,
		"description": req.Description,
		"config":      req.Config,
	}, model.NewCond().Eq("id", id)); err != nil {
		return err
	}
	a.invalidateAgent()
	return nil
}

func (a *pluginInstanceAppImpl) DeleteInstance(ctx context.Context, id uint64) error {
	inst, err := a.GetById(id)
	if err != nil {
		return err
	}
	// 技能实例随技能删除联动移除（DeleteBySkillCode），直删会留下游离技能
	if inst.PluginType == entity.PluginTypeSkill {
		return errorx.NewBiz("skill instance is removed with its skill, delete the skill instead")
	}
	if err := a.DeleteById(ctx, id); err != nil {
		return err
	}
	a.invalidateAgent()
	return nil
}

func (a *pluginInstanceAppImpl) ToggleEnabled(ctx context.Context, id uint64, enabled int) error {
	if enabled != 0 && enabled != 1 {
		return errorx.NewBizf("invalid enabled value: %d", enabled)
	}
	if _, err := a.GetById(id); err != nil {
		return err
	}
	// map 更新：enabled=0 为零值，struct 更新会被 GORM 跳过
	if err := a.UpdateByCond(ctx, map[string]any{"enabled": enabled}, model.NewCond().Eq("id", id)); err != nil {
		return err
	}
	// 启停即运行时总开关（技能注入清单 / MCP 工具装配均受其约束），重置默认 Agent 生效
	a.invalidateAgent()
	return nil
}

// upsertSkillInstance 技能→实例联动注册（by code 幂等，包级辅助供 skill_plugin 与
// plugin_instance 共用，两个 App 互不注入避免循环依赖）。
// 技能为实例的事实源：name/description/config.skillCode 均随技能行同步。
func upsertSkillInstance(ctx context.Context, instRepo repository.PluginInstance, skillRow *entity.Skill) error {
	cfg, err := json.Marshal(SkillInstanceConfig{SkillCode: skillRow.Code})
	if err != nil {
		return err
	}
	existing, err := instRepo.SelectByCode(ctx, skillRow.Code)
	if err != nil {
		// 不存在则注册新实例（默认启用；status 由 discover 维护，技能型恒 unknown）
		return instRepo.Insert(ctx, &entity.PluginInstance{
			Code:        skillRow.Code,
			PluginType:  entity.PluginTypeSkill,
			Name:        skillRow.Name,
			Description: skillRow.Description,
			Config:      string(cfg),
			Enabled:     1,
		})
	}
	// map 更新保零值可写（description 可能被清空）
	return instRepo.UpdateByCond(ctx, map[string]any{
		"name":        skillRow.Name,
		"description": skillRow.Description,
		"config":      string(cfg),
	}, model.NewCond().Eq("id", existing.Id))
}

// deleteSkillInstance 技能删除时联动删除其引用实例
func deleteSkillInstance(ctx context.Context, instRepo repository.PluginInstance, code string) error {
	return instRepo.DeleteByCond(ctx, model.NewCond().Eq("plugin_type", entity.PluginTypeSkill).Eq("code", code))
}

// ============== MCP 运行时 ==============

func (a *pluginInstanceAppImpl) DiscoverTools(ctx context.Context, id uint64) ([]mcpclient.ToolInfo, error) {
	inst, err := a.GetById(id)
	if err != nil {
		return nil, err
	}
	if inst.PluginType != entity.PluginTypeMcp {
		return nil, errorx.NewBizf("plugin type %s does not support tool discovery", inst.PluginType)
	}
	cfg := mustMcpConfig(inst.Config)
	cli, err := mcpclient.Connect(ctx, cfg.Url, parseMcpHeaders(cfg.Headers), time.Duration(cfg.TimeoutSec)*time.Second)
	if err != nil {
		a.writeStatus(ctx, id, entity.PluginStatusError)
		return nil, err
	}
	defer cli.Close()
	tools, err := cli.ListTools(ctx)
	if err != nil {
		a.writeStatus(ctx, id, entity.PluginStatusError)
		return nil, err
	}
	a.writeStatus(ctx, id, entity.PluginStatusHealthy)
	return tools, nil
}

// writeStatus 回写实例健康状态（fail-open：状态回写失败不影响主流程）
func (a *pluginInstanceAppImpl) writeStatus(ctx context.Context, id uint64, status int) {
	if err := a.UpdateByCond(ctx, map[string]any{"status": status}, model.NewCond().Eq("id", id)); err != nil {
		logx.Warnf("[plugin_instance] write instance status failed, id=%d, status=%d: %v", id, status, err)
	}
}

func (a *pluginInstanceAppImpl) ResolveEnabledMcpServers(ctx context.Context) ([]*mcpext.ServerConfig, error) {
	instances, err := a.GetRepo().SelectEnabledByType(ctx, entity.PluginTypeMcp)
	if err != nil {
		return nil, err
	}
	servers := make([]*mcpext.ServerConfig, 0, len(instances))
	for _, inst := range instances {
		cfg := mustMcpConfig(inst.Config)
		if strings.TrimSpace(cfg.Url) == "" {
			// config 解析失败/缺 url：fail-open 跳过该实例并告警，不阻断 Agent 装配
			//（对齐 tokhub parse_capabilities 容错模式）
			logx.Warnf("[plugin_instance] skip mcp instance %s: invalid or empty url config", inst.Code)
			continue
		}
		servers = append(servers, &mcpext.ServerConfig{
			Id:         inst.Id,
			Code:       inst.Code,
			Url:        cfg.Url,
			Headers:    cfg.Headers,
			TimeoutSec: cfg.TimeoutSec,
		})
	}
	return servers, nil
}

// ============== 共享辅助 ==============

// invalidateAgent 插件配置变更后重置默认 Agent 单例（fail-open，不阻断管理操作）：
// 下次对话懒重建 Agent 并重新聚合技能清单与工具列表，配合 mcpext 连接缓存仅重连配置变更的服务器
func invalidateDefaultAgent() {
	agent.ResetDefaultAgent()
	logx.Debug("[plugin_instance] plugin changed, default agent reset for next turn")
}

func (a *pluginInstanceAppImpl) invalidateAgent() { invalidateDefaultAgent() }

// parseMcpHeaders 解析请求头 JSON（空或非法时返回 nil，不阻断连接）
func parseMcpHeaders(headers string) map[string]string {
	if strings.TrimSpace(headers) == "" {
		return nil
	}
	parsed := make(map[string]string)
	if err := json.Unmarshal([]byte(headers), &parsed); err != nil {
		return nil
	}
	if len(parsed) == 0 {
		return nil
	}
	return parsed
}
