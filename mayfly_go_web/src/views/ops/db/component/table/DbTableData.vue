<template>
    <div class="db-table-data mt5" :style="{ height: tableHeight }">
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-table-v2
                    ref="tableRef"
                    :header-height="showColumnTip && dbConfig.showColumnComment ? 45 : 30"
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
                                            <SvgIcon :size="10" name="Clock" style="cursor: unset" />
                                        </span>
                                        <span class="font8" v-else>{{ column.dataTypeSubscript }}</span>
                                    </div>

                                    <div v-if="showColumnTip">
                                        <div class="header-column-title">
                                            <b :title="column.remark" class="el-text" style="cursor: pointer">
                                                {{ column.title }}
                                            </b>
                                        </div>

                                        <!-- 字段备注信息 -->
                                        <span
                                            v-if="dbConfig.showColumnComment"
                                            style="color: var(--el-color-info-light-3)"
                                            class="font10 el-text el-text--small is-truncated"
                                        >
                                            {{ column.columnComment }}
                                        </span>
                                    </div>

                                    <div v-else class="header-column-title">
                                        <b class="el-text">
                                            {{ column.title }}
                                        </b>
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

                                <div v-else :class="isUpdated(rowIndex, column.dataKey) ? 'update_field_active' : ''">
                                    <span v-if="rowData[column.dataKey!] === null" style="color: var(--el-color-info-light-5)"> NULL </span>

                                    <span v-else :title="rowData[column.dataKey!]" class="el-text el-text--small is-truncated">
                                        {{ rowData[column.dataKey!] }}
                                    </span>
                                </div>
                            </div>
                        </div>
                    </template>

                    <template v-if="state.loading" #overlay>
                        <div class="el-loading-mask" style="display: flex; flex-direction: column; align-items: center; justify-content: center">
                            <div>
                                <SvgIcon class="is-loading" name="loading" color="var(--el-color-primary)" :size="28" />
                                <el-text class="ml5" tag="b">执行时间 - {{ state.execTime.toFixed(1) }}s</el-text>
                            </div>
                            <div v-if="loading && abortFn" class="mt10">
                                <el-button @click="cancelLoading" type="info" size="small" plain>取 消</el-button>
                            </div>
                        </div>
                    </template>

                    <template #empty>
                        <div style="text-align: center">
                            <el-empty class="h100" :description="state.emptyText" :image-size="100" />
                        </div>
                    </template>
                </el-table-v2>
            </template>
        </el-auto-resizer>

        <el-dialog @close="state.genTxtDialog.visible = false" v-model="state.genTxtDialog.visible" :title="state.genTxtDialog.title" width="1000px">
            <template #header>
                <div class="mr15" style="display: flex; justify-content: flex-end">
                    <el-button id="copyValue" @click="copyGenTxt(state.genTxtDialog.txt)" icon="CopyDocument" type="success" size="small">一键复制</el-button>
                </div>
            </template>
            <el-input v-model="state.genTxtDialog.txt" type="textarea" rows="20" />
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
import { onBeforeUnmount, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { ElInput, ElMessage } from 'element-plus';
import { copyToClipboard } from '@/common/utils/string';
import { DbInst } from '@/views/ops/db/db';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import SvgIcon from '@/components/svgIcon/index.vue';
import { exportCsv, exportFile } from '@/common/utils/export';
import { dateStrFormat } from '@/common/utils/date';
import { useIntervalFn, useStorage } from '@vueuse/core';
import { ColumnTypeSubscript, compatibleMysql, DataType, DbDialect, getDbDialect } from '../../dialect/index';
import ColumnFormItem from './ColumnFormItem.vue';
import DbTableDataForm from './DbTableDataForm.vue';

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
        default: '暂无数据',
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

const cmHeaderAsc = new ContextmenuItem('asc', '升序')
    .withIcon('top')
    .withOnClick((data: any) => {
        onTableSortChange({ columnName: data.dataKey, order: 'asc' });
    })
    .withHideFunc(() => !props.showColumnTip);

const cmHeaderDesc = new ContextmenuItem('desc', '降序')
    .withIcon('bottom')
    .withOnClick((data: any) => {
        onTableSortChange({ columnName: data.dataKey, order: 'desc' });
    })
    .withHideFunc(() => !props.showColumnTip);

const cmHeaderFixed = new ContextmenuItem('fixed', '固定')
    .withIcon('Paperclip')
    .withOnClick((data: any) => {
        data.fixed = true;
    })
    .withHideFunc((data: any) => data.fixed);

const cmHeaderCancenFixed = new ContextmenuItem('cancelFixed', '取消固定')
    .withIcon('Minus')
    .withOnClick((data: any) => (data.fixed = false))
    .withHideFunc((data: any) => !data.fixed);

/**  表数据 contextmenu items  **/

const cmDataCopyCell = new ContextmenuItem('copyValue', '复制')
    .withIcon('CopyDocument')
    .withOnClick(async (data: any) => {
        await copyToClipboard(data.rowData[data.column.dataKey]);
    })
    .withHideFunc(() => {
        // 选中多条则隐藏该复制按钮
        return selectionRowsMap.size > 1;
    });

const cmDataDel = new ContextmenuItem('deleteData', '删除')
    .withIcon('delete')
    .withOnClick(() => onDeleteData())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataEdit = new ContextmenuItem('editData', '编辑行')
    .withIcon('edit')
    .withOnClick(() => onEditRowData())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenInsertSql = new ContextmenuItem('genInsertSql', 'Insert SQL')
    .withIcon('tickets')
    .withOnClick(() => onGenerateInsertSql())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenJson = new ContextmenuItem('genJson', '生成JSON').withIcon('tickets').withOnClick(() => onGenerateJson());

const cmDataExportCsv = new ContextmenuItem('exportCsv', '导出CSV').withIcon('document').withOnClick(() => onExportCsv());

const cmDataExportSql = new ContextmenuItem('exportSql', '导出SQL')
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
    columnsMap: Map<string, TableCellData> = new Map();
}

class TableCellData {
    /**
     * 旧值
     */
    oldValue: any;
}

let dbDialect: DbDialect = null as any;

let nowSortColumn = null as any;

// 当前正在更新的单元格
let nowUpdateCell: NowUpdateCell = null as any;

// 选中的数据， key->rowIndex  value->primaryKeyValue
const selectionRowsMap: Map<number, any> = new Map();

// 更新单元格  key-> rowIndex  value -> 更新行
const cellUpdateMap: Map<number, UpdatedRow> = new Map();

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
    emptyText: '',

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

const dbConfig = useStorage('dbConfig', { showColumnComment: false });

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
    state.emptyText = props.emptyText;

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

const formatDataValues = (datas: any) => {
    // mysql数据暂不做处理
    if (compatibleMysql(getNowDbInst().type)) {
        return;
    }

    for (let data of datas) {
        for (let column of props.columns!) {
            data[column.columnName] = getFormatTimeValue(dbDialect.getDataType(column.columnType), data[column.columnName]);
        }
    }
};

const setTableData = (datas: any) => {
    tableRef.value?.scrollTo({ scrollLeft: 0, scrollTop: 0 });
    selectionRowsMap.clear();
    cellUpdateMap.clear();
    formatDataValues(datas);
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
            align: 'center',
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
    return state.table && nowUpdateCell?.rowIndex == rowIndex && nowUpdateCell?.colIndex == colIndex;
};

/**
 * 判断当前单元格是否被更新了
 * @param rowIndex ri
 * @param columnName cn
 */
const isUpdated = (rowIndex: number, columnName: string) => {
    return cellUpdateMap.get(rowIndex)?.columnsMap.get(columnName);
};

/**
 * 判断当前行是否被选中
 * @param rowIndex
 */
const isSelection = (rowIndex: number): boolean => {
    return selectionRowsMap.get(rowIndex);
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
        if (selectionRowsMap.get(rowIndex)) {
            selectionRowsMap.delete(rowIndex);
            triggerRefresh();
            return;
        }
    } else {
        selectionRowsMap.clear();
    }
    selectionRowsMap.set(rowIndex, rowData);
    triggerRefresh();
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
    state.contextmenu.items = [cmHeaderAsc, cmHeaderDesc, cmHeaderFixed, cmHeaderCancenFixed];
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
    state.contextmenu.items = [cmDataCopyCell, cmDataDel, cmDataEdit, cmDataGenInsertSql, cmDataGenJson, cmDataExportCsv, cmDataExportSql];
    contextmenuRef.value.openContextmenu({ column, rowData: data });
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: any) => {
    nowSortColumn = sort;
    cancelUpdateFields();
    emits('sortChange', sort);
};

/**
 * 执行删除数据事件
 */
const onDeleteData = async () => {
    const deleteDatas = Array.from(selectionRowsMap.values());
    const db = state.db;
    const dbInst = getNowDbInst();
    dbInst.promptExeSql(db, await dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas as any), null, () => {
        emits('dataDelete', deleteDatas);
    });
};

const onEditRowData = () => {
    const selectionDatas = Array.from(selectionRowsMap.values());
    if (selectionDatas.length > 1) {
        ElMessage.warning('只能编辑一行数据');
        return;
    }
    const data = selectionDatas[0];
    state.tableDataFormDialog.data = data;
    state.tableDataFormDialog.title = `编辑表'${props.table}'数据`;
    state.tableDataFormDialog.visible = true;
};

const onGenerateInsertSql = async () => {
    const selectionDatas = Array.from(selectionRowsMap.values());
    state.genTxtDialog.txt = await getNowDbInst().genInsertSql(state.db, state.table, selectionDatas);
    state.genTxtDialog.title = 'SQL';
    state.genTxtDialog.visible = true;
};

const onGenerateJson = async () => {
    const selectionDatas = Array.from(selectionRowsMap.values());
    // 按列字段重新排序对象key
    const jsonObj = [];
    for (let selectionData of selectionDatas) {
        let obj = {};
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
    exportCsv(`数据导出-${state.table}-${dateStrFormat('yyyyMMddHHmm', new Date().toString())}`, columnNames, dataList);
};

const onExportSql = async () => {
    const selectionDatas = state.datas;
    exportFile(
        `数据导出-${state.table}-${dateStrFormat('yyyyMMddHHmm', new Date().toString())}.sql`,
        await getNowDbInst().genInsertSql(state.db, state.table, selectionDatas)
    );
};

const onEnterEditMode = (rowData: any, column: any, rowIndex = 0, columnIndex = 0) => {
    if (!state.table) {
        return;
    }

    triggerRefresh();
    nowUpdateCell = {
        rowIndex: rowIndex,
        colIndex: columnIndex,
        oldValue: rowData[column.dataKey],
        dataType: column.dataType,
    };
};

const onExitEditMode = (rowData: any, column: any, rowIndex = 0) => {
    if (!nowUpdateCell) {
        return;
    }
    const oldValue = nowUpdateCell.oldValue;
    const newValue = rowData[column.dataKey];

    // 未改变单元格值
    if (oldValue == newValue) {
        nowUpdateCell = null as any;
        triggerRefresh();
        return;
    }

    let updatedRow = cellUpdateMap.get(rowIndex);
    if (!updatedRow) {
        updatedRow = new UpdatedRow();
        updatedRow.rowData = rowData;
        cellUpdateMap.set(rowIndex, updatedRow);
    }

    const columnName = column.dataKey;
    let cellData = updatedRow.columnsMap.get(columnName);
    if (cellData) {
        // 多次修改情况，可能又修改回原值，则移除该修改单元格
        if (cellData.oldValue == newValue) {
            cellUpdateMap.delete(rowIndex);
        }
    } else {
        cellData = new TableCellData();
        cellData.oldValue = oldValue;
        updatedRow.columnsMap.set(columnName, cellData);
    }

    nowUpdateCell = null as any;
    triggerRefresh();
    changeUpdatedField();
};

const submitUpdateFields = async () => {
    const dbInst = getNowDbInst();
    if (cellUpdateMap.size == 0) {
        return;
    }

    const db = state.db;
    let res = '';

    for (let updateRow of cellUpdateMap.values()) {
        const rowData = { ...updateRow.rowData };
        let updateColumnValue = {};

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

    dbInst.promptExeSql(
        db,
        res,
        () => {},
        () => {
            triggerRefresh();
            cellUpdateMap.clear();
            changeUpdatedField();
        }
    );
};

const cancelUpdateFields = () => {
    const updateRows = cellUpdateMap.values();
    // 恢复原值
    for (let updateRow of updateRows) {
        const rowData = updateRow.rowData;
        updateRow.columnsMap.forEach((v: TableCellData, k: string) => {
            rowData[k] = v.oldValue;
        });
    }
    cellUpdateMap.clear();
    changeUpdatedField();
};

const changeUpdatedField = () => {
    emits('changeUpdatedField', cellUpdateMap);
};

const rowClass = (row: any) => {
    if (isSelection(row.rowIndex)) {
        return 'data-selection';
    }
    return '';
};

/**
 * 根据数据库返回的时间字段类型，获取格式化后的时间值
 * @param dataType getDataType返回的数据类型
 * @param originValue 原始值
 * @return 格式化后的值
 */
const getFormatTimeValue = (dataType: DataType, originValue: string): string => {
    if (!originValue || dataType === DataType.Number || dataType === DataType.String) {
        return originValue;
    }

    // 把Z去掉
    originValue = originValue.replace('Z', '');

    switch (dataType) {
        case DataType.Time:
            return dateStrFormat('HH:mm:ss', originValue);
        case DataType.Date:
            return dateStrFormat('yyyy-MM-dd', originValue);
        case DataType.DateTime:
            return dateStrFormat('yyyy-MM-dd HH:mm:ss', originValue);
        default:
            return originValue;
    }
};

/**
 * 触发响应式实时刷新，否则需要滑动或移动才能使样式实时生效
 */
const triggerRefresh = () => {
    // 改变columns等属性值，才能触发slot中的if条件等, 暂不知为啥
    if (state.columns[0].opTimes) {
        state.columns[0].opTimes = state.columns[0].opTimes + 1;
    } else {
        state.columns[0].opTimes = 1;
    }
};

const getNowDbInst = () => {
    return DbInst.getInst(state.dbId);
};

defineExpose({
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
        top: -5px;
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
