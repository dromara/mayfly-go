// Package ext 内置扩展装配入口（对齐 tokhub 各扩展 crate 的 install(builder) 模式）
//
// 每个扩展位于 agent/ext/<name> 子包，实现 contributor 契约包定义的
// 贡献通道接口，并暴露 Install(builder) 注册自身；本包按 tokhub 的
// 装配顺序（内置在前、业务插件在后，后注册者覆盖先注册者）聚合全部
// 内置扩展，供宿主（agent 包）构建默认注册中心。
//
// 业务扩展接入：调用 RegisterHostInstaller 注册安装器（对齐 tokhub
// HostExtensionInstaller），在内置扩展之后追加装配，可覆盖内置工具/贡献者
// 而无需修改宿主代码（开闭原则）。
package ext

import (
	"slices"
	"sync"

	"mayfly-go/internal/ai/agent/contributor"
	environmentext "mayfly-go/internal/ai/agent/ext/environment"
	historyext "mayfly-go/internal/ai/agent/ext/history"
	interruptext "mayfly-go/internal/ai/agent/ext/interrupt"
	memoryext "mayfly-go/internal/ai/agent/ext/memory"
	skillext "mayfly-go/internal/ai/agent/ext/skill"
	"mayfly-go/internal/ai/agent/middleware"
	"mayfly-go/internal/ai/memory"
	"mayfly-go/internal/ai/session"

	"github.com/cloudwego/eino/components/model"
)

// Deps 扩展装配依赖（对应依赖允许为 nil，未装配时该扩展自动跳过注册）
type Deps struct {
	// MemoryManager 记忆管理器（memoryext 注入依赖；为 nil 时记忆片段不注入）
	MemoryManager *memory.Manager
	// SessionManager 会话管理器（historyext 注入依赖；为 nil 时历史扩展跳过注册）
	SessionManager *session.Manager
	// ChatModel 聊天模型（memoryext 长期记忆提取依赖；为 nil 时提取能力降级不可用）
	ChatModel model.ToolCallingChatModel
}

// InstallAll 按内置优先顺序注册全部内置扩展到 Builder
//
// 装配期激活型扩展（ChannelService：中断类型扩展装载、记忆提取装配）
// 经 Registry.Activate 在 WithFilter 裁剪后激活，被裁剪的扩展不会激活。
//
// 业务工具扩展（db_tools / machine_tools）因依赖业务应用层（db/machine），
// 为避免引入 agent → 业务层循环依赖，不在本包静态安装，而是由 ai/init
// 经 RegisterHostInstaller 注册（对齐 tokhub 宿主端私有 Contributor 注入）。
func InstallAll(b *contributor.Builder, deps Deps) {
	// 中间件通道：安全工具拦截（safety_middleware）
	middleware.Install(b)
	memoryext.Install(b, deps.MemoryManager)
	// 记忆搜索工具：memory_search（memory_tools，memoryManager 非 nil 时贡献）
	memoryext.InstallTools(b, deps.MemoryManager)
	// 记忆提取：装配期激活型（memory_extraction），非 nil 时注册
	if e := memoryext.NewExtractionExtension(deps.MemoryManager, deps.SessionManager, deps.ChatModel); e != nil {
		b.RegisterService(e)
	}
	historyext.Install(b, deps.SessionManager)
	// 技能：L1 目录注入（skill_injection）+ skill_read 工具（skill_tools）
	skillext.Install(b)
	skillext.InstallTools(b)
	environmentext.Install(b)
	// 中断类型扩展：审批/参数补全（interrupt_approval / interrupt_param_completion）
	interruptext.Install(b)
}

// hostInstallers 宿主级扩展安装器（对齐 tokhub HostExtensionInstaller）
//
// 业务扩展在内置扩展之后、Build 之前追加装配：注册顺序即覆盖优先级，
// 后注册的插件可直接覆盖内置工具实现，无需修改宿主代码。
var (
	hostInstallerMu sync.Mutex
	hostInstallers  []func(b *contributor.Builder)
)

// RegisterHostInstaller 注册宿主级扩展安装器（须在宿主装配完成前调用）
func RegisterHostInstaller(install func(b *contributor.Builder)) {
	hostInstallerMu.Lock()
	defer hostInstallerMu.Unlock()
	hostInstallers = append(hostInstallers, install)
}

// RunHostInstallers 执行全部宿主级扩展安装器
func RunHostInstallers(b *contributor.Builder) {
	hostInstallerMu.Lock()
	installers := slices.Clone(hostInstallers)
	hostInstallerMu.Unlock()
	for _, install := range installers {
		install(b)
	}
}
