<template>
    <div ref="containerRef" class="db-table-data" :style="height ? { height } : {}">
        <el-table-v2
            ref="tableRef"
            :header-height="showColumnTip && dbConfig.showColumnComment ? 48 : 30"
            :row-height="30"
            :row-class="rowClass"
            :row-key="null"
            :columns="state.columns"
            :data="datas"
            :width="state.containerWidth"
            :height="state.containerHeight"
            fixed
            class="table"
            :row-event-handlers="rowEventHandlers"
            @scroll="onTableScroll"
        >
            <template #header="{ columns }">
                <div v-for="(column, i) in columns" :key="i">
                    <div
                        :style="{
                            width: `${column.width}px`,
                            height: '100%',
                            textAlign: 'center',
                            borderRight: 'var(--el-table-border)',
                            borderTop: 'var(--el-table-border)',
                        }"
                    >
                        <!-- 行号列 -->
                        <div v-if="column.key == rowNoColumn.key" class="header-column-title">
                            <b class="el-text" tag="b"> {{ column.title }} </b>
                        </div>

                        <!-- 字段名列 -->
                        <div v-else style="position: relative" @mouseenter="showColumnAction(column)" @mouseleave="hideColumnAction">
                            <!-- 字段列的数据类型 -->
                            <div class="column-type">
                                <span v-if="column.dataTypeSubscript === 'icon-clock'">
                                    <SvgIcon :size="9" name="Clock" style="cursor: unset" />
                                </span>
                                <span class="text-[8px]!" v-else>{{ column.dataTypeSubscript }}</span>
                            </div>

                            <div v-if="showColumnTip">
                                <div class="header-column-title">
                                    <b :title="column.remark" class="el-text cursor-pointer">
                                        {{ column.title }}
                                    </b>
                                </div>

                                <!-- 字段备注信息 -->
                                <div
                                    v-if="dbConfig.showColumnComment"
                                    style="color: var(--el-color-info-light-3)"
                                    class="text-[10px]! el-text el-text--small is-truncated"
                                >
                                    {{ column.columnComment }}
                                </div>
                            </div>

                            <div v-else class="header-column-title">
                                <b class="el-text"> {{ column.title }} </b>
                            </div>

                            <!-- 字段列右部分内容 -->
                            <div class="column-right">
                                <el-dropdown
                                    @command="handleColumnCommand(column, $event)"
                                    @visibleChange="onColumnActionVisibleChange(column, $event)"
                                    trigger="click"
                                    v-if="column.key !== rowNoColumn.key"
                                    size="small"
                                    placement="bottom-start"
                                >
                                    <span class="column-actions-trigger">
                                        <!-- 排序箭头图标 -->
                                        <SvgIcon
                                            v-if="
                                                column.key == nowSortColumn?.key && !showColumnActions[column.key] && !columnActionVisible[column.key]
                                            "
                                            :color="'var(--el-color-primary)'"
                                            :name="nowSortColumn?.order == 'asc' ? 'top' : 'bottom'"
                                            :size="14"
                                        />
                                        <!-- 更多操作图标 -->
                                        <SvgIcon
                                            v-if="columnActionVisible[column.key] || showColumnActions[column.key]"
                                            name="MoreFilled"
                                            :size="14"
                                            :color="'var(--el-color-primary)'"
                                            class="column-more-icon"
                                            :class="{ 'column-more-icon-visible': columnActionVisible[column.key] || showColumnActions[column.key] }"
                                        />
                                    </span>
                                    <template #dropdown>
                                        <el-dropdown-menu>
                                            <el-dropdown-item v-if="showColumnActionSort" command="sort-asc">
                                                <SvgIcon name="top" class="mr-1" />
                                                {{ $t('db.asc') }}
                                            </el-dropdown-item>
                                            <el-dropdown-item v-if="showColumnActionSort" command="sort-desc">
                                                <SvgIcon name="bottom" class="mr-1" />
                                                {{ $t('db.desc') }}
                                            </el-dropdown-item>
                                            <el-dropdown-item v-if="showColumnActionFixed && !column.fixed" command="fix">
                                                <SvgIcon name="Paperclip" class="mr-1" />
                                                {{ $t('db.fixed') }}
                                            </el-dropdown-item>
                                            <el-dropdown-item v-if="showColumnActionFixed && column.fixed" command="unfix">
                                                <SvgIcon name="Minus" class="mr-1" />
                                                {{ $t('db.cancelFiexd') }}
                                            </el-dropdown-item>
                                        </el-dropdown-menu>
                                    </template>
                                </el-dropdown>
                            </div>
                        </div>
                    </div>
                </div>
            </template>

            <template #cell="{ rowData, column, rowIndex, columnIndex }">
                <div @contextmenu="dataContextmenuClick($event, rowIndex, column, rowData)" class="table-data-cell">
                    <!-- 行号列 -->
                    <div v-if="column.key == rowNoColumn.key">
                        <b class="el-text el-text--small">
                            {{ (pageNum - 1) * pageSize + rowIndex + 1 }}
                        </b>
                    </div>

                    <!-- 数据列 -->
                    <div v-else @dblclick="onEnterEditMode(rowData, column, rowIndex, columnIndex)">
                        <div v-if="canEdit(rowIndex, columnIndex)">
                            <ColumnFormItem
                                v-model="rowData[column.key!]"
                                :data-type="column.dataType"
                                @blur="onExitEditMode(rowData, column, rowIndex)"
                                :column-name="column.columnName"
                                focus
                            />
                        </div>

                        <div v-else :class="isUpdated(rowIndex, column.key) ? 'update_field_active ml-0.5 mr-0.5' : 'ml-0.5 mr-0.5'">
                            <span v-if="rowData[column.key!] === null" style="color: var(--el-color-info-light-5)"> NULL </span>

                            <span v-else :title="rowData[column.key!]" class="el-text el-text--small is-truncated">
                                {{ rowData[column.key!] }}
                            </span>
                        </div>
                    </div>
                </div>
            </template>

            <template v-if="state.loading" #overlay>
                <div class="el-loading-mask flex flex-col items-center justify-center">
                    <div>
                        <SvgIcon class="is-loading" name="loading" color="var(--el-color-primary)" :size="28" />
                        <el-text class="ml-1" tag="b">{{ $t('db.execTime') }} - {{ state.execTime.toFixed(1) }}s</el-text>
                    </div>
                    <div v-if="loading && abortFn" class="mt-2!">
                        <el-button @click="cancelLoading" type="info" size="small" plain>{{ $t('common.cancel') }}</el-button>
                    </div>
                </div>
            </template>

            <template #empty>
                <el-empty class="text-center" :description="props.emptyText" :image-size="60" />
            </template>
        </el-table-v2>

        <el-dialog @close="state.genTxtDialog.visible = false" v-model="state.genTxtDialog.visible" :title="state.genTxtDialog.title" width="1000px">
            <template #header>
                <div class="mr-2" style="display: flex; justify-content: flex-end">
                    <el-button id="copyValue" @click="copyGenTxt(state.genTxtDialog.txt)" icon="CopyDocument" type="success" size="small">
                        {{ $t('db.oneClickCopy') }}
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
            @submit-success="emits('changeUpdatedField')"
        />

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { copyToClipboard } from '@/common/utils/string';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svg-icon/index.vue';
import { DbInst, DbThemeConfig } from '@/views/ops/db/db';
import { useIntervalFn, useStorage } from '@vueuse/core';
import { computed, onBeforeUnmount, onMounted, reactive, ref, toRefs, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { Msg } from '../../../../../hooks/useI18n';
import { ColumnTypeSubscript, DataType, DbDialect, getDbDialect } from '../../dialect/index';
import ColumnFormItem from './ColumnFormItem.vue';
import DbTableDataForm from './DbTableDataForm.vue';
import { useTableSelection } from './composables/useTableSelection';
import { useTableEdit } from './composables/useTableEdit';
import { useTableExport } from './composables/useTableExport';

import type { ColumnMetadata, TableColumnDef } from '../../types';

interface TableColumn extends ColumnMetadata {
    hidden?: boolean;
    show?: boolean;
    showComment?: boolean;
    comment?: string;
    key?: string;
    order?: string;
    /** 数据类型角标 */
    dataTypeSubscript?: string;
    /** 列备注 (类型 + 注释) */
    remark?: string;
}

/**
 * el-table-v2 渲染列：setTableColumns 动态生成的列 + 行号列，
 * 在 TableColumn 基础上补充渲染所需字段（列元数据字段均为可选，行号列无 columnName 等）
 */
interface RenderTableColumn extends Partial<TableColumn> {
    key: string;
    title: string;
    width?: number;
    fixed?: boolean;
    align?: string;
    headerClass?: string;
    class?: string;
    sortable?: boolean;
}

interface ContextmenuRowData {
    rowData: Record<string, unknown>;
    column: { key: string };
    rowIndex: number;
}

const { t } = useI18n();

const emits = defineEmits(['dataDelete', 'sortChange', 'deleteData', 'selectionChange', 'changeUpdatedField']);

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

const contextmenuRef = ref();
const tableRef = ref();
const containerRef = useTemplateRef<HTMLElement>('containerRef');
let resizeObserver: ResizeObserver | null = null;

// 用于控制列操作按钮的显示
const showColumnActions = ref<Record<string, boolean>>({});
const columnActionVisible = ref<Record<string, boolean>>({});

// Composables
const { selectionRowsMap, isSelection, selectionRow, rowEventHandlers, rowClass, clearSelection } = useTableSelection(() => state.datas);

const { nowUpdateCell, cellUpdateMap, canEdit, isUpdated, onEnterEditMode, onExitEditMode: exitEdit, submitUpdateFields: doSubmitUpdate, cancelUpdateFields: doCancelUpdate, clearEditState } = useTableEdit({
    dbId: () => state.dbId,
    db: () => state.db,
    table: () => state.table,
});

const { onExportCsv, onExportExcel, onExportSql, onGenerateJson, onGenerateInsertSql } = useTableExport({
    dbId: () => state.dbId,
    db: () => state.db,
    table: () => state.table,
    datas: () => state.datas,
    columns: () => state.columns,
});

const cmDataCopyCell = new ContextmenuItem('copyValue', 'common.copy')
    .withIcon('CopyDocument')
    .withOnClick(async (data: ContextmenuRowData) => {
        await copyToClipboard(data.rowData[data.column.key] as string);
    })
    .withHideFunc(() => {
        return selectionRowsMap.value.size > 1;
    });

const cmDataDel = new ContextmenuItem('deleteData', 'common.delete')
    .withIcon('delete')
    .withOnClick(() => onDeleteData())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmFormView = new ContextmenuItem('formView', 'db.formView').withIcon('Document').withOnClick(() => onEditRowData());

const cmDataGenInsertSql = new ContextmenuItem('genInsertSql', 'Insert SQL')
    .withIcon('tickets')
    .withOnClick(() => handleGenerateInsertSql())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenJson = new ContextmenuItem('genJson', 'db.genJson').withIcon('tickets').withOnClick(() => handleGenerateJson());

const cmDataExportCsv = new ContextmenuItem('exportCsv', 'db.exportCsv')
    .withIcon('document')
    .withOnClick(() => onExportCsv())
    .withPermission('db:data:export');

const cmDataExportExcel = new ContextmenuItem('exportExcel', 'db.exportExcel')
    .withIcon('document')
    .withOnClick(() => onExportExcel())
    .withPermission('db:data:export');

const cmDataExportSql = new ContextmenuItem('exportSql', 'db.exportSql')
    .withIcon('document')
    .withOnClick(() => onExportSql())
    .withHideFunc(() => {
        return state.table == '';
    })
    .withPermission('db:data:export');

let dbDialect: DbDialect = null!;

let nowSortColumn = ref<{ key: string; order: string } | null>(null);

// 数据加载时间计时器
const { pause, resume } = useIntervalFn(() => {
    state.execTime += 0.1;
}, 100);

const state = reactive({
    dbId: 0, // 当前选中操作的数据库实例
    dbType: '',
    db: '', // 数据库名
    table: '', // 当前的表名
    datas: [] as Record<string, unknown>[],
    columns: [] as RenderTableColumn[],
    loading: false,
    tableHeight: '600px',
    containerHeight: 600,
    containerWidth: 800,
    execTime: 0,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
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

const { containerHeight, containerWidth, datas } = toRefs(state);

const dbConfig = useStorage('dbConfig', DbThemeConfig);

/**
 * 行号字段列
 */
const rowNoColumn = {
    title: 'No.',
    key: 'tableDataRowNo',
    width: 45,
    fixed: true,
    align: 'center',
    headerClass: 'table-column',
    class: 'table-column',
};

watch(
    () => props.data,
    (newValue: unknown) => {
        setTableData(newValue as Record<string, unknown>[]);
    }
);

watch(
    () => props.columns,
    (newValue: TableColumnDef[] | undefined) => {
        // 赋值列字段值是否隐藏，state.columns多了一列索引列
        if (newValue && newValue.length + 1 == state.columns.length) {
            for (let i = 0; i < newValue.length; i++) {
                state.columns[i + 1].hidden = !newValue[i].show;
            }
        }
    },
    {
        deep: true,
    }
);

watch(
    () => props.table,
    (newValue: string) => {
        state.table = newValue;
    }
);

watch(
    () => props.height,
    (newValue: string) => {
        state.tableHeight = newValue;
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

// 显示列排序
const showColumnActionSort = computed(() => {
    return props.columnMoreActions.includes('sort');
});

// 显示列固定
const showColumnActionFixed = computed(() => {
    return props.columnMoreActions.includes('fixed');
});

onMounted(async () => {
    state.tableHeight = props.height;
    state.loading = props.loading;

    // 使用 ResizeObserver 自动测量容器尺寸，确保 el-table-v2 固定表头 + body滚动
    if (containerRef.value) {
        const rect = containerRef.value?.getBoundingClientRect();
        state.containerHeight = rect.height || 600;
        state.containerWidth = rect.width || 800;

        resizeObserver = new ResizeObserver((entries) => {
            for (const entry of entries) {
                const { height, width } = entry.contentRect;
                if (height > 0) state.containerHeight = height;
                if (width > 0) state.containerWidth = width;
            }
        });
        resizeObserver.observe(containerRef.value);
    }

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
    resizeObserver?.disconnect();
});

const setTableData = (datas: Record<string, unknown>[]) => {
    tableRef.value?.scrollTo({ scrollLeft: 0, scrollTop: 0 });
    clearSelection();
    clearEditState();
    state.datas = datas;
    setTableColumns(props.columns ?? []);
};

const setTableColumns = (columns: TableColumnDef[]) => {
    state.columns = columns.map((x: TableColumnDef) => {
        const columnName = x.columnName;
        // 数据类型
        x.dataType = dbDialect.getDataType(x.columnType ?? '');
        x.dataTypeSubscript = ColumnTypeSubscript[x.dataType];
        x.remark = `${x.columnType} ${x.columnComment ? ' |  ' + x.columnComment : ''}`;
        return {
            ...x,
            key: x.key ?? columnName,
            width: DbInst.flexColumnWidth(columnName, state.datas),
            title: columnName,
            align: x.dataType == DataType.Number ? 'right' : 'left',
            headerClass: 'table-column',
            class: 'table-column',
            sortable: true,
            hidden: !x.show,
        };
    });
    if (state.columns.length > 0) {
        state.columns.unshift(rowNoColumn);
    }
};

const startLoading = () => {
    state.execTime = 0;
    resume();
};

const endLoading = () => {
    pause();
};

const cancelLoading = async () => {
    props.abortFn && props.abortFn();
    endLoading();
};

/**
 * 显示列操作按钮
 */
const showColumnAction = (column: RenderTableColumn) => {
    showColumnActions.value[column.key] = true;
};

/**
 * 隐藏列操作按钮
 */
const hideColumnAction = () => {
    showColumnActions.value = {};
};

/**
 * 处理列操作命令
 */
const handleColumnCommand = (column: RenderTableColumn, command: string) => {
    switch (command) {
        case 'sort-asc':
            onTableSortChange({ key: column.key, order: 'asc' });
            break;
        case 'sort-desc':
            onTableSortChange({ key: column.key, order: 'desc' });
            break;
        case 'fix':
            state.columns.forEach((col: RenderTableColumn) => {
                if (col.key == column.key) {
                    col.fixed = true;
                }
            });
            break;
        case 'unfix':
            state.columns.forEach((col: RenderTableColumn) => {
                if (col.key == column.key) {
                    col.fixed = false;
                }
            });
            break;
    }
    // 点击了取消固定等操作后，可能更多的icon还是显示在列上，所以需要重新置为空对象。暂时不懂是组件bug还是啥
    columnActionVisible.value = {};
};

const onColumnActionVisibleChange = (column: RenderTableColumn, visible: boolean) => {
    columnActionVisible.value = {}; // 只显示一个列的更多icon
    columnActionVisible.value[column.key] = visible;
};

const dataContextmenuClick = (event: MouseEvent, rowIndex: number, column: TableColumn, data: Record<string, unknown>) => {
    event.preventDefault(); // 阻止默认的右击菜单行为

    // 当前行未选中，则单行选中该行
    if (!isSelection(rowIndex)) {
        selectionRow(rowIndex, data);
    }
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmDataCopyCell, cmDataDel, cmFormView, cmDataGenInsertSql, cmDataGenJson, cmDataExportExcel, cmDataExportCsv, cmDataExportSql];
    contextmenuRef.value?.openContextmenu({ column, rowData: data });
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: { key: string; order: string }) => {
    nowSortColumn.value = sort;
    cancelUpdateFields();
    emits('sortChange', sort);
};

/**
 * 执行删除数据事件
 */
const onDeleteData = async () => {
    const deleteDatas = Array.from(selectionRowsMap.value.values());
    const db = state.db;
    const dbInst = getNowDbInst();
    dbInst.promptExeSql(db, await dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas as Record<string, unknown>[]), undefined, () => {
        emits('dataDelete', deleteDatas);
    });
};

const onEditRowData = () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    if (selectionDatas.length > 1) {
        Msg.warning('db.onlySelectOneData');
        return;
    }
    const data = selectionDatas[0];
    state.tableDataFormDialog.data = { ...data };
    state.tableDataFormDialog.title = state.table ? `'${props.table}' ${t('db.formView')}` : t('db.formView');
    state.tableDataFormDialog.visible = true;
};

const handleGenerateInsertSql = async () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    state.genTxtDialog.txt = await onGenerateInsertSql(selectionDatas);
    state.genTxtDialog.title = 'SQL';
    state.genTxtDialog.visible = true;
};

const handleGenerateJson = () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    state.genTxtDialog.txt = onGenerateJson(selectionDatas);
    state.genTxtDialog.title = 'JSON';
    state.genTxtDialog.visible = true;
};

const copyGenTxt = async (txt: string) => {
    await copyToClipboard(txt);
    state.genTxtDialog.visible = false;
};

const onExitEditMode = (rowData: Record<string, unknown>, column: RenderTableColumn, rowIndex = 0) => {
    exitEdit(rowData, column, rowIndex);
    changeUpdatedField();
};

const submitUpdateFields = async () => {
    await doSubmitUpdate(() => changeUpdatedField());
};

const cancelUpdateFields = () => {
    doCancelUpdate(() => changeUpdatedField());
};

const changeUpdatedField = () => {
    emits('changeUpdatedField', cellUpdateMap.value);
};

const scrollLeftValue = ref(0);
const onTableScroll = (param: { scrollLeft: number }) => {
    scrollLeftValue.value = param.scrollLeft;
};

/**
 * 激活表格，恢复滚动位置，否则会造成表头与数据单元格错位(暂不知为啥，先这样解决)
 */
const active = () => {
    setTimeout(() => tableRef.value?.scrollToLeft(scrollLeftValue.value));
};

const getNowDbInst = () => {
    return DbInst.getInst(state.dbId);
};

defineExpose({
    active,
    submitUpdateFields,
    cancelUpdateFields,
});
</script>

<style lang="scss">
.db-table-data {
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
