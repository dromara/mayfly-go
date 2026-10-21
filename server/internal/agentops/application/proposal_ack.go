package application

// 审批回调（FlowBizHandle）的回执契约——跨语言线协议，唯一真源。
//
// 为什么必须集中声明：这些键值被 flow 落进 `t_flow_procinst.biz_handle_res`，再由前端审批面板
// 读出来判断「这条回执是有结论的结果，还是只是受理凭据」。此前它们是四段散落的英文字面量，
// 前端 `AgentRemediationBiz.vue` 再独立抄一份字符串数组——后端任何一次措辞调整（或顺手把它 i18n 掉）
// 都会让面板从「等待提示」静默退化成**裸直出 JSON**，审批人看到一坨回执而不知道变更正在异步跑。
// 两侧都不报错，只是显示不对，属于典型的隐式契约漂移。
//
// 值刻意取 snake_case 标识符而不是英文句子（旧形态 `already processed` / `already claimed`）：
// 机器可读的枚举值不该长得像待翻译的文案，否则下一个改它的人会本能地去把它搬进 i18n 字典，
// 而那会直接打断跨语言契约。新值不含空格、不是自然语言句子，i18n 化它一眼可辨是错的。
//
// 变更纪律：改键名或改值都要同步前端 proposalAck.ts 与两侧测试（proposal_ack_test.go 与
// agentRemediationBiz.test.ts 各钉了一份字面值，任一侧改动都会红）。存量工单里已落库的旧句子
// 形态由前端的历史清单继续识别，不在这里保留别名——本包只会产出新值。
const (
	// AckKeyProposalId 提案 id
	AckKeyProposalId = "proposalId"
	// AckKeyStatus 提案当前状态（值域见 entity.ProposalStatus*）
	AckKeyStatus = "status"
	// AckKeyExecution 执行投递结果位：仅「已受理、尚无结论」的回执携带
	AckKeyExecution = "execution"
	// AckKeyMessage 幂等/重投说明位：仅「本次没有新动作」的回执携带
	AckKeyMessage = "message"

	// AckExecutionQueued 审批通过并已投递受控执行（异步跑，终态稍后回写）
	AckExecutionQueued = "queued"
	// AckExecutionRequeued 上一次领取后崩在执行之前，本次重新投递
	AckExecutionRequeued = "requeued"

	// AckMessageAlreadyProcessed 提案已是终态，本次审批回调是重驱动，不再重复执行
	AckMessageAlreadyProcessed = "already_processed"
	// AckMessageAlreadyClaimed CAS 领取落败（并发审批/重复驱动），执行权归胜出者
	AckMessageAlreadyClaimed = "already_claimed"
)
