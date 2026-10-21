import { EnumValue } from '@/common/Enum';

/** 任务状态（对应后端 entity.AgentTaskStatus*） */
export const AgentTaskStatusEnum = {
    Enable: EnumValue.of(1, 'common.enable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'common.disable').tagTypeDanger(),
};

/** 结果通知方式（对应后端 entity.TaskNotify*） */
export const AgentTaskNotifyTypeEnum = {
    None: EnumValue.of(0, 'agentops.notifyNone'),
    Fail: EnumValue.of(1, 'agentops.notifyFail').tagTypeWarning(),
    Always: EnumValue.of(2, 'agentops.notifyAlways').tagTypeSuccess(),
};

/** 错过触发（misfire）策略（对应后端 entity.MisfirePolicy*） */
export const MisfirePolicyEnum = {
    Skip: EnumValue.of(0, 'agentops.misfireSkip'),
    CatchUpOnce: EnumValue.of(1, 'agentops.misfireCatchUpOnce').tagTypeWarning(),
};

/** Agent 装配档（对应后端 headless.Profile*；executor 仅用于审批后的受控执行） */
export const ProfileEnum = {
    Diagnostician: EnumValue.of('diagnostician', 'agentops.profileDiagnostician').tagTypeInfo(),
    Reviewer: EnumValue.of('reviewer', 'agentops.profileReviewer').tagTypeInfo(),
    Executor: EnumValue.of('executor', 'agentops.profileExecutor').tagTypeWarning(),
    Auditor: EnumValue.of('auditor', 'agentops.profileAuditor').tagTypeInfo(),
};

/** 触发类型（对应后端 entity.TriggerType*） */
export const TriggerTypeEnum = {
    Cron: EnumValue.of('cron', 'agentops.triggerCron').tagTypeInfo(),
    Event: EnumValue.of('event', 'agentops.triggerEvent').tagTypeWarning(),
    Manual: EnumValue.of('manual', 'agentops.triggerManual').tagTypeSuccess(),
    /** 多 Agent 编排：父运行与规划/步骤/审查子运行共用该触发方式（与后端 entity.TriggerType* 对齐），
     *  执行历史据此可直接筛出整棵编排树，不必按 triggerRef 前缀模糊匹配 */
    Orchestration: EnumValue.of('orchestration', 'agentops.triggerOrchestration').tagTypeInfo(),
};

/** 事件驱动触发源类型（对应后端 trigger.SourceType* / event.OpsTriggerSource*） */
export const TriggerSourceEnum = {
    Alert: EnumValue.of('alert', 'agentops.sourceAlert').tagTypeDanger(),
    Webhook: EnumValue.of('webhook', 'agentops.sourceWebhook').tagTypeInfo(),
};

/**
 * 运行状态机（对应后端 entity.RunStatus*）。
 *
 * 这里不得多出后端永远不会写入的状态：筛选项与治理分布都直接取本枚举，多一项就多一个
 * 永远查不到数据的选项（选它得到空列表，看起来像系统没跑出过运行）。`waiting_approval` 曾如此，
 * 已两侧同步删除；可达性由后端 application.TestEveryRunStatusHasRuntimeReference 守住
 */
export const RunStatusEnum = {
    Queued: EnumValue.of('queued', 'agentops.runQueued').tagTypeInfo(),
    Running: EnumValue.of('running', 'agentops.runRunning').tagTypeWarning(),
    Succeeded: EnumValue.of('succeeded', 'agentops.runSucceeded').tagTypeSuccess(),
    Failed: EnumValue.of('failed', 'agentops.runFailed').tagTypeDanger(),
    Canceled: EnumValue.of('canceled', 'agentops.runCanceled').tagTypeDanger(),
};

/**
 * 人类反馈（对应后端 entity.FeedbackUp/FeedbackDown）：值集合只有两个，
 * **空串代表尚未反馈、不列入枚举**：列表侧未反馈就是空单元格，给每个未反馈的运行
 * 打一个「未反馈」标签只会在历史多的时候刷屏。筛选项也同源于此枚举，
 * 所以只能选 up/down（基座 Eq 跳零值，“筛未反馈”需哨兵值，不为此造一个）
 */
export const RunFeedbackEnum = {
    Up: EnumValue.of('up', 'agentops.feedbackUp').tagTypeSuccess(),
    Down: EnumValue.of('down', 'agentops.feedbackDown').tagTypeDanger(),
};

/**
 * 运行形态（对应后端 entity.RunRole*）：多阶段流程里这条运行扮演的角色。
 *
 * 为什么不是标题前缀：形态以前写在 `[编排]`、`[评测回放]` 这类中文前缀里并落进数据库列，
 * 英文界面永远看到中文，分类也只能靠文本 LIKE。现在后端回传结构化 `runRole`，标签在此按 i18n
 * 渲染。值集合由 `runRole.test.ts` 钉住与后端声明一致（新增角色两边要同步，否则这里漏项会
 * 退化成显示原始键串而不是报错）
 */
export const RunRoleEnum = {
    EvalReplay: EnumValue.of('eval_replay', 'agentops.roleEvalReplay').tagTypeInfo(),
    // 容器节点取中性 info 色：显式登记而不是依赖「未命中时回退」——旧注释写的回退值是 primary，
    // 而 enumTagType 的实际回退是 info，这种「注释描述的能力与实际不一致」就是本仓反复修的假文档源
    OrchParent: EnumValue.of('orch_parent', 'agentops.roleOrchParent').tagTypeInfo(),
    OrchPlan: EnumValue.of('orch_plan', 'agentops.roleOrchPlan').tagTypeInfo(),
    OrchStep: EnumValue.of('orch_step', 'agentops.roleOrchStep').tagTypeInfo(),
    OrchReview: EnumValue.of('orch_review', 'agentops.roleOrchReview').tagTypeWarning(),
    Exec: EnumValue.of('exec', 'agentops.roleExec').tagTypeSuccess(),
    Verify: EnumValue.of('verify', 'agentops.roleVerify').tagTypeWarning(),
};

/** 全部运行形态值（供守卫用例比对后端声明集，也供需要枚举形态的地方取用） */
export const RUN_ROLES: string[] = Object.values(RunRoleEnum).map((v) => v.value as string);

/**
 * 按 key 从枚举对象取状态值（泛化到任意枚举）。
 *
 * 存在理由：同一状态字面量在导出集合里手写一遍，就是第二个真源——后端改了枚举值或新增状态时，
 * 手抄那份会静默错配（集合取错成员不报错，只表现为「该状态永远进不了这个判定」）
 */
const enumValues = <T extends Record<string, EnumValue>>(enumObj: T, ...keys: (keyof T)[]): string[] =>
    keys.map((k) => enumObj[k].value as string);

/**
 * 运行非终态集合：与后端 `entity.ActiveRunStatuses` 同一口径（它同时是「可取消」「仍在途」「占重叠防护位」
 * 的判定集）。前端不再手写子集：曾出现「急停清场与取消白名单各自凑子集」导致 0 行更新仍报已取消
 */
export const ACTIVE_RUN_STATUS: string[] = enumValues(RunStatusEnum, 'Queued', 'Running');

/** 运行终态集合：可反馈/评测（与后端 `entity.IsTerminal` 一致） */
export const TERMINAL_RUN_STATUS: string[] = enumValues(RunStatusEnum, 'Succeeded', 'Failed', 'Canceled');

/** 可取消的运行状态：与后端 Cancel 的条件更新白名单同源（均取非终态集） */
export const CANCELABLE_RUN_STATUS = ACTIVE_RUN_STATUS;

/** 提案状态机（对应后端 entity.ProposalStatus*） */
export const ProposalStatusEnum = {
    Pending: EnumValue.of('pending', 'agentops.proposalPending').tagTypeWarning(),
    Approved: EnumValue.of('approved', 'agentops.proposalApproved').tagTypeInfo(),
    Rejected: EnumValue.of('rejected', 'agentops.proposalRejected').tagTypeDanger(),
    Executed: EnumValue.of('executed', 'agentops.proposalExecuted').tagTypeSuccess(),
    Failed: EnumValue.of('failed', 'agentops.proposalFailed').tagTypeDanger(),
};

/** 提案风险级（对应后端 entity.ProposalRisk*） */
export const ProposalRiskEnum = {
    Low: EnumValue.of('low', 'agentops.riskLow').tagTypeSuccess(),
    Medium: EnumValue.of('medium', 'agentops.riskMedium').tagTypeInfo(),
    High: EnumValue.of('high', 'agentops.riskHigh').tagTypeWarning(),
    Critical: EnumValue.of('critical', 'agentops.riskCritical').tagTypeDanger(),
};

/** 提案终态集合：仅这些状态可删（与后端 entity.IsProposalTerminal 一致，同样由枚举取值而非手写字面量） */
export const DELETABLE_PROPOSAL_STATUS: string[] = enumValues(ProposalStatusEnum, 'Rejected', 'Executed', 'Failed');
