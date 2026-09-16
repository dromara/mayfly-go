import { ref } from 'vue';
import { labelApi, fillLabelsToItems } from '@/views/ops/label/api';

// ==================== 模块级单例状态（全局共享，只加载一次） ====================

/**
 * 标签颜色映射表。
 *
 * 同时收录两种粒度的键：
 * - `key:value`：值级别颜色（同一 key 下不同 value 可有不同颜色），优先级最高
 * - `key`：键级别颜色，作为兜底（如标签注册时只按 key 语义着色）
 */
const colorMap = ref<Record<string, string>>({});

/** 标签描述映射表，键粒度同上 */
const descriptionMap = ref<Record<string, string>>({});

let colorLoaded = false;

/** 加载标签颜色映射（全局只执行一次） */
async function ensureColorLoaded() {
    if (colorLoaded) return;
    colorLoaded = true;
    try {
        const items = await labelApi.autocomplete.request({ key: '' });
        const colors: Record<string, string> = {};
        const descriptions: Record<string, string> = {};
        for (const item of items ?? []) {
            if (item.color) colors[item.key] = item.color;
            if (item.description) descriptions[item.key] = item.description;
            for (const detail of item.valueDetails ?? []) {
                if (detail.color) colors[`${item.key}:${detail.value}`] = detail.color;
                if (detail.description) descriptions[`${item.key}:${detail.value}`] = detail.description;
            }
        }
        colorMap.value = colors;
        descriptionMap.value = descriptions;
    } catch {
        // 加载失败允许下次重试
        colorLoaded = false;
    }
}

/** 强制刷新颜色缓存（标签注册信息变更后调用） */
export async function refreshLabelColors() {
    colorLoaded = false;
    await ensureColorLoaded();
}

// ==================== 公共 API ====================

/**
 * 获取标签颜色映射（响应式）
 * 首次调用时自动触发颜色加载
 */
export function useLabelColors() {
    ensureColorLoaded();
    return { colorMap, descriptionMap };
}

/**
 * 获取标签颜色：优先取 `key:value` 级别，回落到 `key` 级别
 */
export function getLabelColor(key: string, value?: string): string {
    if (value) {
        const exact = colorMap.value[`${key}:${value}`];
        if (exact) return exact;
    }
    return colorMap.value[key] || '';
}

/**
 * 获取标签描述：优先取 `key:value` 级别，回落到 `key` 级别
 */
export function getLabelDescription(key: string, value?: string): string {
    if (value) {
        const exact = descriptionMap.value[`${key}:${value}`];
        if (exact) return exact;
    }
    return descriptionMap.value[key] || '';
}

/**
 * 获取标签 Tag 内联样式（有颜色时返回着色样式，否则返回默认灰色）
 */
export function getLabelTagStyle(key: string, value?: string): Record<string, string> {
    const color = getLabelColor(key, value);
    if (!color) {
        // 标签不存在或无颜色定义，使用默认灰色
        return {
            backgroundColor: '#f5f7fa',
            borderColor: '#dcdfe6',
            color: '#909399',
        };
    }
    return {
        backgroundColor: color + '20',
        borderColor: color,
        color: color,
    };
}

/**
 * 解析标签 JSON 字符串为 key-value 数组
 */
export function parseLabelPairs(labelsJson: string): { key: string; value: string }[] {
    if (!labelsJson || labelsJson === '{}') return [];
    try {
        const obj = JSON.parse(labelsJson);
        if (typeof obj !== 'object' || obj === null) return [];
        return Object.entries(obj).map(([key, value]) => ({ key, value: String(value) }));
    } catch {
        return [];
    }
}

/**
 * 创建标签填充函数（用于 PageTable 的 dataHandlerFn）
 *
 * @param targetType 目标类型（如 'alert_rule', 'alert_silence'）
 * @param options.labelField 标签字段名，默认 'labels'
 *
 * @example
 * ```vue
 * <PageTable :data-handler-fn="useLabelFill('alert_rule')" ... />
 * ```
 */
export function useLabelFill<T extends { id: number }>(
    targetType: string,
    options?: { labelField?: string }
): (data: any) => Promise<any> {
    ensureColorLoaded();
    const labelField = (options?.labelField || 'labels') as keyof T;

    return async (data: any): Promise<any> => {
        if (data?.list?.length) {
            await fillLabelsToItems<T>(targetType, data.list, labelField);
        }
        return data;
    };
}
