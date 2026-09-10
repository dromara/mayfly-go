<template>
    <div class="auto-form-schema-edit w-full!">
        <ATable :data="fieldList" stripe class="w-full!">
            <ATableColumn prop="prop" :label="$t('components.af.modelField')" min-width="110px">
                <template #header>
                    <AButton class="ml0" type="primary" circle size="small" icon="plus" @click="addItem()"> </AButton>
                    <span class="ml-2">{{ $t('components.af.modelField') }}</span>
                </template>
                <template #default="scope">
                    <AInput v-model="scope.row.prop" :placeholder="$t('components.af.fieldModelPlaceholder')" clearable> </AInput>
                </template>
            </ATableColumn>

            <ATableColumn prop="label" :label="$t('components.af.fieldLabel')" min-width="100px">
                <template #default="scope">
                    <AInput v-model="scope.row.label" clearable> </AInput>
                </template>
            </ATableColumn>

            <ATableColumn prop="type" :label="$t('components.af.fieldType')" min-width="110px">
                <template #default="scope">
                    <ASelect :model-value="scope.row.type ?? 'input'" @update:model-value="(v: AutoFormItemType) => (scope.row.type = v)" style="width: 100%">
                        <AOption v-for="t in fieldTypes" :key="t.value" :label="$t(t.label)" :value="t.value" />
                    </ASelect>
                </template>
            </ATableColumn>

            <ATableColumn prop="placeholder" :label="$t('components.af.fieldPlaceholder')" min-width="140px">
                <template #default="scope">
                    <AInput v-model="scope.row.placeholder" clearable> </AInput>
                </template>
            </ATableColumn>

            <ATableColumn prop="options" :label="$t('components.af.optionalValues')" min-width="140px">
                <template #default="scope">
                    <AInput :model-value="optionsOf(scope.row as JsonField)" :placeholder="$t('components.af.optionalValuesPlaceholder')" clearable @update:model-value="(v: string) => setOptions(scope.row as JsonField, v)"> </AInput>
                </template>
            </ATableColumn>

            <ATableColumn prop="required" :label="$t('components.af.required')" min-width="65px">
                <template #default="scope">
                    <ACheckbox :model-value="requiredOf(scope.row as JsonField)" @update:model-value="(v: unknown) => setRequired(scope.row as JsonField, !!v)" />
                </template>
            </ATableColumn>

            <ATableColumn :label="$t('common.operation')" width="80px">
                <template #default="scope">
                    <AButton type="danger" @click="deleteItem(scope.$index)" icon="delete" plain></AButton>
                </template>
            </ATableColumn>
        </ATable>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { ATable, ATableColumn, AButton, AInput, ASelect, AOption, ACheckbox } from './ui/adapter';
import type { AutoFormItemType } from './types';
import type { AutoFormJsonSchema, JsonField } from './json';

/** 可选择的控件类型（默认 input，穷尽 AutoFormItemType 中 JSON 可表达的控件类型） */
const fieldTypes: { value: AutoFormItemType; label: string }[] = [
    { value: 'input', label: 'components.af.typeInput' },
    { value: 'password', label: 'components.af.typePassword' },
    { value: 'number', label: 'components.af.typeNumber' },
    { value: 'textarea', label: 'components.af.typeTextarea' },
    { value: 'select', label: 'components.af.typeSelect' },
    { value: 'radio', label: 'components.af.typeRadio' },
    { value: 'switch', label: 'components.af.typeSwitch' },
    { value: 'date', label: 'components.af.typeDate' },
    { value: 'datetime', label: 'components.af.typeDatetime' },
    { value: 'time', label: 'components.af.typeTime' },
    { value: 'tags', label: 'components.af.typeTags' },
    { value: 'monaco', label: 'components.af.typeMonaco' },
];

/** 表单定义（v1 JSON Schema），表格直接编辑 fields 数组 */
const fields = defineModel<AutoFormJsonSchema>('modelValue', { default: () => ({ version: 1, fields: [] as JsonField[] }) });

/** 字段列表（纯读取：缺 fields 的 schema 对象兑底空数组，写入统一走 addItem/deleteItem） */
const fieldList = computed(() => fields.value.fields ?? []);

const addItem = () => {
    // 兼容调用方传入缺 fields 的 schema 对象（如仅 { version: 1 }），先补齐再追加
    if (!fields.value.fields) {
        fields.value.fields = [];
    }
    fields.value.fields.push({ prop: '', type: 'input' });
};

const deleteItem = (index: number) => {
    fields.value.fields?.splice(index, 1);
};

// ── 嵌套属性的便捷读写（required 位于 rules 内、options 为选项数组） ──

const requiredOf = (field: JsonField): boolean => field.rules?.required === true;

const setRequired = (field: JsonField, required: boolean) => {
    if (required) {
        field.rules = { ...field.rules, required: true };
        return;
    }
    if (field.rules) {
        delete field.rules.required;
        field.rules = Object.keys(field.rules).length > 0 ? field.rules : undefined;
    }
};

/** 选项编辑沿用逗号分隔字符串（编译为静态 options，value 与 label 相同） */
const optionsOf = (field: JsonField): string => (field.options ?? []).map((o) => String(o.value)).join(',');

const setOptions = (field: JsonField, value: string) => {
    const parts = value
        .split(',')
        .map((p) => p.trim())
        .filter((p) => p !== '');
    field.options = parts.length > 0 ? parts.map((p) => ({ value: p, label: p })) : undefined;
};
</script>
