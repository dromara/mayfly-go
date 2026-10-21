/** 指令段复用聊天侧的芯片段类型（同一形状，避免定义第二份真源） */
import type { ChipSegment } from '@/views/ai/input/chipRegistry';

/** Agent 装配档目录条目（对应后端 headless.ProfileInfo；档位集合以后端注册表为准） */
export interface AgentProfileInfo {
    id: string;
    /** 后端登记的档名（中文），仅用于 i18n 未覆盖的新增档兜底展示 */
    name: string;
    description: string;
    /** 该档必须由一次性运行许可驱动：无人值守派发（cron/事件/手动）不下发许可，表单需置灰 */
    requireRunGrant: boolean;
}

/** 定时/计划触发的 Agent 任务（对应后端 entity.AgentTask / vo.AgentTaskVO） */
export interface AgentTask {
    id: number;
    name: string;
    key: string;
    instruction: string;
    cron: string;
    profileId: string;
    /** 指令芯片引用段（技能码 / 资源定位信息），派发时展开为模型可直接取参的文本 */
    instructionSegments?: ChipSegment[];
    ownerAccountId: number;
    ownerUsername: string;
    status: number;
    timeoutSeconds: number;
    retryTimes: number;
    maxStep: number;
    notifyType: number;
    notifyTmplCode: string;
    misfirePolicy: number;
    lastFireTime?: string | null;
    remark: string;
    createTime?: string;
    creator?: string;
    updateTime?: string;
    modifier?: string;
    /** cron 是否已注册在调度器（后端按 scheduler.ExistKey 回填） */
    running?: boolean;
    /** 影子模式（演练）：触发的运行只读执行，仅产出「将执行什么」不落地变更 */
    dryRun?: boolean;
    /** 单次运行 token 预算熔断（0=不熔断） */
    tokenBudget?: number;
}

/** 任务保存表单（对应后端 form.AgentTaskForm；ownerAccountId 留空则以当前登录账号为运行身份） */
export interface AgentTaskForm {
    id?: number;
    name: string;
    instruction: string;
    cron?: string;
    profileId?: string;
    /** 指令芯片引用段（保存时与 instruction 同源提交；缺省表示纯文本指令） */
    instructionSegments?: ChipSegment[];
    ownerAccountId?: number;
    status?: number;
    timeoutSeconds?: number;
    retryTimes?: number;
    maxStep?: number;
    notifyType?: number;
    notifyTmplCode?: string;
    misfirePolicy?: number;
    dryRun?: boolean;
    tokenBudget?: number;
    remark?: string;
}

/** 一次 Agent 无头运行记录（对应后端 entity.Run / vo.RunVO；instruction/input 仅详情接口返回） */
export interface AgentRun {
    id: number;
    runNo: string;
    triggerType: string;
    triggerRef: string;
    /** 编排子运行的父运行 id（0=非编排子运行）；配合 triggerType=orchestration 定位整棵编排树 */
    parentRunId?: number;
    profileId: string;
    title: string;
    /** 运行形态（编排规划/步骤/审查、评测重放、提案执行/校验；空=普通运行）。
     *  形态不靠标题中文前缀传递（那样英文界面无法本地化），由后端结构化回传，见 enums.ts RunRoleEnum */
    runRole?: string;
    status: string;
    /** 人类反馈（up=点赞 down=点踩；空/缺失=尚未反馈）。列表必须带：否则看不出哪条已给过反馈，
     *  会重复反馈、也无法按反馈态筛（真源 entity.Run.Feedback，字段名必须与后端一致） */
    feedback?: string;
    ownerAccountId: number;
    ownerUsername: string;
    conversationId: number;
    turnId: string;
    summary: string;
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
    startedAt?: string | null;
    finishedAt?: string | null;
    costMs: number;
    err: string;
    createTime?: string;
    /** 以下为详情接口（GET /agentops/runs/{id}）额外返回的字段（与后端 vo.RunDetailVO 同源，
     *  不在 VO 里的字段不得声明：据虚字段展示永远拿到 undefined，会被误判成后端没落库） */
    instruction?: string;
    input?: string;
    /** 影子模式（演练）：只读执行，仅产出「将执行什么」不落地变更 */
    dryRun?: boolean;
}

/** 事件驱动触发器（对应后端 entity.Trigger / vo.TriggerVO）：把某来源的某类事件绑定到一次 Agent 运行配置 */
export interface AgentTrigger {
    id: number;
    name: string;
    /** 触发源类型：alert / webhook（对应后端 trigger.SourceType*） */
    sourceType: string;
    /** 匹配条件 JSON（空=命中该来源全部事件） */
    matchCond: string;
    /** 关联任务 id（>0 复用任务全套运行配置；0=用内联字段） */
    taskId: number;
    /**
     * 关联任务名（后端 VO 批量回查后回传，列表与详情同源）：界面拿它展示「关联任务：某巡检」。
     * 任务已被删除时为空串（此时 taskId 仍非零：删任务只禁用触发器、不解绑），
     * 展示侧必须按「任务已不存在」给文案，不得退回亮裸 id
     */
    taskName: string;
    instruction: string;
    /**
     * 指令的芯片引用段（与 AgentTask 同源语义）：后端 VO 现在会回传它。
     * 不回传则编辑抽屉打开时看不到已有引用，用户保存一次就把已存芯片段静默清空
     */
    instructionSegments?: ChipSegment[];
    profileId: string;
    /**
     * 内联模式的结果通知配置（关联任务时以任务为准）：没有它，
     * webhook/告警驱动的诊断运行结果永远无处可达（后端存 Extra，非查询维度）
     */
    notifyType: number;
    notifyTmplCode: string;
    ownerAccountId: number;
    ownerUsername: string;
    cooldownSeconds: number;
    /**
     * webhook 推送令牌：只在详情接口回传（该接口与保存同一权限点把守），列表接口为空。
     * 它是免鉴权 webhook 路由的唯一凭据，不能出现在无权限码的列表返回体里
     */
    webhookToken?: string;
    enabled: number;
    lastFireTime?: string | null;
    remark: string;
    createTime?: string;
    creator?: string;
    updateTime?: string;
    modifier?: string;
}

/**
 * 触发器保存表单（对应后端 form.TriggerForm）。
 * 运行配置二选一：taskId>0 复用关联任务配置（无需 owner/instruction）；否则内联 instruction（ownerAccountId 留空取当前登录账号）。
 */
export interface AgentTriggerForm {
    id?: number;
    name: string;
    sourceType: string;
    matchCond?: string;
    taskId?: number;
    instruction?: string;
    instructionSegments?: ChipSegment[];
    profileId?: string;
    notifyType?: number;
    notifyTmplCode?: string;
    ownerAccountId?: number;
    cooldownSeconds?: number;
    /** true=显式轮换推送令牌（旧地址立即失效）；不传则一律保留库中现值 */
    rotateWebhookToken?: boolean;
    enabled?: number;
    remark?: string;
}

/**
 * 变更提案（对应后端 entity.Proposal / vo.ProposalVO）：自主运维闭环「诊断 → 提案 → 审批 → 受控执行 → 验证」的记录。
 * 由只读诊断档经 propose_remediation 工具登记（非用户手建），故前端只读展示 + 删除终态，审批动作在 flow 工单页完成。
 */
export interface AgentProposal {
    id: number;
    /** 发起提案的无头运行 id（0=无关联） */
    runId: number;
    /** 关联会话 id（回放入口；无痕运行时为 0） */
    conversationId: number;
    title: string;
    /** 变更动作（工具名或语义动作，如 ExecSql / MachineCommandExec） */
    action: string;
    /** 锁定参数 JSON（审批通过后执行档严格按此执行，不得自由发挥） */
    params: string;
    /** 风险级：low / medium / high / critical */
    risk: string;
    /** 提案理由（诊断依据 + 预期效果） */
    reason: string;
    /** 状态：pending / approved / rejected / executed / failed */
    status: string;
    ownerAccountId: number;
    ownerUsername: string;
    /** 审批流程实例 id（0=未绑定审批流） */
    procInstId: number;
    /** 审批流程定义 id（发起工单时快照） */
    procDefId: number;
    /** 受控执行运行 id（0=尚未执行） */
    execRunId: number;
    /** 只读校验运行 id（0=尚未校验） */
    verifyRunId: number;
    /** 执行结果摘要（执行档最终输出） */
    execResult: string;
    /** 校验结论（目标是否达成/是否需回滚） */
    verifyConclusion: string;
    err: string;
    createTime?: string;
    creator?: string;
    updateTime?: string;
    modifier?: string;
}

/** 治理看板统计（对应后端 application.GovernanceStats）：急停状态 + 窗口内运行 KPI + 全量资源盘点 */
export interface GovernanceStats {
    /** 急停是否已拉起（true = 所有 task/trigger 停止派发新运行） */
    killSwitch: boolean;
    /** 运行 KPI 统计窗口天数 */
    windowDays: number;
    /** 窗口内运行总数 */
    runTotal: number;
    /** 窗口内运行状态分布（后端 entity.ActiveRunStatuses + 三个终态） */
    runByStatus: Record<string, number>;
    /** 当前非终态运行数（全量） */
    activeRuns: number;
    /** 窗口内终态运行成功率（0-1） */
    successRate: number;
    /** 窗口内消耗总 token */
    totalTokens: number;
    /** 窗口内输入 token */
    promptTokens: number;
    /** 窗口内输出 token */
    completionTokens: number;
    /** 窗口内已完结运行平均耗时（毫秒） */
    avgCostMs: number;
    /** 待审批变更提案数（全量） */
    pendingProposals: number;
    /**
     * 停滞提案数：审批已领取（approved）但受控执行超过 stallAfterMinutes 仍未落库（execRunId=0）。
     * 判据与后端停滞清扫共用同一份，故这个数字恰好等于「下一轮清扫会去重投的那批」。
     * 成因是执行前进程退出丢了单（受控执行走进程内队列），而工单在回调那一刻已记为处理成功，
     * flow 侧的重推入口对它全部不可达——由 agentops 自己的停滞清扫自动重投（幂等键防重复执行），
     * 故本计数是「尚未被清扫收走」的瞬时值，长期不归零才说明重投本身在失败（去提案详情看 err）
     */
    stuckApproved: number;
    /**
     * 停滞判定的静默期（分钟）。展示口径一律用响应里的这个值而不是手抄前端常量，
     * 与 windowDays 同一取向：后端调阈值时界面文案不会静默说谎
     */
    stallAfterMinutes: number;
    /** 启用中的定时任务数（全量） */
    enabledTasks: number;
    /** 启用中的事件触发器数（全量） */
    enabledTriggers: number;
}

/** 可引用资源（后端 vo.ChatResourceVO）：聊天 `@` 菜单与触发器匹配条件的资源白名单共用 */
export interface ChatResource {
    /** 资源 id（字符串统一承载，消费侧按数值用） */
    id: string;
    /** 资源类型稳定键串（machine/db/redis/...） */
    resourceType: string;
    name: string;
    code: string;
    ip?: string;
    port?: number;
    description?: string;
}
