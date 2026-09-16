import EnumValue from '@/common/Enum';

/** 告警规则状态 */
export const AlertRuleStatusEnum = {
    Enable: EnumValue.of(1, 'common.enable'),
    Disable: EnumValue.of(-1, 'common.disable'),
};

/** 告警事件状态 */
export const AlertEventStatusEnum = {
    Firing: EnumValue.of(1, 'alert.firing').tagTypeDanger(),
    Acknowledged: EnumValue.of(2, 'alert.acknowledged').tagTypeWarning(),
    Recovered: EnumValue.of(3, 'alert.recovered').tagTypeSuccess(),
    Closed: EnumValue.of(4, 'alert.closed').tagTypeInfo(),
};

/** 告警优先级 */
export const AlertPriorityEnum = {
    Critical: EnumValue.of(0, 'alert.priorityCritical').tagTypeDanger(),
    High: EnumValue.of(1, 'alert.priorityHigh').tagTypeWarning(),
    Medium: EnumValue.of(2, 'alert.priorityMedium').tagTypeInfo(),
    Low: EnumValue.of(3, 'alert.priorityLow').tagTypeSuccess(),
};

/** 关联范围类型：统一使用标签路径多选（TagTreeCheck），不再支持单资源 ID */
export const AlertScopeTypeEnum = {
    TagPath: EnumValue.of(2, 'alert.scopeTagPath'),
};

/** 条件比较方式 */
export const AlertCompareEnum = {
    GT: EnumValue.of('gt', '>'),
    GTE: EnumValue.of('gte', '>='),
    LT: EnumValue.of('lt', '<'),
    LTE: EnumValue.of('lte', '<='),
    EQ: EnumValue.of('eq', '='),
    NEQ: EnumValue.of('neq', '!='),
};

/** 条件组合方式 */
export const AlertOperatorEnum = {
    AND: EnumValue.of('and', 'alert.operatorAnd'),
    OR: EnumValue.of('or', 'alert.operatorOr'),
};

/** 指标展示元信息 */
export interface AlertMetricMeta {
    /** i18n label key */
    label: string;
    /** 展示单位，空表示无单位 */
    unit: string;
}

/**
 * 指标 key -> 展示元信息。
 * 事件列表等只读场景需要把后端返回的原始指标 key（如 cpu_rate）渲染成本地化名称与带单位的数值。
 * 运行时的权威来源是 GET /alert-rules/metrics，该映射作为请求失败或历史数据场景的兜底，
 * 新增指标时需与后端 MetricDefinition.Key/Unit 保持一致
 */
export const AlertMetricMetaEnum: Record<string, AlertMetricMeta> = {
    cpu_rate: { label: 'alert.metricCpuRate', unit: '%' },
    mem_rate: { label: 'alert.metricMemRate', unit: '%' },
    disk_usage: { label: 'alert.metricDiskUsage', unit: '%' },
    status: { label: 'alert.metricStatus', unit: '' },
};
