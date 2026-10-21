/**
 * 状态枚举注册表
 *
 * 收敛散落在各处的状态字符串字面量，提供统一的：
 * - ToolCallStatus：工具调用执行状态
 * - InterruptResumeStatus：中断恢复决策状态
 * - action → 恢复状态映射（决策解释的单一出处）
 */

// ==================== 工具调用状态 ====================

/** 工具调用执行状态（对齐后端 TurnItemStatus） */
export const ToolCallStatus = {
    /** 待执行 */
    Pending: 'pending',
    /** 执行中（流式 arguments 增量） */
    Running: 'running',
    /** 执行成功 */
    Success: 'success',
    /** 执行失败 */
    Failed: 'failed',
    /** 已取消 */
    Cancelled: 'cancelled',
    /** 被中断等待用户输入 */
    Interrupted: 'interrupted',
} as const;

export type ToolCallStatusValue = (typeof ToolCallStatus)[keyof typeof ToolCallStatus];

// ==================== 中断恢复决策状态 ====================

/** 中断恢复决策状态（InterruptResumeStatus：tag snake_case） */
export const InterruptResumeStatus = {
    /** 待用户决策 */
    Pending: 'pending',
    /** 已批准（审批通过） */
    Approved: 'approved',
    /** 已拒绝 */
    Rejected: 'rejected',
    /** 已取消 */
    Cancelled: 'cancelled',
    /** 已解决（参数补全等非二元决策） */
    Resolved: 'resolved',
    /** 已完善（参数补全提交，params_completed） */
    ParamsCompleted: 'params_completed',
    /** 已回答（人工输入提交，answered） */
    Answered: 'answered',
    /** 已跳过（skipped） */
    Skipped: 'skipped',
} as const;

export type InterruptResumeStatusValue = (typeof InterruptResumeStatus)[keyof typeof InterruptResumeStatus];

/** 判断中断是否已决策（非 Pending 即视为已决策） */
export function isInterruptDecided(status?: string): boolean {
    return !!status && status !== InterruptResumeStatus.Pending;
}

/**
 * 「已决议且恢复执行」状态集合（单一权威来源，INTERRUPT_EXECUTION_RESUMED_STATUSES）
 *
 * 集合内状态表示决策已作出且执行继续：载体 tool_call 翻转为执行中（Pending）；
 * 集合外决议（rejected / cancelled 等）表示事件终止（Cancelled）。
 *
 * 新增「恢复执行」类决策状态时只需在此追加，
 * eventWriter 决策落地全链路自动对齐，无需多处同步。
 */
export const INTERRUPT_EXECUTION_RESUMED_STATUSES: ReadonlySet<string> = new Set([
    InterruptResumeStatus.Approved,
    InterruptResumeStatus.Resolved,
    InterruptResumeStatus.ParamsCompleted,
    InterruptResumeStatus.Answered,
]);

/** 判断恢复状态是否属于「已决议且已恢复执行」集合 */
export function isInterruptResolvedStatus(status: string): boolean {
    return INTERRUPT_EXECUTION_RESUMED_STATUSES.has(status);
}

// ==================== turn 级终态 ====================

/** turn_completed 事件的 status 取值（对齐后端 protocol.TurnStatus，与 item 级 ToolCallStatus 是另一值域） */
export const TurnStatus = {
    Success: 'success',
    Stopped: 'stopped',
    /**
     * 用户主动终止（取消参数补全 / 拒绝审批）导致的收尾。
     *
     * 必须与 Stopped 分开：Stopped 是「整轮生成被打断」，Cancelled 是「这一轮里某个动作被人否掉」，
     * 二者运维含义不同；且收尾文案由服务端落的 i18n 气泡承担，前端只对 Stopped 写「已停止生成」，
     * 两者混用会让用户只是拒了一条命令却看到「已停止生成」
     */
    Cancelled: 'cancelled',
    Failed: 'failed',
} as const;

// ==================== 决策解释（action → 恢复状态） ====================

/**
 * 用户操作 action 到恢复状态的映射
 *
 * 注意：这是无 handler 时的兜底解释（interpretDecisionWithFallback）。
 * 决策解释的权威出处是各中断类型 handler 的 interpretDecision（见 interrupt/handlers/），
 * eventWriter 委托 handler 解释，仅无 handler 时回退到此表
 */
const ACTION_TO_RESUME_STATUS: Record<string, string> = {
    approve: InterruptResumeStatus.Approved,
    approved: InterruptResumeStatus.Approved,
    reject: InterruptResumeStatus.Rejected,
    rejected: InterruptResumeStatus.Rejected,
    cancel: InterruptResumeStatus.Cancelled,
    // 提交类动作：对齐后端 ResumeTypeOfAction（complete → params_completed），
    // 保证实时解释结果与持久化 resume.type 同值域（历史/实时展示一致）
    complete: InterruptResumeStatus.ParamsCompleted,
    // 确认类动作（选项确认）：语义等同批准
    confirm: InterruptResumeStatus.Approved,
    // 恢复决策类型直传（后端 resume.type snake_case）
    params_completed: InterruptResumeStatus.ParamsCompleted,
    answered: InterruptResumeStatus.Answered,
    skipped: InterruptResumeStatus.Skipped,
};

/** 将用户操作解释为恢复状态（未知 action 视为已解决） */
export function interpretActionStatus(action: string): string {
    return ACTION_TO_RESUME_STATUS[action] ?? InterruptResumeStatus.Resolved;
}

// ==================== resume.type → UI status 归一化（单一权威来源） ====================

/**
 * 后端 InterruptResume.type → 前端 UI status 映射（RESUME_TYPE_TO_STATUS）
 *
 * 后端持久化 `{ type: "approved" | "rejected" | "params_completed" ... }`，
 * 前端统一经 normalizeResumeStatus 归一后驱动 UI 状态机。
 * 多数 type 与 status 同值（直接透传），仅语义不等价的需要显式映射。
 *
 * 当前后端所有 resume type 都与对应 status 同值，所以表为空——它是扩展点而不是占位：
 * normalizeResumeStatus 的行为就是「未登记则同值透传」，空表下所有值都走透传分支。
 *
 * 新增后端 resume 变体时：
 * 1. 若 type === status（同值）→ 无需修改，自动透传
 * 2. 若语义不等价 → 在此追加一行，全链路（历史还原/实时决策）自动对齐
 */
export const RESUME_TYPE_TO_STATUS: Readonly<Record<string, string>> = {};

/** 将持久化 resume.type 归一化为 UI 恢复状态：空值视为待决策，未登记类型同值透传 */
export function normalizeResumeStatus(resumeType: string): string {
    if (!resumeType) return InterruptResumeStatus.Pending;
    return RESUME_TYPE_TO_STATUS[resumeType] ?? resumeType;
}

/**
 * 决议徽章配置（i18n key + 语义色淡底）：工具行与中断卡片共用同一份。
 *
 * 此前配置只在 ToolCallPart 内部，三个中断组件另写一套「待处理 / 已处理」两态粗粒度
 * 徽章，同一个 status 在两处呈现粒度不同（第二真源）；而它们又拿 readonly 当「已决策」，
 * 导致轨迹里一条从未被批准的审批显示成「已处理」。
 */
export const RESUME_STATUS_BADGES: Readonly<Record<string, { class: string; labelKey: string }>> = {
    [InterruptResumeStatus.Approved]: { class: 'bg-success/10 text-success border-success/20', labelKey: 'ai.chat.resumeStatus.approved' },
    [InterruptResumeStatus.Resolved]: { class: 'bg-success/10 text-success border-success/20', labelKey: 'ai.chat.resumeStatus.approved' },
    [InterruptResumeStatus.ParamsCompleted]: { class: 'bg-success/10 text-success border-success/20', labelKey: 'ai.chat.resumeStatus.paramsCompleted' },
    [InterruptResumeStatus.Answered]: { class: 'bg-success/10 text-success border-success/20', labelKey: 'ai.chat.resumeStatus.answered' },
    [InterruptResumeStatus.Rejected]: { class: 'bg-destructive/10 text-destructive border-destructive/20', labelKey: 'ai.chat.resumeStatus.rejected' },
    [InterruptResumeStatus.Skipped]: { class: 'bg-muted text-muted-foreground border-border', labelKey: 'ai.chat.resumeStatus.skipped' },
};

/** 未登记状态的兜底徽章（与历史实现一致，不抛错也不显示空白） */
export const FALLBACK_RESUME_BADGE = { class: 'bg-muted text-muted-foreground border-border', labelKey: 'ai.chat.resumeStatus.cancelled' };
