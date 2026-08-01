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
    />

    <!-- 数字输入框 -->
    <el-input-number v-else-if="type == 'number'" v-model="value" :min="item.min" :max="item.max" :disabled="disabled" v-bind="item.props" />

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
        <el-option v-for="option in state.options" :key="`${option.value}`" :label="option.label" :value="option.value" />
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
        <el-option v-for="ev in enumValues" :key="`${ev.value}`" :label="$t(ev.label)" :value="ev.value" />
    </el-select>

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
    <monaco-editor v-else-if="type == 'monaco'" v-model="value" class="w-full" v-bind="item.props" />
</template>

<script lang="ts" setup>
import { computed, reactive, watchEffect } from 'vue';
import { useI18n } from 'vue-i18n';
import { useI18nPleaseInput, useI18nPleaseSelect } from '@/hooks/useI18n';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { isSelectLikeItem, type AutoFormData, type AutoFormItem, type AutoFormSelectOption } from './types';

const props = defineProps<{
    item: AutoFormItem;
    form: AutoFormData;
}>();

/** 字段值（由 AutoForm 通过 v-model="model[item.prop]" 绑定） */
const modelValue = defineModel<any>();

const { t } = useI18n();

const type = computed(() => props.item.type ?? 'input');

/** 值代理：写入表单数据并触发 onChange 联动回调 */
const value = computed({
    get: () => modelValue.value,
    set: (v: unknown) => {
        modelValue.value = v;
        props.item.onChange?.(v, props.form);
    },
});

/** 禁用状态（支持函数形式动态计算） */
const disabled = computed(() => {
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
    return isSelectLikeItem(props.item) ? useI18nPleaseSelect(props.item.label) : useI18nPleaseInput(props.item.label);
});

/** 枚举选项列表 */
const enumValues = computed(() => (props.item.enums ? Object.values(props.item.enums) : []));

// ── select 选项加载（静态数组直接使用，函数则异步加载并跟踪表单依赖） ──
const state = reactive({
    options: [] as AutoFormSelectOption[],
    loading: false,
});

watchEffect(async () => {
    const options = props.item.options;
    if (!options) {
        return;
    }
    if (Array.isArray(options)) {
        state.options = options;
        return;
    }
    try {
        state.loading = true;
        state.options = await options(props.form);
    } finally {
        state.loading = false;
    }
});
</script>
<style lang="scss" scoped></style>
