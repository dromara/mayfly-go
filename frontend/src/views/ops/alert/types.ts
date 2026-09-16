import type { BaseModel, PageResult } from '@/types/common';

/** 告警规则 */
export interface AlertRuleVO extends BaseModel {
    id: number;
    name: string;
    status: number;
    priority: number;
    remark: string;
    resourceType: number;
    scopeType: number;
    scopeValue: string;
    condition: AlertCondition;
    notifyConfig: AlertNotifyConfig;
    evalInterval: number;
    triggerCount: number;
    recoveryCount: number;
    labels: string;
}

/** 告警规则表单 */
export interface AlertRuleForm {
    id?: number | null;
    name: string;
    status: number;
    priority: number;
    remark: string;
    resourceType: number;
    scopeType: number;
    scopeValue: string;
    condition: AlertCondition;
    notifyConfig: AlertNotifyConfig;
    evalInterval: number;
    triggerCount: number;
    recoveryCount: number;
    labels: string;
}

/** 告警条件 */
export interface AlertCondition {
    operator: string; // and / or
    items: ConditionItem[];
}

/** 条件项 */
export interface ConditionItem {
    metric: string;
    compare: string;
    value: number;
    duration: number;
}

/** 通知配置（仅分组/重试时序参数，通知路由由独立的通知策略负责） */
export interface AlertNotifyConfig {
    groupWait: number;
    groupInterval: number;
    repeatInterval: number;
}

/** 告警事件 */
export interface AlertEventVO extends BaseModel {
    id: number;
    ruleId: number;
    ruleName: string;
    priority: number;
    resourceType: number;
    resourceId: number;
    resourceName: string;
    status: number;
    metric: string;
    currentValue: number;
    threshold: string;
    firstTriggerTime: string;
    lastTriggerTime: string;
    recoverTime: string | null;
    ackUserId: number | null;
    ackTime: string | null;
    triggerCount: number;
    notifyCount: number;
    lastNotifyTime: string | null;
    escalationLevel: number;
    labels: string;
}

/** 指标元信息（由后端评估器提供，前端据此约束输入并渲染单位） */
export interface MetricDefinition {
    key: string;
    /** i18n label key，如 alert.metricCpuRate */
    label: string;
    unit: string;
    isPercent: boolean;
    hasRange: boolean;
    min: number;
    max: number;
}

/** 资源类型对应的可用指标 */
export interface ResourceMetrics {
    resourceType: number;
    metrics: MetricDefinition[];
}

/** 告警规则查询参数 */
export interface AlertRuleQuery {
    name?: string;
    status?: number;
    resourceType?: number;
    keyword?: string;
    pageNum?: number;
    pageSize?: number;
}

/** 告警事件查询参数 */
export interface AlertEventQuery {
    ruleId?: number;
    resourceType?: number;
    resourceId?: number;
    status?: number;
    priority?: number;
    keyword?: string;
    pageNum?: number;
    pageSize?: number;
}

/** 静默规则 */
export interface AlertSilenceVO extends BaseModel {
    id: number;
    name: string;
    status: number;
    matchLabels: string;
    startTime: string;
    endTime: string;
    remark: string;
}

export interface AlertSilenceForm {
    id?: number | null;
    name: string;
    status: number;
    matchLabels: string;
    startTime: string;
    endTime: string;
    remark: string;
}

export interface AlertSilenceQuery {
    name?: string;
    status?: number;
    keyword?: string;
    pageNum?: number;
    pageSize?: number;
}

/** 升级策略 */
export interface AlertEscalationVO extends BaseModel {
    id: number;
    name: string;
    status: number;
    matchLabels: string;
    rules: EscalationRule[];
    remark: string;
}

export interface EscalationRule {
    delayMinutes: number;
    channelIds: number[];
    receiverIds: number[];
}

export interface AlertEscalationForm {
    id?: number | null;
    name: string;
    status: number;
    matchLabels: string;
    rules: EscalationRule[];
    remark: string;
}

export interface AlertEscalationQuery {
    name?: string;
    status?: number;
    keyword?: string;
    pageNum?: number;
    pageSize?: number;
}

/** 抑制规则 */
export interface AlertInhibitionVO extends BaseModel {
    id: number;
    name: string;
    status: number;
    sourceMatch: string;
    targetMatch: string;
    equal: string;
    remark: string;
}

export interface AlertInhibitionForm {
    id?: number | null;
    name: string;
    status: number;
    sourceMatch: string;
    targetMatch: string;
    equal: string;
    remark: string;
}

export interface AlertInhibitionQuery {
    name?: string;
    status?: number;
    pageNum?: number;
    pageSize?: number;
}

/** 告警概览数据 */
export interface AlertOverviewData {
    // 事件状态统计
    firingCount: number;
    acknowledgedCount: number;
    recoveredCount: number;
    closedCount: number;

    // MTTR (平均恢复时间，单位秒)
    avgRecoveryTime: number;

    // 今日统计
    todayNotifyCount: number;

    // 规则统计
    ruleStats: RuleStats;

    // 通知统计（阶段一 + 阶段二 + 阶段三）
    notifyStats: NotifyStats;

    // 静默/抑制/升级统计
    silenceStats: SilenceStats;
    inhibitionStats: InhibitionStats;
    escalationStats: EscalationStats;

    // Top 告警
    topFrequent: AlertEventVO[];
    topLongest: AlertEventVO[];
    topNotified: AlertEventVO[];

    // 最近事件
    recentEvents: PageResult<AlertEventVO>;

    // 阶段三：历史趋势（最近 24 小时）
    notifyTrend: NotifyTrendPoint[];

    // 阶段三：通知质量报告
    qualityReport: NotifyQualityReport;
}

/** 规则统计 */
export interface RuleStats {
    total: number;
    enabled: number;
    disabled: number;
    byPriority: Record<string, number>;
    byResourceType: Record<number, number>;
}


/** 通知统计（阶段一 + 阶段二） */
export interface NotifyStats {
    // 阶段一：基础统计
    todaySent: number;
    successRate: number;
    policyMatched: number;
    totalEvents: number;
    unmatchedCount: number;

    // 阶段一：渠道分布（基于策略配置）
    channelDistribution: Record<string, number>;

    // 阶段二：精确统计（基于日志表）
    todaySuccess: number;
    todayFailed: number;
    byChannel: Record<string, ChannelStat>;
}

/** 渠道统计（阶段二） */
export interface ChannelStat {
    sent: number;
    success: number;
    failed: number;
    rate: number;
}

/** 通知趋势点（阶段三） */
export interface NotifyTrendPoint {
    hour: string;
    sent: number;
    success: number;
    failed: number;
}

/** 通知质量报告（阶段三） */
export interface NotifyQualityReport {
    avgSendDelay: number;
    maxSendDelay: number;
    queueLength: number;
    weeklySuccess: number;
    monthlySuccess: number;
    topFailReasons: FailReason[];
}

/** 失败原因统计（阶段三） */
export interface FailReason {
    reason: string;
    count: number;
}

/** 静默统计 */
export interface SilenceStats {
    total: number;
    active: number;
    silencedCount: number;
}

/** 抑制统计 */
export interface InhibitionStats {
    total: number;
    active: number;
    inhibitedCount: number;
}

/** 升级统计 */
export interface EscalationStats {
    total: number;
    active: number;
    escalatedCount: number;
    maxLevelReached: number;
}

/** 通知策略 */
export interface AlertNotifyPolicyVO extends BaseModel {
    id: number;
    name: string;
    status: number;
    matchLabels: string;
    channelIds: number[];
    receiverIds: number[];
    repeatInterval: number;
    remark: string;
}

export interface AlertNotifyPolicyForm {
    id?: number | null;
    name: string;
    status: number;
    matchLabels: string;
    channelIds: number[];
    receiverIds: number[];
    repeatInterval: number;
    remark: string;
}

export interface AlertNotifyPolicyQuery {
    name?: string;
    status?: number;
    keyword?: string;
    pageNum?: number;
    pageSize?: number;
}
