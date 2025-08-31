<template>
    <div>
        <el-row>
            <el-col :span="8">
                <div class="mt-1">
                    <el-link :disabled="state.loading" @click="onRefresh()" icon="refresh" underline="never" class="ml-1"> </el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-popover
                        popper-style="max-height: 550px; overflow: auto; max-width: 450px"
                        placement="bottom"
                        width="auto"
                        :title="$t('db.tableFieldConf')"
                        trigger="click"
                        @hide="triggerCheckedColumns"
                    >
                        <div><el-input v-model="checkedShowColumns.searchKey" size="small" :placeholder="$t('db.columnFilterPlaceholder')" /></div>
                        <div>
                            <el-checkbox
                                v-model="checkedShowColumns.checkedAllColumn"
                                :indeterminate="checkedShowColumns.isIndeterminate"
                                @change="handleCheckAllColumnChange"
                                size="small"
                            >
                                {{ $t('db.selectAll') }}
                            </el-checkbox>

                            <el-checkbox-group v-model="checkedShowColumns.columnNames" @change="handleCheckedColumnChange">
                                <div v-for="(item, index) in filterCheckedColumns" :key="index">
                                    <el-checkbox
                                        :key="index"
                                        :label="`${!item.columnComment ? item.columnName : item.columnName + ' [' + item.columnComment + ']'}`"
                                        :value="item.columnName"
                                        size="small"
                                    />
                                </div>
                            </el-checkbox-group>
                        </div>
                        <template #reference>
                            <el-link icon="Operation" size="small" underline="never"></el-link>
                        </template>
                    </el-popover>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-link @click="onShowAddDataDialog()" type="primary" icon="plus" underline="never"></el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" effect="dark" content="commit" placement="top">
                        <el-link @click="onCommit()" type="success" icon="CircleCheck" underline="never"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" v-if="hasUpdatedFileds" :content="$t('db.submitUpdate')" placement="top">
                        <el-link @click="submitUpdateFields()" type="success" underline="never" class="!text-[12px]">{{ $t('common.submit') }}</el-link>
                    </el-tooltip>
                    <el-divider v-if="hasUpdatedFileds" direction="vertical" border-style="dashed" />
                    <el-tooltip :show-after="500" v-if="hasUpdatedFileds" :content="$t('db.cancelUpdate')" placement="top">
                        <el-link @click="cancelUpdateFields" type="warning" underline="never" class="!text-[12px]">{{ $t('common.cancel') }}</el-link>
                    </el-tooltip>
                </div>
            </el-col>
            <el-col :span="16">
                <el-autocomplete
                    v-model="condition"
                    :fetch-suggestions="getColumnTips"
                    @keyup.enter.native="onSelectByCondition"
                    @select="handlerColumnSelect"
                    popper-class="my-autocomplete"
                    :placeholder="$t('db.autoCompleteColumnPlaceholder')"
                    @clear="selectData"
                    size="small"
                    clearable
                    class="!w-full"
                    highlight-first-item
                    value-key="columnName"
                    ref="condInputRef"
                >
                    <template #suffix>
                        <SvgIcon @click="onSelectByCondition" name="search" />
                    </template>

                    <template #default="{ item }">
                        <el-text tag="b"> {{ item.columnName }}</el-text>

                        <el-divider direction="vertical" />

                        <span style="color: var(--el-color-info-light-3)">
                            {{ item.columnType }}

                            <template v-if="item.columnComment">
                                <el-divider direction="vertical" />
                                {{ item.columnComment }}
                            </template>
                        </span>
                    </template>

                    <template #prepend>
                        <el-popover :visible="state.condPopVisible" trigger="click" :width="320" placement="right">
                            <template #reference>
                                <el-button @click.stop="chooseCondColumnName" style="color: var(--el-color-success)" text size="small">
                                    {{ $t('db.selectColumn') }}
                                </el-button>
                            </template>
                            <el-table
                                :data="filterCondColumns"
                                max-height="500"
                                size="small"
                                @row-click="
                                    (...event: any) => {
                                        onConditionRowClick(event);
                                    }
                                "
                                class="cursor-pointer"
                            >
                                <el-table-column property="columnName" :label="$t('db.columnName')" show-overflow-tooltip>
                                    <template #header>
                                        <el-input
                                            ref="columnNameSearchInputRef"
                                            v-model="state.columnNameSearch"
                                            size="small"
                                            :placeholder="$t('db.columnFilterPlaceholder')"
                                            @click.stop="(e: any) => e.preventDefault()"
                                        />
                                    </template>
                                </el-table-column>
                                <el-table-column property="columnComment" :label="$t('common.remark')" show-overflow-tooltip> </el-table-column>
                            </el-table>
                        </el-popover>
                    </template>
                </el-autocomplete>
            </el-col>
        </el-row>

        <db-table-data
            ref="dbTableRef"
            :db-id="dbId"
            :db="dbName"
            :data="datas"
            :table="tableName"
            :columns="columns"
            :loading="loading"
            :height="tableHeight"
            :show-column-tip="true"
            @sort-change="(sort: any) => onTableSortChange(sort)"
            @selection-change="onDataSelectionChange"
            @change-updated-field="changeUpdatedField"
            @data-delete="onRefresh"
        ></db-table-data>

        <el-row type="flex" class="mt-2" :gutter="10" justify="space-between" style="user-select: none">
            <el-col :span="12">
                <el-text
                    id="copyValue"
                    style="color: var(--el-color-info-light-3)"
                    class="is-truncated !text-[12px] mt-1"
                    @click="copyToClipboard(sql)"
                    :title="sql"
                    >{{ sql }}</el-text
                >
            </el-col>
            <el-col :span="12">
                <el-row :gutter="10" justify="start">
                    <el-link class="op-page" underline="never" @click="pageNum = 1" :disabled="pageNum == 1" icon="DArrowLeft" :title="$t('db.homePage')" />
                    <el-link
                        class="op-page"
                        underline="never"
                        @click="pageNum = --pageNum || 1"
                        :disabled="pageNum == 1"
                        icon="Back"
                        :title="$t('db.previousPage')"
                    />
                    <div class="op-page">
                        <el-input-number
                            style="width: 50px"
                            :controls="false"
                            :min="1"
                            v-model="state.setPageNum"
                            size="small"
                            @blur="handleSetPageNum"
                            @keydown.enter="handleSetPageNum"
                        />
                    </div>
                    <el-link class="op-page" underline="never" @click="++pageNum" :disabled="datas.length < pageSize" icon="Right" />
                    <el-link class="op-page" underline="never" @click="handleEndPage" :disabled="datas.length < pageSize" icon="DArrowRight" />
                    <div style="width: 90px" class="op-page ml-2">
                        <el-select size="small" :default-first-option="true" v-model="pageSize" @change="handleSizeChange">
                            <el-option
                                style="font-size: 12px; height: 24px; line-height: 24px"
                                v-for="(op, i) in pageSizes"
                                :key="i"
                                :label="op + $t('db.rowsPage')"
                                :value="op"
                            />
                        </el-select>
                    </div>

                    <el-button @click="handleCount" :loading="state.counting" class="ml-2" text bg size="small">
                        {{ state.showTotal ? `${state.total} ${$t('db.rows')}` : 'count' }}
                    </el-button>
                </el-row>
            </el-col>
        </el-row>

        <el-dialog v-model="conditionDialog.visible" :title="conditionDialog.title" width="500px">
            <el-row gutter="5">
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
                    <el-input
                        @keyup.enter.native="onConfirmCondition"
                        ref="condDialogInputRef"
                        v-model="conditionDialog.value"
                        :placeholder="conditionDialog.placeholder"
                    />
                </el-col>
            </el-row>
            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="onCancelCondition">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" @click="onConfirmCondition">{{ $t('common.confirm') }}</el-button>
                </span>
            </template>
        </el-dialog>

        <DbTableDataForm
            :db-inst="getNowDbInst()"
            :db-name="dbName"
            :columns="columns"
            :title="addDataDialog.title"
            :table-name="tableName"
            v-model:visible="addDataDialog.visible"
            v-model="addDataDialog.data"
            @submit-success="onRefresh"
        />
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, Ref, ref, toRefs, watch } from 'vue';
import { ElMessage } from 'element-plus';

import { DbInst } from '@/views/ops/db/db';
import DbTableData from './DbTableData.vue';
import { DbDialect } from '@/views/ops/db/dialect';
import SvgIcon from '@/components/svgIcon/index.vue';
import { useEventListener } from '@vueuse/core';
import { copyToClipboard, fuzzyMatchField } from '@/common/utils/string';
import DbTableDataForm from './DbTableDataForm.vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    dbName: {
        type: String,
        required: true,
    },
    tableName: {
        type: String,
        required: true,
    },
    tableHeight: {
        type: [String],
        default: '600px',
    },
});

const dbTableRef: Ref = ref(null);
const condInputRef: Ref = ref(null);
const columnNameSearchInputRef: Ref = ref(null);
const condDialogInputRef: Ref = ref(null);

const defaultPageSize = DbInst.DefaultLimit;

const state = reactive({
    datas: [],
    sql: '', // 当前数据tab执行的sql
    orderBy: '',
    condition: '', // 当前条件框的条件
    loading: false, // 是否在加载数据
    columns: [] as any,
    pageNum: 1,
    pageSize: defaultPageSize,
    pageSizes: [
        defaultPageSize,
        defaultPageSize * 2,
        defaultPageSize * 4,
        defaultPageSize * 8,
        defaultPageSize * 20,
        defaultPageSize * 40,
        defaultPageSize * 80,
    ],
    setPageNum: 0,
    total: 0,
    showTotal: false,
    counting: false,
    selectionDatas: [] as any,
    condPopVisible: false,
    columnNameSearch: '',
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
        visible: false,
    },
    tableHeight: '600px',
    hasUpdatedFileds: false,
    dbDialect: {} as DbDialect,

    checkedShowColumns: {
        searchKey: '',
        checkedAllColumn: true,
        isIndeterminate: false,
        columnNames: [] as any,
    },
});

const { datas, condition, loading, columns, checkedShowColumns, pageNum, pageSize, pageSizes, sql, hasUpdatedFileds, conditionDialog, addDataDialog } =
    toRefs(state);

watch(
    () => props.tableHeight,
    (newValue: any) => {
        state.tableHeight = newValue;
    }
);

const getNowDbInst = () => {
    return DbInst.getInst(props.dbId);
};

onMounted(async () => {
    console.log('in table data mounted');
    state.tableHeight = props.tableHeight;
    await onRefresh();

    state.dbDialect = getNowDbInst().getDialect();
    useEventListener('click', handlerWindowClick);

    state.checkedShowColumns.columnNames = state.columns.map((item: any) => item.columnName);
});

const handlerWindowClick = () => {
    if (state.condPopVisible) {
        state.condPopVisible = false;
    }
};

const onRefresh = async () => {
    state.pageNum = 1;
    await selectData();
};

watch(
    () => state.pageNum,
    async () => {
        await selectData();
    }
);

/**
 * 单表数据信息查询数据
 */
const selectData = async () => {
    state.loading = true;
    state.setPageNum = state.pageNum;
    const dbInst = getNowDbInst();
    const db = props.dbName;
    const table = props.tableName;
    try {
        if (state.columns.length == 0) {
            const columns = await getNowDbInst().loadColumns(props.dbName, props.tableName);
            columns.forEach((x: any) => {
                x.show = true;
            });
            state.columns = columns;
        }

        let sql = dbInst.getDefaultSelectSql(db, table, state.condition, state.orderBy, state.pageNum, state.pageSize);
        state.sql = sql;
        const res: any = await dbInst.runSql(db, sql);
        const colAndData: any = res[0];
        state.datas = colAndData.res;
    } finally {
        state.loading = false;
    }
};

const handleSizeChange = async (size: any) => {
    state.pageNum = 1;
    state.pageSize = size;
    await selectData();
};

const handleEndPage = async () => {
    await handleCount();
    state.pageNum = Math.ceil(state.total / state.pageSize);
    await selectData();
};

const handleSetPageNum = async () => {
    state.pageNum = state.setPageNum;
    await selectData();
};

const handleCount = async () => {
    state.counting = true;

    try {
        const db = props.dbName;
        const table = props.tableName;
        const dbInst = getNowDbInst();
        let countRes = await dbInst.runSql(db, dbInst.getDefaultCountSql(table, state.condition));
        countRes = countRes[0];
        state.total = parseInt(countRes.res[0].count || countRes.res[0].COUNT || 0);
        state.showTotal = true;
    } catch (e) {
        /* empty */
    }

    state.counting = false;
};

const handleCheckAllColumnChange = (val: boolean) => {
    state.checkedShowColumns.columnNames = val ? state.columns.map((x: any) => x.columnName) : [];
    state.checkedShowColumns.isIndeterminate = false;
};

const handleCheckedColumnChange = (value: string[]) => {
    const checkedCount = value.length;
    state.checkedShowColumns.checkedAllColumn = checkedCount === state.columns.length;
    state.checkedShowColumns.isIndeterminate = checkedCount > 0 && checkedCount < state.columns.length;
};

const triggerCheckedColumns = () => {
    const checkedColumnNames = state.checkedShowColumns.columnNames;
    for (let column of state.columns) {
        column.show = checkedColumnNames.includes(column.columnName);
    }
};

// 完整的条件,每次选中后会重置条件框内容，故需要这个变量在获取建议时将文本框内容保存
let completeCond = '';
// 是否存在列建议
let existSuggestion = false;

const getColumnTips = (queryString: string, callback: any) => {
    const columns = state.columns;

    var words = queryString.split(' '); // 使用空格分割字符串为数组
    let columnNameSearch = words[words.length - 1]; // 获取最后一个元素

    let res = [];
    if (columnNameSearch) {
        res = fuzzyMatchField(columnNameSearch, columns, (x: any) => x.columnName);
    }

    completeCond = condition.value;
    callback(res);

    existSuggestion = res.length > 0;
};

const handlerColumnSelect = (column: any) => {
    // 获取最后一个空格的索引
    var lastSpaceIndex = completeCond.lastIndexOf(' ');

    // 默认拼接上 columnName =
    let value = column.columnName + ' = ';
    // 不是数字类型默认拼接上''
    if (!DbInst.isNumber(column.dataType)) {
        value = `${value} ''`;
    }

    if (lastSpaceIndex != -1) {
        // 获取最后一个空格之前的文本,拼上当前选中的建议列
        condition.value = `${completeCond.slice(0, lastSpaceIndex)} ${value}`;
    } else {
        condition.value = value;
    }
};

/**
 * 选择条件列
 */
const chooseCondColumnName = () => {
    state.condPopVisible = !state.condPopVisible;
    if (state.condPopVisible) {
        columnNameSearchInputRef.value.clear();
        columnNameSearchInputRef.value.focus();
    }
};

/**
 * 过滤条件列名
 */
const filterCondColumns = computed(() => {
    return filterColumns(state.columnNameSearch);
});

const filterCheckedColumns = computed(() => {
    return filterColumns(state.checkedShowColumns.searchKey);
});

const filterColumns = (searchKey: string) => {
    const columns = state.columns;
    if (!searchKey) {
        return columns;
    }
    return fuzzyMatchField(
        searchKey,
        columns,
        (x: any) => x.columnName,
        (x: any) => x.columnComment
    );
};

/**
 * 条件查询，点击列信息后显示输入对应的值
 */
const onConditionRowClick = (event: any) => {
    const row = event[0];
    state.conditionDialog.title = t('db.conditionInputDialogTitle', { columnName: row.columnName });
    state.conditionDialog.placeholder = `${row.columnType}  ${row.columnComment}`;
    state.conditionDialog.columnRow = row;
    state.conditionDialog.visible = true;
    setTimeout(() => {
        condDialogInputRef.value.focus();
    }, 100);
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
    state.condition = condition + state.dbDialect.wrapValue(row.dataType, conditionDialog.value!);
    onCancelCondition();
    condInputRef.value.focus();
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
    getNowDbInst().runSql(props.dbName, 'COMMIT;');
    ElMessage.success('COMMIT success');
};

const onSelectByCondition = async () => {
    if (!existSuggestion) {
        state.pageNum = 1;
        await selectData();
    }
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: any) => {
    const sortType = sort.order == 'desc' ? 'DESC' : 'ASC';
    state.orderBy = `ORDER BY ${state.dbDialect.quoteIdentifier(sort.columnName)} ${sortType}`;
    await onRefresh();
};

const onDataSelectionChange = (datas: []) => {
    state.selectionDatas = datas;
};

const changeUpdatedField = (updatedFields: any) => {
    // 如果存在要更新字段，则显示提交和取消按钮
    state.hasUpdatedFileds = updatedFields && updatedFields.size > 0;
};

const submitUpdateFields = () => {
    dbTableRef.value.submitUpdateFields();
};

const cancelUpdateFields = () => {
    dbTableRef.value.cancelUpdateFields();
};

const onShowAddDataDialog = async () => {
    state.addDataDialog.title = t('db.addDataDialogTitle', { tableName: props.tableName });
    state.addDataDialog.visible = true;
};

defineExpose({
    active: () => dbTableRef.value.active(),
});
</script>

<style lang="scss">
.op-page {
    margin-left: 5px;
}
</style>
