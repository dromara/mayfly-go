<template>
    <component
        :is="item?.render ?? `el-${item.type}`"
        v-bind="{ ...handleSearchProps, ...placeholder, ...optionsBind, clearable: true }"
        v-on="{ ...handleEvents }"
        v-model.trim="itemValue"
    >
        <!-- 选项子节点的渲染完全由 searchTypeMeta 驱动：新增可带选项的类型只需在注册表登记，不改本模板 -->
        <template v-if="meta.optionSlot === 'cascader'" #default="{ data }">
            <span>{{ data[fieldNames.label] }}</span>
        </template>

        <template v-if="meta.optionSlot === 'select'">
            <component
                :is="`el-option`"
                v-for="(col, index) in item.options"
                :key="index"
                :label="$t(col[fieldNames.label])"
                :value="col[fieldNames.value]"
            ></component>
        </template>

        <slot v-else></slot>
    </component>
</template>

<script setup lang="ts" name="SearchFormItem">
import { computed } from 'vue';
import { SearchItem } from '../index';
import { searchTypeMeta } from '../../searchItemTypes';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

interface SearchFormItemProps {
    item: SearchItem;
}
const props = defineProps<SearchFormItemProps>();

const itemValue = defineModel('modelValue');

// 该类型的渲染元数据（绑哪个 prop、子节点怎么渲染、要不要合并 fieldNames），未登记类型即普通控件默认
const meta = computed(() => searchTypeMeta(props.item));

// fieldNames 兜底：SearchItem 未显式声明时用 EP 约定键位
const fieldNames = computed(() => {
    return {
        label: props.item?.fieldNames?.label ?? 'label',
        value: props.item?.fieldNames?.value ?? 'value',
        children: props.item.fieldNames?.children ?? 'children',
    };
});

// 选项数组按 meta 绑到组件对应 prop（select-v2/cascader→options，tree-select→data；select 走子节点不绑）
const optionsBind = computed<Record<string, unknown>>(() => {
    const prop = meta.value.optionsProp;
    return prop ? { [prop]: props.item.options } : {};
});

// 透传的 searchProps：tree-select、cascader 需按 meta 合并 fieldNames 到组件 props.props
const handleSearchProps = computed(() => {
    const { label, value, children } = fieldNames.value;
    let searchProps = props.item?.props ?? {};
    switch (meta.value.fieldNames) {
        case 'tree-select':
            searchProps = { ...searchProps, props: { ...(searchProps.props as Record<string, unknown>), label, children }, nodeKey: value };
            break;
        case 'cascader':
            searchProps = { ...searchProps, props: { ...(searchProps.props as Record<string, unknown>), label, value, children } };
            break;
    }
    return searchProps;
});

// 透传的组件事件
const handleEvents = computed(() => {
    return props.item?.events ?? {};
});

// 默认 placeholder（区间型日期/时间选择器走 range 占位；其余按 input/select 语义）
const placeholder = computed(() => {
    const search = props.item;
    const label = t(search.label);
    if (['datetimerange', 'daterange', 'monthrange'].includes(search?.props?.type as string) || search?.props?.isRange) {
        return {
            rangeSeparator: search?.props?.rangeSeparator ?? t('common.rangeSeparator'),
            startPlaceholder: search?.props?.startPlaceholder ?? t('common.startTime'),
            endPlaceholder: search?.props?.endPlaceholder ?? t('common.endTime'),
        };
    }

    const placeholder =
        (search?.props?.placeholder as string | undefined) ??
        (search?.type?.includes('input') ? t('common.pleaseInput', { label }) : t('common.pleaseSelect', { label }));
    return { placeholder: t(placeholder) };
});
</script>
