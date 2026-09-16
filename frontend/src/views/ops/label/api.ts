import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type { LabelVO, LabelQuery, LabelForm, LabelBindingVO, LabelAutocompleteItem } from './types';

/** 标签管理 API */
export const labelApi = {
    /** 分页查询标签 */
    list: Api.newGet<PageResult<LabelVO>, LabelQuery>('/labels'),
    /** 获取所有标签键 */
    keys: Api.newGet<string[]>('/labels/keys'),
    /** 获取指定 key 的所有值 */
    values: Api.newGet<string[]>('/labels/values'),
    /** 自动补全 */
    autocomplete: Api.newGet<LabelAutocompleteItem[]>('/labels/autocomplete'),
    /** 创建标签 */
    save: Api.newPost<number, LabelForm>('/labels'),
    /** 更新标签 */
    update: Api.newPut<void, LabelForm>('/labels/{id}'),
    /** 删除标签 */
    del: Api.newDelete<void>('/labels/{id}'),
    /** 获取目标的绑定标签 */
    bindings: Api.newGet<LabelBindingVO[]>('/labels/bindings/{targetType}/{targetId}'),
    /** 批量保存绑定 */
    saveBindings: Api.newPost<void, LabelBindingVO[]>('/labels/bindings'),
    /** 删除绑定 */
    deleteBinding: Api.newDelete<void>('/labels/bindings/{labelId}/{targetType}/{targetId}'),
    /** 批量填充标签绑定（类似 TagTree.FillTagInfo） */
    fillBindings: Api.newPost<Record<number, LabelBindingVO[]>>('/labels/bindings/fill'),
};

/**
 * 批量填充标签到列表项（类似 TagTree.FillTagInfo 模式）
 * @param targetType 目标类型（如 'alert_rule'）
 * @param items 列表项数组（需要有 id 字段）
 * @param labelField 标签字段名（默认 'labels'）
 */
export async function fillLabelsToItems<T extends { id: number }>(
    targetType: string,
    items: T[],
    labelField: keyof T = 'labels' as keyof T
): Promise<void> {
    if (items.length === 0) return;

    const targetIds = items.map((item) => item.id);
    try {
        const result = await labelApi.fillBindings.request({ targetType, targetIds });
        // 将标签绑定转换为 JSON 字符串并填充到每个项
        for (const item of items) {
            const bindings = result[item.id] || [];
            if (bindings.length > 0) {
                const labelsObj: Record<string, string> = {};
                for (const b of bindings) {
                    labelsObj[b.labelKey] = b.labelValue;
                }
                (item as any)[labelField] = JSON.stringify(labelsObj);
            } else {
                (item as any)[labelField] = '';
            }
        }
    } catch (e) {
        // 标签填充失败不影响列表显示，但记录警告日志便于排查
        console.warn('[label] fillLabelsToItems failed:', e);
    }
}
