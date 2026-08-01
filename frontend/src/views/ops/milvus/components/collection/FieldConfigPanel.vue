<template>
    <div class="schema-right">
        <el-tabs v-model="ctx.activeTab.value" v-if="ctx.selectedField.value || (ctx.dynamicFieldEnabled.value && ctx.dynamicField.value)">
            <el-tab-pane :label="$t('milvus.properties')" name="properties">
                <!-- 普通字段属性 -->
                <el-form :model="ctx.selectedField.value" label-width="auto" v-if="ctx.selectedField.value && !ctx.isDynamicFieldSelected.value">
                    <el-form-item :label="$t('milvus.field')" required>
                        <el-input
                            v-model="ctx.selectedField.value.name"
                            :placeholder="$t('milvus.fieldNamePlaceholder')"
                            :disabled="ctx.isEditMode.value && ctx.selectedField.value.readonly"
                        ></el-input>
                    </el-form-item>

                    <el-form-item :label="$t('milvus.dataType')">
                        <el-select v-model="ctx.selectedField.value.dataType" filterable :disabled="ctx.isEditMode.value && ctx.selectedField.value.readonly">
                            <el-option-group v-for="(types, category) in fieldTypeGroups" :key="category" :label="category">
                                <el-option v-for="type in types" :key="type.value" :label="type.label" :value="type.value"></el-option>
                            </el-option-group>
                        </el-select>
                    </el-form-item>

                    <!-- 向量类型特有属性 -->
                    <template v-if="ctx.isVectorType(ctx.selectedField.value.dataType)">
                        <el-form-item :label="$t('milvus.dim')" prop="dim">
                            <el-input-number v-model="ctx.selectedField.value.dim" :min="1" style="width: 100%" />
                        </el-form-item>
                    </template>

                    <!-- VarChar 类型特有属性 -->
                    <template v-if="ctx.isVarCharType(ctx.selectedField.value.dataType)">
                        <el-form-item :label="$t('milvus.maxLength')" prop="maxLength">
                            <el-input-number v-model="ctx.selectedField.value.maxLength" :min="1" :max="65535" style="width: 100%" />
                        </el-form-item>
                    </template>

                    <!-- 数组类型特有属性 -->
                    <template v-if="ctx.isArrayType(ctx.selectedField.value.dataType)">
                        <el-form-item :label="$t('milvus.elementType')" prop="elementType">
                            <el-select v-model="ctx.selectedField.value.elementType" style="width: 100%" filterable>
                                <el-option label="Bool" value="Bool" />
                                <el-option label="Int8" value="Int8" />
                                <el-option label="Int16" value="Int16" />
                                <el-option label="Int32" value="Int32" />
                                <el-option label="Int64" value="Int64" />
                                <el-option label="Float" value="Float" />
                                <el-option label="Double" value="Double" />
                                <el-option label="VarChar" value="VarChar" />
                                <el-option label="Struct" value="Struct" />
                            </el-select>
                        </el-form-item>
                        <el-form-item :label="$t('milvus.maxCapacity')" prop="maxCapacity">
                            <el-input-number v-model="ctx.selectedField.value.maxCapacity" :min="1" style="width: 100%" />
                        </el-form-item>
                    </template>

                    <el-divider />

                    <el-form-item :label="$t('milvus.primaryKey')">
                        <el-checkbox
                            v-model="ctx.selectedField.value.isPrimaryKey"
                            :disabled="
                                (ctx.isEditMode.value && ctx.selectedField.value.readonly) ||
                                !ctx.canBePrimaryKey(ctx.selectedField.value.dataType) ||
                                (!!ctx.getPrimaryKeyField() && ctx.getPrimaryKeyField() !== ctx.selectedField.value)
                            "
                        />
                        <el-tooltip
                            v-if="!ctx.canBePrimaryKey(ctx.selectedField.value.dataType)"
                            :content="$t('milvus.primaryKeyOnlyInt64OrVarChar')"
                            placement="top"
                        >
                            <SvgIcon name="WarningFilled" :size="14" style="margin-left: 4px; color: var(--el-color-warning)" />
                        </el-tooltip>
                    </el-form-item>

                    <el-form-item v-if="ctx.selectedField.value.isPrimaryKey && ctx.isInt64Type(ctx.selectedField.value.dataType)" :label="$t('milvus.autoID')">
                        <el-switch v-model="ctx.selectedField.value.autoID" :disabled="ctx.isEditMode.value && ctx.selectedField.value.readonly" />
                    </el-form-item>

                    <el-form-item :label="$t('milvus.nullable')">
                        <el-checkbox v-model="ctx.selectedField.value.nullable" />
                    </el-form-item>

                    <el-form-item :label="$t('milvus.partitionKey')">
                        <el-switch v-model="ctx.selectedField.value.isPartitionKey" />
                    </el-form-item>

                    <el-form-item :label="$t('milvus.clusteringKey')">
                        <el-switch v-model="ctx.selectedField.value.isClusteringKey" />
                    </el-form-item>

                    <el-form-item :label="$t('milvus.mmap')">
                        <el-switch v-model="ctx.selectedField.value.mmap" /> <SvgIcon name="QuestionFilled" :size="14" />
                    </el-form-item>

                    <el-divider />

                    <el-form-item :label="$t('milvus.defaultValue')">
                        <el-input v-model="ctx.selectedField.value.defaultValue" :placeholder="$t('milvus.defaultValuePlaceholder')"></el-input>
                    </el-form-item>

                    <el-form-item :label="$t('milvus.description')">
                        <el-input
                            v-model="ctx.selectedField.value.description"
                            type="textarea"
                            :rows="3"
                            :placeholder="$t('milvus.descriptionPlaceholder')"
                        ></el-input>
                    </el-form-item>
                </el-form>

                <!-- 动态字段属性 -->
                <el-form :model="ctx.dynamicField.value" label-width="auto" v-if="ctx.isDynamicFieldSelected.value">
                    <el-form-item :label="$t('milvus.dynamicField')">
                        <el-tag type="warning">$meta (JSON)</el-tag>
                    </el-form-item>
                    <el-alert :title="$t('milvus.dynamicFieldHint')" type="info" :closable="false" show-icon />
                </el-form>
            </el-tab-pane>

            <el-tab-pane :label="$t('milvus.index')" name="index">
                <!-- 动态字段索引配置 -->
                <div v-if="ctx.isDynamicFieldSelected.value" class="index-config">
                    <div>
                        <div>
                            <el-space>
                                <span class="index-title">{{ $t('milvus.dynamicFieldIndexes') }}</span>
                                <el-button type="primary" size="small" @click="ctx.handleAddDynamicIndex">
                                    <SvgIcon name="Plus" :size="14" />
                                    {{ $t('milvus.addIndex') }}
                                </el-button>
                            </el-space>
                        </div>

                        <div v-if="ctx.dynamicField.value.indexes.length === 0" class="index-empty">
                            <el-empty :description="$t('milvus.noIndexCreated')" :image-size="60" />
                        </div>

                        <div v-else>
                            <el-space direction="vertical" alignment="stretch">
                                <!-- 索引列表 -->
                                <div class="index-list-tabs">
                                    <div
                                        v-for="(idx, idxIdx) in ctx.dynamicField.value.indexes"
                                        :key="idxIdx"
                                        class="index-list-tab"
                                        :class="{ active: ctx.dynamicField.value.selectedIdx === idxIdx }"
                                        @click="ctx.dynamicField.value.selectedIdx = idxIdx"
                                    >
                                        <span>{{ idx.indexName || `${$t('common.index')} ${Number(idxIdx) + 1}` }}</span>
                                        <el-popconfirm
                                            :title="$t('milvus.confirmDeleteIndex')"
                                            @confirm="ctx.handleDeleteDynamicIndex(Number(idxIdx))"
                                        >
                                            <template #reference>
                                                <el-icon class="delete-icon"><close /></el-icon>
                                            </template>
                                        </el-popconfirm>
                                    </div>
                                </div>

                                <!-- 当前选中索引的配置 -->
                                <el-form
                                    v-if="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx]"
                                    :model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx]"
                                    label-width="auto"
                                >
                                    <el-form-item :label="$t('milvus.indexName')">
                                        <el-input
                                            v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].indexName"
                                            :placeholder="$t('milvus.indexNamePlaceholder')"
                                        />
                                    </el-form-item>

                                    <el-form-item :label="$t('milvus.indexType')">
                                        <el-select
                                            v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].indexType"
                                            style="width: 100%"
                                            filterable
                                        >
                                            <el-option label="AUTOINDEX" value="AUTOINDEX" />
                                            <el-option label="INVERTED" value="INVERTED" />
                                            <el-option label="Trie" value="Trie" />
                                            <el-option label="STL_SORT" value="STL_SORT" />
                                            <el-option label="BITMAP" value="BITMAP" />
                                        </el-select>
                                    </el-form-item>

                                    <el-form-item :label="$t('milvus.jsonPath')" required>
                                        <el-input
                                            v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].json_path"
                                            :placeholder="$t('milvus.jsonPathPlaceholder')"
                                        />
                                        <div class="form-tip">{{ $t('milvus.jsonPathTip') }}</div>
                                    </el-form-item>

                                    <el-form-item :label="$t('milvus.jsonCastType')" required>
                                        <el-select v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].json_cast_type" style="width: 100%">
                                            <el-option label="varchar" value="varchar" />
                                            <el-option label="double" value="double" />
                                            <el-option label="bool" value="bool" />
                                            <el-option label="array_varchar" value="array_varchar" />
                                            <el-option label="array_double" value="array_double" />
                                            <el-option label="array_bool" value="array_bool" />
                                        </el-select>
                                        <div class="form-tip">{{ $t('milvus.jsonCastTypeTip') }}</div>
                                    </el-form-item>

                                    <el-form-item :label="$t('milvus.jsonCastFunction')">
                                        <el-input
                                            v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].json_cast_function"
                                            :placeholder="$t('milvus.jsonCastFunctionPlaceholder')"
                                        />
                                        <div class="form-tip">{{ $t('milvus.jsonCastFunctionTip') }}</div>
                                    </el-form-item>

                                    <el-form-item :label="$t('milvus.fieldName')" required>
                                        <el-input
                                            v-model="ctx.dynamicField.value.indexes[ctx.dynamicField.value.selectedIdx].field_name"
                                            :placeholder="$t('milvus.fieldNamePlaceholder')"
                                        />
                                        <div class="form-tip">{{ $t('milvus.dynamicFieldNameTip') }}</div>
                                    </el-form-item>
                                </el-form>
                            </el-space>
                        </div>
                    </div>
                </div>

                <!-- 普通字段索引配置 -->
                <div v-else-if="ctx.selectedField.value" class="index-config">
                    <el-alert
                        v-if="!ctx.supportsIndex(ctx.selectedField.value.dataType)"
                        :title="$t('milvus.noIndexSupport')"
                        type="info"
                        :closable="false"
                        show-icon
                    />
                    <div v-else>
                        <!-- 未创建索引时显示创建按钮 -->
                        <el-empty v-if="!ctx.selectedField.value.indexType" :description="$t('milvus.noIndexCreated')" :image-size="80">
                            <el-button type="primary" @click="ctx.handleAddIndex(ctx.selectedField.value!)">
                                {{ $t('milvus.createIndex') }}
                            </el-button>
                        </el-empty>

                        <!-- 已创建索引时显示索引配置 -->
                        <div v-else class="index-detail">
                            <div class="index-header">
                                <span class="index-title">{{ $t('milvus.currentIndex') }}</span>
                                <el-popconfirm
                                    v-if="!ctx.isEditMode.value"
                                    :title="$t('milvus.confirmDeleteIndex')"
                                    @confirm="ctx.handleRemoveIndex(ctx.selectedField.value!)"
                                >
                                    <template #reference>
                                        <el-button type="danger" size="small" text>
                                            <SvgIcon name="Delete" :size="14" />
                                            {{ $t('milvus.deleteIndex') }}
                                        </el-button>
                                    </template>
                                </el-popconfirm>
                            </div>

                            <el-form :model="ctx.selectedField.value" label-width="auto" class="index-form">
                                <el-form-item :label="$t('milvus.indexType')">
                                    <el-select v-model="ctx.selectedField.value.indexType" style="width: 100%" filterable :disabled="ctx.isEditMode.value">
                                        <el-option-group
                                            v-for="(indexes, group) in ctx.getIndexesByGroup(ctx.selectedField.value.dataType)"
                                            :key="group"
                                            :label="group"
                                        >
                                            <el-option v-for="idx in indexes" :key="idx.value" :label="idx.label" :value="idx.value" />
                                        </el-option-group>
                                    </el-select>
                                </el-form-item>

                                <!-- 向量类型显示度量类型 -->
                                <el-form-item v-if="ctx.getMetricOptions(ctx.selectedField.value.dataType).length > 0" :label="$t('milvus.metricType')">
                                    <el-select v-model="ctx.selectedField.value.metricType" style="width: 100%" :disabled="ctx.isEditMode.value">
                                        <el-option
                                            v-for="metric in ctx.getMetricOptions(ctx.selectedField.value.dataType)"
                                            :key="metric"
                                            :label="metric"
                                            :value="metric"
                                        />
                                    </el-select>
                                </el-form-item>

                                <!-- HNSW 参数 -->
                                <template v-if="ctx.selectedField.value.indexType?.startsWith('HNSW')">
                                    <el-form-item label="M">
                                        <el-input-number
                                            v-model="ctx.selectedField.value.indexParams.M"
                                            :min="1"
                                            :max="200"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                    <el-form-item label="efConstruction">
                                        <el-input-number
                                            v-model="ctx.selectedField.value.indexParams.efConstruction"
                                            :min="1"
                                            :max="65535"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                    <el-form-item
                                        v-if="ctx.selectedField.value.indexType?.includes('PQ') || ctx.selectedField.value.indexType?.includes('PRQ')"
                                        label="m"
                                    >
                                        <el-input-number
                                            v-model="ctx.selectedField.value.indexParams.m"
                                            :min="1"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                    <el-form-item
                                        v-if="ctx.selectedField.value.indexType?.includes('PQ') || ctx.selectedField.value.indexType?.includes('PRQ')"
                                        label="nbits"
                                    >
                                        <el-input-number
                                            v-model="ctx.selectedField.value.indexParams.nbits"
                                            :min="1"
                                            :max="16"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                </template>

                                <!-- IVF 参数 -->
                                <template v-if="ctx.selectedField.value.indexType?.startsWith('IVF') || ctx.selectedField.value.indexType === 'SCANN'">
                                    <el-form-item label="nlist">
                                        <el-input
                                            v-model="ctx.selectedField.value.indexParams.nlist"
                                            :min="1"
                                            :max="65536"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                    <el-form-item v-if="ctx.selectedField.value.indexType?.includes('PQ')" label="m">
                                        <el-input v-model="ctx.selectedField.value.indexParams.m" :min="1" style="width: 100%" :disabled="ctx.isEditMode.value" />
                                    </el-form-item>
                                    <el-form-item v-if="ctx.selectedField.value.indexType?.includes('PQ')" label="nbits">
                                        <el-input
                                            v-model="ctx.selectedField.value.indexParams.nbits"
                                            :min="1"
                                            :max="16"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                </template>

                                <!-- SPARSE_INVERTED_INDEX 参数 -->
                                <template v-if="ctx.selectedField.value.indexType === 'SPARSE_INVERTED_INDEX'">
                                    <el-form-item label="drop_ratio_build">
                                        <el-input-number
                                            v-model="ctx.selectedField.value.indexParams.drop_ratio_build"
                                            :min="0"
                                            :max="1"
                                            :step="0.1"
                                            style="width: 100%"
                                            :disabled="ctx.isEditMode.value"
                                        />
                                    </el-form-item>
                                </template>
                            </el-form>
                        </div>
                    </div>
                </div>
            </el-tab-pane>
        </el-tabs>

        <el-empty v-else :description="$t('milvus.selectFieldToConfig')" />
    </div>
</template>

<script lang="ts" setup>
import SvgIcon from '@/components/svg-icon/index.vue';
import { fieldTypeGroups } from './constants';
import { useCollectionFormInject } from '../composables/useCollectionForm';

const ctx = useCollectionFormInject();
</script>
