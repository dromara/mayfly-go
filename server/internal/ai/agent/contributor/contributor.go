// Package contributor 定义 Agent 能力扩展的贡献者契约
//
// Agent 的能力不再由单一模块硬编码拼装，而是由一组 Contributor 通过
// 各自的贡献通道注入，经 Registry 统一聚合与调度：
//
//   - ContextContributor       统一 Prompt 注入通道（记忆/技能目录/环境等轮次级片段）
//   - ToolContributor          注册工具执行器（内置/条件工具组统一走此通道聚合）
//   - HistoryContributor       对话历史消息段通道（短期记忆进历史的唯一通道）
//   - PreambleFooterContributor preamble 后置注入通道（窗口余量提醒等）
//   - TurnLifecycleContributor 轮次生命周期钩子（轮次开始/结束按因施策）
//   - ToolMiddlewareContributor 工具执行中间件通道（安全拦截等）
//   - TokenUsageContributor    token 用量回调通道（计量/计费/统计唯一出口）
//   - Activatable              装配期激活（无运行期调度，如中断类型扩展装载、记忆提取装配）
//
// 另有 Sandbox/SuggestQuestions/ToolPlanner/HandoffSignal 等潜在通道，
// mayfly-go 暂无对应运行时概念，待能力落地时按需补充，扩展方式与上述通道一致。
//
// ## Registry 设计原则（ExtensionRegistry）
//
//   - 通过 Builder 显式注册，Build 后只读，并发共享
//   - 注册顺序即覆盖优先级（后注册胜出）：内置注册在前、业务扩展注册在后，
//     插件无需修改宿主即可覆盖内置实现
//   - 每个贡献者带稳定 Id，宿主可按标识裁剪（WithFilter）与覆盖诊断
//   - Registry 只存数据与聚合，调度（遍历/去重/fail-open）由本包 dispatch 提供
//
// ## 分层约定
//
// 本包仅为契约层（接口 + Registry + 调度），不含任何具体扩展实现；
// 内置扩展位于 agent/ext/<name> 子包（各自 Install 到 Builder），
// 宿主（agent 包）负责组装。
package contributor

// Contributor 全部贡献通道的基础接口
//
// 每个贡献者带一个稳定 Id：重复注册时覆盖旧实例，供 Registry
// 覆盖诊断与 WithFilter 按标识裁剪（ContributorId）。
type Contributor interface {
	// Id 贡献者稳定标识
	Id() string
}
