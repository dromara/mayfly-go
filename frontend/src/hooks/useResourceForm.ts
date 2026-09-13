import { computed, ref, type Ref } from 'vue';
import { useRoute } from 'vue-router';
import { useI18nCreateTitle, useI18nEditTitle } from './useI18n';
import type { AutoFormData } from '@/components/auto-form/types';

/**
 * SSH 隧道机器 id 表单转换：空值/非正数 → -1（后端约定）。
 *
 * 各资源编辑组件（DB/Machine/Redis/Mongo/ES/Milvus/Kafka）共享同一转换逻辑，
 * 避免 getReqForm 中重复书写 if (!sshTunnelMachineId || sshTunnelMachineId <= 0) 样板。
 *
 * @param form 当前表单数据（Ref 或 computed）
 * @returns 转换后的表单（computed，原表单不被修改）
 */
export function useSshTunnelTransform(form: Ref<AutoFormData>) {
    return computed(() => {
        const reqForm = { ...form.value };
        const id = reqForm.sshTunnelMachineId as number | null | undefined;
        if (!id || id <= 0) {
            reqForm.sshTunnelMachineId = -1;
        }
        return reqForm;
    });
}

/**
 * 路由 tagPath 注入：从当前路由 query 中读取 tagPath 并注入分页查询参数。
 *
 * 列表组件通过 before-query-fn 调用此 composable 返回的处理函数，
 * 实现「从标签树点击跳转时自动按 tagPath 过滤」的统一行为。
 */
export function useRouteTagPath() {
    const route = useRoute();
    return (query: Record<string, unknown>) => {
        if (route.query.tagPath) {
            query.tagPath = route.query.tagPath as string;
        }
        return query;
    };
}

/**
 * 列表页编辑弹窗状态管理 composable。
 *
 * 收敛所有列表组件中重复的 { visible, data, title } 三件套与 edit 函数样板，
 * 统一使用 i18n 生成弹窗标题。配合 AutoFormDrawer 的 v-model:visible 使用。
 *
 * @param entityI18nKey 实体名称的 i18n key（如 'mongo.mongo'、'milvus'）
 *
 * @example
 * ```ts
 * const { editDialog, editEntity } = useEditDialog<Mongo>('mongo.mongo');
 * // 模板中：<MongoEdit v-model:visible="editDialog.visible" :data="editDialog.data" @val-change="search" />
 * // 操作按钮：@click="editEntity(null)" 新增 / @click="editEntity(row)" 编辑
 * ```
 */
export interface EditDialogState<T> {
    visible: boolean;
    data: T | null;
    title: string;
}

export function useEditDialog<T>(entityI18nKey: string) {
    const editDialog = ref<EditDialogState<T>>({
        visible: false,
        data: null,
        title: '',
    });

    const editEntity = (data: T | null | false = null) => {
        editDialog.value = {
            data: (data || null) as T | null,
            title: data ? useI18nEditTitle(entityI18nKey) : useI18nCreateTitle(entityI18nKey),
            visible: true,
        };
    };

    return { editDialog, editEntity };
}
