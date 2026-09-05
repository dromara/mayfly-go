/**
 * AutoForm 弹层宿主（Dialog / Drawer）统一宿主逻辑
 *
 * 收敛 AutoFormDialog / AutoFormDrawer 完全对称的字段解析、编辑回填、
 * confirmApi 统一提交、confirm 事件兼容（含返回 Promise 的防重等待）、
 * expose 契约（AutoFormInstance 含 submit/submitting），保证宿主能力单一出处：
 * 新增宿主行为只需改这里，Dialog/Drawer 仅保留模板与形态差异（宽高/方向/头部结构等），
 * 杜绝两宿主契约漂移（历史上 opened/body-extra 曾漏补 Dialog 一轮）。
 *
 * 放置于 hooks/ 而非 auto-form 目录内：auto-form 下纯 .ts 属 core 层
 * （由 __tests__/architecture.test.ts 守护 UI 框架无关），本 composable 依赖
 * Msg/useI18nFormValidate（element-plus ElMessage），属宿主层实现。
 */
import { getCurrentInstance, reactive, ref, watch } from 'vue';
import type { Ref } from 'vue';
import { Msg, useI18nFormValidate } from './useI18n';
import { cloneFormData, resolveFormItems, type AutoFormItemsProps } from '@/components/auto-form/shared';
import { buildDefaultForm, type AutoFormData, type AutoFormInstance } from '@/components/auto-form/types';

/** 宿主公共 props 子集（items/schema/tabs + data/confirmApi；title/width 等形态 props 由各宿主自行声明） */
export interface AutoFormHostProps extends AutoFormItemsProps {
    /** 编辑数据（对象回填表单；false/null 表示新增，按字段 defaultValue 回填） */
    data?: AutoFormData | boolean | null;
    /** 统一提交 API（可选）：传入后确认走内置默认提交逻辑，confirm 事件不再触发 */
    confirmApi?: (form: AutoFormData) => Promise<unknown>;
}

/** 宿主回调（由宿主转发为同名 emit，composable 不直接依赖 defineEmits） */
export interface AutoFormHostCallbacks {
    /** 点击取消/关闭（visible 置 false 由 composable 负责，此处通常转发 emit('cancel')） */
    onCancel?: () => void;
    /** 弹层打开且回填完成后触发，参数为内部表单引用（供父组件异步补充回填字段） */
    onOpened?: (form: AutoFormData) => void;
    /** confirmApi 默认提交成功后触发（父组件通常在此刷新列表） */
    onSubmitted?: (form: AutoFormData) => void;
}

/**
 * 宿主逻辑复用入口
 *
 * @param options.props 宿主公共 props（结构化子集，直接传宿主 defineProps 结果）
 * @param options.visible 宿主 visible model（defineModel 返回值）
 * @param options.autoFormRef 宿主内部 AutoForm 的模板 ref
 */
export const useAutoFormHost = (options: {
    props: AutoFormHostProps;
    visible: Ref<boolean>;
    autoFormRef: { readonly value: AutoFormInstance | null | undefined };
} & AutoFormHostCallbacks) => {
    const { props, visible, autoFormRef } = options;

    /** 生效的字段配置：schema 优先编译，否则使用 items（tabs 模式下合并所有 Tab 字段，供 buildDefaultForm 使用，与 AutoForm 共用解析逻辑） */
    const items = resolveFormItems(props);

    const state = reactive({
        form: {} as AutoFormData,
    });

    // 弹层打开时回填编辑数据或应用字段默认值（深拷贝，避免嵌套对象编辑中突变污染外部行数据）。
    // 仅在打开瞬间回填：打开期间外部 data 引用变化（如列表刷新）不重置表单，避免丢失用户已填内容
    watch(visible, (v) => {
        if (!v) {
            return;
        }
        if (props.data && typeof props.data === 'object') {
            state.form = cloneFormData(props.data);
        } else {
            state.form = buildDefaultForm(items.value);
        }
        // 回填完成后抛出内部表单引用（Dialog/Drawer 契约一致）
        options.onOpened?.(state.form);
    });

    const onCancel = () => {
        visible.value = false;
        options.onCancel?.();
    };

    /** confirm 提交期状态：从点击确认（含校验）到提交动作 settle 期间按钮 loading 且忽略重复点击。
     *  失败路径自动恢复：保存请求 reject 后按钮解除 loading，弹层保留供修改重提 */
    const confirming = ref(false);

    // confirm 处理器经 vnode props 直调（@confirm 编译为 onConfirm prop）以捕获返回的 Promise；
    // 事件回调阶段 instance 恒存在，setup 期引用安全
    const instance = getCurrentInstance();

    const onConfirm = async () => {
        if (confirming.value) {
            return;
        }
        confirming.value = true;
        try {
            // 校验失败内部已 toast 并抛出；返回 false 表示表单 ref 未就绪（如销毁中），静默忽略本次确认
            const valid = await useI18nFormValidate(autoFormRef).catch(() => false);
            if (valid === false) {
                return;
            }
            // 统一提交 API 优先：内置默认提交逻辑（成功提示 → 通知父组件刷新 → 关闭弹层），
            // 失败由 API 层统一 toast，confirming 解除后弹层保留供修改重提
            if (props.confirmApi) {
                try {
                    await props.confirmApi(state.form);
                } catch {
                    // 保存失败（请求层已 toast），弹层保留供修改重试
                    return;
                }
                Msg.saveSuccess();
                options.onSubmitted?.(state.form);
                visible.value = false;
                return;
            }
            // 直接调用父组件处理器（@confirm 编译为 onConfirm prop，等价 emit 且可捕获返回值）：
            // 返回 Promise（保存请求）时保持 loading 至其 settle，覆盖请求期防重复提交；
            // 同步处理器（无返回值）立即恢复，行为与纯校验期防重一致。老调用点无需任何改动
            const handler = instance?.vnode.props?.onConfirm;
            const listeners = (Array.isArray(handler) ? handler : [handler]).filter(Boolean) as ((form: AutoFormData) => unknown)[];
            await Promise.all(listeners.map((fn) => fn(state.form))).catch(() => {}); // 保存失败由调用方 toast，弹层保留供修改重提
        } finally {
            confirming.value = false;
        }
    };

    /** 暴露内部表单方法（AutoFormInstance 契约编译期校验，外部编程式校验/单字段校验/重置/清校验/统一提交） */
    const exposed: AutoFormInstance = {
        validate: async () => autoFormRef.value?.validate(),
        validateField: async (fields?: string | string[]) => autoFormRef.value?.validateField?.(fields),
        resetFields: () => autoFormRef.value?.resetFields(),
        clearValidate: () => autoFormRef.value?.clearValidate(),
        // #footer 自定义确认按钮统一从 submit 触发完整提交流程（含防重守卫）
        submit: onConfirm,
        get submitting() {
            return confirming.value;
        },
    };

    return { items, state, confirming, onCancel, onConfirm, exposed };
};
