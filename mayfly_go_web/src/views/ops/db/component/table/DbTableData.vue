<template>
    <div class="db-table-data mt5" :style="{ height: `${tableHeight}px` }">
        <el-auto-resizer>
            <template #default="{ height, width }">
                <el-table-v2
                    ref="tableRef"
                    :header-height="32"
                    :row-height="32"
                    :row-class="rowClass"
                    :columns="state.columns"
                    :data="datas"
                    :width="width"
                    :height="height"
                    fixed
                    :row-event-handlers="rowEventHandlers"
                >
                    <template #header="{ columns }">
                        <div v-for="(column, i) in columns" :key="i">
                            <div
                                :style="{
                                    width: `${column.width}px`,
                                    height: '100%',
                                    lineHeight: '32px',
                                    textAlign: 'center',
                                    borderRight: 'var(--el-table-border)',
                                }"
                            >
                                <!-- 行号列表头 -->
                                <div v-if="column.key == rowNoColumn.key || !showColumnTip">
                                    <el-text tag="b"> {{ column.title }} </el-text>
                                </div>

                                <div v-else @contextmenu="headerContextmenuClick($event, column)">
                                    <div v-if="showColumnTip" @mouseover="column.showSetting = true" @mouseleave="column.showSetting = false">
                                        <el-tooltip :show-after="500" raw-content placement="top">
                                            <template #content> {{ getColumnTip(column) }} </template>
                                            <el-text tag="b" style="cursor: pointer"> {{ column.title }} </el-text>
                                        </el-tooltip>

                                        <span>
                                            <SvgIcon
                                                color="var(--el-color-primary)"
                                                v-if="column.title == nowSortColumn?.columnName"
                                                :name="nowSortColumn?.order == 'asc' ? 'top' : 'bottom'"
                                            ></SvgIcon>
                                        </span>
                                    </div>

                                    <div v-else>
                                        <el-text tag="b" style="cursor: pointer"> {{ column.title }} </el-text>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </template>

                    <template #cell="{ rowData, column, rowIndex, columnIndex }">
                        <div style="width: 100%; height: 100%; line-height: 32px">
                            <!-- 行号列 -->
                            <div v-if="column.key == 'tableDataRowNo'">
                                <el-text tag="b" size="small">
                                    {{ rowIndex + 1 }}
                                </el-text>
                            </div>

                            <!-- 数据列 -->
                            <div v-else @dblclick="onEnterEditMode($event.target, rowData, column, rowIndex, columnIndex)">
                                <div v-if="canEdit(rowIndex, columnIndex)">
                                    <el-input
                                        :ref="(el: any) => el?.focus()"
                                        @blur="onExitEditMode(rowData, column, rowIndex)"
                                        class="w100"
                                        input-style="text-align: center"
                                        size="small"
                                        v-model="rowData[column.dataKey!]"
                                    ></el-input>
                                </div>

                                <div v-else :class="isUpdated(rowIndex, column.dataKey) ? 'update_field_active' : ''">
                                    <el-text :title="rowData[column.dataKey!]" size="small" truncated>
                                        {{ rowData[column.dataKey!] }}
                                    </el-text>
                                </div>
                            </div>
                        </div>
                    </template>

                    <template v-if="loading" #overlay>
                        <div class="el-loading-mask" style="display: flex; align-items: center; justify-content: center">
                            <SvgIcon name="loading" color="var(--el-color-primary)" :size="26" />
                        </div>
                    </template>

                    <template #empty>
                        <div style="text-align: center">
                            <el-empty :description="state.emptyText" :image-size="100" />
                        </div>
                    </template>
                </el-table-v2>
            </template>
        </el-auto-resizer>

        <el-dialog @close="state.genSqlDialog.visible = false" v-model="state.genSqlDialog.visible" :title="state.genSqlDialog.title" width="1000px">
            <el-input v-model="state.genSqlDialog.sql" type="textarea" rows="20" />
        </el-dialog>

        <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="headerContextmenuRef" />
    </div>
</template>

<script lang="ts" setup>
import { ref, onMounted, watch, reactive, toRefs } from 'vue';
import { ElInput } from 'element-plus';
import { DbInst } from '@/views/ops/db/db';
import Contextmenu from '@/components/contextmenu/index.vue';
import { ContextmenuItem } from '@/components/contextmenu/index';
import SvgIcon from '@/components/svgIcon/index.vue';

const emits = defineEmits(['dataDelete', 'sortChange', 'deleteData', 'selectionChange', 'changeUpdatedField']);

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    dbType: {
        type: String,
        default: '',
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
    sortable: {
        type: [String, Boolean],
        default: false,
    },
    loading: {
        type: Boolean,
        default: false,
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
        type: Number,
        default: 600,
    },
});

const headerContextmenuRef = ref();
const tableRef = ref();

const cmHeaderAsc = new ContextmenuItem('asc', '升序').withIcon('top').withOnClick((data: any) => {
    onTableSortChange({ columnName: data.dataKey, order: 'asc' });
});

const cmHeaderDesc = new ContextmenuItem('desc', '降序').withIcon('bottom').withOnClick((data: any) => {
    onTableSortChange({ columnName: data.dataKey, order: 'desc' });
});

const cmDataDel = new ContextmenuItem('desc', '删除')
    .withIcon('delete')
    .withOnClick(() => onDeleteData())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenInsertSql = new ContextmenuItem('genInsertSql', 'Insert SQL')
    .withIcon('document')
    .withOnClick(() => onGenerateInsertSql())
    .withHideFunc(() => {
        return state.table == '';
    });

const cmDataGenJson = new ContextmenuItem('genJson', '生成JSON').withIcon('document').withOnClick(() => onGenerateJson());

class NowUpdateCell {
    rowIndex: number;
    colIndex: number;
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

let nowSortColumn = null as any;

// 当前正在更新的单元格
let nowUpdateCell: NowUpdateCell = null as any;

// 选中的数据， key->rowIndex  value->primaryKeyValue
const selectionRowsMap: Map<number, any> = new Map();

// 更新单元格  key-> rowIndex  value -> 更新行
const cellUpdateMap: Map<number, UpdatedRow> = new Map();

const state = reactive({
    dbId: 0, // 当前选中操作的数据库实例
    dbType: '',
    db: '', // 数据库名
    table: '', // 当前的表名
    datas: [],
    columns: [] as any,
    sortable: false,
    loading: false,
    showColumnTip: false,
    tableHeight: 600,
    emptyText: '',

    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [] as ContextmenuItem[],
    },

    genSqlDialog: {
        title: 'SQL',
        visible: false,
        sql: '',
    },
});

const { tableHeight, datas } = toRefs(state);

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
    headerClass: 'table-data-cell',
    class: 'table-data-cell',
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
        setTableColumns(newValue);
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
    }
);

onMounted(async () => {
    console.log('in DbTable mounted');
    state.tableHeight = props.height;
    state.sortable = props.sortable as any;
    state.loading = props.loading;
    state.showColumnTip = props.showColumnTip;
    state.emptyText = props.emptyText;

    state.dbId = props.dbId;
    state.dbType = props.dbType;
    state.db = props.db;
    state.table = props.table;
});

const setTableData = (datas: any) => {
    console.log('set table datas', props);
    tableRef.value.scrollTo({ scrollLeft: 0, scrollTop: 0 });
    selectionRowsMap.clear();
    cellUpdateMap.clear();
    state.datas = datas;
    setTableColumns(props.columns);
};

const setTableColumns = (columns: any) => {
    state.columns = columns.map((x: any) => {
        const columnName = x.columnName;
        return {
            ...x,
            key: columnName,
            dataKey: columnName,
            width: DbInst.flexColumnWidth(columnName, state.datas),
            title: columnName,
            align: 'center',
            headerClass: 'table-data-cell',
            class: 'table-data-cell',
            sortable: true,
            hidden: !x.show,
        };
    });
    state.columns.unshift(rowNoColumn);
};

/**
 * 当前单元格是否允许编辑
 * @param rowIndex ri
 * @param colIndex ci
 */
const canEdit = (rowIndex: number, colIndex: number) => {
    return state.table && nowUpdateCell && nowUpdateCell.rowIndex == rowIndex && nowUpdateCell.colIndex == colIndex;
};

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
    onContextmenu: (e: any) => {
        dataContextmenuClick(e.event, e.rowIndex, e.rowData);
    },
};

const headerContextmenuClick = (event: any, data: any) => {
    event.preventDefault(); // 阻止默认的右击菜单行为

    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmHeaderAsc, cmHeaderDesc];
    headerContextmenuRef.value.openContextmenu(data);
};

const dataContextmenuClick = (event: any, rowIndex: number, data: any) => {
    event.preventDefault(); // 阻止默认的右击菜单行为

    // 当前行未选中，则单行选中该行
    if (!isSelection(rowIndex)) {
        selectionRow(rowIndex, data);
    }
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    state.contextmenu.items = [cmDataDel, cmDataGenInsertSql, cmDataGenJson];
    headerContextmenuRef.value.openContextmenu(data);
};

const onEnterEditMode = (el: any, rowData: any, column: any, rowIndex = 0, columnIndex = 0) => {
    if (!state.table) {
        return;
    }

    triggerRefresh();

    const oldVal = rowData[column.dataKey];
    nowUpdateCell = {
        rowIndex: rowIndex,
        colIndex: columnIndex,
        oldValue: oldVal,
    };
};

const onExitEditMode = (rowData: any, column: any, rowIndex = 0) => {
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

const rowClass = (row: any) => {
    if (isSelection(row.rowIndex)) {
        return 'data-selection';
    }
    if (row.rowIndex % 2 != 0) {
        return 'data-spacing';
    }
    return '';
};

const getColumnTip = (column: any) => {
    const comment = column.columnComment;
    return `${column.columnType} ${comment ? ' |  ' + comment : ''}`;
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

const onGenerateInsertSql = async () => {
    const selectionDatas = Array.from(selectionRowsMap.values());
    state.genSqlDialog.sql = await getNowDbInst().genInsertSql(state.db, state.table, selectionDatas);
    state.genSqlDialog.title = 'SQL';
    state.genSqlDialog.visible = true;
};

const onGenerateJson = async () => {
    const selectionDatas = Array.from(selectionRowsMap.values());
    // 按列字段重新排序对象key
    const jsonObj = [];
    for (let selectionData of selectionDatas) {
        let obj = {};
        for (let column of state.columns) {
            obj[column.title] = selectionData[column.dataKey];
        }
        jsonObj.push(obj);
    }
    state.genSqlDialog.sql = JSON.stringify(jsonObj, null, 4);
    state.genSqlDialog.title = 'JSON';
    state.genSqlDialog.visible = true;
};

const submitUpdateFields = async () => {
    const dbInst = getNowDbInst();
    if (cellUpdateMap.size == 0) {
        return;
    }

    const db = state.db;
    let res = '';

    for (let updateRow of cellUpdateMap.values()) {
        let sql = `UPDATE ${dbInst.wrapName(state.table)} SET `;
        const rowData = updateRow.rowData;
        // 主键列信息
        const primaryKey = await dbInst.loadTableColumn(db, state.table);
        let primaryKeyType = primaryKey.columnType;
        let primaryKeyName = primaryKey.columnName;
        let primaryKeyValue = rowData[primaryKeyName];

        for (let k of updateRow.columnsMap.keys()) {
            const v = updateRow.columnsMap.get(k);
            if (!v) {
                continue;
            }
            // 更新字段列信息
            const updateColumn = await dbInst.loadTableColumn(db, state.table, k);
            sql += ` ${dbInst.wrapName(k)} = ${DbInst.wrapColumnValue(updateColumn.columnType, rowData[k])},`;

            // 如果修改的字段是主键
            if (k === primaryKeyName) {
                primaryKeyValue = v.oldValue;
            }
        }

        sql = sql.substring(0, sql.length - 1);
        sql += ` WHERE ${dbInst.wrapName(primaryKeyName)} = ${DbInst.wrapColumnValue(primaryKeyType, primaryKeyValue)} ;`;
        res += sql;
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

const getNowDbInst = () => {
    return DbInst.getInst(state.dbId);
};

defineExpose({
    submitUpdateFields,
    cancelUpdateFields,
});
</script>

<style>
.db-table-data {
    .table-data-cell {
        padding: 0 2px;
        font-size: 12px;
        border-right: var(--el-table-border);
    }
    .data-selection {
        background-color: var(--el-color-success-light-8);
    }
    .data-spacing {
        background-color: var(--el-fill-color-lighter);
    }

    .update_field_active {
        background-color: var(--el-color-success);
    }
}
</style>
