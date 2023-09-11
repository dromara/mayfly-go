<template>
    <div>
        <el-table
            @cell-dblclick="(row: any, column: any, cell: any, event: any) => cellClick(row, column, cell)"
            @sort-change="(sort: any) => onTableSortChange(sort)"
            @selection-change="onDataSelectionChange"
            :data="datas"
            size="small"
            :max-height="tableHeight"
            v-loading="loading"
            element-loading-text="查询中..."
            :empty-text="emptyText"
            highlight-current-row
            stripe
            border
            class="mt5"
        >
            <el-table-column v-if="datas.length > 0 && table" type="selection" width="35" />

            <template v-for="(item, index) in columns">
                <el-table-column
                    min-width="100"
                    :width="DbInst.flexColumnWidth(item.columnName, datas)"
                    align="center"
                    v-if="item.show"
                    :key="index"
                    :prop="item.columnName"
                    :label="item.columnName"
                    show-overflow-tooltip
                    :sortable="sortable"
                >
                    <template #header v-if="showColumnTip">
                        <el-tooltip raw-content placement="top" effect="customized">
                            <template #content> {{ getColumnTip(item) }} </template>
                            {{ item.columnName }}
                        </el-tooltip>
                    </template>
                </el-table-column>
            </template>
        </el-table>
    </div>
</template>

<script lang="ts" setup>
import { onMounted, watch, reactive, toRefs } from 'vue';
import { DbInst, UpdateFieldsMeta, FieldsMeta } from '../db';

const emits = defineEmits(['sortChange', 'deleteData', 'selectionChange', 'changeUpdatedField'])

const props = defineProps({
    dbId: {
        type: Number,
        required: true,
    },
    dbType: {
        type: String,
        default: ''
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
        type: String,
        default: '600'
    }
})

const state = reactive({
    dbId: 0, // 当前选中操作的数据库实例
    dbType: '',
    db: '',  // 数据库名
    table: '', // 当前的表名
    datas: [],
    columns: [],
    sortable: false,
    loading: false,
    selectionDatas: [] as any,
    showColumnTip: false,
    tableHeight: '600',
    emptyText: '',
    updatedFields: [] as UpdateFieldsMeta[],// 各个tab表被修改的字段信息
});

const {
    tableHeight,
    datas,
    sortable,
    loading,
    showColumnTip,
} = toRefs(state);

watch(props, (newValue: any) => {
    setState(newValue);
});

onMounted(async () => {
    console.log('in DbTable mounted');
    setState(props);
})

const setState = (props: any) => {
    state.dbId = props.dbId;
    state.dbType = props.dbType;
    state.db = props.db;
    state.table = props.table;
    state.datas = props.data;
    state.tableHeight = props.height;
    state.sortable = props.sortable;
    state.loading = props.loading;
    state.columns = props.columns;
    state.showColumnTip = props.showColumnTip;
    state.emptyText = props.emptyText;
}

const getColumnTip = (column: any) => {
    const comment = column.columnComment;
    return `${column.columnType} ${comment ? ' |  ' + comment : ''}`;
};

/**
 * 表排序字段变更
 */
const onTableSortChange = async (sort: any) => {
    if (!sort.prop) {
        return;
    }
    cancelUpdateFields();
    emits('sortChange', sort);
};

const onDataSelectionChange = (datas: []) => {
    state.selectionDatas = datas;
    emits('selectionChange', datas);
};

// 监听单元格点击事件
const cellClick = (row: any, column: any, cell: any) => {
    const property = column.property;
    // 如果当前操作的表名不存在 或者 当前列的property不存在(如多选框)，则不允许修改当前单元格内容
    if (!state.table || !property) {
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
                const dbInst = getNowDbInst();
                // 主键
                const primaryKey = await dbInst.loadTableColumn(state.db, state.table);
                const primaryKeyValue = row[primaryKey.columnName];
                // 更新字段列信息
                const updateColumn = await dbInst.loadTableColumn(state.db, state.table, property);
                const newField = {
                    div, row,
                    fieldName: property,
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
                    if (a.fieldName === column.property) {
                        a.newValue = input.value
                    }
                    return a.fieldName === column.property
                })

                const field = fieldsParam.length > 0 && fieldsParam[0] || {} as FieldsMeta
                if (field.oldValue === input.value) { // 新值=旧值
                    // 删除数据
                    div.classList.remove('update_field_active')
                    let delIndex: number[] = [];
                    currentUpdatedFields.forEach((a, i) => {
                        if (a.primaryKey === primaryKeyValue) {
                            a.fields = a.fields && a.fields.length > 0 ? a.fields.filter(f => f.fieldName !== column.property) : [];
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
                changeUpdatedField();
            }
        });
    }
};

const submitUpdateFields = () => {
    const dbInst = DbInst.getInst(state.dbId)
    let currentUpdatedFields = state.updatedFields;
    if (currentUpdatedFields.length <= 0) {
        return;
    }
    const db = state.db;
    let res = '';
    let divs: HTMLElement[] = [];
    currentUpdatedFields.forEach(a => {
        let sql = `UPDATE ${dbInst.wrapName(state.table)} SET `;
        let primaryKey = a.primaryKey;
        let primaryKeyType = a.primaryKeyType;
        let primaryKeyName = a.primaryKeyName;
        a.fields.forEach(f => {
            sql += ` ${dbInst.wrapName(f.fieldName)} = ${DbInst.wrapColumnValue(f.fieldType, f.newValue)},`
            // 如果修改的字段是主键
            if (f.fieldName === primaryKeyName) {
                primaryKey = f.oldValue
            }
            divs.push(f.div)
        })
        sql = sql.substring(0, sql.length - 1)
        sql += ` WHERE ${dbInst.wrapName(primaryKeyName)} = ${DbInst.wrapColumnValue(primaryKeyType, primaryKey)} ;`
        res += sql;
    })

    dbInst.promptExeSql(db, res, () => { }, () => {
        currentUpdatedFields = [];
        divs.forEach(a => {
            a.classList.remove('update_field_active');
        })
        state.updatedFields = [];
        changeUpdatedField();
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
    changeUpdatedField();
}


const changeUpdatedField = () => {
    emits('changeUpdatedField', state.updatedFields);
}

const getNowDbInst = () => {
    return DbInst.getInst(state.dbId);
}

defineExpose({
    submitUpdateFields,
    cancelUpdateFields
})
</script>

<style lang="scss">
.update_field_active {
    background-color: var(--el-color-success);
}
</style>
