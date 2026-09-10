<template>
    <component :is="fieldComponent" v-model="modelValue" :item="item" :form="form" :readonly="readonly" />
</template>

<script lang="ts" setup>
import { computed, type Component } from 'vue';
import type { AutoFormData, AutoFormItem, AutoFormItemType } from './types';
import InputField from './fields/input/index.vue';
import NumberField from './fields/number/index.vue';
import TextareaField from './fields/textarea/index.vue';
import SelectField from './fields/select/index.vue';
import RadioField from './fields/radio/index.vue';
import SwitchField from './fields/switch/index.vue';
import DateField from './fields/date/index.vue';
import MonacoField from './fields/monaco/index.vue';
import TagsField from './fields/tags/index.vue';

defineOptions({
    inheritAttrs: false,
});

const props = defineProps<{
    item: AutoFormItem;
    form: AutoFormData;
    /** 只读（由 AutoForm 解析全局 readonly + 字段级 readonly 后传入） */
    readonly?: boolean;
}>();

const modelValue = defineModel<any>();

/** 字段类型 → 家族组件映射（新增类型只需在这注册对应家族组件；Partial 因 divider/group/custom 等结构类型由 AutoFormFieldCol 拦截，无需注册） */
const FIELD_MAP: Partial<Record<AutoFormItemType, Component>> = {
    input: InputField,
    password: InputField,
    number: NumberField,
    textarea: TextareaField,
    select: SelectField,
    enum: SelectField,
    radio: RadioField,
    switch: SwitchField,
    date: DateField,
    datetime: DateField,
    time: DateField,
    monaco: MonacoField,
    tags: TagsField,
};

/** 根据当前字段类型查找对应家族组件，缺省回退到输入家族并输出开发环境警告 */
const fieldComponent = computed<Component>(() => {
    const type = props.item.type ?? 'input';
    const component = FIELD_MAP[type];
    if (!component && import.meta.env.DEV) {
        console.warn(`[AutoForm] 未注册的控件类型 "${type}"，已回退到输入组件。请在 FIELD_MAP 中注册对应家族组件。`);
    }
    return component ?? InputField;
});
</script>
