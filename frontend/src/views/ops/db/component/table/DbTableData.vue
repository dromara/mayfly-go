<template>
    <div class="db-table-data mt-1" :style="{ height: tableHeight }">
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-table-v2
                    ref="tableRef"
                    :header-height="showColumnTip && dbConfig.showColumnComment ? 48 : 30"
                    :row-height="30"
                    :row-class="rowClass"
                    :row-key="null"
                    :columns="state.columns"
                    :data="datas"
                    :width="width"
                    :height="height"
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
                                }"
                            >
                                <!-- 行号列 -->
                                <div v-if="column.key == rowNoColumn.key" class="header-column-title">
                                    <b class="el-text" tag="b"> {{ column.title }} </b>
                                </div>

                                <!-- 字段名列 -->
                                <div v-else @contextmenu="headerContextmenuClick($event, column)" style="position: relative">
                                    <!-- 字段列的数据类型 -->
                                    <div class="column-type">
                                        <span v-if="column.dataTypeSubscript === 'icon-clock'">
                                            <SvgIcon :size="9" name="Clock" style="cursor: unset" />
                                        </span>
                                        <span class="!text-[8px]" v-else>{{ column.dataTypeSubscript }}</span>
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
                                            class="!text-[10px] el-text el-text--small is-truncated"
                                        >
                                            {{ column.columnComment }}
                                        </div>
                                    </div>

                                    <div v-else class="header-column-title">
                                        <b class="el-text"> {{ column.title }} </b>
                                    </div>

                                    <!-- 字段列右部分内容 -->
                                    <div class="column-right">
                                        <span v-if="column.title == nowSortColumn?.columnName">
                                            <SvgIcon color="var(--el-color-primary)" :name="nowSortColumn?.order == 'asc' ? 'top' : 'bottom'"></SvgIcon>
                                        </span>
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
                                    {{ rowIndex + 1 }}
                                </b>
                            </div>

                            <!-- 数据列 -->
                            <div v-else @dblclick="onEnterEditMode(rowData, column, rowIndex, columnIndex)">
                                <div v-if="canEdit(rowIndex, columnIndex)">
                                    <ColumnFormItem
                                        v-model="rowData[column.dataKey!]"
                                        :data-type="column.dataType"
                                        @blur="onExitEditMode(rowData, column, rowIndex)"
                                        :column-name="column.columnName"
                                        focus
                                    />
                                </div>

                                <div v-else :class="isUpdated(rowIndex, column.dataKey) ? 'update_field_active ml-0.5 mr-0.5' : 'ml-0.5 mr-0.5'">
                                    <span v-if="rowData[column.dataKey!] === null" style="color: var(--el-color-info-light-5)"> NULL </span>

                                    <span v-else :title="rowData[column.dataKey!]" class="el-text el-text--small is-truncated">
                                        {{ rowData[column.dataKey!] }}
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
                            <div v-if="loading && abortFn" class="!mt-2">
                                <el-button @click="cancelLoading" type="info" size="small" plain>{{ $t('common.cancel') }}</el-button>
                            </div>
                        </div>
                    </template>

                    <template #empty>
                        <el-empty class="text-center" :description="props.emptyText" :image-size="60" />
                    </template>
                </el-table-v2>
            </template>
        </el-auto-resizer>

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
import { onBeforeUnmount, onMounted, reactive, ref, toRefs, watch, Ref } from 'vue';
import { ElInput, ElMessage } from 'element-plus';
import { copyToClipboard } from '@/common/utils/string';
import { DbInst, DbThemeConfig } from '@/views/ops/db/db';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svgIcon/index.vue';
import { exportCsv, exportFile } from '@/common/utils/export';
import { formatDate } from '@/common/utils/format';
import { useIntervalFn, useStorage } from '@vueuse/core';
import { ColumnTypeSubscript, DataType, DbDialect, getDbDialect } from '../../dialect/index';
import ColumnFormItem from './ColumnFormItem.vue';
import DbTableDataForm from './DbTableDataForm.vue';
import { useI18n } from 'vue-i18n';

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
        type: Array<any>,
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
        default: '600px',
    },
});

const contextmenuRef = ref();
const tableRef = ref();

/**  表头 menu items  **/

const cmHeaderAsc = new ContextmenuItem('asc', 'db.asc')
    .withIcon('top')
    .withOnClick((data: any) => {
        onTableSortChange({ columnName: data.dataKey, order: 'asc' });
    })
    .withHideFunc(() => !props.showColumnTip);

const cmHeaderDesc = new ContextmenuItem('desc', 'db.desc')
    .withIcon('bottom')
    .withOnClick((data: any) => {
        onTableSortChange({ columnName: data.dataKey, order: 'desc' });
    })
    .withHideFunc(() => !props.showColumnTip);

const cmHeaderFixed = new ContextmenuItem('fixed', 'db.fixed')
    .withIcon('Paperclip')
    .withOnClick((data: any) => {
        state.columns.forEach((column: any) => {
            if (column.dataKey == data.dataKey) {
                column.fixed = true;
            }
        });
    })
    .withHideFunc((data: any) => data.fixed);

const cmHeaderCancelFixed = new ContextmenuItem('cancelFixed', 'db.cancelFiexd')
    .withIcon('Minus')
    .withOnClick((data: any) => {
        state.columns.forEach((column: any) => {
            if (column.dataKey == data.dataKey) {
                column.fixed = false;
            }
        });
    })
    .withHideFunc((data: any) => !data.fixed);

/**  表数据 contextmenu items  **/

const cmDataCopyCell = new ContextmenuItem('copyValue', 'common.copy')
    .withIcon('CopyDocument')
    .withOnClick(async (data: any) => {
        await copyToClipboard(data.rowData[data.column.dataKey]);
    })
    .withHideFunc(() => {
        // 选中多条则隐藏该复制按钮
        return selectionRowsMap.value.size > 1;
    });

const cmDataDel = new ContextmenuItem('deleteData', 'common.delete')
    .withIcon('delete')
    .withOnClick(() => onDeleteData())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmFormView = new ContextmenuItem('formView', 'db.formView').withIcon('Document').withOnClick(() => onEditRowData());
// .withHideFunc(() => {
//     return state.table == '';
// });

const cmDataGenInsertSql = new ContextmenuItem('genInsertSql', 'Insert SQL')
    .withIcon('tickets')
    .withOnClick(() => onGenerateInsertSql())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenJson = new ContextmenuItem('genJson', 'db.genJson').withIcon('tickets').withOnClick(() => onGenerateJson());

const cmDataExportCsv = new ContextmenuItem('exportCsv', 'db.exportCsv').withIcon('document').withOnClick(() => onExportCsv());

const cmDataExportSql = new ContextmenuItem('exportSql', 'db.exportSql')
    .withIcon('document')
    .withOnClick(() => onExportSql())
    .withHideFunc(() => {
        return state.table == '';
    });

class NowUpdateCell {
    rowIndex: number;
    colIndex: number;
    dataType: DataType;
    oldValue: any;
}

class UpdatedRow {
    /**
     * 主键值
     */
    primaryValue: any;

    /**
     * 行数据
     */
    rowData: any;

    /**
     * 修改到的列信息, columnName -> tablecelldata
     */
    columnsMap = new Map<string, TableCellData>();
}

class TableCellData {
    /**
     * 旧值
     */
    oldValue: any;
}

let dbDialect: DbDialect = null as any;

let nowSortColumn = ref(null) as any;

// 当前正在更新的单元格
let nowUpdateCell: Ref<NowUpdateCell> = ref(null) as any;

// 选中的数据， key->rowIndex  value->primaryKeyValue
const selectionRowsMap = ref(new Map<number, any>());

// 更新单元格  key-> rowIndex  value -> 更新行
const cellUpdateMap = ref(new Map<number, UpdatedRow>());

// 数据加载时间计时器
const { pause, resume } = useIntervalFn(() => {
    state.execTime += 0.1;
}, 100);

const state = reactive({
    dbId: 0, // 当前选中操作的数据库实例
    dbType: '',
    db: '', // 数据库名
    table: '', // 当前的表名
    datas: [],
    columns: [] as any,
    loading: false,
    tableHeight: '600px',
    execTime: 0,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [] as ContextmenuItem[],
    },
    tableDataFormDialog: {
        data: {},
        title: '',
        visible: false,
    },
    genTxtDialog: {
        title: 'SQL',
        visible: false,
        txt: '',
    },
});

const { tableHeight, datas } = toRefs(state);

const dbConfig = useStorage('dbConfig', DbThemeConfig);

/**
 * 行号字段列
 */
const rowNoColumn = {
    title: 'No.',
    key: 'tableDataRowNo',
    dataKey: 'tableDataRowNo',
    width: 45,
    fixed: true,
    align: 'center',
    headerClass: 'table-column',
    class: 'table-column',
};

watch(
    () => props.data,
    (newValue: any) => {
        setTableData(newValue);
    }
);

watch(
    () => props.columns,
    (newValue: any) => {
        // 赋值列字段值是否隐藏，state.columns多了一列索引列
        if (newValue.length + 1 == state.columns.length) {
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
    (newValue: any) => {
        state.table = newValue;
    }
);

watch(
    () => props.height,
    (newValue: any) => {
        state.tableHeight = newValue;
    }
);

watch(
    () => props.loading,
    (newValue: any) => {
        state.loading = newValue;
        if (newValue) {
            startLoading();
        } else {
            endLoading();
        }
    }
);

onMounted(async () => {
    console.log('in DbTable mounted');
    state.tableHeight = props.height;
    state.loading = props.loading;

    state.dbId = props.dbId;
    state.dbType = getNowDbInst().type;
    dbDialect = getDbDialect(state.dbType);

    state.db = props.db;
    state.table = props.table;
    setTableData(props.data);

    if (state.loading) {
        startLoading();
    }
});

onBeforeUnmount(() => {
    endLoading();
});

const setTableData = (datas: any) => {
    tableRef.value?.scrollTo({ scrollLeft: 0, scrollTop: 0 });
    selectionRowsMap.value.clear();
    cellUpdateMap.value.clear();
    // formatDataValues(datas);
    state.datas = datas;
    setTableColumns(props.columns);
};

const setTableColumns = (columns: any) => {
    state.columns = columns.map((x: any) => {
        const columnName = x.columnName;
        // 数据类型
        x.dataType = dbDialect.getDataType(x.columnType);
        x.dataTypeSubscript = ColumnTypeSubscript[x.dataType];
        x.remark = `${x.columnType} ${x.columnComment ? ' |  ' + x.columnComment : ''}`;
        return {
            ...x,
            key: columnName,
            dataKey: columnName,
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
 * 当前单元格是否允许编辑
 * @param rowIndex ri
 * @param colIndex ci
 */
const canEdit = (rowIndex: number, colIndex: number) => {
    return state.table && nowUpdateCell.value?.rowIndex == rowIndex && nowUpdateCell.value?.colIndex == colIndex;
};

/**
 * 判断当前单元格是否被更新了
 * @param rowIndex ri
 * @param columnName cn
 */
const isUpdated = (rowIndex: number, columnName: string) => {
    return cellUpdateMap.value.get(rowIndex)?.columnsMap.get(columnName);
};

/**
 * 判断当前行是否被选中
 * @param rowIndex
 */
const isSelection = (rowIndex: number): boolean => {
    return selectionRowsMap.value.get(rowIndex);
};

/**
 * 选中指定行
 * @param rowIndex
 * @param rowData
 * @param isMultiple 是否允许多选
 */
const selectionRow = (rowIndex: number, rowData: any, isMultiple = false) => {
    if (isMultiple) {
        // 如果重复点击，则取消改选中数据
        if (selectionRowsMap.value.get(rowIndex)) {
            selectionRowsMap.value.delete(rowIndex);
            return;
        }
    } else {
        selectionRowsMap.value.clear();
    }
    selectionRowsMap.value.set(rowIndex, rowData);
};

/**
 * 行事件处理
 */
const rowEventHandlers = {
    onClick: (e: any) => {
        const event = e.event;
        const rowIndex = e.rowIndex;
        const rowData = e.rowData;
        // 按住ctrl点击，则新建标签页打开, metaKey对应mac command键
        if (event.ctrlKey || event.metaKey) {
            selectionRow(rowIndex, rowData, true);
            return;
        }
        selectionRow(rowIndex, rowData);
    },
};

const headerContextmenuClick = (event: any, data: any) => {
    event.preventDefault(); // 阻止默认的右击菜单行为

    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmHeaderAsc, cmHeaderDesc, cmHeaderFixed, cmHeaderCancelFixed];
    contextmenuRef.value.openContextmenu(data);
};

const dataContextmenuClick = (event: any, rowIndex: number, column: any, data: any) => {
    event.preventDefault(); // 阻止默认的右击菜单行为

    // 当前行未选中，则单行选中该行
    if (!isSelection(rowIndex)) {
        selectionRow(rowIndex, data);
    }
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmDataCopyCell, cmDataDel, cmFormView, cmDataGenInsertSql, cmDataGenJson, cmDataExportCsv, cmDataExportSql];
    contextmenuRef.value.openContextmenu({ column, rowData: data });
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: any) => {
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
    dbInst.promptExeSql(db, await dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas as any), null, () => {
        emits('dataDelete', deleteDatas);
    });
};

const onEditRowData = () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    if (selectionDatas.length > 1) {
        ElMessage.warning(t('db.onlySelectOneData'));
        return;
    }
    const data = selectionDatas[0];
    state.tableDataFormDialog.data = { ...data };
    state.tableDataFormDialog.title = state.table ? `'${props.table}' ${t('db.formView')}` : t('db.formView');
    state.tableDataFormDialog.visible = true;
};

const onGenerateInsertSql = async () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    state.genTxtDialog.txt = await getNowDbInst().genInsertSql(state.db, state.table, selectionDatas);
    state.genTxtDialog.title = 'SQL';
    state.genTxtDialog.visible = true;
};

const onGenerateJson = async () => {
    const selectionDatas = Array.from(selectionRowsMap.value.values());
    // 按列字段重新排序对象key
    const jsonObj = [];
    for (let selectionData of selectionDatas) {
        let obj: any = {};
        for (let column of state.columns) {
            if (column.show) {
                obj[column.title] = selectionData[column.dataKey];
            }
        }
        jsonObj.push(obj);
    }
    state.genTxtDialog.txt = JSON.stringify(jsonObj, null, 4);
    state.genTxtDialog.title = 'JSON';
    state.genTxtDialog.visible = true;
};

const copyGenTxt = async (txt: string) => {
    await copyToClipboard(txt);
    state.genTxtDialog.visible = false;
};

/**
 * 导出当前页数据
 */
const onExportCsv = () => {
    const dataList = state.datas as any;
    let columnNames = [];
    for (let column of state.columns) {
        if (column.show) {
            columnNames.push(column.columnName);
        }
    }
    exportCsv(`Data-${state.table}-${formatDate(new Date(), 'YYYYMMDDHHmm')}`, columnNames, dataList);
};

const onExportSql = async () => {
    const selectionDatas = state.datas;
    exportFile(`Data-${state.table}-${formatDate(new Date(), 'YYYYMMDDHHmm')}.sql`, await getNowDbInst().genInsertSql(state.db, state.table, selectionDatas));
};

const onEnterEditMode = (rowData: any, column: any, rowIndex = 0, columnIndex = 0) => {
    // 不存在表，或者已经在编辑中，则不处理
    if (!state.table || nowUpdateCell.value) {
        return;
    }

    nowUpdateCell.value = {
        rowIndex: rowIndex,
        colIndex: columnIndex,
        oldValue: rowData[column.dataKey],
        dataType: column.dataType,
    };
};

const onExitEditMode = (rowData: any, column: any, rowIndex = 0) => {
    if (!nowUpdateCell.value) {
        return;
    }
    const oldValue = nowUpdateCell.value.oldValue;
    const newValue = rowData[column.dataKey];

    // 未改变单元格值
    if (oldValue == newValue) {
        nowUpdateCell.value = null as any;
        return;
    }

    let updatedRow = cellUpdateMap.value.get(rowIndex);
    if (!updatedRow) {
        updatedRow = new UpdatedRow();
        updatedRow.rowData = rowData;
        cellUpdateMap.value.set(rowIndex, updatedRow);
    }

    const columnName = column.dataKey;
    let cellData = updatedRow.columnsMap.get(columnName);
    if (cellData) {
        // 多次修改情况，可能又修改回原值，则移除该修改单元格
        if (cellData.oldValue == newValue) {
            cellUpdateMap.value.delete(rowIndex);
        }
    } else {
        cellData = new TableCellData();
        cellData.oldValue = oldValue;
        updatedRow.columnsMap.set(columnName, cellData);
    }

    nowUpdateCell.value = null as any;
    changeUpdatedField();
};

const submitUpdateFields = async () => {
    const dbInst = getNowDbInst();
    if (cellUpdateMap.value.size == 0) {
        return;
    }

    const db = state.db;
    let res = '';

    for (let updateRow of cellUpdateMap.value.values()) {
        const rowData = { ...updateRow.rowData };
        let updateColumnValue: any = {};

        for (let k of updateRow.columnsMap.keys()) {
            const v = updateRow.columnsMap.get(k);
            if (!v) {
                continue;
            }
            updateColumnValue[k] = rowData[k];
            // 将更新的字段对应的原始数据还原（主要应对可能更新修改了主键等）
            rowData[k] = v.oldValue;
        }
        res += await dbInst.genUpdateSql(db, state.table, updateColumnValue, rowData);
    }

    dbInst.promptExeSql(db, res, null, () => {
        cellUpdateMap.value.clear();
        changeUpdatedField();
    });
};

const cancelUpdateFields = () => {
    const updateRows = cellUpdateMap.value.values();
    // 恢复原值
    for (let updateRow of updateRows) {
        const rowData = updateRow.rowData;
        updateRow.columnsMap.forEach((v: TableCellData, k: string) => {
            rowData[k] = v.oldValue;
        });
    }
    cellUpdateMap.value.clear();
    changeUpdatedField();
};

const changeUpdatedField = () => {
    emits('changeUpdatedField', cellUpdateMap.value);
};

const rowClass = (row: any) => {
    if (isSelection(row.rowIndex)) {
        return 'data-selection';
    }
    return '';
};

const scrollLeftValue = ref(0);
const onTableScroll = (param: any) => {
    scrollLeftValue.value = param.scrollLeft;
};
/**
 * 激活表格，恢复滚动位置，否则会造成表头与数据单元格错位(暂不知为啥，先这样解决)
 */
const active = () => {
    setTimeout(() => tableRef.value.scrollToLeft(scrollLeftValue.value));
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
    .table {
        border-left: var(--el-table-border);
        border-top: var(--el-table-border);
    }

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
    }
}
</style>
