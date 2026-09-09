package init

import (
	"context"

	"mayfly-go/internal/ai/agent"
	"mayfly-go/internal/ai/agent/contributor"
	"mayfly-go/internal/ai/agent/ext"
	dbtoolext "mayfly-go/internal/ai/agent/ext/dbtool"
	machinetoolext "mayfly-go/internal/ai/agent/ext/machinetool"
	mcpext "mayfly-go/internal/ai/agent/ext/mcp/mcpext"
	resourcetoolext "mayfly-go/internal/ai/agent/ext/resourcetool"
	"mayfly-go/internal/ai/api"
	"mayfly-go/internal/ai/application"
	"mayfly-go/internal/ai/infra/persistence"
	"mayfly-go/internal/ai/skill"
	"mayfly-go/pkg/logx"
	"mayfly-go/pkg/starter"
)

func init() {
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
