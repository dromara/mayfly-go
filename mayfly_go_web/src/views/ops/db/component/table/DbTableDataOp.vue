<template>
    <div>
        <el-row>
            <el-col :span="8">
                <div class="mt5">
                    <el-link :disabled="state.loading" @click="onRefresh()" icon="refresh" :underline="false" class="ml5"> </el-link>
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

                    <el-tooltip :show-after="500" class="box-item" effect="dark" content="commit" placement="top">
                        <el-link @click="onCommit()" type="success" icon="CircleCheck" :underline="false"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" class="box-item" effect="dark" content="commit" placement="top">
                        <template #content>
                            1. 右击数据/表头可显示操作菜单 <br />
                            2. 按住Ctrl点击数据则为多选 <br />
                            3. 双击单元格可编辑数据 <br />
                            4. 鼠标悬停字段名或标签树的表名可提示相关备注
                        </template>
                        <el-link icon="QuestionFilled" :underline="false"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" v-if="hasUpdatedFileds" class="box-item" effect="dark" content="提交修改" placement="top">
                        <el-link @click="submitUpdateFields()" type="success" :underline="false" class="font12">提交</el-link>
                    </el-tooltip>
                    <el-divider v-if="hasUpdatedFileds" direction="vertical" border-style="dashed" />
                    <el-tooltip :show-after="500" v-if="hasUpdatedFileds" class="box-item" effect="dark" content="取消修改" placement="top">
                        <el-link @click="cancelUpdateFields" type="warning" :underline="false" class="font12">取消</el-link>
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
                    placeholder="可输入SQL条件表达式后回车或点击查询图标过滤结果, 可根据备注或字段名提示"
                    @clear="selectData"
                    size="small"
                    clearable
                    class="w100"
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

        <el-dialog v-model="addDataDialog.visible" :title="addDataDialog.title" :destroy-on-close="true" width="600px">
            <el-form ref="dataForm" :model="addDataDialog.data" label-width="auto" size="small">
                <el-form-item
                    v-for="column in columns"
                    :key="column.columnName"
                    class="w100"
                    :prop="column.columnName"
                    :label="column.columnName"
                    :required="column.nullable != 'YES' && column.columnKey != 'PRI'"
                >
                    <ColumnFormItem
                        v-model="addDataDialog.data[`${column.columnName}`]"
                        :data-type="dbDialect.getDataType(column.columnType)"
                        :placeholder="`${column.columnType}  ${column.columnComment}`"
                    />
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
import { onMounted, reactive, Ref, ref, toRefs, watch } from 'vue';
import { ElMessage } from 'element-plus';

import { DbInst } from '@/views/ops/db/db';
import DbTableData from './DbTableData.vue';
import { DbDialect, getDbDialect } from '@/views/ops/db/dialect';
import SvgIcon from '@/components/svgIcon/index.vue';
import ColumnFormItem from './ColumnFormItem.vue';

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

const dataForm: any = ref(null);
const dbTableRef: Ref = ref(null);
const condInputRef: Ref = ref(null);

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
    count: 0,
    selectionDatas: [] as any,
    addDataDialog: {
        data: {},
        title: '',
        placeholder: '',
        visible: false,
    },
    tableHeight: '600px',
    hasUpdatedFileds: false,
    dbDialect: {} as DbDialect,
});

const { datas, condition, loading, columns, pageNum, pageSize, pageSizes, count, hasUpdatedFileds, addDataDialog, dbDialect } = toRefs(state);

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
    const columns = await getNowDbInst().loadColumns(props.dbName, props.tableName);
    columns.forEach((x: any) => {
        x.show = true;
    });
    state.columns = columns;
    await onRefresh();

    state.dbDialect = getDbDialect(getNowDbInst().type);
});

const onRefresh = async () => {
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
    const dbInst = getNowDbInst();
    const db = props.dbName;
    const table = props.tableName;
    try {
        const countRes = await dbInst.runSql(db, dbInst.getDefaultCountSql(table, state.condition));
        state.count = countRes.res[0].count || countRes.res[0].COUNT || 0;
        let sql = dbInst.getDefaultSelectSql(table, state.condition, state.orderBy, state.pageNum, state.pageSize);
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
        columnNameSearch = columnNameSearch.toLowerCase();
        res = columns.filter((data: any) => {
            return data.columnName.toLowerCase().includes(columnNameSearch) || data.columnComment.includes(columnNameSearch);
        });
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
    if (!DbInst.isNumber(column.columnType)) {
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
    state.orderBy = `ORDER BY ${sort.columnName} ${sortType}`;
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
    state.addDataDialog.title = `添加'${props.tableName}'表数据`;
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
            const dbInst = getNowDbInst();
            const data = state.addDataDialog.data;
            // key: 字段名，value: 字段名提示
            let obj: any = {};
            for (let item of state.columns) {
                const value = data[item.columnName];
                if (!value) {
                    continue;
                }
                obj[`${dbInst.wrapName(item.columnName)}`] = DbInst.wrapValueByType(value);
            }
            let columnNames = Object.keys(obj).join(',');
            let values = Object.values(obj).join(',');
            let sql = `INSERT INTO ${dbInst.wrapName(props.tableName)} (${columnNames}) VALUES (${values});`;
            dbInst.promptExeSql(props.dbName, sql, null, () => {
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

<style lang="scss"></style>
