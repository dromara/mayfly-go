<template>
    <div>
        <el-row>
            <el-col :span="8">
                <el-link @click="onRefresh()" icon="refresh" :underline="false" class="ml5">
                </el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-link @click="addRow()" type="primary" icon="plus" :underline="false"></el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-link @click="onDeleteData()" type="danger" icon="delete" :underline="false"></el-link>
                <el-divider direction="vertical" border-style="dashed" />

                <el-tooltip class="box-item" effect="dark" content="commit" placement="top">
                    <el-link @click="onCommit()" type="success" icon="CircleCheck" :underline="false">
                    </el-link>
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

                <el-tooltip v-if="state.updatedFields.length > 0" class="box-item" effect="dark" content="提交修改"
                    placement="top">
                    <el-link @click="submitUpdateFields()" type="success" :underline="false" class="f12">提交</el-link>
                </el-tooltip>
                <el-divider v-if="state.updatedFields.length > 0" direction="vertical" border-style="dashed" />
                <el-tooltip v-if="state.updatedFields.length > 0" class="box-item" effect="dark" content="取消修改"
                    placement="top">
                    <el-link @click="cancelUpdateFields" type="warning" :underline="false" class="f12">取消</el-link>
                </el-tooltip>
            </el-col>
            <el-col :span="16">
                <el-input v-model="condition" placeholder="若需条件过滤，可选择列并点击对应的字段并输入需要过滤的内容点击查询按钮即可" clearable size="small"
                    style="width: 100%">
                    <template #prepend>
                        <el-popover trigger="click" :width="320" placement="right">
                            <template #reference>
                                <el-link type="success" :underline="false">选择列</el-link>
                            </template>
                            <el-table :data="columns" max-height="500" size="small" @row-click="
                                (...event: any) => {
                                    onConditionRowClick(event);
                                }
                            " style="cursor: pointer">
                                <el-table-column property="columnName" label="列名" show-overflow-tooltip>
                                </el-table-column>
                                <el-table-column property="columnComment" label="备注" show-overflow-tooltip>
                                </el-table-column>
                            </el-table>
                        </el-popover>
                    </template>

                    <template #append>
                        <el-button @click="onSelectByCondition()" icon="search" size="small"></el-button>
                    </template>
                </el-input>
            </el-col>
        </el-row>
        <el-table @cell-dblclick="(row: any, column: any, cell: any, event: any) => cellClick(row, column, cell)"
            @sort-change="(sort: any) => onTableSortChange(sort)" @selection-change="onDataSelectionChange"
            :data="datas" size="small" :max-height="tableHeight" v-loading="loading" element-loading-text="查询中..."
            empty-text="暂无数据" stripe border class="mt5">
            <el-table-column v-if="datas.length > 0" type="selection" width="35" />
            <el-table-column min-width="100" :width="DbInst.flexColumnWidth(item, datas)" align="center"
                v-for="item in columnNames" :key="item" :prop="item" :label="item" show-overflow-tooltip
                :sortable="'custom'">
                <template #header>
                    <el-tooltip raw-content placement="top" effect="customized">
                        <template #content> {{ getColumnTip(item) }} </template>
                        {{ item }}
                    </el-tooltip>
                </template>
            </el-table-column>
        </el-table>
        <el-row type="flex" class="mt5" justify="center">
            <el-pagination small :total="count" @current-change="pageChange()" layout="prev, pager, next, total, jumper"
                v-model:current-page="pageNum" :page-size="DbInst.DefaultLimit"></el-pagination>
        </el-row>
        <div style=" font-size: 12px; padding: 0 10px; color: #606266"><span>{{ state.sql }}</span>
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
    </div>
</template>

<script lang="ts" setup>
import { onMounted, watch, reactive, toRefs } from 'vue';
import { isTrue, notEmpty } from '@/common/assert';
import { ElMessage } from 'element-plus';

import { DbInst, UpdateFieldsMeta, FieldsMeta, TabInfo } from '../../db';
import { exportCsv } from '@/common/utils/export';
import { dateStrFormat } from '@/common/utils/date';
import { notBlank } from '../../../../../common/assert';

const emits = defineEmits(['genInsertSql', 'clickSqlName', 'clickSchemaTable', 'changeSchema', 'loadSqlNames'])

const props = defineProps({
    data: {
        type: TabInfo,
        required: true
    },
    tableHeight: {
        type: String,
        default: '600'
    }
})

const state = reactive({
    ti: {} as TabInfo,
    dbId: null, // 当前选中操作的数据库实例
    table: '', // 当前的表名
    datas: [],
    sql: '', // 当前数据tab执行的sql
    orderBy: '',
    condition: '', // 当前条件框的条件
    loading: false, // 是否在加载数据
    columnNames: [],
    columns: [],
    pageNum: 1,
    count: 0,
    selectionDatas: [] as any,
    conditionDialog: {
        title: '',
        placeholder: '',
        columnRow: null,
        dataTab: null,
        visible: false,
        condition: '=',
        value: null
    },
    tableHeight: '600',
    updatedFields: [] as UpdateFieldsMeta[],// 各个tab表被修改的字段信息
});

const {
    datas,
    condition,
    loading,
    columns,
    columnNames,
    pageNum,
    count,
    conditionDialog,
} = toRefs(state);

watch(() => props.tableHeight, (newValue: any) => {
    state.tableHeight = newValue;
});

onMounted(async () => {
    console.log('in table data mounted');
    state.ti = props.data;
    state.tableHeight = props.tableHeight;
    state.table = state.ti.other.table;
    notBlank(state.table, "TableData组件other.table信息不能为空")

    const columns = await state.ti.getNowDbInst().loadColumns(state.ti.db, state.table);
    state.columns = columns;
    state.columnNames = columns.map((t: any) => t.columnName);
    await onRefresh();
})

const onRefresh = async () => {
    // 查询条件置空
    state.condition = '';
    state.pageNum = 1;
    await selectData();
}

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
        const countRes = await dbInst.runSql(db, DbInst.getDefaultCountSql(state.table));
        state.count = countRes.res[0].count;
        let sql = dbInst.getDefaultSelectSql(state.table, state.condition, state.orderBy, state.pageNum);
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
}

/**
 * 导出当前页数据
 */
const exportData = () => {
    const dataList = state.datas as any;
    isTrue(dataList.length > 0, '没有数据可导出');
    exportCsv(`数据导出-${state.table}-${dateStrFormat('yyyyMMddHHmm', new Date().toString())}`, state.columnNames, dataList)
};


const getColumnTip = (columnName: string) => {
    // 优先从 table map中获取
    let columns = getColumns();
    if (!columns) {
        return '';
    }

    const column = columns.find((c: any) => c.columnName == columnName);
    const comment = column.columnComment;
    return `${column.columnType} ${comment ? ' |  ' + comment : ''}`;
};

const getColumns = () => {
    return state.ti.getNowDb().getColumns(state.table);
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
}

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

/**
 * 执行删除数据事件
 */
const onDeleteData = async () => {
    const deleteDatas = state.selectionDatas;
    isTrue(deleteDatas && deleteDatas.length > 0, '请先选择要删除的数据');
    const { db } = state.ti;
    const dbInst = state.ti.getNowDbInst()
    dbInst.promptExeSql(db, dbInst.genDeleteByPrimaryKeysSql(db, state.table, deleteDatas), null, () => {
        onRefresh();
    });
};

const onGenerateInsertSql = async () => {
    emits('genInsertSql', state.ti.getNowDbInst().genInsertSql(state.ti.db, state.table, state.selectionDatas));
};


// 监听单元格点击事件
const cellClick = (row: any, column: any, cell: any) => {
    const property = column.property;
    // 如果当前操作的表名不存在 或者 当前列的property不存在(如多选框)，则不允许修改当前单元格内容
    if (!property) {
        return;
    }
    let div: HTMLElement = cell.children[0];
    if (div && div.tagName === 'DIV') {
        // 转为字符串比较,可能存在数字等
        let text = (row[property] || row[property] == 0 ? row[property] : '') + '';
        let input = document.createElement('input');
        input.setAttribute('value', text);
        // 将表格width也赋值于输入框，避免输入框长度超过表格长度
        input.setAttribute('style', 'height:23px;text-align:center;border:none;' + div.getAttribute('style'));
        cell.replaceChildren(input);
        input.focus();
        input.addEventListener('blur', async () => {
            row[property] = input.value;
            cell.replaceChildren(div);
            if (input.value !== text) {
                let currentUpdatedFields = state.updatedFields
                const db = state.ti.getNowDb();
                // 主键
                const primaryKey = db.getColumn(state.table);
                const primaryKeyValue = row[primaryKey.columnName];
                // 更新字段列信息
                const updateColumn = db.getColumn(state.table, property);
                const newField = {
                    div, row,
                    fieldName: column.rawColumnKey,
                    fieldType: updateColumn.columnType,
                    oldValue: text,
                    newValue: input.value
                } as FieldsMeta;

                // 被修改的字段
                const primaryKeyFields = currentUpdatedFields.filter((meta) => meta.primaryKey === primaryKeyValue)
                let hasKey = false;
                if (primaryKeyFields.length <= 0) {
                    primaryKeyFields[0] = {
                        primaryKey: primaryKeyValue,
                        primaryKeyName: primaryKey.columnName,
                        primaryKeyType: primaryKey.columnType,
                        fields: [newField]
                    }
                } else {
                    hasKey = true
                    let hasField = primaryKeyFields[0].fields.some(a => {
                        if (a.fieldName === newField.fieldName) {
                            a.newValue = newField.newValue
                        }
                        return a.fieldName === newField.fieldName
                    })
                    if (!hasField) {
                        primaryKeyFields[0].fields.push(newField)
                    }
                }
                let fields = primaryKeyFields[0].fields

                const fieldsParam = fields.filter((a) => {
                    if (a.fieldName === column.rawColumnKey) {
                        a.newValue = input.value
                    }
                    return a.fieldName === column.rawColumnKey
                })

                const field = fieldsParam.length > 0 && fieldsParam[0] || {} as FieldsMeta
                if (field.oldValue === input.value) { // 新值=旧值
                    // 删除数据
                    div.classList.remove('update_field_active')
                    let delIndex: number[] = [];
                    currentUpdatedFields.forEach((a, i) => {
                        if (a.primaryKey === primaryKeyValue) {
                            a.fields = a.fields && a.fields.length > 0 ? a.fields.filter(f => f.fieldName !== column.rawColumnKey) : [];
                            a.fields.length <= 0 && delIndex.push(i)
                        }
                    });
                    delIndex.forEach(i => delete currentUpdatedFields[i])
                    currentUpdatedFields = currentUpdatedFields.filter(a => a)
                } else {
                    // 新增数据
                    div.classList.add('update_field_active')
                    if (hasKey) {
                        currentUpdatedFields.forEach((value, index, array) => {
                            if (value.primaryKey === primaryKeyValue) {
                                array[index].fields = fields
                            }
                        })
                    } else {
                        currentUpdatedFields.push({
                            primaryKey: primaryKeyValue,
                            primaryKeyName: primaryKey.columnName,
                            primaryKeyType: primaryKey.columnType,
                            fields
                        })
                    }
                }
                state.updatedFields = currentUpdatedFields;
            }
        });
    }
};

const submitUpdateFields = () => {
    let currentUpdatedFields = state.updatedFields;
    if (currentUpdatedFields.length <= 0) {
        return;
    }
    const { db } = state.ti;
    let res = '';
    let divs: HTMLElement[] = [];
    currentUpdatedFields.forEach(a => {
        let sql = `UPDATE ${state.table} SET `;
        let primaryKey = a.primaryKey;
        let primaryKeyType = a.primaryKeyType;
        let primaryKeyName = a.primaryKeyName;
        a.fields.forEach(f => {
            sql += ` ${f.fieldName} = ${DbInst.wrapColumnValue(f.fieldType, f.newValue)},`
            divs.push(f.div)
        })
        sql = sql.substring(0, sql.length - 1)
        sql += ` WHERE ${primaryKeyName} = ${DbInst.wrapColumnValue(primaryKeyType, primaryKey)} ;`
        res += sql;
    })

    state.ti.getNowDbInst().promptExeSql(db, res, () => { }, () => {
        currentUpdatedFields = [];
        divs.forEach(a => {
            a.classList.remove('update_field_active');
        })
        state.updatedFields = [];
    });

}

const cancelUpdateFields = () => {
    state.updatedFields.forEach((a: any) => {
        a.fields.forEach((b: any) => {
            b.div.classList.remove('update_field_active')
            b.row[b.fieldName] = b.oldValue
        })
    })
    state.updatedFields = [];
}

// 添加新数据行
const addRow = async () => {
    const columns = state.ti.getNowDb().getColumns(state.table);
    // key: 字段名，value: 字段名提示
    let obj: any = {};
    columns.forEach((item: any) => {
        obj[`${item.columnName}`] = `'${item.columnComment || ''} ${item.columnName}[${item.columnType}]${item.nullable == 'YES' ? '' : '[not null]'}'`;
    });
    let columnNames = Object.keys(obj).join(',');
    let values = Object.values(obj).join(',');
    let sql = `INSERT INTO ${state.table} (${columnNames}) VALUES (${values});`;
    state.ti.getNowDbInst().promptExeSql(state.ti.db, sql, null, () => {
        onRefresh();
    });
};

</script>

<style lang="scss">
.update_field_active {
    background-color: var(--el-color-success)
}
</style>
