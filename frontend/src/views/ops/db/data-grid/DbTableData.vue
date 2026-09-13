<template>
    <div class="db-table-data" :style="height ? { height } : {}">
        <VirtualTable
            ref="virtualTableRef"
            :data="datas"
            :columns="state.columns"
            :loading="state.loading"
            :header-height="showColumnTip && dbConfig.showColumnComment ? 48 : 30"
            :row-height="30"
            :row-class="rowClass"
            :row-event-handlers="rowEventHandlers"
            :height="height"
            @scroll-change="onScrollChange"
        >
            <!-- DB 特有的表头渲染（含列操作、备注信息） -->
            <template #header="{ columns }">
                <DbTableDataHeader
                    :columns="columns"
                    :row-no-column-key="rowNoColumnKey"
                    :show-column-tip="showColumnTip"
                    :show-column-comment="dbConfig.showColumnComment"
                    :now-sort-column="nowSortColumn"
                    :show-column-action-sort="showColumnActionSort"
                    :show-column-action-fixed="showColumnActionFixed"
                    @sort-change="onHeaderSortChange"
                    @fix-change="onHeaderFixChange"
                />
            </template>

            <!-- DB 特有的单元格渲染（含编辑模式） -->
            <template #cell="{ rowData, column, rowIndex, columnIndex }">
                <div @contextmenu="onCellContextmenu($event, rowIndex, column, rowData)" class="table-data-cell">
                    <!-- 行号列 -->
                    <div v-if="column.key === rowNoColumnKey">
                        <b class="el-text el-text--small">
                            {{ (pageNum - 1) * pageSize + rowIndex + 1 }}
                        </b>
                    </div>

                    <!-- 数据列 -->
                    <div v-else @dblclick="onEnterEditMode(rowData, column, rowIndex, columnIndex)">
                        <div v-if="canEdit(rowIndex, columnIndex)">
                            <ColumnValueInput
                                v-model="rowData[column.key!]"
                                :data-type="column.dataType"
                                @blur="onExitEditMode(rowData, column, rowIndex)"
                                :column-name="column.columnName"
                                focus
                            />
                        </div>

                        <div v-else :class="isUpdated(rowIndex, column.key) ? 'update_field_active ml-0.5 mr-0.5' : 'ml-0.5 mr-0.5'">
                            <span v-if="rowData[column.key!] === null" style="color: var(--el-color-info-light-5)"> NULL </span>
                            <span v-else :title="String(rowData[column.key!])" class="el-text el-text--small is-truncated">
                                {{ rowData[column.key!] }}
                            </span>
                        </div>
                    </div>
                </div>
            </template>

            <!-- 加载覆盖层 -->
            <template #overlay>
                <div class="el-loading-mask flex flex-col items-center justify-center">
                    <div>
                        <SvgIcon class="is-loading" name="loading" color="var(--el-color-primary)" :size="28" />
                        <el-text class="ml-1" tag="b">{{ t('db.execTime') }} - {{ state.execTime.toFixed(1) }}s</el-text>
                    </div>
                    <div v-if="abortFn" class="mt-2!">
                        <el-button @click="abortFn()" type="info" size="small" plain>{{ t('common.cancel') }}</el-button>
                    </div>
                </div>
            </template>
        </VirtualTable>

        <el-dialog @close="state.genTxtDialog.visible = false" v-model="state.genTxtDialog.visible" :title="state.genTxtDialog.title" width="1000px">
            <template #header>
                <div class="mr-2" style="display: flex; justify-content: flex-end">
                    <el-button id="copyValue" @click="copyGenTxt(state.genTxtDialog.txt)" icon="CopyDocument" type="success" size="small">
                        {{ t('db.oneClickCopy') }}
                    </el-button>
                </div>
            </template>
            <el-input v-model="state.genTxtDialog.txt" type="textarea" :rows="20" />
        </el-dialog>

        <DbTableDataForm
            v-if="state.tableDataFormDialog.visible"
            :db-inst="getNowDbInst()"
            :db-name="db"
            :columns="columns!"
            :title="state.tableDataFormDialog.title"
            :table-name="table"
            v-model:visible="state.tableDataFormDialog.visible"
            v-model="state.tableDataFormDialog.data"
            @submit-success="changeUpdatedField"
        />

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { copyToClipboard } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { VirtualTable } from '@/components/virtual-table';
import type { VirtualTableColumn } from '@/components/virtual-table/adapters';
import { DbInst, DbThemeConfig } from '@/views/ops/db/db';
import { useIntervalFn, useStorage } from '@vueuse/core';
import { computed, onBeforeUnmount, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '../../../../hooks/useI18n';
import { ColumnTypeSubscript, DataType, DbDialect, getDbDialect } from '../dialect/index';
import type { ColumnMetadata, TableColumnDef, TableExportStrategy } from '../types';
import ColumnValueInput from '../widgets/ColumnValueInput.vue';
import DbTableDataForm from './DbTableDataForm.vue';
import DbTableDataHeader from './DbTableDataHeader.vue';
import { useTableSelection } from './composables/useTableSelection';
import { useTableEdit } from './composables/useTableEdit';
import { useTableExport } from './composables/useTableExport';

const { t } = useI18n();

/**
 * 事件契约（必须用类型式声明）
 *
 * 这里曾是数组式 `defineEmits(['changeUpdatedField', ...])`，payload 不进类型检查，
 * 于是重构时漏传参数只表现为「工具条的提交/取消按钮不见了」，编译与测试都不报错。
 * 改成类型式后，漏参或父级签名不符会直接被 vue-tsc 拦下。
 */
const emits = defineEmits<{
    /** 待提交变更集合变化，父级据此显隐工具条的「提交/取消」 */
    changeUpdatedField: [hasUpdatedFields: boolean];
    /** 按主键删除成功，回传被删行 */
    dataDelete: [deleteDatas: Record<string, unknown>[]];
    /** 表头排序变更 */
    sortChange: [sort: { key: string; order: string }];
}>();

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    db: {
        type: String,
        required: true,
    },
    table: {
        type: String,
        default: '',
    },
    data: {
        type: Array,
    },
    columns: {
        type: Array<TableColumnDef>,
    },
    columnMoreActions: {
        type: Array,
        default: () => ['sort', 'fixed'],
    },
    loading: {
        type: Boolean,
        default: false,
    },
    abortFn: {
        type: Function,
    },
    emptyText: {
        type: String,
        default: 'No Data',
    },
    showColumnTip: {
        type: Boolean,
        default: false,
    },
    height: {
        type: String,
        default: '',
    },
    pageSize: {
        type: Number,
        default: 25,
    },
    pageNum: {
        type: Number,
        default: 1,
    },
});

// ==================== Refs ====================

const contextmenuRef = ref();
const virtualTableRef = ref<InstanceType<typeof VirtualTable>>();

// 行号列 key 常量
const rowNoColumnKey = 'tableDataRowNo';

// ==================== 状态 ====================

let dbDialect: DbDialect = null!;

const nowSortColumn = ref<{ key: string; order: string } | null>(null);

// 数据加载时间计时器
const { pause, resume } = useIntervalFn(() => {
    state.execTime += 0.1;
}, 100);

const state = reactive({
    dbId: 0,
    dbType: '',
    db: '',
    table: '',
    datas: [] as Record<string, unknown>[],
    columns: [] as VirtualTableColumn[],
    loading: false,
    execTime: 0,
    contextmenu: {
        dropdown: { x: 0, y: 0 },
        items: [] as ContextmenuItem[],
    },
    tableDataFormDialog: {
        data: {} as Record<string, unknown>,
        title: '',
        visible: false,
    },
    genTxtDialog: {
        title: 'SQL',
        visible: false,
        txt: '',
    },
});

const { datas } = toRefs(state);
const dbConfig = useStorage('dbConfig', DbThemeConfig);

// ==================== Composables ====================

const { selectionRowsMap, isSelection, selectionRow, rowEventHandlers, rowClass, clearSelection } = useTableSelection(() => state.datas);

const { canEdit, isUpdated, hasUpdatedFields, onEnterEditMode, onExitEditMode: exitEdit, submitUpdateFields: doSubmitUpdate, cancelUpdateFields: doCancelUpdate, clearEditState } = useTableEdit({
    dbId: () => state.dbId,
    db: () => state.db,
    table: () => state.table,
});

const { executeStrategy, exportByKey, getExportStrategies, getGenStrategies } = useTableExport({
    dbId: () => state.dbId,
    db: () => state.db,
    table: () => state.table,
    datas: () => state.datas,
    columns: () => state.columns,
});

// 右键菜单项
const cmDataCopyCell = new ContextmenuItem('copyValue', 'common.copy')
    .withIcon('CopyDocument')
    .withOnClick(async (data: { rowData: Record<string, unknown>; column: { key: string } }) => {
        await copyToClipboard(data.rowData[data.column.key] as string);
    })
    .withHideFunc(() => selectionRowsMap.value.size > 1);

const cmDataDel = new ContextmenuItem('deleteData', 'common.delete')
    .withIcon('delete')
    .withOnClick(() => onDeleteData())
    .withHideFunc(() => state.table === '');

const cmFormView = new ContextmenuItem('formView', 'db.formView').withIcon('Document').withOnClick(() => onEditRowData());

/** 把导出策略翻译为菜单项：文案/图标/权限/可见性全部取自策略自描述，本组件不认识任何具体格式 */
const toStrategyMenuItem = (strategy: TableExportStrategy, onClick: () => void) => {
    const item = new ContextmenuItem(strategy.key, strategy.labelI18nKey).withOnClick(onClick);
    if (strategy.icon) {
        item.withIcon(strategy.icon);
    }
    if (strategy.permission) {
        item.withPermission(strategy.permission);
    }
    if (strategy.requireTable) {
        item.withHideFunc(() => state.table === '');
    }
    return item;
};

/** 生成类策略：对选中行执行，把返回的文本展示到弹窗 */
const onGenStrategy = async (strategy: TableExportStrategy) => {
    const txt = await executeStrategy(strategy, Array.from(selectionRowsMap.value.values()));
    if (typeof txt !== 'string') {
        return;
    }
    state.genTxtDialog.title = strategy.resultTitle ?? t(strategy.labelI18nKey);
    state.genTxtDialog.txt = txt;
    state.genTxtDialog.visible = true;
};

const cmDataGen = new ContextmenuItem('gen', 'db.gen')
    .withIcon('tickets')
    .withChildren(getGenStrategies().map((s) => toStrategyMenuItem(s, () => onGenStrategy(s))));

const cmDataExport = new ContextmenuItem('export', 'db.export')
    .withIcon('Download')
    .withChildren(getExportStrategies().map((s) => toStrategyMenuItem(s, () => exportByKey(s.key))));

// ==================== 计算属性 ====================

const showColumnActionSort = computed(() => props.columnMoreActions.includes('sort'));
const showColumnActionFixed = computed(() => props.columnMoreActions.includes('fixed'));

// ==================== Watchers ====================

watch(
    () => props.data,
    (newValue: unknown) => {
        setTableData(newValue as Record<string, unknown>[]);
    }
);

watch(
    () => props.columns,
    (newValue: TableColumnDef[] | undefined) => {
        if (newValue && newValue.length + 1 === state.columns.length) {
            for (let i = 0; i < newValue.length; i++) {
                state.columns[i + 1].hidden = !newValue[i].show;
            }
        }
    },
    { deep: true }
);

watch(
    () => props.table,
    (newValue: string) => {
        state.table = newValue;
    }
);

watch(
    () => props.loading,
    (newValue: boolean) => {
        state.loading = newValue;
        if (newValue) {
            startLoading();
        } else {
            endLoading();
        }
    }
);

// ==================== 生命周期 ====================

onMounted(() => {
    state.dbId = props.dbId;
    state.dbType = getNowDbInst().type;
    dbDialect = getDbDialect(state.dbType);
    state.db = props.db;
    state.table = props.table;
    setTableData((props.data ?? []) as Record<string, unknown>[]);

    if (state.loading) {
        startLoading();
    }
});

onBeforeUnmount(() => {
    endLoading();
});

// ==================== 数据与列管理 ====================

const setTableData = (datas: Record<string, unknown>[]) => {
    virtualTableRef.value?.scrollTo({ scrollLeft: 0, scrollTop: 0 });
    clearSelection();
    clearEditState();
    state.datas = datas;
    setTableColumns(props.columns ?? []);
};

const setTableColumns = (columns: TableColumnDef[]) => {
    state.columns = columns.map((x: TableColumnDef) => {
        const columnName = x.columnName;
        const dataType = dbDialect.getDataType(x.columnType ?? '');
        const dataTypeSubscript = ColumnTypeSubscript[dataType];
        const remark = `${x.columnType} ${x.columnComment ? ' |  ' + x.columnComment : ''}`;
        const title = x.masked ? `${columnName} [${t('db.maskedTag')}]` : columnName;
        return {
            ...x,
            dataType,
            dataTypeSubscript,
            remark,
            key: x.key ?? columnName,
            width: DbInst.flexColumnWidth(title, state.datas),
            title,
            align: dataType === DataType.Number ? 'right' : 'left',
            headerClass: 'table-column',
            class: 'table-column',
            sortable: true,
            hidden: !x.show,
        };
    });
    if (state.columns.length > 0) {
        state.columns.unshift({
            title: 'No.',
            key: rowNoColumnKey,
            width: 45,
            fixed: true,
            align: 'center',
            headerClass: 'table-column',
            class: 'table-column',
        });
    }
};

// ==================== 加载控制 ====================

const startLoading = () => {
    state.execTime = 0;
    resume();
};

const endLoading = () => {
    pause();
};

// ==================== 滚动处理 ====================

let scrollLeftValue = 0;

const onScrollChange = (scroll: { scrollLeft: number; scrollTop: number }) => {
    scrollLeftValue = scroll.scrollLeft;
};

// ==================== 表头事件 ====================

const onHeaderSortChange = (sort: { key: string; order: string }) => {
    nowSortColumn.value = sort;
    cancelUpdateFields();
    emits('sortChange', sort);
};

const onHeaderFixChange = (column: VirtualTableColumn, fixed: boolean) => {
    state.columns.forEach((col) => {
        if (col.key === column.key) {
            col.fixed = fixed;
        }
    });
};

// ==================== 单元格事件 ====================

const onCellContextmenu = (event: MouseEvent, rowIndex: number, column: { key: string }, rowData: Record<string, unknown>) => {
    event.preventDefault();
    if (!isSelection(rowIndex)) {
        selectionRow(rowIndex, rowData);
    }
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmDataCopyCell, cmDataDel, cmFormView, cmDataGen, cmDataExport];
    contextmenuRef.value?.openContextmenu({ column, rowData });
};

const onExitEditMode = (rowData: Record<string, unknown>, column: VirtualTableColumn, rowIndex = 0) => {
    exitEdit(rowData, column, rowIndex);
    changeUpdatedField();
};

// ==================== 数据操作 ====================

const onDeleteData = async () => {
    const deleteDatas = Array.from(selectionRowsMap.value.values());
    const dbInst = getNowDbInst();
    dbInst.promptExeSql(state.db, await dbInst.genDeleteByPrimaryKeysSql(state.db, state.table, deleteDatas as Record<string, unknown>[]), undefined, () => {
        emits('dataDelete', deleteDatas);
    });
};

const onEditRowData = () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    if (selectionDatas.length > 1) {
        Msg.warning('db.onlySelectOneData');
        return;
    }
    state.tableDataFormDialog.data = { ...selectionDatas[0] };
    state.tableDataFormDialog.title = state.table ? `'${props.table}' ${t('db.formView')}` : t('db.formView');
    state.tableDataFormDialog.visible = true;
};

const copyGenTxt = async (txt: string) => {
    await copyToClipboard(txt);
    state.genTxtDialog.visible = false;
};

const submitUpdateFields = async () => {
    await doSubmitUpdate(() => changeUpdatedField());
};

const cancelUpdateFields = () => {
    doCancelUpdate(() => changeUpdatedField());
};

const changeUpdatedField = () => {
    emits('changeUpdatedField', hasUpdatedFields.value);
};

const getNowDbInst = () => DbInst.getInst(state.dbId);

// ==================== 公开 API ====================

/** 激活表格，恢复滚动位置 */
const active = () => {
    setTimeout(() => virtualTableRef.value?.scrollToLeft(scrollLeftValue));
};

defineExpose({
    active,
    submitUpdateFields,
    cancelUpdateFields,
});
</script>

<style lang="scss">
.db-table-data {
    height: 100%;
    display: flex;
    flex-direction: column;

    .table-column {
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

    .data-selection {
        background-color: var(--el-table-current-row-bg-color);
    }

    .update_field_active {
        background-color: var(--el-color-success-light-3);
    }

    .column-type {
        color: var(--el-color-info-light-3);
        font-weight: bold;
        position: absolute;
        top: -7px;
        padding: 2px;
    }

    .column-right {
        position: absolute;
        top: 2px;
        right: 0;
        padding: 2px;
        display: flex;
        align-items: center;
    }

    .column-actions-trigger {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        width: 16px;
        height: 16px;
        border-radius: 50%;
        cursor: pointer;

        &:hover {
            background-color: var(--el-fill-color-light);
        }
    }

    .column-more-icon {
        opacity: 0;
        transition: opacity 0.2s;
    }

    .column-more-icon-visible {
        opacity: 1 !important;
    }
}
</style>
