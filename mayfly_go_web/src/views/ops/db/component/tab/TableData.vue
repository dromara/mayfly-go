<template>
    <div>
        <el-row>
            <el-col :span="8">
                <el-link @click="onRefresh()" icon="refresh" :underline="false" class="ml5"> </el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-popover
                    popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                    placement="bottom"
                    width="auto"
                    title="表格字段配置"
                    trigger="click"
                >
                    <div v-for="(item, index) in columns" :key="index">
                        <el-checkbox
                            v-model="item.show"
                            :label="`${!item.columnComment ? item.columnName : item.columnName + ' [' + item.columnComment + ']'}`"
                            :true-label="true"
                            :false-label="false"
                            size="small"
                        />
                    </div>
                    <template #reference>
                        <el-link icon="Operation" size="small" :underline="false"></el-link>
                    </template>
                </el-popover>
                <el-divider direction="vertical" border-style="dashed" />

                <el-link @click="onShowAddDataDialog()" type="primary" icon="plus" :underline="false"></el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-link @click="onDeleteData()" type="danger" icon="delete" :underline="false"></el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                    <el-link @click="onCommit()" type="success" icon="CircleCheck" :underline="false"> </el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip class="box-item" effect="dark" content="生成insert sql" placement="top">
                    <el-link @click="onGenerateInsertSql()" type="success" :underline="false">gi</el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip class="box-item" effect="dark" content="导出当前页的csv文件" placement="top">
                    <el-link type="success" :underline="false" @click="exportData"><span class="f12">导出</span></el-link>
                </el-tooltip>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip v-if="hasUpdatedFileds" class="box-item" effect="dark" content="提交修改" placement="top">
                    <el-link @click="submitUpdateFields()" type="success" :underline="false" class="f12">提交</el-link>
                </el-tooltip>
                <el-divider v-if="hasUpdatedFileds" direction="vertical" border-style="dashed" />
                <el-tooltip v-if="hasUpdatedFileds" class="box-item" effect="dark" content="取消修改" placement="top">
                    <el-link @click="cancelUpdateFields" type="warning" :underline="false" class="f12">取消</el-link>
                </el-tooltip>
            </el-col>
            <el-col :span="16">
                <el-input
                    v-model="condition"
                    placeholder="若需条件过滤，可选择列并点击对应的字段并输入需要过滤的内容点击查询按钮即可"
                    clearable
                    @clear="selectData"
                    size="small"
                    style="width: 100%"
                >
                    <template #prepend>
                        <el-popover trigger="click" :width="320" placement="right">
                            <template #reference>
                                <el-link type="success" :underline="false">选择列</el-link>
                            </template>
                            <el-table
                                :data="columns"
                                max-height="500"
                                size="small"
                                @row-click="(...event: any) => {
                                onConditionRowClick(event);
                            }
                                "
                                style="cursor: pointer"
                            >
                                <el-table-column property="columnName" label="列名" show-overflow-tooltip> </el-table-column>
                                <el-table-column property="columnComment" label="备注" show-overflow-tooltip> </el-table-column>
                            </el-table>
                        </el-popover>
                    </template>

                    <template #append>
                        <el-button @click="onSelectByCondition()" icon="search" size="small"></el-button>
                    </template>
                </el-input>
            </el-col>
        </el-row>

        <db-table
            ref="dbTableRef"
            :db-id="state.ti.dbId"
            :db="state.ti.db"
            :data="datas"
            :table="state.table"
            :columns="columns"
            :loading="loading"
            :height="tableHeight"
            :show-column-tip="true"
            :sortable="'custom'"
            @sort-change="(sort: any) => onTableSortChange(sort)"
            @selection-change="onDataSelectionChange"
            @change-updated-field="changeUpdatedField"
        ></db-table>

        <el-row type="flex" class="mt5" justify="center">
            <el-pagination
                small
                :total="count"
                @size-change="handleSizeChange"
                @current-change="pageChange()"
                layout="prev, pager, next, total, sizes, jumper"
                v-model:current-page="pageNum"
                v-model:page-size="pageSize"
                :page-sizes="pageSizes"
            ></el-pagination>
        </el-row>
        <div style="font-size: 12px; padding: 0 10px; color: #606266">
            <span>{{ state.sql }}</span>
        </div>

        <el-dialog v-model="conditionDialog.visible" :title="conditionDialog.title" width="420px">
            <el-row>
                <el-col :span="5">
                    <el-select v-model="conditionDialog.condition">
                        <el-option label="=" value="="> </el-option>
                        <el-option label="LIKE" value="LIKE"> </el-option>
                        <el-option label=">" value=">"> </el-option>
                        <el-option label=">=" value=">="> </el-option>
                        <el-option label="<" value="<"> </el-option>
                        <el-option label="<=" value="<="> </el-option>
                    </el-select>
                </el-col>
                <el-col :span="19">
                    <el-input v-model="conditionDialog.value" :placeholder="conditionDialog.placeholder" />
                </el-col>
            </el-row>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCancelCondition">取消</el-button>
                    <el-button type="primary" @click="onConfirmCondition">确定</el-button>
                </span>
            </template>
        </el-dialog>

        <el-dialog v-model="addDataDialog.visible" :title="addDataDialog.title" :destroy-on-close="true" width="600px">
            <el-form ref="dataForm" :model="addDataDialog.data" label-width="auto" size="small">
                <el-form-item
                    v-for="column in columns"
                    class="w100"
                    :prop="column.columnName"
                    :label="column.columnName"
                    :required="column.nullable != 'YES' && column.columnKey != 'PRI'"
                >
                    <el-input-number
                        v-if="DbInst.isNumber(column.columnType)"
                        v-model="addDataDialog.data[`${column.columnName}`]"
                        :placeholder="`${column.columnType}  ${column.columnComment}`"
                        class="w100"
                    />

                    <el-input v-else v-model="addDataDialog.data[`${column.columnName}`]" :placeholder="`${column.columnType}  ${column.columnComment}`" />
                </el-form-item>
            </el-form>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="closeAddDataDialog">取消</el-button>
                    <el-button type="primary" @click="addRow">确定</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, watch, reactive, toRefs, ref, Ref } from 'vue';
import { isTrue, notEmpty, notBlank } from '@/common/assert';
import { ElMessage } from 'element-plus';

import { DbInst, TabInfo } from '../../db';
import { exportCsv } from '@/common/utils/export';
import { dateStrFormat } from '@/common/utils/date';
import DbTable from '../DbTable.vue';

const emits = defineEmits(['genInsertSql']);
const dataForm: any = ref(null);

const props = defineProps({
    data: {
        type: TabInfo,
        required: true,
    },
    tableHeight: {
        type: [String],
        default: '600',
    },
});

const dbTableRef = ref(null) as Ref;

const state = reactive({
    ti: {} as TabInfo,
    table: '', // 当前的表名
    datas: [],
    sql: '', // 当前数据tab执行的sql
    orderBy: '',
    condition: '', // 当前条件框的条件
    loading: false, // 是否在加载数据
    columns: [] as any,
    pageNum: 1,
    pageSize: DbInst.DefaultLimit,
    pageSizes: [20, 40, 80, 100, 200, 300, 400],
    count: 0,
    selectionDatas: [] as any,
    conditionDialog: {
        title: '',
        placeholder: '',
        columnRow: null,
        dataTab: null,
        visible: false,
        condition: '=',
        value: null,
    },
    addDataDialog: {
        data: {},
        title: '',
        placeholder: '',
        visible: false,
    },
    tableHeight: '600',
    hasUpdatedFileds: false,
});

const { datas, condition, loading, columns, pageNum, pageSize, pageSizes, count, hasUpdatedFileds, conditionDialog, addDataDialog } = toRefs(state);

watch(
    () => props.tableHeight,
    (newValue: any) => {
        state.tableHeight = newValue;
    }
);

onMounted(async () => {
    console.log('in table data mounted');
    state.ti = props.data;
    state.tableHeight = props.tableHeight;
    state.table = state.ti.params.table;
    notBlank(state.table, 'TableData组件params.table信息不能为空');

    const columns = await state.ti.getNowDbInst().loadColumns(state.ti.db, state.table);
    columns.forEach((x: any) => {
        x.show = true;
    });
    state.columns = columns;
    await onRefresh();
});

const onRefresh = async () => {
    // 查询条件置空
    state.condition = '';
    state.pageNum = 1;
    await selectData();
};

/**
 * 数据tab修改页数
 */
const pageChange = async () => {
    await selectData();
};

/**
 * 单表数据信息查询数据
 */
const selectData = async () => {
    state.loading = true;
    const dbInst = state.ti.getNowDbInst();
    const { db } = state.ti;
    try {
        const countRes = await dbInst.runSql(db, DbInst.getDefaultCountSql(state.table, state.condition));
        state.count = countRes.res[0].count;
        let sql = dbInst.getDefaultSelectSql(state.table, state.condition, state.orderBy, state.pageNum, state.pageSize);
        state.sql = sql;
        if (state.count > 0) {
            const colAndData: any = await dbInst.runSql(db, sql);
            state.datas = colAndData.res;
        } else {
            state.datas = [];
        }
    } finally {
        state.loading = false;
    }
};

const handleSizeChange = async (size: any) => {
    state.pageNum = 1;
    state.pageSize = size;
    await selectData();
};

/**
 * 导出当前页数据
 */
const exportData = () => {
    const dataList = state.datas as any;
    isTrue(dataList.length > 0, '没有数据可导出');
    let columnNames = [];
    for (let column of state.columns) {
        if (column.show) {
            columnNames.push(column.columnName);
        }
    }
    exportCsv(`数据导出-${state.table}-${dateStrFormat('yyyyMMddHHmm', new Date().toString())}`, columnNames, dataList);
};

/**
 * 条件查询，点击列信息后显示输入对应的值
 */
const onConditionRowClick = (event: any) => {
    const row = event[0];
    state.conditionDialog.title = `请输入 [${row.columnName}] 的值`;
    state.conditionDialog.placeholder = `${row.columnType}  ${row.columnComment}`;
    state.conditionDialog.columnRow = row;
    state.conditionDialog.visible = true;
};

// 确认条件
const onConfirmCondition = () => {
    const conditionDialog = state.conditionDialog;
    let condition = state.condition;
    if (condition) {
        condition += ` AND `;
    }
    const row = conditionDialog.columnRow as any;
    condition += `${row.columnName} ${conditionDialog.condition} `;
    state.condition = condition + DbInst.wrapColumnValue(row.columnType, conditionDialog.value);
    onCancelCondition();
};

const onCancelCondition = () => {
    state.conditionDialog.visible = false;
    state.conditionDialog.title = ``;
    state.conditionDialog.placeholder = ``;
    state.conditionDialog.value = null;
    state.conditionDialog.columnRow = null;
    state.conditionDialog.dataTab = null;
};

/**
 * 提交事务，用于没有开启自动提交事务
 */
const onCommit = () => {
    state.ti.getNowDbInst().runSql(state.ti.db, 'COMMIT;');
    ElMessage.success('COMMIT success');
};

const onSelectByCondition = async () => {
    notEmpty(state.condition, '条件不能为空');
    state.pageNum = 1;
    await selectData();
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: any) => {
    if (!sort.prop) {
        return;
    }
    const sortType = sort.order == 'descending' ? 'DESC' : 'ASC';
    state.orderBy = `ORDER BY ${sort.prop} ${sortType}`;
    await onRefresh();
};

const onDataSelectionChange = (datas: []) => {
    state.selectionDatas = datas;
};

const changeUpdatedField = (updatedFields: []) => {
    // 如果存在要更新字段，则显示提交和取消按钮
    state.hasUpdatedFileds = updatedFields && updatedFields.length > 0;
};

/**
 * 执行删除数据事件
 */
const onDeleteData = async () => {
    const deleteDatas = state.selectionDatas;
    isTrue(deleteDatas && deleteDatas.length > 0, '请先选择要删除的数据');
    const { db } = state.ti;
    const dbInst = state.ti.getNowDbInst();
    dbInst.promptExeSql(db, dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas), null, () => {
        onRefresh();
    });
};

const onGenerateInsertSql = async () => {
    isTrue(state.selectionDatas && state.selectionDatas.length > 0, '请先选择数据');
    emits('genInsertSql', state.ti.getNowDbInst().genInsertSql(state.ti.db, state.table, state.selectionDatas));
};

const submitUpdateFields = () => {
    dbTableRef.value.submitUpdateFields();
};

const cancelUpdateFields = () => {
    dbTableRef.value.cancelUpdateFields();
};

const onShowAddDataDialog = async () => {
    state.addDataDialog.title = `添加'${state.table}'表数据`;
    state.addDataDialog.visible = true;
};

const closeAddDataDialog = () => {
    state.addDataDialog.visible = false;
    state.addDataDialog.data = {};
};

// 添加新数据行
const addRow = async () => {
    dataForm.value.validate(async (valid: boolean) => {
        if (valid) {
            const data = state.addDataDialog.data;
            // key: 字段名，value: 字段名提示
            let obj: any = {};
            for (let item of state.columns) {
                const value = data[item.columnName];
                if (!value) {
                    continue;
                }
                obj[`${item.columnName}`] = DbInst.wrapValueByType(value);
            }
            let columnNames = Object.keys(obj).join(',');
            let values = Object.values(obj).join(',');
            let sql = `INSERT INTO ${state.table} (${columnNames}) VALUES (${values});`;
            state.ti.getNowDbInst().promptExeSql(state.ti.db, sql, null, () => {
                closeAddDataDialog();
                onRefresh();
            });
        } else {
            ElMessage.error('请正确填写数据信息');
            return false;
        }
    });
};
</script>

<style lang="scss">
.update_field_active {
    background-color: var(--el-color-success);
}
</style>
