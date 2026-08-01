<template>
    <el-dialog v-model="visible" :title="title" :destroy-on-close="true" width="600px" body-class="h-[65vh] overflow-auto">
        <el-form ref="dataForm" :model="modelValue" scroll-to-error :show-message="false" label-width="auto" size="small">
            <el-form-item
                v-for="column in columns"
                :key="column.columnName"
                :prop="column.columnName"
                :required="props.tableName != '' && !column.nullable && !column.isPrimaryKey && !column.autoIncrement"
            >
                <template #label>
                    <span class="cursor-pointer" :title="column?.columnComment ? `${column.columnType} | ${column.columnComment}` : column.columnType">
                        {{ column.columnName }}
                    </span>
                </template>

                <ColumnFormItem
                    v-model="modelValue[`${column.columnName}`]"
                    :data-type="dbInst.getDialect().getDataType(column.dataType ?? '')"
                    :placeholder="column?.columnComment ? `${column.columnType} | ${column.columnComment}` : column.columnType"
                    :column-name="column.columnName"
                    :disabled="column.autoIncrement"
                />
            </el-form-item>
        </el-form>
        <template #footer v-if="props.tableName">
            <el-button @click="onCloseDialog">{{ $t('common.cancel') }}</el-button>
            <el-button type="primary" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted, useTemplateRef } from 'vue';
import type { FormInstance } from 'element-plus';
import ColumnFormItem from './ColumnFormItem.vue';
import { DbInst } from '../../db';
import { useI18nFormValidate } from '@/hooks/useI18n';
import type { TableColumnDef } from '../../types';

export interface ColumnFormItemProps {
    dbInst: DbInst;
    dbName: string;
    tableName: string;
    columns: TableColumnDef[];
    title?: string; // dialog title
}

const props = withDefaults(defineProps<ColumnFormItemProps>(), {
    title: '',
});

const modelValue = defineModel<Record<string, unknown>>('modelValue', { default: () => ({}) });

const visible = defineModel<boolean>('visible', {
    default: false,
});

const emit = defineEmits(['submitSuccess']);

const dataForm = useTemplateRef<FormInstance>('dataForm');

let oldValue = null as Record<string, unknown> | null;

onMounted(() => {
    setOldValue();
});

watch(visible, (newValue) => {
    if (newValue) {
        setOldValue();
    }
});

const setOldValue = () => {
    // 空对象则为insert操作，否则为update
    if (Object.keys(modelValue.value).length > 0) {
        oldValue = Object.assign({}, modelValue.value);
    }
};

const onCloseDialog = () => {
    visible.value = false;
    modelValue.value = {};
};

const onConfirm = async () => {
    await useI18nFormValidate(dataForm);

    const dbInst = props.dbInst;
    const data = modelValue.value;
    const db = props.dbName;
    const tableName = props.tableName;

    let sql = '';
    if (oldValue) {
        const old = oldValue;
        const updateColumnValue: Record<string, unknown> = {};
        Object.keys(old).forEach((key) => {
            // 如果新旧值不相等，则为需要更新的字段
            if (old[key] !== modelValue.value[key]) {
                updateColumnValue[key] = modelValue.value[key];
            }
        });
        sql = await dbInst.genUpdateSql(db, tableName, updateColumnValue, old);
    } else {
        sql = await dbInst.genInsertSql(db, tableName, [data], true);
    }

    dbInst.promptExeSql(db, sql, undefined, () => {
        onCloseDialog();
        emit('submitSuccess');
    });
};
</script>

<style lang="scss"></style>
