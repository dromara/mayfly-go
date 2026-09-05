<template>
    <!-- 文本 / 密码输入框 -->
    <el-input
        v-if="type == 'input' || type == 'password'"
        v-model="value"
        :type="type == 'password' ? 'password' : 'text'"
        :show-password="type == 'password'"
        :placeholder="placeholder"
        :disabled="disabled"
        autocomplete="off"
        clearable
        v-bind="item.props"
    >
        <template v-if="item.prefix" #prefix>{{ $t(item.prefix) }}</template>
        <template v-if="item.suffix" #suffix>{{ $t(item.suffix) }}</template>
    </el-input>

    <!-- 数字输入框 -->
    <el-input-number v-else-if="type == 'number'" v-model="value" :min="item.min" :max="item.max" :disabled="disabled" class="w-full!" v-bind="item.props">
        <template v-if="item.prefix" #prefix>{{ $t(item.prefix) }}</template>
        <template v-if="item.suffix" #suffix>{{ $t(item.suffix) }}</template>
    </el-input-number>

    <!-- 文本域 -->
    <el-input
        v-else-if="type == 'textarea'"
        v-model="value"
        type="textarea"
        :rows="item.rows ?? 3"
        :placeholder="placeholder"
        :disabled="disabled"
        v-bind="item.props"
    />

    <!-- 下拉选择（静态 / 异步 options） -->
    <el-select
        v-else-if="type == 'select'"
        v-model="value"
        :placeholder="placeholder"
        :multiple="item.multiple"
        :loading="state.loading"
        :disabled="disabled"
        filterable
        clearable
        class="w-full"
        v-bind="item.props"
    >
        <el-option
            v-for="option in selectOptions"
            :key="`${option.value}`"
            :label="$t(option.label)"
            :value="option.value"
            :disabled="option.disabled || isOptionDisabled(option.value)"
        />
    </el-select>

    <!-- 枚举下拉选择 -->
    <el-select
        v-else-if="type == 'enum'"
        v-model="value"
        :placeholder="placeholder"
        :multiple="item.multiple"
        :disabled="disabled"
        filterable
        clearable
        class="w-full"
        v-bind="item.props"
    >
        <el-option v-for="ev in enumValues" :key="`${ev.value}`" :label="$t(ev.label)" :value="ev.value" :disabled="isOptionDisabled(ev.value)" />
    </el-select>

    <!-- 单选组（enums 枚举或静态 options） -->
    <el-radio-group v-else-if="type == 'radio'" v-model="value" :disabled="disabled" v-bind="item.props">
        <el-radio v-for="opt in radioOptions" :key="`${opt.value}`" :value="opt.value" :disabled="opt.disabled || isOptionDisabled(opt.value)">
            {{ $t(opt.label) }}
        </el-radio>
    </el-radio-group>

    <!-- 开关 -->
    <el-switch v-else-if="type == 'switch'" v-model="value" :disabled="disabled" v-bind="item.props" />

    <!-- 日期 / 日期时间 / 时间 -->
    <el-date-picker
        v-else-if="type == 'date' || type == 'datetime'"
        v-model="value"
        :type="type"
        :placeholder="placeholder"
        :disabled="disabled"
        class="w-full"
        v-bind="item.props"
    />
    <el-time-picker v-else-if="type == 'time'" v-model="value" :placeholder="placeholder" :disabled="disabled" class="w-full" v-bind="item.props" />

    <!-- Monaco 代码编辑器（language/height 等通过 item.props 透传） -->
    <monaco-editor v-else-if="type == 'monaco'" v-model="value" class="w-full" v-bind="item.props" :options="monacoOptions" />

    <!-- 标签输入器（字符串数组） -->
    <el-input-tag v-else-if="type == 'tags'" v-model="value" :placeholder="placeholder" :disabled="disabled" v-bind="item.props" />
</template>

<script lang="ts" setup>
import { computed, onWatcherCleanup, reactive, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { getNestedValue, isNestedPath, isSelectLikeItem, setNestedValue, type AutoFormData, type AutoFormItem, type AutoFormSelectOption } from './types';

const props = defineProps<{
    item: AutoFormItem;
    form: AutoFormData;
    /** 只读（由 AutoForm 解析全局 readonly + 字段级 readonly 后传入） */
    readonly?: boolean;
}>();

/** 字段值（由 AutoForm 通过 v-model="model[item.prop]" 绑定） */
const modelValue = defineModel<any>();

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

/** 占位文本：显式配置优先，否则根据类型自动生成 请输入/请选择{label}（label 翻译带插值参数，与 FieldCol 一致） */
const placeholder = computed(() => {
    if (props.item.placeholder) {
        return t(props.item.placeholder);
    }
    if (!props.item.label) {
        return '';
    }
    const label = t(props.item.label, props.item.labelParams ?? {});
    return isSelectLikeItem(props.item) ? t('common.pleaseSelect', { label }) : t('common.pleaseInput', { label });
});

/** Monaco 编辑器选项：合并调用方自定义 options，并强制注入 readOnly（monaco 只认 editor option）；
 *  调用方显式 readOnly: true 保留，但不可覆盖 disabled/readonly 状态的强制只读 */
const monacoOptions = computed(() => {
    const callerOptions = ((props.item.props as { options?: Record<string, unknown> } | undefined)?.options ?? {}) as { readOnly?: boolean };
    return { ...callerOptions, readOnly: disabled.value || callerOptions.readOnly === true };
});

/** 枚举选项列表（应用 excludeValues 剔除；enums 支持对象或子集数组） */
const enumValues = computed(() => {
    if (!props.item.enums) {
        return [];
    }
    const all = Object.values(props.item.enums);
    return props.item.excludeValues?.length ? all.filter((ev) => !props.item.excludeValues!.includes(ev.value)) : all;
});

/** 单选组选项（enums 枚举优先，否则取静态 options；两者均应用 excludeValues 剔除） */
const radioOptions = computed<AutoFormSelectOption[]>(() => {
    const opts = props.item.enums ? enumValues.value : Array.isArray(props.item.options) ? props.item.options : [];
    return props.item.excludeValues?.length ? opts.filter((o) => !props.item.excludeValues!.includes(o.value)) : opts;
});

/** 选项级禁用（select / enum 通用） */
const isOptionDisabled = (value: unknown): boolean => props.item.optionDisabled?.(value, props.form) ?? false;

/** select 选项（应用 excludeValues 剔除，与 enum/radio 行为一致，兑现 types.ts「select / enum 通用」契约） */
const selectOptions = computed<AutoFormSelectOption[]>(() =>
    props.item.excludeValues?.length ? state.options.filter((o) => !props.item.excludeValues!.includes(o.value)) : state.options
);

// ── select 选项加载（静态数组直接使用，函数则异步加载并跟踪表单依赖） ──
const state = reactive({
    options: [] as AutoFormSelectOption[],
    loading: false,
});

/** 加载序号：依赖快速变化时丢弃乱序返回的旧响应，避免竞态覆盖新结果 */
let optionsSeq = 0;

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
    // Vue 3.5 onWatcherCleanup：依赖重跑或作用域停止时使本次请求失效，
    // 替代在每处比较 seq，并补齐组件卸载后响应不再写入的清理
    onWatcherCleanup(() => ++optionsSeq);
    try {
        state.loading = true;
        const result = await options(props.form);
        if (seq === optionsSeq) {
            state.options = result;
        }
    } catch (e) {
        // 加载失败：清空选项并留痕，避免残留旧选项误导用户选择
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
</script>
<style lang="scss" scoped></style>
