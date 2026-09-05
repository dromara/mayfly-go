<template>
    <div class="create-collection-drawer">
        <auto-form-drawer
            ref="setFormRef"
            v-model:visible="visible"
            :title="ctx.drawerTitle.value"
            :items="collectionItems"
            size="80%"
            @opened="ctx.initForm"
            @cancel="ctx.handleClose"
        >
            <template #schema>
                <div class="schema-container">
                    <!-- 左侧：字段列表 -->
                    <SchemaFieldEditor />

                    <!-- 右侧：字段属性配置 -->
                    <FieldConfigPanel />
                </div>
            </template>

            <template #footer>
                <div class="drawer-footer">
                    <el-button @click="ctx.handleClose">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" @click="ctx.handleSubmit" :loading="ctx.loading.value">{{ $t('common.confirm') }}</el-button>
                    <el-button type="primary" @click="ctx.handlePreview" :loading="ctx.loading.value">{{ 'Json' + $t('common.preview') }}</el-button>
                </div>
            </template>
        </auto-form-drawer>
    </div>
</template>

<script setup lang="ts">
import FieldConfigPanel from './collection/FieldConfigPanel.vue';
import SchemaFieldEditor from './collection/SchemaFieldEditor.vue';
import { AutoFormDrawer, type AutoFormItem } from '@/components/auto-form';
import type { ApiCollection } from './collection/types';
import { provideCollectionForm, useCollectionForm } from './composables/useCollectionForm';
import type { FormInstance } from 'element-plus';

const props = defineProps<{
    milvusId: number;
    mode?: 'create' | 'edit' | 'copy';
    editData?: ApiCollection | null;
}>();

const emits = defineEmits(['success']);

const visible = defineModel<boolean>('visible', { default: false });

const ctx = useCollectionForm({
    milvusId: props.milvusId,
    get mode() {
        return props.mode;
    },
    get editData() {
        return props.editData;
    },
    visible,
    onSuccess: () => emits('success'),
});

provideCollectionForm(ctx);

// AutoFormDrawer 实例绑定到 composable 的 formRef（其暴露的 validate 与 FormInstance 方法兼容）
const setFormRef = (el: unknown) => {
    ctx.formRef.value = el as FormInstance | undefined;
};

/** 建 collection 表单声明（Schema 字段编辑区为 custom 插槽） */
const collectionItems: AutoFormItem[] = [
    { prop: 'basicDivider', label: 'common.basic', type: 'divider' },
    { prop: 'name', label: 'common.name', required: true, span: 10, placeholder: 'milvus.collectionNamePlaceholder' },
    { prop: 'shardsNum', label: 'milvus.shardsNum', type: 'number', min: 1, max: 64, span: 6 },
    { prop: 'consistency_level', label: 'milvus.consistencyLevel', type: 'select', span: 8, options: [{ label: 'Bounded', value: 'Bounded' }, { label: 'Strong', value: 'Strong' }, { label: 'Session', value: 'Session' }, { label: 'Eventually', value: 'Eventually' }] },
    { prop: 'description', label: 'milvus.description', type: 'textarea', props: { rows: 2 }, placeholder: 'milvus.descriptionPlaceholder' },
    { prop: 'schemaDivider', label: 'Schema', type: 'divider' },
    { prop: 'schema', type: 'custom' },
];
</script>

<style lang="scss" scoped>
.schema-container {
    display: flex;
    gap: 10px;
    height: calc(100vh - 300px);
}

.schema-left {
    flex: 0 0 400px;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    overflow: hidden;
}

.schema-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--el-border-color);
    background: var(--el-fill-color-light);
}

.schema-title {
    font-weight: 600;
    font-size: 14px;
}

.field-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
}

.field-item {
    display: flex;
    align-items: center;
    padding: 6px;
    margin-bottom: 8px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
    background: var(--el-fill-color-lighter);
}

.field-item:hover {
    background: var(--el-fill-color);
}

.field-item.active {
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-5);
}

.field-icon {
    margin-right: 12px;
    font-size: 18px;
    color: var(--el-text-color-regular);
}

.field-info {
    flex: 1;
    min-width: 0;
}

.field-name {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 4px;
}

.field-name-text {
    font-weight: 500;
    font-size: 14px;
}

.field-type {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.field-actions {
    opacity: 0;
    transition: opacity 0.2s;
}

.field-item:hover .field-actions {
    opacity: 1;
}

/* 只读字段样式 */
.field-item.readonly {
    cursor: default;
    opacity: 0.75;
}

.field-item.readonly:hover {
    background: var(--el-fill-color-lighter);
}

/* 动态字段样式 */
.field-item.dynamic-field {
    background: var(--el-color-warning-light-9);
    border: 1px dashed var(--el-color-warning-light-5);
}

.field-item.dynamic-field:hover {
    background: var(--el-color-warning-light-8);
}

.field-item.dynamic-field.active {
    background: var(--el-color-warning-light-9);
    border: 1px solid var(--el-color-warning-light-5);
}

/* 字段索引标签 */
.field-index-tag {
    margin: 0 8px;
    flex-shrink: 0;
}

/* 动态索引列表样式 */
.dynamic-index-list {
    display: flex;
    gap: 16px;
}

.index-list-tabs {
    flex-direction: column;
    gap: 4px;
    max-height: 400px;
    overflow-y: auto;
}

.index-list-tab {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
    background: var(--el-fill-color-lighter);
    font-size: 13px;
    margin-top: 8px;
}

.index-list-tab:hover {
    background: var(--el-fill-color);
}

.index-list-tab.active {
    background: var(--el-color-primary-light-9);
    border: 1px solid var(--el-color-primary-light-5);
    color: var(--el-color-primary);
}

.index-list-tab .delete-icon {
    opacity: 0;
    transition: opacity 0.2s;
    cursor: pointer;
    color: var(--el-color-danger);
}

.index-list-tab:hover .delete-icon {
    opacity: 1;
}

.index-empty {
    padding: 20px 0;
}

/* 表单提示 */
.form-tip {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.5;
    margin-top: 4px;
}

.schema-right {
    flex: 1;
    border: 1px solid var(--el-border-color);
    border-radius: 4px;
    padding: 16px;
    overflow-y: auto;
}

.index-config {
    padding: 8px 0;
}

.drawer-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
}

/* 字段类型选择器样式 */
.field-type-popover {
    padding: 0 !important;
}

.field-type-selector {
    display: flex;
    min-height: 300px;
    max-height: 400px;
}

.category-list {
    width: 140px;
    border-right: 1px solid var(--el-border-color-light);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
}

.type-list {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
}

.category-list-header,
.type-list-header {
    padding: 12px 16px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
    border-bottom: 1px solid var(--el-border-color-light);
    font-weight: 500;
}

.category-list-body,
.type-list-body {
    flex: 1;
    overflow-y: auto;
}

.category-item,
.type-item {
    padding: 10px 16px;
    cursor: pointer;
    transition: all 0.2s;
    font-size: 14px;
    color: var(--el-text-color-regular);
}

.category-item:hover,
.type-item:hover {
    background: var(--el-fill-color);
}

.category-item.active {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    font-weight: 500;
}

.type-item {
    border-bottom: 1px solid var(--el-border-color-lighter);
}

.type-item:last-child {
    border-bottom: none;
}

.type-item:hover {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
}
</style>
