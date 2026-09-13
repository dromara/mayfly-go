<template>
    <ARow :gutter="16">
        <AutoFormFieldCol v-for="item in items" :key="item.prop ?? item.label ?? ''" :item="item" :form="form" :default-span="defaultSpan" :readonly="readonly">
            <template v-for="(_, name) in $slots" :key="name" #[name]="slotProps">
                <slot :name="name" v-bind="slotProps ?? {}" />
            </template>
        </AutoFormFieldCol>
    </ARow>
</template>

<script lang="ts" setup>
import { ARow } from './ui/adapter';
import AutoFormFieldCol from './AutoFormFieldCol.vue';
import type { AutoFormData, AutoFormItem } from './types';

/**
 * 字段行列表（group 分组内层与平铺布局共用的渲染单元）
 *
 * 收敛 AutoFormFields 中 group 分支与平铺分支重复的「ARow + FieldCol 循环 + 插槽转发」，
 * 保证栅格 gutter 与字段渲染单一出处；插槽继续向上透传给 FieldCol 的 custom 具名插槽。
 */
defineProps<{
    items: AutoFormItem[];
    /** 表单数据对象（共享引用） */
    form: AutoFormData;
    /** 每个字段默认栅格跨度（由 AutoFormFields 按 cols 均分） */
    defaultSpan: number;
    /** 只读 */
    readonly?: boolean;
}>();
</script>
