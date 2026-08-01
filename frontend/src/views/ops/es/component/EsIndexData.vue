<template>
    <div class="es-index-data-box">
        <el-descriptions class="w-full! shrink-0" :column="20" size="small" border>
            <el-descriptions-item label-align="center">
                <template #label>
                    <SvgIcon name="Menu" />
                </template>
                <el-select v-model="currentIdxName" filterable :teleported="false" size="small" style="width: 200px" @change="onIndexChange">
                    <el-option v-for="idx in indices" :key="idx.index" :value="idx.index" :label="idx.index" />
                </el-select>
            </el-descriptions-item>
            <el-descriptions-item label-align="center">
                <template #label>
                    <SvgIcon name="PieChart" />
                </template>
                {{ currentIdxInfo?.['store.size'] }}
            </el-descriptions-item>
            <el-descriptions-item label-align="center">
                <template #label>
                    <el-space><SvgIcon name="refresh" @click="onRefreshStats" /> {{ t('es.docs') }}</el-space>
                </template>
                {{ currentIdxInfo?.['docs.count'] }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('es.health')" label-align="center">
                <el-tag size="small" :type="getHealthTagType(currentIdxInfo?.health)">{{ currentIdxInfo?.health }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item :label="t('es.status')" label-align="center">
                <el-tag size="small" :type="currentIdxInfo?.status === 'open' ? 'success' : 'danger'">{{ currentIdxInfo?.status }}</el-tag>
            </el-descriptions-item>
        </el-descriptions>

        <EsIndexDataToolbar
            :state="state"
            :select-keys-len="state.selectKeys.length"
            :perms="perms"
            @refresh-data="onRefreshData"
            @basic-search="onBasicSearch"
            @add-doc="onAddDoc"
            @delete-docs="onDeleteDocs"
            @edit-select-doc="onEditSelectDoc"
            @first-page="onFirstPage"
            @prev-page="onPrevPage"
            @next-page="onNextPage"
            @change-page-size="onChangePageSize"
            @switch-track-total="onSwitchTrackTotal"
            @check-all-columns="onCheckAllColumns"
            @filter-columns="onFilterColumns"
            @check-column-filter="onCheckColumnFilter"
            @open-export-dialog="onOpenExportDialog"
        />

        <div class="es-table-data flex-1 min-h-0">
            <el-auto-resizer>
                <template #default="{ height, width }">
                    <el-table-v2
                        ref="tableRef"
                        :row-height="state.rowHeight"
                        :columns="state.columns"
                        :data="state.datas"
                        :width="width"
                        :height="height"
                        fixed
                        :header-height="22"
                        class="es-table"
                        :row-class="({ rowIndex }: { rowIndex: number }) => (state.datas[rowIndex]?._selected ? 'es-row-selected' : '')"
                        :row-event-handlers="rowEventHandlers"
                    >
                        <template #header="{ columns }">
                            <div
                                v-for="(column, i) in columns"
                                :key="i"
                                :style="{
                                    width: `${column.width}px`,
                                    textAlign: 'center',
                                    borderRight: 'var(--el-table-border)',
                                }"
                            >
                                <el-checkbox
                                    :style="{ height: '100%' }"
                                    v-if="column.key === '_selected'"
                                    v-model="state.selectAll"
                                    @change="onSelectAll"
                                    :indeterminate="state.selectKeys.length > 0 && !state.selectAll"
                                />
                                <b v-else> {{ column.title }} </b>
                            </div>
                        </template>

                        <template #cell="{ rowData, column, rowIndex, columnIndex }">
                            <div v-if="column.key === '_table_index'" class="table-data-cell">
                                <span class="el-text el-text--small is-truncated">
                                    {{ rowIndex + 1 + state.currentFrom }}
                                </span>
                            </div>
                            <div v-if="column.key === '_selected'" class="table-data-cell">
                                <span class="el-text el-text--small is-truncated">
                                    <el-checkbox v-model="rowData._selected" @change="onSelectRow(rowData)" />
                                </span>
                            </div>
                            <div v-else @contextmenu="dataContextmenuClick($event, rowIndex, column, rowData)" class="table-data-cell">
                                <span v-if="rowData[column.dataKey] === null" style="color: var(--el-color-info-light-5)"> NULL </span>
                                <span v-else :title="rowData[column.dataKey]" class="el-text el-text--small is-truncated">
                                    {{ rowData[column.dataKey] }}
                                </span>
                            </div>
                        </template>

                        <template v-if="state.loading" #overlay>
                            <div class="el-loading-mask flex flex-col items-center justify-center">
                                <div>
                                    <SvgIcon class="is-loading" name="loading" color="var(--el-color-primary)" :size="28" />
                                    <el-text class="ml-1" tag="b">{{ t('db.execTime') }} - {{ state.execTime?.toFixed(1) || 0 }}s</el-text>
                                </div>
                                <div v-if="state.loading && state.abortSearch" class="mt-2!">
                                    <el-button @click="state.abortSearch" type="info" size="small" plain>{{ t('common.cancel') }}</el-button>
                                </div>
                            </div>
                        </template>
                    </el-table-v2>
                </template>
            </el-auto-resizer>
        </div>

        <es-search :instId="instId" :idxName="currentIdxName" :fields="state.fields" v-model:visible="state.searchDialogVisible" @search="onEsSearch" />

        <Contextmenu :dropdown="contextmenu.dropdown" :items="contextmenu.items" ref="contextmenuRef" />

        <EsEditRow v-model="docEditDialog" v-model:visible="docEditDialog.visible" @success="onEditRowSuccess" />

        <!-- Export Dialog -->
        <el-dialog v-model="exportDialog.visible" :title="t('es.export.title')" width="480px" :teleported="false">
            <el-space direction="vertical" fill style="width: 100%">
                <el-alert
                    v-if="state.selectKeys.length > 0"
                    :title="t('es.export.selectedCount', { count: state.selectKeys.length })"
                    type="info"
                    :closable="false"
                    show-icon
                />
                <el-radio-group v-model="exportDialog.scope">
                    <el-radio value="selected" :disabled="state.selectKeys.length === 0">{{ t('es.export.exportSelected') }}</el-radio>
                    <el-radio value="query" :disabled="!hasCustomQuery">{{ t('es.export.exportQuery') }}</el-radio>
                    <el-radio value="all">{{ t('es.export.exportAll') }}</el-radio>
                </el-radio-group>
                <el-alert
                    v-if="(exportDialog.scope === 'all' && exportDialog.queryTotal > 10000) || (exportDialog.scope === 'query' && exportDialog.queryTotal > 10000) || (exportDialog.scope === 'selected' && state.selectKeys.length > 10000)"
                    :title="t('es.export.largeExportTip', { total: exportDialog.scope === 'selected' ? state.selectKeys.length : (exportDialog.queryTotal >= 0 ? exportDialog.queryTotal : '...') })"
                    type="warning"
                    :closable="false"
                    show-icon
                />
                <div>
                    <div class="el-text mb-1">{{ t('es.export.exportType') }}</div>
                    <el-radio-group v-model="exportDialog.type">
                        <el-radio-button value="csv">{{ t('es.export.csv') }}</el-radio-button>
                        <el-radio-button value="excel">{{ t('es.export.excel') }}</el-radio-button>
                        <el-radio-button value="json">{{ t('es.export.json') }}</el-radio-button>
                    </el-radio-group>
                </div>
                <div>
                    <div class="el-text mb-1">{{ t('es.export.exportFields') }}</div>
                    
                    <el-checkbox v-model="exportDialog.allFields" @change="onExportFieldsToggle" class="mb-1">
                        {{ t('es.export.selectAllFields') }}
                    </el-checkbox>
                    <div class="export-fields-group">
                        <el-checkbox-group v-model="exportDialog.fields" @change="onExportFieldsChange">
                            <el-checkbox v-for="field in state.fields" :key="field" :value="field" :label="field" />
                        </el-checkbox-group>
                    </div>
                </div>
            </el-space>
            <template #footer>
                <div v-if="exportDialog.progress" class="mb-2">
                    <div class="flex items-center justify-between mb-1">
                        <span class="el-text el-text--small">
                            {{ t(`es.export.phase.${exportDialog.progress.phase}`) }}
                        </span>
                        <span class="el-text el-text--small" v-if="exportDialog.progress.total > 0">
                            {{ exportDialog.progress.processed }} / {{ exportDialog.progress.total }}
                            ({{ Math.round((exportDialog.progress.processed / exportDialog.progress.total) * 100) }}%)
                        </span>
                    </div>
                    <el-progress
                        :percentage="exportDialog.progress.total > 0 ? Math.round((exportDialog.progress.processed / exportDialog.progress.total) * 100) : 0"
                        :status="exportDialog.progress.error ? 'exception' : exportDialog.progress.done ? 'success' : undefined"
                        :stroke-width="6"
                    />
                </div>
                <el-button @click="exportDialog.visible = false">{{ t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="exportDialog.loading" :disabled="exportDialog.fields.length === 0" @click="onConfirmExport">
                    {{ t('es.export.confirm') }}
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="tsx" setup>
import { copyToClipboard } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { useI18nDeleteConfirm } from '@/hooks/useI18n';
import type { EsColumn, EsDoc } from '@/views/ops/es/types';
import { defineAsyncComponent, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useEsDocCrud } from './composables/useEsDocCrud';
import { useEsSearch } from './composables/useEsSearch';
import EsIndexDataToolbar from './EsIndexDataToolbar.vue';

const EsSearch = defineAsyncComponent(() => import('./EsSearch.vue'));
const EsEditRow = defineAsyncComponent(() => import('../component/EsEditRow.vue'));

const props = defineProps<{
    instId: number;
}>();

const i18n = useI18n();
const { t } = i18n;

const perms = {
    saveData: 'es:data:save',
    delData: 'es:data:del',
};

// ---- Composables ----

const {
    state,
    currentIdxName,
    currentIdxInfo,
    indices,
    onIndexChange,
    fetchIndexData,
    onChangePageSize,
    onFirstPage,
    onNextPage,
    onPrevPage,
    onSwitchTrackTotal,
    refreshIndex,
    onRefreshData,
    onRefreshStats,
    onBasicSearch,
    onEsSearch,
    onCheckColumnFilter,
    onCheckAllColumns,
    onFilterColumns,
    onSelectAll,
    onSelectRow,
    rowEventHandlers,
    selectIndex,
} = useEsSearch({ instId: props.instId, i18n });

const { docEditDialog, exportDialog, hasCustomQuery, onAddDoc, onEditDoc, onEditSelectDoc, onEditRowSuccess, onDeleteDocs, doDeleteDoc, onOpenExportDialog, onExportFieldsToggle, onExportFieldsChange, onConfirmExport } = useEsDocCrud({
    instId: props.instId,
    get currentIdxName() {
        return currentIdxName.value;
    },
    state,
    refreshIndex,
    fetchIndexData,
});

const contextmenu = reactive({ items: [] as ContextmenuItem[], dropdown: { x: 0, y: 0 } });

const contextmenuRef = ref();
const tableRef = ref();

// ---- Context menu ----

const copyCell = new ContextmenuItem('copyCell', 'common.copyCell').withIcon('CopyDocument').withOnClick(async (data: Record<string, unknown>) => {
    const rowData = data.rowData as EsDoc;
    const column = data.column as EsColumn;
    await copyToClipboard(String(rowData[column.dataKey || ''] ?? ''));
});

const copyLineJson = new ContextmenuItem('copyLineJson', 'es.contextmenu.index.copyLineJson').withIcon('CopyDocument').withOnClick(async (data: Record<string, unknown>) => {
    const rowData = data.rowData as EsDoc;
    await copyToClipboard(String(rowData.src ?? ''));
});

const copySelectLineJson = new ContextmenuItem('copySelectLineJson', 'es.contextmenu.index.copySelectLineJson')
    .withIcon('CopyDocument')
    .withHideFunc(() => state.selectKeys?.length == 0)
    .withOnClick(async () => {
        await copyToClipboard(
            JSON.stringify(
                state.selectKeys.map((a: EsDoc) => JSON.parse(String(a.src))),
                null,
                2
            )
        );
    });

const editLineJson = new ContextmenuItem('editLineJson', 'common.edit').withIcon('EditPen').withOnClick(async (data: Record<string, unknown>) => {
    const rowData = data.rowData as EsDoc;
    return onEditDoc(String(rowData.src ?? ''));
});

const deleteLine = new ContextmenuItem('deleteLine', 'common.delete').withIcon('Delete').withOnClick(async (data: Record<string, unknown>) => {
    const rowData = data.rowData as EsDoc;
    let ids = [rowData._id];
    await useI18nDeleteConfirm(ids.join(', '));
    await doDeleteDoc(ids);
});

const deleteSelectLine = new ContextmenuItem('deleteLine', 'es.contextmenu.index.DeleteSelectLine')
    .withIcon('Delete')
    .withHideFunc(() => state.selectKeys.length == 0)
    .withOnClick(async () => {
        let ids = state.selectKeys.map((a: EsDoc) => a._id);
        await useI18nDeleteConfirm(ids.join(', '));
        await doDeleteDoc(ids);
    });

const dataContextmenuClick = (event: MouseEvent, rowIndex: number, column: EsColumn, data: EsDoc) => {
    event.preventDefault();
    const { clientX, clientY } = event;
    contextmenu.dropdown.x = clientX;
    contextmenu.dropdown.y = clientY;
    contextmenu.items = [copyCell, copyLineJson, copySelectLineJson, editLineJson, deleteLine, deleteSelectLine];
    contextmenuRef.value?.openContextmenu({ column, rowData: data });
};

// ---- Helpers ----

const getHealthTagType = (health?: string) => {
    return health == 'green' ? 'success' : health == 'yellow' ? 'warning' : 'danger';
};

defineExpose({
    onBasicSearch,
    onRefresh: fetchIndexData,
    fetchIndexData,
    selectIndex,
});
</script>

<style lang="scss">
.es-index-data-box {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.es-op-header {
    padding: 4px 0;
}

.es-table-data {
    overflow: hidden;

    .es-table {
        border-left: var(--el-table-border);
        border-top: var(--el-table-border);
    }

    .es-table-column {
        padding: 0 2px;
        font-size: 12px;
        border-right: var(--el-table-border);
    }

    .header-column-title {
        height: 30px;
        display: flex;
        justify-content: center;
    }

    .table-data-cell {
        width: 100%;
        height: 100%;
        line-height: 30px;
        cursor: pointer;
    }

    .es-row-selected {
        background-color: var(--el-table-current-row-bg-color);
    }

    .es-row-hover:hover {
        background-color: var(--el-fill-color-light);
    }

    .data-selection {
        background-color: var(--el-table-current-row-bg-color);
    }

    .update_field_active {
        background-color: var(--el-color-success-light-3);
    }

    .el-table-v2__overlay {
        z-index: 1;
    }
}

.export-fields-group {
    max-height: 180px;
    overflow-y: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 4px;
    padding: 8px;

    .el-checkbox {
        display: block;
        margin-right: 0;
        margin-bottom: 4px;
    }
}
</style>
