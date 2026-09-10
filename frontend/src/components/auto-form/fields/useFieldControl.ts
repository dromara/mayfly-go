/**
 * 字段家族公共组合式函数
 *
 * 从 AutoFormControl 提取的共享逻辑，供各字段家族组件（fields 下各子目录 index.vue）调用：
 * - value：嵌套路径读写 + onChange 联动（所有家族通用）
 * - disabled：字段级 + readonly 合并（所有家族通用）
 * - placeholder：自动 请输入/请选择（所有家族通用）
 * - type：字段类型解析（所有家族通用）
 * - enumValues / selectOptions / isOptionDisabled：选项相关（select-like 家族使用）
 * - state + watchOptionsLoader：异步选项加载（select 家族使用）
 */
import { computed, onWatcherCleanup, reactive, watchEffect, type ComputedRef, type Ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { getNestedValue, isNestedPath, isSelectPromptItem, setNestedValue, type AutoFormData, type AutoFormItem, type AutoFormSelectOption } from '../types';

/** 字段家族组件的公共 props（各家族组件 defineProps 的类型基础） */
export interface FieldControlProps {
    item: AutoFormItem;
    form: AutoFormData;
    /** 只读（由 AutoForm 解析全局 readonly + 字段级 readonly 后传入） */
    readonly?: boolean;
}

/** useFieldControl 返回值 */
export interface FieldControlReturn {
    /** 当前字段类型（缺省 'input'） */
    type: ComputedRef<string>;
    /** 字段值双向代理：嵌套路径走 form 读写，平铺 prop 走 modelValue；写入触发 onChange */
    value: Ref<any>;
    /** 禁用状态：readonly 或字段级 disabled（支持函数动态计算） */
    disabled: ComputedRef<boolean>;
    /** 占位文本：显式配置优先，否则自动生成 请输入/请选择{label} */
    placeholder: ComputedRef<string>;
    /** 枚举选项列表（enums 解析 + excludeValues 剔除；select / enum / radio 使用） */
    enumValues: ComputedRef<AutoFormSelectOption[]>;
    /** 单选组选项（enums 优先，否则静态 options；均应用 excludeValues 剔除） */
    radioOptions: ComputedRef<AutoFormSelectOption[]>;
    /** 选项级禁用判定 */
    isOptionDisabled: (value: unknown) => boolean;
    /** select 选项（异步加载结果 + excludeValues 剔除） */
    selectOptions: ComputedRef<AutoFormSelectOption[]>;
    /** 异步选项加载状态 */
    state: { options: AutoFormSelectOption[]; loading: boolean };
    /** 启动异步选项加载的 watchEffect（select 家族在 setup 中调用） */
    watchOptionsLoader: () => void;
}

/**
 * 字段家族公共逻辑提取
 *
 * 各家族组件在 setup 中调用，获取共享的计算属性与状态。
 * 家族组件只需关注自身模板渲染，无需重复维护值代理、禁用判定、选项加载等逻辑。
 */
export const useFieldControl = (
    props: FieldControlProps,
    modelValue: Ref<any>,
): FieldControlReturn => {
    const { t } = useI18n();

    const type = computed(() => props.item.type ?? 'input');

    /** 嵌套路径 prop（如 'extra.qywxUserId'）经 form 对象读写，平铺 prop 走 v-model */
    const nested = computed(() => isNestedPath(props.item.prop));

    /** 值代理：写入表单数据并触发 onChange 联动回调 */
    const value = computed({
        get: () => (nested.value ? getNestedValue(props.form, props.item.prop!) : modelValue.value),
        set: (v: unknown) => {
            if (nested.value) {
                setNestedValue(props.form, props.item.prop!, v);
            } else {
                modelValue.value = v;
            }
            props.item.onChange?.(v, props.form);
        },
    });

    /** 禁用状态（字段级 disabled 支持函数形式动态计算，readonly 语义等同禁用） */
    const disabled = computed(() => {
        if (props.readonly) {
            return true;
        }
        const d = props.item.disabled;
        return typeof d === 'function' ? d(props.form) : !!d;
    });

    /** 占位文本：显式配置优先，否则根据类型自动生成 请输入/请选择{label} */
    const placeholder = computed(() => {
        if (props.item.placeholder) {
            return t(props.item.placeholder);
        }
        if (!props.item.label) {
            return '';
        }
        const label = t(props.item.label, props.item.labelParams ?? {});
        return isSelectPromptItem(props.item) ? t('common.pleaseSelect', { label }) : t('common.pleaseInput', { label });
    });

    // ── 选项相关（select / enum / radio 家族使用） ──

    /** 枚举选项列表（应用 excludeValues 剔除；enums 支持对象或子集数组） */
    const enumValues = computed<AutoFormSelectOption[]>(() => {
        if (!props.item.enums) {
            return [];
        }
        const all = Object.values(props.item.enums) as AutoFormSelectOption[];
        return props.item.excludeValues?.length ? all.filter((ev) => !props.item.excludeValues!.includes(ev.value)) : all;
    });

    /** 单选组选项（enums 枚举优先，否则取静态 options；两者均应用 excludeValues 剔除） */
    const radioOptions = computed<AutoFormSelectOption[]>(() => {
        const opts = props.item.enums ? enumValues.value : Array.isArray(props.item.options) ? props.item.options : [];
        return props.item.excludeValues?.length ? opts.filter((o) => !props.item.excludeValues!.includes(o.value)) : opts;
    });

    /** 选项级禁用（select / enum / radio 通用） */
    const isOptionDisabled = (optValue: unknown): boolean => props.item.optionDisabled?.(optValue, props.form) ?? false;

    // ── select 异步选项加载 ──

    const state = reactive({
        options: [] as AutoFormSelectOption[],
        loading: false,
    });

    /** 加载序号：依赖快速变化时丢弃乱序返回的旧响应，避免竞态覆盖新结果 */
    let optionsSeq = 0;

    /** 启动异步选项加载的 watchEffect（仅 select 家族调用） */
    const watchOptionsLoader = () => {
        watchEffect(async () => {
            const options = props.item.options;
            if (!options) {
                return;
            }
            if (Array.isArray(options)) {
                state.options = options;
                return;
            }
            const seq = ++optionsSeq;
            // Vue 3.5 onWatcherCleanup：依赖重跑或作用域停止时使本次请求失效
            onWatcherCleanup(() => ++optionsSeq);
            try {
                state.loading = true;
                const result = await options(props.form);
                if (seq === optionsSeq) {
                    state.options = result;
                }
            } catch (e) {
                if (seq === optionsSeq) {
                    state.options = [];
                    console.error(`[AutoForm] 字段 ${props.item.prop} 的 options 加载失败:`, e);
                }
            } finally {
                if (seq === optionsSeq) {
                    state.loading = false;
                }
            }
        });
    };

    /** select 选项（应用 excludeValues 剔除，与 enum/radio 行为一致） */
    const selectOptions = computed<AutoFormSelectOption[]>(() =>
        props.item.excludeValues?.length ? state.options.filter((o) => !props.item.excludeValues!.includes(o.value)) : state.options,
    );

    return {
        type,
        value,
        disabled,
        placeholder,
        enumValues,
        radioOptions,
        isOptionDisabled,
        selectOptions,
        state,
        watchOptionsLoader,
    };
};
