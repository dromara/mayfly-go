<template>
    <div class="h-full flex flex-col gap-1">
        <el-row class="flex-shrink-0">
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

                    <el-link v-if="!readonly" @click="onShowAddDataDialog()" type="primary" icon="plus" underline="never"></el-link>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip v-if="!readonly" :show-after="500" effect="dark" :content="$t('db.importData')" placement="top">
                        <el-link v-auth="'db:sqlscript:run'" @click="openImportDialog" type="warning" icon="upload" underline="never"></el-link>
                    </el-tooltip>
                    <el-divider v-if="!readonly" direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" effect="dark" content="commit" placement="top">
                        <el-link @click="onCommit()" type="success" icon="CircleCheck" underline="never"> </el-link>
                    </el-tooltip>
                    <el-divider direction="vertical" border-style="dashed" />

                    <el-tooltip :show-after="500" v-if="hasUpdatedFields" :content="$t('db.submitUpdate')" placement="top">
                        <el-link @click="submitUpdateFields()" type="success" underline="never" class="text-[12px]!">{{ $t('common.submit') }}</el-link>
                    </el-tooltip>
                    <el-divider v-if="hasUpdatedFields" direction="vertical" border-style="dashed" />
                    <el-tooltip :show-after="500" v-if="hasUpdatedFields" :content="$t('db.cancelUpdate')" placement="top">
                        <el-link @click="cancelUpdateFields" type="warning" underline="never" class="text-[12px]!">{{ $t('common.cancel') }}</el-link>
                    </el-tooltip>
                </div>
            </el-col>
            <el-col :span="16">
                <div class="flex items-center gap-2 flex-wrap w-full min-w-0">
                    <!-- 过滤双轨：可视化构建器为主，SQL 表达式退为高级模式 -->
                    <el-radio-group v-model="state.conditionMode" size="small" class="shrink-0">
                        <el-radio-button value="builder">{{ $t('db.filterModeBuilder') }}</el-radio-button>
                        <el-radio-button value="sql">{{ $t('db.filterModeSql') }}</el-radio-button>
                    </el-radio-group>

                    <template v-if="state.conditionMode === 'builder'">
                        <el-popover v-model:visible="state.filterPopVisible" placement="bottom-start" :width="520" trigger="click">
                            <template #reference>
                                <el-badge :value="filterChips.length" :hidden="!filterChips.length || customSqlActive" :offset="[-4, 2]">
                                    <el-button size="small" icon="Filter">{{ $t('db.filter') }}</el-button>
                                </el-badge>
                            </template>
                            <DbTableFilterBuilder
                                v-if="state.filterPopVisible"
                                :columns="state.columns"
                                :model-value="state.filterGroup"
                                :dialect="state.dbDialect"
                                @apply="applyBuilderFilter"
                            />
                        </el-popover>

                        <!-- 生效条件 chips：点击回构建器编辑，× 单条删除并立即生效；
                             手写 SQL 生效期间隐藏（与实际 WHERE 不一致，由 chip-sql 表达） -->
                        <template v-if="!customSqlActive">
                            <span v-for="chip in filterChips" :key="chip.id" class="filter-chip">
                                <button type="button" class="chip-text" :title="chip.text" @click="state.filterPopVisible = true">{{ chip.text }}</button>
                                <button
                                    type="button"
                                    class="chip-close"
                                    :aria-label="$t('common.delete')"
                                    :title="$t('common.delete')"
                                    @click="removeFilterChip(chip.id)"
                                >
                                    <el-icon><Close /></el-icon>
                                </button>
                            </span>
                        </template>
                        <!-- 当前生效条件为 SQL 模式下手写的表达式，与构建器条件组不一致 -->
                        <span v-if="customSqlActive" class="filter-chip chip-sql">
                            <button type="button" class="chip-text" :title="state.condition" @click="state.conditionMode = 'sql'">
                                {{ $t('db.customSqlCondition') }}
                            </button>
                        </span>
                        <span v-if="!filterChips.length && !customSqlActive" class="text-[12px] text-gray-400">{{ $t('db.noFilter') }}</span>
                    </template>

                    <el-autocomplete
                        v-else
                        v-model="condition"
                        :fetch-suggestions="getColumnTips"
                        @keyup.enter="onSelectByCondition"
                        @select="handlerColumnSelect"
                        popper-class="my-autocomplete"
                        :placeholder="$t('db.sqlConditionPlaceholder')"
                        @clear="selectData"
                        size="small"
                        clearable
                        class="flex-1 min-w-0"
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
                </div>
            </el-col>
        </el-row>

        <db-table-data
            ref="dbTableRef"
            class="flex-1 min-h-0 overflow-hidden"
            :db-id="dbId"
            :db="dbName"
            :data="datas"
            :table="tableName"
            :columns="columns"
            :readonly="readonly"
            :loading="loading"
            :page-size="pageSize"
            :page-num="pageNum"
            :show-column-tip="true"
            @sort-change="(sort: { key: string; order: string }) => onTableSortChange(sort)"
            @change-updated-field="changeUpdatedField"
            @data-delete="onRefresh"
        ></db-table-data>

        <el-row type="flex" class="flex-shrink-0" :gutter="10" justify="space-between" style="user-select: none">
            <el-col :span="12">
                <el-text
                    id="copyValue"
                    style="color: var(--el-color-info-light-3)"
                    class="is-truncated text-[12px]! mt-1"
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
                        @click="pageNum = Math.max(1, pageNum - 1)"
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

        <DbTableDataImport
            v-if="!readonly"
            :db-id="dbId"
            :db-name="dbName"
            :table-name="tableName"
            :columns="columns"
            v-model:visible="importDialogVisible"
            @success="onRefresh"
        />
    </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive, Ref, ref, toRefs, watch } from 'vue';

import { Close } from '@element-plus/icons-vue';
import { copyToClipboard, fuzzyMatchField } from '@/common/utils/string';
import SvgIcon from '@/components/svg-icon/index.vue';
import { Msg } from '@/hooks/useI18n';
import { DbInst } from '@/views/ops/db/db';
import { DbDialect } from '@/views/ops/db/dialect';
import type { ColumnMetadata, TableColumnDef } from '@/views/ops/db/types';
import { useI18n } from 'vue-i18n';
import DbTableData from './DbTableData.vue';
import DbTableDataForm from './DbTableDataForm.vue';
import DbTableDataImport from './DbTableDataImport.vue';
import DbTableFilterBuilder from './DbTableFilterBuilder.vue';
import { createFilterGroup, getOperatorDef, isConditionUsable, serializeFilterGroup, type TableFilterCondition, type TableFilterGroup } from './filterModel';

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
    // 只读（如视图）：隐藏新增、禁用单元格编辑与删除
    readonly: {
        type: Boolean,
        default: false,
    },
});

const dbTableRef: Ref = ref(null);
const condInputRef: Ref = ref(null);

const defaultPageSize = DbInst.DefaultLimit;

const state = reactive({
    datas: [] as Record<string, unknown>[],
    sql: '', // 当前数据tab执行的sql
    orderBy: '',
    condition: '', // 当前条件框的条件
    loading: false, // 是否在加载数据
    columns: [] as TableColumnDef[],
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
    // 过滤双轨模式：builder 可视化构建器 / sql 手写表达式
    conditionMode: 'builder' as 'builder' | 'sql',
    // 可视化构建器条件组（chips 展示与 WHERE 序列化的单一真源）
    filterGroup: createFilterGroup(),
    filterPopVisible: false,
    addDataDialog: {
        data: {},
        title: '',
        visible: false,
    },
    hasUpdatedFields: false,
    dbDialect: {} as DbDialect,

    checkedShowColumns: {
        searchKey: '',
        checkedAllColumn: true,
        isIndeterminate: false,
        columnNames: [] as string[],
    },
});

const { datas, condition, loading, columns, checkedShowColumns, pageNum, pageSize, pageSizes, sql, hasUpdatedFields, addDataDialog } = toRefs(state);

const getNowDbInst = () => {
    return DbInst.getInst(props.dbId);
};

onMounted(async () => {
    await onRefresh();

    state.dbDialect = getNowDbInst().getDialect();

    state.checkedShowColumns.columnNames = state.columns.map((item: TableColumnDef) => item.columnName);
});

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

// SQL 模式清空表达式即「无过滤」：同步重置可视化条件组，避免残留 chips 在下次应用时复活
watch(
    () => state.condition,
    (v) => {
        if (state.conditionMode === 'sql' && !v.trim() && state.filterGroup.conditions.length) {
            state.filterGroup = createFilterGroup();
        }
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
            const columns = (await getNowDbInst().loadColumns(props.dbName, props.tableName)) as TableColumnDef[];
            columns.forEach((x: TableColumnDef) => {
                x.show = true;
                x.key = x.columnName;
            });
            state.columns = columns;
        }

        let sql = dbInst.getDefaultSelectSql(db, table, state.condition, state.orderBy, state.pageNum, state.pageSize);
        state.sql = sql;
        const res = await dbInst.runSql(db, sql);
        const colAndData = res[0];
        state.datas = colAndData.res ?? [];
    } finally {
        state.loading = false;
    }
};

const handleSizeChange = async (size: number) => {
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
        const countRes = (await dbInst.runSql(db, dbInst.getDefaultCountSql(table, state.condition)))[0];
        const countRow = countRes.res?.[0];
        state.total = parseInt(String(countRow?.count || countRow?.COUNT || 0));
        state.showTotal = true;
    } catch (e) {
        /* empty */
    }

    state.counting = false;
};

const handleCheckAllColumnChange = (val: boolean) => {
    state.checkedShowColumns.columnNames = val ? state.columns.map((x: TableColumnDef) => x.columnName) : [];
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

const getColumnTips = (queryString: string, callback: (res: TableColumnDef[]) => void) => {
    const columns = state.columns;

    var words = queryString.split(' '); // 使用空格分割字符串为数组
    let columnNameSearch = words[words.length - 1]; // 获取最后一个元素

    let res: TableColumnDef[] = [];
    if (columnNameSearch) {
        res = fuzzyMatchField(columnNameSearch, columns, (x: TableColumnDef) => x.columnName);
    }

    completeCond = condition.value;
    callback(res);

    existSuggestion = res.length > 0;
};

const handlerColumnSelect = (column: ColumnMetadata) => {
    // 获取最后一个空格的索引
    var lastSpaceIndex = completeCond.lastIndexOf(' ');

    // 默认拼接上 columnName =
    let value = column.columnName + ' = ';
    // 不是数字类型默认拼接上''
    if (!DbInst.isNumber(column.dataType)) {
        value = `${value}''`;
    }

    if (lastSpaceIndex != -1) {
        // 获取最后一个空格之前的文本,拼上当前选中的建议列
        condition.value = `${completeCond.slice(0, lastSpaceIndex)} ${value}`;
    } else {
        condition.value = value;
    }
};

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
        (x: TableColumnDef) => x.columnName,
        (x: TableColumnDef) => x.columnComment
    );
};

/** 生效条件 chips：文案形如 `列 操作符 值`，与构建器读同一份条件组 */
const chipText = (cond: TableFilterCondition) => {
    const def = getOperatorDef(cond.operator);
    const operator = def?.symbol ?? t(def?.labelKey ?? '');
    const value =
        def?.valueKind === 'range'
            ? `${cond.value} ~ ${cond.value2}`
            : def?.valueKind === 'multi'
              ? cond.inValues.join(', ')
              : def?.valueKind === 'single'
                ? cond.value
                : '';
    return value ? `${cond.columnName} ${operator} ${value}` : `${cond.columnName} ${operator}`;
};

const filterChips = computed(() =>
    state.filterGroup.conditions.filter((x) => isConditionUsable(x, state.columns, state.dbDialect)).map((x) => ({ id: x.id, text: chipText(x) }))
);

/** 当前生效条件为 SQL 模式下手写的表达式，与构建器条件组序列化结果不一致 */
const customSqlActive = computed(
    () =>
        state.conditionMode === 'builder' &&
        !!state.condition.trim() &&
        state.condition.trim() !== serializeFilterGroup(state.filterGroup, state.columns, state.dbDialect)
);

/** 应用构建器条件组：序列化为 WHERE 并回第一页查询；不传 group 表示按当前条件组应用（如单条 chip 删除后） */
const applyBuilderFilter = async (group?: TableFilterGroup) => {
    if (group) {
        // 不完整条件行（缺列/缺值/数值列非法文本）不参与查询，剪除以免 chips 与实际 SQL 不一致
        // 须包一层箭头函数：直传会把 Array.filter 的下标当第二个参数传入
        group.conditions = group.conditions.filter((cond) => isConditionUsable(cond, state.columns, state.dbDialect));
        state.filterGroup = group;
    }
    state.filterPopVisible = false;
    state.condition = serializeFilterGroup(state.filterGroup, state.columns, state.dbDialect);
    await onRefresh();
};

/** 单条 chip 删除后立即生效 */
const removeFilterChip = async (id: number) => {
    state.filterGroup.conditions = state.filterGroup.conditions.filter((x) => x.id !== id);
    await applyBuilderFilter();
};

/**
 * 提交事务，用于没有开启自动提交事务
 */
const onCommit = () => {
    getNowDbInst().runSql(props.dbName, 'COMMIT;');
    Msg.success('COMMIT success');
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
const onTableSortChange = async (sort: { key: string; order: string }) => {
    const sortType = sort.order == 'desc' ? 'DESC' : 'ASC';
    state.orderBy = `ORDER BY ${state.dbDialect.quoteIdentifier(sort.key)} ${sortType}`;
    await onRefresh();
};

const changeUpdatedField = (hasUpdatedFields: boolean) => {
    // 存在待提交的单元格变更时，工具条才出现「提交/取消」（选中行由 DbTableData 自己维护，无需在此镜像一份）
    state.hasUpdatedFields = hasUpdatedFields;
};

const submitUpdateFields = () => {
    dbTableRef.value?.submitUpdateFields();
};

const cancelUpdateFields = () => {
    dbTableRef.value?.cancelUpdateFields();
};

const onShowAddDataDialog = async () => {
    state.addDataDialog.title = t('db.addDataDialogTitle', { tableName: props.tableName });
    state.addDataDialog.visible = true;
};

// 数据文件导入弹窗（CSV/Excel）
const importDialogVisible = ref(false);
const openImportDialog = () => {
    importDialogVisible.value = true;
};

defineExpose({
    active: () => dbTableRef.value?.active(),
});
</script>

<style lang="scss">
.op-page {
    margin-left: 5px;
}
</style>

<style lang="scss" scoped>
// 生效过滤条件 chips：与 mongo 条件条同一语汇，单条可删、点击回构建器编辑
.filter-chip {
    display: inline-flex;
    align-items: center;
    max-width: 260px;
    height: 24px;
    padding-left: 8px;
    border: 1px solid var(--el-border-color);
    border-radius: 12px;
    background: var(--el-fill-color-light);
    font-size: 12px;
    color: var(--el-text-color-regular);

    .chip-text {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        cursor: pointer;
    }

    .chip-close {
        display: inline-flex;
        align-items: center;
        margin: 0 4px 0 2px;
        color: var(--el-text-color-secondary);
        cursor: pointer;

        &:hover {
            color: var(--el-color-danger);
        }
    }

    &.chip-sql {
        border-style: dashed;
        color: var(--el-color-warning);
    }
}
</style>
