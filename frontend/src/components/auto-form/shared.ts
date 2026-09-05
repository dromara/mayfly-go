/**
 * AutoForm 系列共享逻辑
 *
 * 收敛 AutoForm / AutoFormDialog / AutoFormDrawer 三处重复的字段配置解析、
 * 动态 required 判定与编辑数据回填逻辑，保证行为单一出处（改一处全站生效）。
 */
import { computed, isProxy, toRaw, type ComputedRef } from 'vue';
import { compileJsonForm, type AutoFormJsonSchema, type JsonField } from './json';
import type { AutoFormData, AutoFormItem, AutoFormTab } from './types';

/** 宿主组件的字段配置来源（AutoForm / AutoFormDialog / AutoFormDrawer 的公共 props 子集） */
export interface AutoFormItemsProps {
    /** 字段配置（与 schema 二选一） */
    items?: AutoFormItem[];
    /** v1 JSON Schema 表单定义（编译为 items，与 items 二选一，优先 schema） */
    schema?: AutoFormJsonSchema | JsonField[];
    /** Tab 页签布局（每个 Tab 为一组字段，共享表单数据与校验） */
    tabs?: AutoFormTab[];
}

/**
 * 解析生效的字段配置：tabs 合并优先，其次 schema 编译，最后 items
 *
 * AutoForm 用于渲染与校验规则生成，Dialog/Drawer 用于 buildDefaultForm 默认值构建。
 */
export const resolveFormItems = (props: AutoFormItemsProps): ComputedRef<AutoFormItem[]> =>
    computed<AutoFormItem[]>(() => {
        if (props.tabs?.length) {
            return props.tabs.flatMap((tab) => tab.items);
        }
        const schema = props.schema;
        // schema 内置 tabs：以 tabs 为准（与渲染层一致），合并所有 Tab 字段，
        // 否则 Dialog/Drawer 的 buildDefaultForm 会拿到空 items 丢失各字段 defaultValue
        if (schema && !Array.isArray(schema) && schema.tabs?.length) {
            return schema.tabs.flatMap((tab) => compileJsonForm(tab.fields));
        }
        return schema ? compileJsonForm(schema) : (props.items ?? []);
    });

/**
 * 动态 required 解析（支持根据表单值计算的条件必填）
 *
 * AutoForm 生成校验规则与 AutoFormFieldCol 星号显示共用，避免两处判定逻辑漂移。
 */
export const isItemRequired = (item: AutoFormItem, form: AutoFormData): boolean => {
    const r = item.required;
    return typeof r === 'function' ? r(form) : !!r;
};

/**
 * 字段可见性判定（hidden 不渲染但保留字段值；when 返回 false 时隐藏且不参与校验）
 *
 * AutoFormFieldCol 渲染与 AutoFormFields 分组空壳判定（组内全部隐藏时整组不渲染）共用，
 * 避免两处显隐逻辑漂移。
 */
export const isItemVisible = (item: AutoFormItem, form: AutoFormData): boolean => !item.hidden && (!item.when || item.when(form));

/**
 * 编辑数据回填拷贝：深拷贝避免嵌套对象与源数据共享引用
 *
 * 嵌套路径 prop（如 'meta.icon'）经 setNestedValue 原地写入，浅拷贝会把
 * 编辑中的中间突变直接污染外部列表行数据（确认保存前数据已被改动）。
 *
 * 注意：props 传入的是响应式 Proxy，structuredClone 无法克隆 Proxy（抛 DataCloneError），
 * 必须先 toRaw 还原原始对象再克隆，否则深拷贝静默退化为浅拷贝。
 */
export const cloneFormData = (data: AutoFormData): AutoFormData => {
    const raw = isProxy(data) ? toRaw(data) : data;
    try {
        return structuredClone(raw);
    } catch {
        // 含不可结构化克隆的值（File/Blob 等）时退化为浅拷贝
        return { ...raw };
    }
};
