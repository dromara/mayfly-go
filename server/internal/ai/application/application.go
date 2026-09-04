package application

import (
	"mayfly-go/internal/ai/application/resource"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/session"
	"mayfly-go/pkg/ioc"
)

func Init() {
	// 注册 session 存储（基于 conversation + turn_item 两张表实现）
	sessionStore := new(sessionStoreImpl)
	ioc.Register(sessionStore)
	session.DefaultSessionStore = sessionStore

	// 注册记忆存储（基于 t_ai_memory 表实现，跨会话、跨实例共享）
	memoryStore := new(memoryStoreImpl)
	ioc.Register(memoryStore)
	memory.DefaultStore = memoryStore

	// 注册 Conversation App
	conversationAppImpl := new(conversationAppImpl)
	ioc.Register(conversationAppImpl)

	// 注册 TurnItem App
	turnItemAppImpl := new(turnItemAppImpl)
	ioc.Register(turnItemAppImpl)

	// 注册压缩事件项记录器：订阅摘要完成事件，落库 context_compaction TurnItem
	recorder := new(compactionRecorder)
	ioc.Register(recorder)
	registerCompactionRecorder(recorder)

	// 注册技能插件 App（ai/init 经 GetSkillPlugin 取用，注入技能 Registry DB provider）
	skillPluginApp = new(skillPluginAppImpl)
	ioc.Register(skillPluginApp)

	// 注册插件实例 App（ai/init 经 GetPluginInstanceApp 取用，注入 MCP 服务器 loader）
	pluginInstanceApp = new(pluginInstanceAppImpl)
	ioc.Register(pluginInstanceApp)

	// 装配插件类型注册表（skill/mcp 内置类型；新增类型 = 实现 PluginTypeHandler + 此处注册一行）
	RegisterPluginType(skillInstanceType{})
	RegisterPluginType(mcpInstanceType{})

	// 装配统一资源查询服务（内置 machine/db provider，资源清单工具与参数补全选项共用）
	resource.Init()
}

var (
	skillPluginApp    SkillPlugin
	pluginInstanceApp PluginInstanceApp
)

// GetSkillPlugin 技能插件管理服务
func GetSkillPlugin() SkillPlugin { return skillPluginApp }

// GetPluginInstanceApp 插件实例统一管理服务
func GetPluginInstanceApp() PluginInstanceApp { return pluginInstanceApp }
