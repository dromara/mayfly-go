<template>
    <div class="auto-form-schema-edit w-full!">
        <el-table :data="fieldList" stripe class="w-full!">
            <el-table-column prop="prop" :label="$t('components.af.modelField')" min-width="110px">
                <template #header>
                    <el-button class="ml0" type="primary" circle size="small" icon="plus" @click="addItem()"> </el-button>
                    <span class="ml-2">{{ $t('components.af.modelField') }}</span>
                </template>
                <template #default="scope">
                    <el-input v-model="scope.row.prop" :placeholder="$t('components.af.fieldModelPlaceholder')" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="label" :label="$t('components.af.fieldLabel')" min-width="100px">
                <template #default="scope">
                    <el-input v-model="scope.row.label" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="type" :label="$t('components.af.fieldType')" min-width="110px">
                <template #default="scope">
                    <el-select :model-value="scope.row.type ?? 'input'" @update:model-value="(v: AutoFormItemType) => (scope.row.type = v)" style="width: 100%">
                        <el-option v-for="t in fieldTypes" :key="t.value" :label="$t(t.label)" :value="t.value" />
                    </el-select>
                </template>
            </el-table-column>

            <el-table-column prop="placeholder" :label="$t('components.af.fieldPlaceholder')" min-width="140px">
                <template #default="scope">
                    <el-input v-model="scope.row.placeholder" clearable> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="options" :label="$t('components.af.optionalValues')" min-width="140px">
                <template #default="scope">
                    <el-input :model-value="optionsOf(scope.row)" :placeholder="$t('components.af.optionalValuesPlaceholder')" clearable @update:model-value="(v: string) => setOptions(scope.row, v)"> </el-input>
                </template>
            </el-table-column>

            <el-table-column prop="required" :label="$t('components.af.required')" min-width="65px">
                <template #default="scope">
                    <el-checkbox :model-value="requiredOf(scope.row)" @update:model-value="(v: unknown) => setRequired(scope.row, !!v)" />
                </template>
            </el-table-column>

            <el-table-column :label="$t('common.operation')" width="80px">
                <template #default="scope">
                    <el-button type="danger" @click="deleteItem(scope.$index)" icon="delete" plain></el-button>
                </template>
            </el-table-column>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import type { AutoFormItemType } from './types';
import type { AutoFormJsonSchema, JsonField } from './json';

/** 可选择的控件类型（默认 input） */
const fieldTypes: { value: AutoFormItemType; label: string }[] = [
    { value: 'input', label: 'components.af.typeInput' },
    { value: 'password', label: 'components.af.typePassword' },
    { value: 'number', label: 'components.af.typeNumber' },
    { value: 'textarea', label: 'components.af.typeTextarea' },
    { value: 'select', label: 'components.af.typeSelect' },
    { value: 'switch', label: 'components.af.typeSwitch' },
    { value: 'date', label: 'components.af.typeDate' },
    { value: 'datetime', label: 'components.af.typeDatetime' },
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
