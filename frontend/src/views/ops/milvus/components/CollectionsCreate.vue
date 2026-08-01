<template>
    <div class="create-collection-drawer">
        <el-drawer
            :title="ctx.drawerTitle.value"
            v-model="visible"
            :before-close="ctx.handleClose"
            :destroy-on-close="false"
            :close-on-click-modal="false"
            :append-to-body="false"
            size="80%"
        >
            <el-form :ref="setFormRef" :model="ctx.form.value" :rules="ctx.rules" label-width="auto">
                <!-- 基本信息 -->
                <el-divider content-position="left">{{ $t('common.basic') }}</el-divider>
                <el-row :gutter="20">
                    <el-col :span="10">
                        <el-form-item :label="$t('common.name')" prop="name">
                            <el-input v-model="ctx.form.value.name" :placeholder="$t('milvus.collectionNamePlaceholder')"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :span="6">
                        <el-form-item :label="$t('milvus.shardsNum')" prop="shardsNum">
                            <el-input-number v-model="ctx.form.value.shardsNum" :min="1" :max="64" style="width: 100%" />
                        </el-form-item>
                    </el-col>
                    <el-col :span="8">
                        <el-form-item :label="$t('milvus.consistencyLevel')" prop="consistency_level">
                            <el-select v-model="ctx.form.value.consistency_level">
                                <el-option label="Bounded" value="Bounded" />
                                <el-option label="Strong" value="Strong" />
                                <el-option label="Session" value="Session" />
                                <el-option label="Eventually" value="Eventually" />
                                <!--                      <el-option label="Customized" value="Customized" />-->
                            </el-select>
                        </el-form-item>
                    </el-col>
                </el-row>
                <el-form-item :label="$t('milvus.description')" prop="description">
                    <el-input v-model="ctx.form.value.description" type="textarea" :rows="2" :placeholder="$t('milvus.descriptionPlaceholder')"></el-input>
                </el-form-item>

                <!-- Schema 配置 -->
                <el-divider content-position="left">Schema</el-divider>

                <div class="schema-container">
                    <!-- 左侧：字段列表 -->
                    <SchemaFieldEditor />

                    <!-- 右侧：字段属性配置 -->
                    <FieldConfigPanel />
                </div>
            </el-form>

            <template #footer>
                <div class="drawer-footer">
                    <el-button @click="ctx.handleClose">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" @click="ctx.handleSubmit" :loading="ctx.loading.value">{{ $t('common.confirm') }}</el-button>
                    <el-button type="primary" @click="ctx.handlePreview" :loading="ctx.loading.value">{{ 'Json' + $t('common.preview') }}</el-button>
                </div>
            </template>
        </el-drawer>
    </div>
</template>

<script setup lang="ts">
import FieldConfigPanel from './collection/FieldConfigPanel.vue';
import SchemaFieldEditor from './collection/SchemaFieldEditor.vue';
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

// el-form ref 绑定到 composable 的 formRef
const setFormRef = (el: unknown) => {
    ctx.formRef.value = el as FormInstance | undefined;
};
</script>

<style lang="scss" scoped>
:deep(.create-collection-drawer) {
    .el-drawer__header {
        margin-bottom: 20px;
    }
    .el-drawer__body {
        padding-top: 0;
    }
}

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
