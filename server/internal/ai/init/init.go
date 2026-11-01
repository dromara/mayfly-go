package init

import (
	"context"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/agent/ext"
	dbtoolext "mayfly-go/internal/ai/agent/ext/dbtool"
	machinetoolext "mayfly-go/internal/ai/agent/ext/machinetool"
	mcpext "mayfly-go/internal/ai/agent/ext/mcp/mcpext"
	monitortoolsExt "mayfly-go/internal/ai/agent/ext/monitortool"
	resourcetoolext "mayfly-go/internal/ai/agent/ext/resourcetool"
	"mayfly-go/internal/ai/api"
	"mayfly-go/internal/ai/application"
	aiconfig "mayfly-go/internal/ai/config"
	"mayfly-go/internal/ai/infra/persistence"
	"mayfly-go/internal/ai/skill"
	sysapp "mayfly-go/internal/sys/application"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/starter"
)

func init() {
	// 系统配置热生效：模型 / Agent 配置保存后刷新 AI 运行时（下一轮对话即用新配置）
	//
	// 依赖倒置：sys 只提供「配置已变更」钩子，不感知 AI 模块；此处把配置键与刷新
	// 动作绑定，新增一类需要热生效的 AI 配置只需在这里多注册一行。
	// 注册在包初始化期完成（早于任何请求），钩子本身在保存成功后才被触发
	sysapp.RegisterConfigChangeHook(aiconfig.ConfigKeyModel, func(_ context.Context, _ string) {
		application.ReloadModelConfig()
	})
	sysapp.RegisterConfigChangeHook(aiconfig.ConfigKeyAgent, func(_ context.Context, _ string) {
		application.ReloadAgentConfig()
	})

	// 注册AI模块的IoC组件
	starter.AddInitIocFunc(func() {
		persistence.InitIoc()
		application.Init()
		api.InitIoc()
	})

	// 启动预热：装配默认运行时（扩展激活、中断扩展注册表冻结、存储就绪），
	// 确保任意业务请求到达前扩展已就位（多实例部署下实例启动即完成装配，
	// 避免首个请求承担装配开销与激活时序问题）。
	// 失败不阻断启动（fail-open），首次调用 Agent 时会重试装配。
	starter.AddInitFunc(func() {
		if _, err := agent.AssembleDefault(context.Background()); err != nil {
			logx.Warnf("[ai] warm up default runtime failed (will retry on first use): %v", err)
		}
	})

	// 业务工具扩展经宿主级安装器注册（HostExtensionInstaller）：
	// dbtool/machinetool/mcp_tools 依赖业务应用层（db/machine/插件管理），为避免
	// agent → 业务层循环依赖，由本包（main 的直接依赖）注册，在内置扩展之后追加装配，
	// 后注册的插件仍可覆盖其工具实现（后注册胜出）。
	ext.RegisterHostInstaller(func(b *contributor.Builder) {
		dbtoolext.Install(b)
		machinetoolext.Install(b)
		// 可观测数据工具（QueryMetrics/QueryAlertHistory）：依赖 alert 应用层，同范式宿主级注册
		monitortoolsExt.Install(b)
		resourcetoolext.Install(b)
		mcpext.Install(b)
	})

	// MCP 服务器 loader 接线：mcp_tools 装配期读取启用中的 MCP 插件实例（仅 enabled=1，
	// 停用即从工具清单下线；连接配置内联在实例 config 中）。
	// （迁移阶段应用层未初始化时 fail-open 返回空，不阻断启动）
	mcpext.SetServerLoader(func(ctx context.Context) ([]*mcpext.ServerConfig, error) {
		instanceApp := application.GetPluginInstanceApp()
		if instanceApp == nil {
			return nil, nil
		}
		return instanceApp.ResolveEnabledMcpServers(ctx)
	})

	// 技能 Registry DB provider 接线：技能目录/正文全部读 DB（含迁移内置的默认技能），
	// 读失败退回上次成功缓存。
	// 注意：迁移阶段应用层尚未初始化（GetSkillPlugin 为 nil），此闭包在迁移期间被
	// 触发时 fail-open 返回空列表（由调用方退回上次缓存），不阻断启动。
	skill.DefaultRegistry.SetProvider(func() ([]*skill.Skill, error) {
		skillPlugin := application.GetSkillPlugin()
		if skillPlugin == nil {
			return nil, nil
		}
		return skillPlugin.ListPublishedSkills(context.Background())
	})
}
