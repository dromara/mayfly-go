<template>
    <component
        :is="item?.render ?? `el-${item.type}`"
        v-bind="{ ...handleSearchProps, ...placeholder, clearable: true }"
        v-model.trim="itemValue"
        :data="item.type === 'tree-select' ? item.options : []"
        :options="['cascader', 'select-v2'].includes(item.type!) ? item.options : []"
    >
        <template v-if="item.type === 'cascader'" #default="{ data }">
            <span>{{ data[fieldNames.label] }}</span>
        </template>

        <template v-if="item.type === 'select'">
            <component
                :is="`el-option`"
                v-for="(col, index) in item.options"
                :key="index"
                :label="col[fieldNames.label]"
                :value="col[fieldNames.value]"
            ></component>
        </template>

        <slot v-else></slot>
    </component>
</template>

<script setup lang="ts" name="SearchFormItem">
import { computed } from 'vue';
import { SearchItem } from '../index';
import { useVModel } from '@vueuse/core';

interface SearchFormItemProps {
    modelValue: any;
    item: SearchItem;
}
const props = defineProps<SearchFormItemProps>();

const emit = defineEmits(['update:modelValue']);

const itemValue = useVModel(props, 'modelValue', emit);

// 判断 fieldNames 设置 label && value && children 的 key 值
const fieldNames = computed(() => {
    return {
        label: props.item?.fieldNames?.label ?? 'label',
        value: props.item?.fieldNames?.value ?? 'value',
        children: props.item.fieldNames?.children ?? 'children',
    };
});

// 接收 enumMap (el 为 select-v2 需单独处理 enumData)
// const enumMap = inject('enumMap', ref(new Map()));
// const columnEnum = computed(() => {
//     let enumData = enumMap.value.get(props.item.prop);
//     if (!enumData) return [];
//     if (props.item?.type === 'select-v2' && props.item.fieldNames) {
//         enumData = enumData.map((item: { [key: string]: any }) => {
//             return { ...item, label: item[fieldNames.value.label], value: item[fieldNames.value.value] };
//         });
//     }
//     return enumData;
// });

// 处理透传的 searchProps (type 为 tree-select、cascader 的时候需要给下默认 label && value && children)
const handleSearchProps = computed(() => {
    const label = fieldNames.value.label;
    const value = fieldNames.value.value;
    const children = fieldNames.value.children;
    const searchEl = props.item?.type;
    let searchProps = props.item?.props ?? {};
    if (searchEl === 'tree-select') {
        searchProps = { ...searchProps, props: { ...searchProps.props, label, children }, nodeKey: value };
    }
    if (searchEl === 'cascader') {
        searchProps = { ...searchProps, props: { ...searchProps.props, label, value, children } };
    }
    return searchProps;
});

// 处理默认 placeholder
const placeholder = computed(() => {
    const search = props.item;
    const label = search.label;
    if (['datetimerange', 'daterange', 'monthrange'].includes(search?.props?.type) || search?.props?.isRange) {
        return {
            rangeSeparator: search?.props?.rangeSeparator ?? '至',
            startPlaceholder: search?.props?.startPlaceholder ?? '开始时间',
            endPlaceholder: search?.props?.endPlaceholder ?? '结束时间',
        };
    }
    const placeholder = search?.props?.placeholder ?? (search?.type?.includes('input') ? `请输入${label}` : `请选择${label}`);
    return { placeholder };
});
</script>
