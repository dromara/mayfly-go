import { ref } from 'vue';
import { i18n } from '@/i18n';
import { AlertPriorityEnum, AlertEventStatusEnum, AlertMetricMetaEnum } from './enums';
import { alertRuleApi } from './api';
import type { MetricDefinition, ResourceMetrics } from './types';
import { ResourceTypeEnum } from '@/common/commonEnum';

const t = i18n.global.t;

/** 获取优先级展示文本 */
export function getPriorityLabel(priority: number): string {
    const item = Object.values(AlertPriorityEnum).find((e) => e.value === priority);
    return item ? t(item.label) : String(priority);
}

/** 获取优先级对应的 Tag 类型 */
export function getPriorityTagType(priority: number): string {
    switch (priority) {
        case 0:
            return 'danger';
        case 1:
            return 'warning';
        case 2:
            return 'info';
        default:
            return 'success';
    }
}

/** 获取事件状态展示文本 */
export function getStatusLabel(status: number): string {
    const item = Object.values(AlertEventStatusEnum).find((e) => e.value === status);
    return item ? t(item.label) : String(status);
}

/** 获取事件状态对应的 Tag 类型 */
export function getStatusTagType(status: number): string {
    const { Firing, Acknowledged, Recovered } = AlertEventStatusEnum;
    switch (status) {
        case Firing.value:
            return 'danger';
        case Acknowledged.value:
            return 'warning';
        case Recovered.value:
            return 'success';
        default:
            return 'info';
    }
}

/** 获取资源类型展示文本 */
export function getResourceTypeLabel(type: number): string {
    const item = Object.values(ResourceTypeEnum).find((e) => Number(e.value) === type);
    return item ? t(item.label) : String(type);
}

/** 获取指标本地化名称，未知指标原样返回 */
export function getMetricLabel(metric: string): string {
    const meta = AlertMetricMetaEnum[metric];
    return meta ? t(meta.label) : metric || '-';
}

/**
 * 格式化指标数值。
 * 后端返回的是未经舍入的浮点数（如 38.239999999999995），直接展示会出现超长小数，
 * 百分比指标统一保留两位小数并补单位，status 指标渲染为在线/离线
 */
export function formatMetricValue(metric: string, value: number | null | undefined): string {
    if (value === null || value === undefined || Number.isNaN(Number(value))) {
        return '-';
    }
    if (metric === 'status') {
        return Number(value) >= 1 ? t('alert.statusOnline') : t('alert.statusOffline');
    }
    const meta = AlertMetricMetaEnum[metric];
    const text = formatNumber(Number(value));
    return meta?.unit ? `${text}${meta.unit}` : text;
}

function formatNumber(value: number): string {
    // 整数不补小数位；浮点数最多保留两位
    return Number.isInteger(value) ? String(value) : value.toFixed(2);
}

/**
 * 校验标签字段：必须是不含嵌套的 JSON 对象。
 * 后端用 map[string]string 解析，配置成数组或嵌套对象会导致解析失败、静默规则永不命中
 */
export function parseLabelJson(text: string): Record<string, string> | null {
    const trimmed = (text || '').trim();
    if (!trimmed) {
        return {};
    }
    try {
        const parsed = JSON.parse(trimmed);
        if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
            return null;
        }
        // 值必须是标量：后端以 map[string]string 解析，嵌套对象会导致解析失败
        const hasInvalidValue = Object.values(parsed).some((v) => typeof v !== 'string' && typeof v !== 'number');
        if (hasInvalidValue) {
            return null;
        }
        return parsed as Record<string, string>;
    } catch {
        return null;
    }
}

/** 资源类型下拉选项 */
export interface ResourceTypeOption {
    value: number;
    /** i18n key，由控件层负责翻译 */
    label: string;
}

/**
 * 后端已注册评估器的资源类型选项。
 *
 * 初始为空数组，完全由后端 GET /alert-rules/metrics 接口驱动：
 * 无评估器的资源类型保存后不会被任何评估周期处理（后端也会直接拒绝），
 * 因此可选项必须以后端为准，否则用户能配出永不生效的规则
 */
export const supportedResourceTypes = ref<ResourceTypeOption[]>([]);

/** 后端下发的资源类型与指标元信息，进程内只请求一次 */
let resourceMetricsRequest: Promise<ResourceMetrics[]> | null = null;

export function fetchResourceMetrics(): Promise<ResourceMetrics[]> {
    if (!resourceMetricsRequest) {
        resourceMetricsRequest = alertRuleApi.metrics
            .request({})
            .then((res) => res || [])
            .catch(() => {
                // 失败时清空以便下次重新加载，降级期间表单仍可使用前端枚举与兜底指标
                resourceMetricsRequest = Promise.resolve([]);
                return [];
            });
    }
    return resourceMetricsRequest;
}

/** 加载资源类型可选项与指标元信息，多个视图共用同一份结果 */
export async function ensureResourceMetadata(): Promise<ResourceMetrics[]> {
    const list = await fetchResourceMetrics();
    if (list.length) {
        supportedResourceTypes.value = list
            .map((it) => ({ value: Number(it.resourceType), label: getResourceTypeI18nKey(Number(it.resourceType)) }))
            .sort((a, b) => a.value - b.value);
    }
    return list;
}

/** 资源类型值对应的 i18n key，未知类型回退为原始值字符串 */
export function getResourceTypeI18nKey(type: number): string {
    const item = Object.values(ResourceTypeEnum).find((e) => Number(e.value) === type);
    return item ? item.label : String(type);
}

/** 指定资源类型的指标定义；后端无数据时返回空数组，由调用方决定兜底 */
export async function resourceMetricsOf(resourceType?: number): Promise<MetricDefinition[]> {
    if (!resourceType) {
        return [];
    }
    const list = await fetchResourceMetrics();
    return list.find((it) => Number(it.resourceType) === Number(resourceType))?.metrics ?? [];
}

/**
 * 比较操作符映射：技术符号 → 人类可读符号
 * 参考 Grafana/Prometheus Alertmanager 的展示方式
 */
const compareOperatorMap: Record<string, string> = {
    gt: '>',
    gte: '≥',
    lt: '<',
    lte: '≤',
    eq: '=',
    neq: '≠',
};

/**
 * 格式化阈值描述为人类可读文本。
 *
 * 后端返回格式："{metric} {operator} {value}"，如 "status gte 1.00"、"cpu_rate gt 80"
 * 转换为："{指标名} {操作符} {值}{单位}"，如 "在线状态 ≥ 1"、"CPU 使用率 > 80%"
 */
export function formatThreshold(threshold: string): string {
    if (!threshold) {
        return '-';
    }

    // 解析 "metric operator value" 格式
    const parts = threshold.trim().split(/\s+/);
    if (parts.length < 3) {
        return threshold; // 无法解析，原样返回
    }

    const metric = parts[0];
    const operator = parts[1];
    const value = parts[2];

    // 获取指标名称
    const metricLabel = getMetricLabel(metric);

    // 获取操作符符号
    const operatorSymbol = compareOperatorMap[operator] || operator;

    // 格式化数值（去掉不必要的小数位）
    const formattedValue = formatNumber(Number(value));

    // 获取单位
    const meta = AlertMetricMetaEnum[metric];
    const unit = meta?.unit || '';

    return `${metricLabel} ${operatorSymbol} ${formattedValue}${unit}`;
}
