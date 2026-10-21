package session

// session.Message.Extra 的运行时键（跨层契约，唯一真源）。
//
// 这些键由 agent 层写入、被 session/application/tools 多层读取，此前各处字面量各写一遍；
// 抽到本包是因 Message.Extra 归属 session 模型，且 tools/agent/application 均已依赖 session，引用方向不成环
const (
	// ExtraKeyTurnId 消息所属 turn id
	ExtraKeyTurnId = "turnId"
	// ExtraKeyActionId 中断动作 id（与 eino interrupt id 同源，恢复匹配与审计用）
	ExtraKeyActionId = "actionId"
	// ExtraKeyToolStatus 工具调用状态标记（error / interrupted / 空）
	ExtraKeyToolStatus = "toolStatus"
	// ExtraKeyResumeInfo 中断恢复决策（resume 时回写，供历史还原与重试读取）
	ExtraKeyResumeInfo = "resumeInfo"
)
