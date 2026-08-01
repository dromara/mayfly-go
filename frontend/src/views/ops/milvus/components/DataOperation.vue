<template>
    <div ref="tableContainerRef" style="height: 100%">
        <!-- 顶部查询工具栏 -->
        <el-space>
            <el-button size="small" type="primary" @click="() => handleQuery()" :loading="queryLoading" icon="search">
                {{ $t('milvus.query') }}
            </el-button>

            <el-button size="small" text @click="handleReset" icon="refresh" :disabled="queryLoading">
                {{ $t('milvus.reset') }}
            </el-button>

            <el-tooltip content="collection" placement="top" :teleported="false">
                <el-select size="small" v-model="selectedCollection" style="min-width: 200px" filterable clearable :teleported="false">
                    <el-option v-for="item in collections" :key="item" :label="item" :value="item" />
                </el-select>
            </el-tooltip>

            <el-input
                size="small"
                style="min-width: 180px"
                v-model="queryExpr"
                :placeholder="$t('milvus.queryExpr') + $t('milvus.queryExprPlaceholder')"
                clearable
                @keyup.enter="() => handleQuery()"
            >
            </el-input>

            <el-tooltip :content="$t('milvus.consistencyLevel')" placement="top" :teleported="false">
                <el-select size="small" v-model="consistencyLevel" style="width: 130px" :teleported="false">
                    <el-option label="Strong" :value="0" />
                    <el-option label="Session" :value="1" />
                    <el-option label="Bounded" :value="2" />
                    <el-option label="Eventually" :value="3" />
                    <el-option label="Customized" :value="4" />
                </el-select>
            </el-tooltip>

            <el-select size="small" v-model="selectedPartition" :placeholder="$t('milvus.partitionManagement')" style="width: 120px" clearable :teleported="false">
                <el-option size="small" :label="$t('milvus.allPartitions')" value="" />
                <el-option size="small" v-for="partition in partitions" :key="partition" :label="partition" :value="partition" />
            </el-select>

            <el-dropdown size="small" trigger="click" :teleported="false">
                <el-button size="small" text icon="grid"> {{ $t('milvus.outputFields') }} ({{ selectedFields.length }}/{{ collectionFields.length }}) </el-button>
                <template #dropdown>
                    <el-dropdown-menu class="fields-dropdown-menu">
                        <div class="fields-dropdown-header">
                            <el-checkbox
                                :model-value="selectedFields.length === collectionFields.length && collectionFields.length > 0"
                                :indeterminate="selectedFields.length > 0 && selectedFields.length < collectionFields.length"
                                @change="handleSelectAll"
                            >
                                {{ $t('milvus.outputFields') }}
                            </el-checkbox>
                        </div>
                        <div class="fields-dropdown-list">
                            <div v-for="field in collectionFields" :key="field" class="field-item" @click="toggleField(field)">
                                <el-checkbox :model-value="selectedFields.includes(field)">
                                    <span class="field-label">
                                        <SvgIcon v-if="isPrimaryKey(field)" name="Key" :size="14" class="field-icon primary" />
                                        <SvgIcon v-else-if="isVectorField(field)" name="DataAnalysis" :size="14" class="field-icon vector" />
                                        <SvgIcon v-else-if="isDynamicField(field)" name="InfoFilled" :size="14" class="field-icon dynamic" />
                                        <SvgIcon v-else name="Grid" :size="14" class="field-icon normal" />
                                        {{ field }}
                                    </span>
                                </el-checkbox>
                            </div>
                        </div>
                    </el-dropdown-menu>
                </template>
            </el-dropdown>
        </el-space>

        <!-- 操作工具栏 -->
        <el-space style="padding: 5px 0">
            <el-button text size="small" icon="upload" @click="handleImportFile">
                {{ $t('milvus.importFile') }}
            </el-button>
            <el-button text size="small" icon="plus" @click="handleInsertSample">
                {{ $t('milvus.insertSampleData') }}
            </el-button>
            <el-button text size="small" icon="delete" @click="handleClearData" :disabled="queryResults.length === 0">
                {{ $t('milvus.clearData') }}
            </el-button>
            <el-button text size="small" icon="edit" @click="handleEditData" :disabled="selectedRows.length === 0">
                {{ $t('common.edit') }}
            </el-button>
            <el-button text size="small" icon="download" :disabled="queryResults.length === 0"> {{ $t('milvus.export') }} ({{ selectedRows.length }}) </el-button>
            <el-button text size="small" icon="document-copy" :disabled="selectedRows.length === 0" @click="handleCopySelected">
                {{ $t('common.copy') }} JSON
            </el-button>
            <el-button text size="small" icon="delete" :disabled="selectedRows.length === 0" @click="handleDeleteSelected">
                {{ $t('common.delete') }}
            </el-button>
        </el-space>

        <el-table
            v-loading="queryLoading"
            :data="queryResults"
            style="width: 100%"
            @selection-change="handleSelectionChange"
            border
            stripe
            :loading="queryLoading"
            :height="tableHeight"
        >
            <el-table-column type="selection" width="55" />

            <el-table-column v-for="field in displayFields" :key="field" :label="field" :min-width="getMinWidth(field)">
                <template #header>
                    <span class="field-label">
                        <SvgIcon v-if="isPrimaryKey(field)" name="Key" :size="14" title="Primary Key" />
                        <SvgIcon v-else-if="isVectorField(field)" name="DataAnalysis" :size="14" title="Vector Field" />
                        <SvgIcon v-else-if="isDynamicField(field)" name="InfoFilled" :size="14" title="Dynamic Fields" />
                        <SvgIcon v-else name="Grid" :size="14" />
                        {{ getDisplayLabel(field) }}
                    </span>
                </template>
                <template #default="{ row }">
                    <div class="cell-content">
                        <div
                            class="cell-value"
                            :class="{ 'url-link': isUrl(row[field]), 'vector-cell': isVectorField(field) }"
                            :title="formatCellValue(row[field], field)"
                            @click="handleCellClick(row[field], field)"
                        >
                            {{ formatCellValue(row[field], field) }}
                        </div>
                        <div class="cell-actions">
                            <SvgIcon name="DocumentCopy" :size="14" class="copy-icon" @click.stop="copyToClipboard(row[field], field)" />
                        </div>
                    </div>
                </template>
            </el-table-column>
        </el-table>

        <!-- 分页控件 -->
        <div class="pagination-container">
            <div class="pagination-info">
                <span>{{ $t('milvus.paginationInfo', { total: totalRecords, current: currentPage, pages: totalPages }) }}</span>
                <el-select size="small" v-model="pageSize" @change="handlePageSizeChange" style="width: 110px; margin-left: 12px" :teleported="false">
                    <el-option size="small" :label="$t('milvus.pageSize10')" :value="10" />
                    <el-option size="small" :label="$t('milvus.pageSize20')" :value="20" />
                    <el-option size="small" :label="$t('milvus.pageSize50')" :value="50" />
                    <el-option size="small" :label="$t('milvus.pageSize100')" :value="100" />
                </el-select>
            </div>
            <div class="pagination-controls">
                <el-button text :disabled="currentPage <= 1" @click="handlePageChange(currentPage - 1)" icon="ArrowLeft" size="small">
                    {{ $t('milvus.prevPage') }}
                </el-button>
                <div class="pagination-jump">
                    <span>{{ $t('milvus.jumpTo') }}</span>
                    <el-input-number
                        v-model="jumpPage"
                        :min="1"
                        :max="totalPages || 999999"
                        size="small"
                        style="width: 80px"
                        @change="handleJumpPage"
                        @keyup.enter="handleJumpPage"
                    />
                </div>
                <el-button text :disabled="currentPage >= totalPages" @click="handlePageChange(currentPage + 1)" icon="ArrowRight" size="small">
                    {{ $t('milvus.nextPage') }}
                </el-button>
            </div>
        </div>
    </div>

    <!-- 样本数据导入对话框 -->
    <el-dialog
        v-model="importDialogVisible"
        :title="$t('milvus.importSampleDialogTitle', { name: selectedCollection })"
        width="650px"
        :close-on-click-modal="false"
        @close="handleCloseImportDialog"
    >
        <div class="import-dialog-content">
            <p class="import-description">
                {{ $t('milvus.importSampleDialogDesc') }}
            </p>

            <!-- 分区选择 -->
            <div class="import-field">
                <label class="field-label">{{ $t('milvus.partition') }}</label>
                <el-select v-model="selectedPartitionForImport" style="width: 100%" :teleported="false">
                    <el-option v-for="partition in partitions" :key="partition" :label="partition" :value="partition" />
                </el-select>
            </div>

            <!-- 样本数据大小 -->
            <div class="import-field">
                <label class="field-label">{{ $t('milvus.sampleSizeLabel') }}</label>
                <el-input-number v-model="sampleSize" :min="1" :max="10000" :placeholder="$t('milvus.sampleSizePlaceholder')" />
            </div>

            <!-- 下载选项 -->
            <div class="import-download-options">
                <label class="field-label">{{ $t('milvus.downloadOptions') }}</label>
                <el-space>
                    <el-button
                        type="info"
                        @click="handleDownloadCSV"
                        icon="Download"
                        :loading="downloadingCSV"
                        :disabled="sampleSize < 1 || sampleSize > 10000"
                    >
                        {{ selectedCollection }}.sample.{{ sampleSize }}.csv
                    </el-button>
                    <el-button
                        type="info"
                        @click="handleDownloadJSON"
                        icon="Download"
                        :loading="downloadingJSON"
                        :disabled="sampleSize < 1 || sampleSize > 10000"
                    >
                        {{ selectedCollection }}.sample.{{ sampleSize }}.json
                    </el-button>
                </el-space>
                <p class="download-hint">{{ $t('milvus.downloadHint') }}</p>
            </div>
        </div>

        <template #footer>
            <div class="dialog-footer">
                <el-button @click="handleCloseImportDialog">{{ $t('common.cancel') }}</el-button>
                <el-button type="success" @click="executeImport" :loading="loading">
                    {{ $t('milvus.import') }}
                </el-button>
            </div>
        </template>
    </el-dialog>
</template>

<script setup lang="ts">
import MonacoEditorBox from '@/components/monaco/MonacoEditorBox';
import { Msg } from '@/hooks/useI18n';
import { useMilvusStore } from '@/views/ops/milvus/resource/store';
import SvgIcon from '@/components/svg-icon/index.vue';
import { useClipboard, useResizeObserver } from '@vueuse/core';
import { ElMessageBox } from 'element-plus';
import { storeToRefs } from 'pinia';
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { milvusApi } from '../api';

// 表格动态高度（仅 body 滚动，表头固定）
const tableContainerRef = ref<HTMLElement>();
const tableHeight = ref(300);
let resizeObserver: ReturnType<typeof useResizeObserver> | undefined;

const calcTableHeight = () => {
    if (!tableContainerRef.value) return;
    const table = tableContainerRef.value?.querySelector('.el-table');
    if (!table) return;
    const containerRect = tableContainerRef.value?.getBoundingClientRect();
    const tableRect = table.getBoundingClientRect();
    const offset = tableRect.top - containerRect.top;
    const paginationEl = tableContainerRef.value?.querySelector('.pagination-container') as HTMLElement;
    const paginationH = paginationEl ? paginationEl.offsetHeight + 15 : 0;
    const newHeight = containerRect.height - offset - paginationH;
    if (newHeight > 100) tableHeight.value = newHeight;
};

onMounted(() => {
    resizeObserver = useResizeObserver(tableContainerRef, () => {
        nextTick(calcTableHeight);
    });
});

onBeforeUnmount(() => {
    resizeObserver?.stop();
});

const { t } = useI18n();
const { copy } = useClipboard();

const props = defineProps<{
    milvusId: number;
    tabKey?: string;
}>();

const milvusStore = useMilvusStore(props.tabKey || 'milvusStore');
const { collections, selectedCollection } = storeToRefs(milvusStore);

// 查询状态
const queryLoading = ref(false);
const loading = ref(false);
const hasQueried = ref(false);
const queryExpr = ref('');
const consistencyLevel = ref<number>(0);
const selectedPartition = ref<string>('');
const selectedFields = ref<string[]>([]);

// 分页状态
const currentPage = ref(1);
const pageSize = ref(20);
const totalRecords = ref(0);

// 集合信息
const collectionFields = ref<string[]>([]);
const partitions = ref<string[]>([]);
const primaryKey = ref<string>('');

// 数据
const queryResults = ref<Record<string, unknown>[]>([]);
const selectedRows = ref<Record<string, unknown>[]>([]);

// 显示字段(动态计算，按照 collectionFields 的原始顺序)
const displayFields = computed(() => {
    const fields = selectedFields.value.length > 0 ? selectedFields.value : collectionFields.value;
    // 按照 collectionFields 的顺序排序
    return collectionFields.value.filter((field) => fields.includes(field));
});

// 加载集合详情
const loadCollectionDetail = async (collectionName: string) => {
    if (!collectionName) return;

    const res = await milvusApi.describeCollection(props.milvusId, collectionName);

    // 提取字段信息
    if (res.Schema && res.Schema.Fields) {
        const fields: string[] = [];
        let dynamicFieldCount = 0;

        res.Schema.Fields.forEach((field) => {
            if (field.IsDynamic) {
                dynamicFieldCount++;
                if (dynamicFieldCount === 1) {
                    fields.push('dynamicFields');
                }
            } else {
                fields.push(field.Name);
                if (field.PrimaryKey) {
                    primaryKey.value = field.Name;
                }
            }
        });

        collectionFields.value = fields;
        // 默认选中所有字段
        selectedFields.value = [...fields];
    }

    const pts = await milvusApi.listPartitions(props.milvusId, collectionName);

    // 加载分区信息
    if (pts) {
        partitions.value = pts.map((a) => a.name);
    }
};

// 切换单个字段选中状态
const toggleField = (field: string) => {
    const index = selectedFields.value.indexOf(field);
    if (index > -1) {
        selectedFields.value.splice(index, 1);
    } else {
        selectedFields.value.push(field);
    }
};

// 全选/取消全选
const handleSelectAll = (val: boolean | string | number) => {
    if (val) {
        // 全选
        selectedFields.value = [...collectionFields.value];
    } else {
        // 取消全选
        selectedFields.value = [];
    }
};

// 查询数据
const handleQuery = async (page?: number) => {
    if (!selectedCollection.value) {
        Msg.warning('milvus.selectCollectionHint');
        return;
    }

    queryLoading.value = true;
    hasQueried.value = true;
    selectedRows.value = [];

    // 如果没有传 page，使用当前页码
    const targetPage = page !== undefined ? page : currentPage.value;

    try {
        const params: Record<string, unknown> = {
            consistency_level: consistencyLevel.value,
            partitionNames: selectedPartition.value ? [selectedPartition.value] : [],
            page: targetPage,
            pageSize: pageSize.value,
        };

        // expr 字段可选，空字符串表示查询所有数据
        if (queryExpr.value && queryExpr.value.trim()) {
            params.expr = queryExpr.value.trim();
        }

        // outputFields 必填，默认使用所有字段
        params.outputFields = selectedFields.value.length > 0 ? selectedFields.value : collectionFields.value;

        const res = await milvusApi.query(props.milvusId, selectedCollection.value, params);
        // 后端返回的是 { count, fields, data, page, pageSize, total } 结构
        if (res && res.data) {
            queryResults.value = res.data || [];
            currentPage.value = res.page || targetPage;
            // 使用后端返回的真实 total
            totalRecords.value = res.total || 0;
            if (res.count === 0) {
                Msg.info('milvus.noData');
            }
        } else {
            queryResults.value = [];
            totalRecords.value = 0;
            Msg.info('milvus.noData');
        }
    } catch (error: unknown) {
        queryResults.value = [];
        totalRecords.value = 0;
    } finally {
        queryLoading.value = false;
    }
};

// 计算总页数
const totalPages = computed(() => {
    if (totalRecords.value === 0) return 0;
    return Math.ceil(totalRecords.value / pageSize.value);
});

// 页码改变
const handlePageChange = (val: number) => {
    if (val < 1 || val > totalPages.value) return;
    handleQuery(val);
};

// 每页条数改变
const handlePageSizeChange = (val: number) => {
    pageSize.value = val;
    currentPage.value = 1;
    handleQuery(1);
};

// 页码跳转
const jumpPage = ref<number>(1);
const handleJumpPage = () => {
    if (jumpPage.value && jumpPage.value > 0 && jumpPage.value <= totalPages.value) {
        handleQuery(jumpPage.value);
        jumpPage.value = currentPage.value;
    }
};

// 重置查询状态
const resetQueryState = () => {
    queryExpr.value = '';
    consistencyLevel.value = 0;
    selectedPartition.value = '';
    selectedFields.value = [];
    // 重置分页
    currentPage.value = 1;
    pageSize.value = 20;
    totalRecords.value = 0;
    jumpPage.value = 1;
};

// 重置查询
const handleReset = () => {
    resetQueryState();
    queryResults.value = [];
    hasQueried.value = false;
    selectedRows.value = [];
    totalRecords.value = 0;
};

// 选择变化
const handleSelectionChange = (selection: Record<string, unknown>[]) => {
    selectedRows.value = selection;
};

// 复制单个数据
const copyToClipboard = async (value: unknown, field: string) => {
    await copy(JSON.stringify(value));
    Msg.success('common.copySuccess');
};

// 复制选中行
const handleCopySelected = async () => {
    await copy(JSON.stringify(selectedRows.value, null, 2));
    Msg.success('common.copySuccess');
};

// 删除选中行
const handleDeleteSelected = async () => {
    if (!selectedCollection.value || selectedRows.value.length === 0) return;

    await ElMessageBox.confirm(t('common.confirmDelete', { count: selectedRows.value.length }), t('common.warning'), {
        type: 'warning',
    });

    const ids = selectedRows.value.map((row) => row[primaryKey.value]).filter((id): id is string => id != null && id !== undefined);
    if (ids.length === 0) {
        Msg.warning('milvus.noPrimaryKey');
        return;
    }

    const expr = `${primaryKey.value} in [${ids.join(',')}]`;
    await milvusApi.deleteData(props.milvusId, selectedCollection.value, { expr });

    Msg.success('common.deleteSuccess');
    selectedRows.value = [];
    setTimeout(handleQuery, 500);
};

// 格式化单元格值
const formatCellValue = (value: unknown, field: string): string => {
    if (value === null || value === undefined) {
        return '';
    }

    // 动态字段
    if (field === 'dynamicFields') {
        if (typeof value === 'object') {
            return JSON.stringify(value);
        }
        return String(value);
    }

    // 向量字段
    if (Array.isArray(value)) {
        const str = JSON.stringify(value);
        return str.length > 100 ? str.substring(0, 100) + '...' : str;
    }
    // 其余对象数据
    if (typeof value === 'object') {
        return JSON.stringify(value);
    }
    return String(value);
};

// 判断是否为主键
const isPrimaryKey = (field: string) => {
    return field === primaryKey.value;
};

// 判断是否为动态字段
const isDynamicField = (field: string) => {
    return field === 'dynamicFields';
};

// 判断是否为向量字段
const isVectorField = (field: string) => {
    return field === 'vector' || field.toLowerCase().includes('vector');
};

// 获取最小列宽
const getMinWidth = (field: string) => {
    if (isVectorField(field)) return 300;
    if (isDynamicField(field)) return 250;
    if (isPrimaryKey(field)) return 180;
    return 150;
};

// 获取显示标签
const getDisplayLabel = (field: string) => {
    // 主键字段显示 # 前缀
    if (isPrimaryKey(field)) {
        return `# ${field}`;
    }
    // 向量字段显示 ~ 前缀
    if (isVectorField(field)) {
        return `${field}`;
    }
    // 动态字段显示 "" 前缀
    if (isDynamicField(field)) {
        return `"${field}`;
    }
    return field;
};

// 判断是否为 URL
const isUrl = (value: unknown): value is string => {
    return typeof value === 'string' && (value.startsWith('http://') || value.startsWith('https://'));
};

// 单元格点击事件
const handleCellClick = (value: unknown, field: string) => {
    if (isUrl(value)) {
        window.open(value, '_blank');
    }
};

// 样本数据导入配置对话框
const importDialogVisible = ref(false);
const sampleSize = ref<number>(10);
const selectedPartitionForImport = ref('');
const generatedSampleData = ref<Record<string, unknown>[]>([]);
const downloadingCSV = ref(false);
const downloadingJSON = ref(false);

// 插入样本数据 - 打开配置对话框
const handleInsertSample = async () => {
    if (!selectedCollection.value) {
        Msg.warning('milvus.selectCollectionHint');
        return;
    }

    // 重置配置
    sampleSize.value = 10;
    selectedPartitionForImport.value = '';

    // 打开配置对话框
    importDialogVisible.value = true;
};

// 生成样本数据（调用后端 Mock API）
const generateSampleData = async () => {
    if (sampleSize.value < 1 || sampleSize.value > 10000) {
        Msg.error('样本数据大小必须在 1-10000 之间');
        return;
    }

    try {
        loading.value = true;

        // 调用后端 Mock API 生成样本数据
        const response = await milvusApi.generateMockData(props.milvusId, selectedCollection.value, {
            count: sampleSize.value,
            partitionName: selectedPartitionForImport.value,
        });
        generatedSampleData.value = response.data;

        Msg.success(`已生成 ${sampleSize.value} 条样本数据`);
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || '生成样本数据失败' : '生成样本数据失败');
    } finally {
        loading.value = false;
    }
};

// 处理 CSV 下载（预加载：调用接口生成数据并直接下载）
const handleDownloadCSV = async () => {
    if (sampleSize.value < 1 || sampleSize.value > 10000) {
        Msg.error('样本数据大小必须在 1-10000 之间');
        return;
    }

    try {
        downloadingCSV.value = true;

        // 调用后端 Mock API 生成样本数据
        const response = await milvusApi.generateMockData(props.milvusId, selectedCollection.value, {
            count: sampleSize.value,
            partitionName: selectedPartitionForImport.value,
        });
        const data = response.data;

        // 直接下载 CSV
        const headers = Object.keys(data[0] || {});
        const csvRows = [
            headers.join(','),
            ...data.map((row: Record<string, unknown>) =>
                headers
                    .map((header) => {
                        const value = row[header];
                        const stringValue = typeof value === 'object' ? JSON.stringify(value) : String(value);
                        // 如果包含逗号或引号，需要用引号包裹
                        return stringValue.includes(',') || stringValue.includes('"') ? `"${stringValue.replace(/"/g, '""')}"` : stringValue;
                    })
                    .join(',')
            ),
        ];

        const csvContent = csvRows.join('\n');
        const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `${selectedCollection.value}.sample.${sampleSize.value}.csv`;
        link.click();
        URL.revokeObjectURL(link.href);

        Msg.success(`CSV 文件下载成功`);
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || '下载 CSV 失败' : '下载 CSV 失败');
    } finally {
        downloadingCSV.value = false;
    }
};

// 处理 JSON 下载（预加载：调用接口生成数据并直接下载）
const handleDownloadJSON = async () => {
    if (sampleSize.value < 1 || sampleSize.value > 10000) {
        Msg.error('样本数据大小必须在 1-10000 之间');
        return;
    }

    try {
        downloadingJSON.value = true;

        // 调用后端 Mock API 生成样本数据
        const response = await milvusApi.generateMockData(props.milvusId, selectedCollection.value, {
            count: sampleSize.value,
            partitionName: selectedPartitionForImport.value,
        });
        const data = response.data;

        // 直接下载 JSON
        const jsonContent = JSON.stringify(data, null, 2);
        const blob = new Blob([jsonContent], { type: 'application/json;charset=utf-8;' });
        const link = document.createElement('a');
        link.href = URL.createObjectURL(blob);
        link.download = `${selectedCollection.value}.sample.${sampleSize.value}.json`;
        link.click();
        URL.revokeObjectURL(link.href);

        Msg.success(`JSON 文件下载成功`);
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || '下载 JSON 失败' : '下载 JSON 失败');
    } finally {
        downloadingJSON.value = false;
    }
};

// 下载 CSV 文件（已生成数据时使用）
const downloadCSV = () => {
    if (generatedSampleData.value.length === 0) {
        Msg.warning('请先生成样本数据');
        return;
    }

    const headers = Object.keys(generatedSampleData.value[0]);
    const csvRows = [
        headers.join(','),
        ...generatedSampleData.value.map((row) =>
            headers
                .map((header) => {
                    const value = row[header];
                    const stringValue = typeof value === 'object' ? JSON.stringify(value) : String(value);
                    // 如果包含逗号或引号，需要用引号包裹
                    return stringValue.includes(',') || stringValue.includes('"') ? `"${stringValue.replace(/"/g, '""')}"` : stringValue;
                })
                .join(',')
        ),
    ];

    const csvContent = csvRows.join('\n');
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `${selectedCollection.value}.sample.${sampleSize.value}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
};

// 下载 JSON 文件（已生成数据时使用）
const downloadJSON = () => {
    if (generatedSampleData.value.length === 0) {
        Msg.warning('请先生成样本数据');
        return;
    }

    const jsonContent = JSON.stringify(generatedSampleData.value, null, 2);
    const blob = new Blob([jsonContent], { type: 'application/json;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `${selectedCollection.value}.sample.${sampleSize.value}.json`;
    link.click();
    URL.revokeObjectURL(link.href);
};

// 执行导入（后端直接 mock 数据并入库）
const executeImport = async () => {
    if (sampleSize.value < 1 || sampleSize.value > 10000) {
        Msg.error('样本数据大小必须在 1-10000 之间');
        return;
    }

    try {
        loading.value = true;

        const response = await milvusApi.insertSampleData(props.milvusId, selectedCollection.value, {
            count: sampleSize.value,
            partitionName: selectedPartitionForImport.value,
        });

        Msg.success(`成功导入 ${response.insertCount} 条数据`);
        importDialogVisible.value = false;

        // 刷新数据
        await handleQuery();
    } catch (error: unknown) {
        Msg.error(error instanceof Error ? error.message || '导入失败' : '导入失败');
    } finally {
        loading.value = false;
    }
};

// 关闭对话框
const handleCloseImportDialog = () => {
    importDialogVisible.value = false;
};

// 导入文件 - 直接上传到后端解析
const handleImportFile = () => {
    const input = document.createElement('input');
    input.type = 'file';
    input.accept = '.json,.csv';

    input.onchange = async (e: Event) => {
        const target = e.target as HTMLInputElement;
        const file = target.files?.[0];
        if (!file) return;

        // 检查文件类型
        if (!file.name.endsWith('.json') && !file.name.endsWith('.csv')) {
            Msg.error('仅支持 CSV 和 JSON 格式的文件');
            return;
        }

        try {
            loading.value = true;

            // 创建 FormData 上传文件
            const formData = new FormData();
            formData.append('file', file);
            formData.append('partitionName', selectedPartition.value);

            // 调用后端接口导入文件
            const response = await milvusApi.importFile(props.milvusId, selectedCollection.value, formData);

            Msg.success(`成功导入 ${response.insertCount} 条数据`);
            setTimeout(handleQuery, 500);
        } finally {
            loading.value = false;
        }
    };

    input.click();
};

// 清空数据
const handleClearData = async () => {
    if (!selectedCollection.value) return;

    await ElMessageBox.confirm(t('milvus.confirmClearData'), t('common.warning'), {
        type: 'warning',
    });

    if (!primaryKey.value) {
        Msg.warning('milvus.noPrimaryKey');
        return;
    }

    // 清空所有数据
    const expr = `${primaryKey.value} > 0 || ${primaryKey.value} < 0`;
    await milvusApi.deleteData(props.milvusId, selectedCollection.value, { expr });

    Msg.success('common.deleteSuccess');
    await handleQuery();
};

const handleEditData = () => {
    MonacoEditorBox({
        content: JSON.stringify(selectedRows.value, null, 2),
        title: t('milvus.viewData'),
        language: 'json',
        canChangeLang: false,
        options: { wordWrap: 'on', tabSize: 2, readOnly: true }, // 自动换行
        useDrawer: true,
        confirmFn: async (_value: string) => {
            Msg.info('common.developing');
        },
    });
};

// 监听 milvusId 或授权凭证变化
watch(
    [() => props.milvusId, () => milvusStore.authCertName],
    async () => {
        // 重置状态
        collectionFields.value = [];
        partitions.value = [];
        primaryKey.value = '';
        queryResults.value = [];
        hasQueried.value = false;
        selectedRows.value = [];
        resetQueryState();
    }
);

onMounted(async () => {
    // 加载 collections 列表
    if (props.milvusId > 0) {
        const res = await milvusApi.listCollections(props.milvusId);
        milvusStore.setCollections(res.map((a) => a.name));
    }
});

// 监听 selectedCollection 变化，加载字段详情
watch(
    () => milvusStore.selectedCollection,
    async (newVal, oldVal) => {
        if (!newVal || newVal !== oldVal) {
            // 清空选择时重置所有状态
            collectionFields.value = [];
            partitions.value = [];
            primaryKey.value = '';
            queryResults.value = [];
            hasQueried.value = false;
            selectedRows.value = [];
            selectedFields.value = [];
            resetQueryState();
        }

        if (newVal !== oldVal) {
            await loadCollectionDetail(newVal);
            // 自动查询
            await handleQuery();
        }

        return;
    },
    { immediate: true } // 组件创建时立即执行一次
);

// 同步 jumpPage 与 currentPage
watch(
    () => currentPage.value,
    (newVal) => {
        jumpPage.value = newVal;
    }
);
</script>

<style scoped lang="scss">
.empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 400px;
}

.no-collection-hint {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 400px;
}

.cell-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding-right: 4px;
}

.cell-value {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    line-height: 1.5;

    &.url-link {
        color: #409eff;
        cursor: pointer;
        text-decoration: underline;

        &:hover {
            color: #66b1ff;
        }
    }
    &.vector-cell {
        font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
    }
}

.cell-actions {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
    opacity: 0;
    transition: opacity 0.2s;

    .cell-content:hover & {
        opacity: 1;
    }
}

.copy-icon {
    cursor: pointer;
    color: #888;
    font-size: 14px;
    padding: 2px;
    transition: color 0.2s;

    &:hover {
        color: #00ff88;
    }
}

.field-label {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    font-weight: 500;
}

// 字段选择 Dropdown 样式
.fields-dropdown-menu {
    min-width: 240px;
    max-width: 400px;
    padding: 0;
}

.fields-dropdown-header {
    padding: 5px 16px;
    position: sticky;
    top: 0;
    z-index: 10;
}

.fields-dropdown-list {
    max-height: 400px;
    overflow-y: auto;
}

.field-item {
    padding: 2px 16px;
    cursor: pointer;
    transition: background 0.2s;
}

.field-label {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
}

.field-icon {
    font-size: 14px;
    flex-shrink: 0;
}

// 滚动条样式
.fields-dropdown-list {
    &::-webkit-scrollbar {
        width: 6px;
    }
    &::-webkit-scrollbar-thumb {
        border-radius: 3px;
    }
}

// 分页控件样式
.pagination-container {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 15px;
}

.pagination-info {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 13px;
}

.pagination-controls {
    display: flex;
    align-items: center;
    gap: 12px;
}

.pagination-jump {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
}

// 样本数据导入对话框样式
.import-dialog-content {
    .import-description {
        color: #909399;
        font-size: 14px;
        line-height: 1.6;
        margin-bottom: 20px;
    }

    .import-field {
        margin-bottom: 20px;

        .field-label {
            display: block;
            font-size: 14px;
            color: #606266;
            margin-bottom: 8px;
            font-weight: 500;
        }
    }

    .import-download-buttons {
        display: flex;
        gap: 12px;
        margin-top: 24px;
        padding-top: 20px;
        border-top: 1px solid #e4e7ed;

        .el-button {
            flex: 1;
        }
    }

    .import-download-options {
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid #e4e7ed;

        .field-label {
            display: block;
            font-size: 14px;
            color: #606266;
            margin-bottom: 12px;
            font-weight: 500;
        }

        .download-hint {
            margin-top: 8px;
            font-size: 12px;
            color: #909399;
        }
    }
}

.dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
}
</style>
