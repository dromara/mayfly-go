<template>
    <div class="schema-left">
        <div class="schema-header">
            <span class="schema-title">{{ $t('milvus.fields') }}</span>
            <el-space>
                <!-- 动态字段开关 -->
                <el-tooltip :content="ctx.dynamicFieldEnabled.value ? $t('milvus.disableDynamicField') : $t('milvus.enableDynamicField')" placement="top">
                    <el-switch
                        v-model="ctx.dynamicFieldEnabled.value"
                        :disabled="ctx.isEditMode.value"
                        @change="ctx.handleDynamicFieldToggle"
                        active-text=""
                        inactive-text=""
                    >
                        <template #active-action>
                            <SvgIcon name="check" :size="14" />
                        </template>
                        <template #inactive-action>
                            <SvgIcon name="Close" :size="14" />
                        </template>
                    </el-switch>
                </el-tooltip>
                <el-popover
                    v-model:visible="ctx.fieldTypePopoverVisible.value"
                    placement="right-start"
                    :width="400"
                    trigger="click"
                    popper-class="field-type-popover"
                >
                    <template #reference>
                        <el-button type="primary" size="small">
                            {{ $t('common.add') }}{{ $t('milvus.field') }}
                            <SvgIcon name="ArrowDown" :size="14" class="el-icon--right" />
                        </el-button>
                    </template>

                    <div class="field-type-selector">
                        <!-- 左侧：分类列表 -->
                        <div class="category-list">
                            <div class="category-list-header">分类</div>
                            <div class="category-list-body">
                                <div
                                    v-for="(types, category) in fieldTypeGroups"
                                    :key="category"
                                    class="category-item"
                                    :class="{ active: ctx.selectedCategory.value === category }"
                                    @mouseenter="ctx.selectedCategory.value = category"
                                >
                                    {{ category }}
                                </div>
                            </div>
                        </div>
                        <!-- 右侧：类型列表 -->
                        <div class="type-list">
                            <div class="type-list-header">类型</div>
                            <div class="type-list-body">
                                <div
                                    v-for="type in fieldTypeGroups[ctx.selectedCategory.value]"
                                    :key="type.value"
                                    class="type-item"
                                    @click="ctx.handleSelectFieldType(type.value)"
                                >
                                    {{ type.label }}
                                </div>
                            </div>
                        </div>
                    </div>
                </el-popover>
            </el-space>
        </div>

        <div class="field-list">
            <!-- 普通字段列表 -->
            <div
                v-for="(field, index) in ctx.normalFields.value"
                :key="'normal-' + index"
                class="field-item"
                :class="{ active: ctx.selectedFieldIndex.value === index, readonly: ctx.isEditMode.value && field.readonly }"
                @click="ctx.selectField(index)"
            >
                <div class="field-icon">
                    <SvgIcon v-if="field.isPrimaryKey" name="Key" :size="14" />
                    <SvgIcon v-else-if="ctx.isVectorType(field.dataType)" name="TrendCharts" :size="14" />
                    <SvgIcon v-else-if="ctx.isGeometryType(field.dataType)" name="Location" :size="14" />
                    <SvgIcon v-else-if="ctx.isClockType(field.dataType)" name="Clock" :size="14" />
                    <SvgIcon v-else-if="ctx.isArrayType(field.dataType)" name="Memo" :size="14" />
                    <SvgIcon v-else name="Document" :size="14" />
                </div>
                <div class="field-info">
                    <div class="field-name">
                        <span class="field-name-text">{{ field.name }}</span>
                        <el-tag v-if="field.isPrimaryKey" size="small" type="primary">PK</el-tag>
                        <el-tag v-else-if="field.isPartitionKey" size="small" type="success">Partition Key</el-tag>
                        <el-tag v-if="field.readonly" size="small" type="info">{{ $t('milvus.readonly') }}</el-tag>
                    </div>
                    <div class="field-type">{{ ctx.formatFieldType(field) }}</div>
                </div>
                <div class="field-index-tag">
                    <el-tag v-if="field.indexType" size="small" type="success">{{ field.indexType }}</el-tag>
                    <el-tag v-else size="small" type="info">{{ $t('milvus.noIndex') }}</el-tag>
                </div>
                <div class="field-actions">
                    <el-space>
                        <el-button
                            text
                            size="small"
                            type="primary"
                            @click.stop="ctx.handleCopyField(index)"
                            :disabled="(field.isPrimaryKey && field.autoID) || (ctx.isEditMode.value && field.readonly)"
                        >
                            <SvgIcon name="CopyDocument" :size="14" />
                        </el-button>
                        <el-button
                            text
                            size="small"
                            type="danger"
                            @click.stop="ctx.handleDeleteField(index)"
                            :disabled="(field.isPrimaryKey && field.autoID) || (ctx.isEditMode.value && field.readonly)"
                        >
                            <SvgIcon name="Delete" :size="14" />
                        </el-button>
                    </el-space>
                </div>
            </div>

            <!-- 动态字段 -->
            <div
                v-if="ctx.dynamicFieldEnabled.value && ctx.dynamicField.value"
                class="field-item dynamic-field"
                :class="{ active: ctx.selectedFieldIndex.value === -2 }"
                @click="ctx.selectDynamicField"
            >
                <div class="field-icon">
                    <SvgIcon name="CirclePlus" :size="14" />
                </div>
                <div class="field-info">
                    <div class="field-name">
                        <span class="field-name-text">$meta</span>
                        <el-tag size="small" type="warning">{{ $t('milvus.dynamicField') }}</el-tag>
                    </div>
                    <div class="field-type">JSON</div>
                </div>
                <div class="field-index-tag">
                    <el-tag v-if="ctx.dynamicField.value.indexes && ctx.dynamicField.value.indexes.length > 0" size="small" type="success">
                        {{ ctx.dynamicField.value.indexes.length }} {{ $t('milvus.indexes') }}
                    </el-tag>
                    <el-tag v-else size="small" type="info">{{ $t('milvus.noIndex') }}</el-tag>
                </div>
                <div class="field-actions">
                    <el-space>
                        <el-button text size="small" type="danger" @click.stop="ctx.handleDeleteDynamicField" :disabled="ctx.isEditMode.value">
                            <SvgIcon name="Delete" :size="14" />
                        </el-button>
                    </el-space>
                </div>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import { fieldTypeGroups } from './constants';
import { useCollectionFormInject } from '../composables/useCollectionForm';

const ctx = useCollectionFormInject();
</script>
