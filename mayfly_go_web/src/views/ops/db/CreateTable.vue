<template>
    <div>
        <el-dialog :title="title" v-model="dialogVisible" :before-close="cancel" width="90%">
            <el-form label-position="left" ref="formRef" :model="tableData" label-width="80px">
                <el-row>
                    <el-col :span="12">
                        <el-form-item prop="tableName" label="表名">
                            <el-input style="width: 80%" v-model="tableData.tableName" size="small"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item prop="tableComment" label="备注">
                            <el-input style="width: 80%" v-model="tableData.tableComment" size="small"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item prop="characterSet" label="charset">
                            <el-select filterable style="width: 80%" v-model="tableData.characterSet" size="small">
                                <el-option v-for="item in characterSetNameList" :key="item" :label="item" :value="item"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :span="12">
                        <el-form-item prop="characterSet" label="collation">
                            <el-select filterable style="width: 80%" v-model="tableData.collation" size="small">
                                <el-option
                                    v-for="item in collationNameList"
                                    :key="item"
                                    :label="tableData.characterSet + '_' + item"
                                    :value="tableData.characterSet + '_' + item"
                                >
                                </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-tabs v-model="activeName">
                    <el-tab-pane label="字段" name="1">
                        <el-table :data="tableData.fields.res" :max-height="tableData.height">
                            <el-table-column :prop="item.prop" :label="item.label" v-for="item in tableData.fields.colNames" :key="item.prop">
                                <template #default="scope">
                                    <el-input v-if="item.prop === 'name'" size="small" v-model="scope.row.name"> </el-input>

                                    <el-select v-if="item.prop === 'type'" filterable size="small" v-model="scope.row.type">
                                        <el-option v-for="typeValue in columnTypeList" :key="typeValue" :value="typeValue">{{ typeValue }}</el-option>
                                    </el-select>

                                    <el-input v-if="item.prop === 'value'" size="small" v-model="scope.row.value"> </el-input>

                                    <el-input v-if="item.prop === 'length'" size="small" v-model="scope.row.length"> </el-input>

                                    <el-checkbox v-if="item.prop === 'notNull'" size="small" v-model="scope.row.notNull"> </el-checkbox>

                                    <el-checkbox v-if="item.prop === 'pri'" size="small" v-model="scope.row.pri"> </el-checkbox>

                                    <el-checkbox v-if="item.prop === 'auto_increment'" size="small" v-model="scope.row.auto_increment"> </el-checkbox>

                                    <el-input v-if="item.prop === 'remark'" size="small" v-model="scope.row.remark"> </el-input>

                                    <el-link
                                        v-if="item.prop === 'action'"
                                        type="danger"
                                        plain
                                        size="small"
                                        :underline="false"
                                        @click.prevent="deleteRow(scope.$index)"
                                        >删除</el-link
                                    >
                                </template>
                            </el-table-column>
                        </el-table>
                        <el-row style="margin-top: 20px">
                            <el-button @click="addDefaultRows()" link type="warning" icon="plus">添加默认列</el-button>
                            <el-button @click="addRow()" link type="primary" icon="plus">添加列</el-button>
                        </el-row>
                    </el-tab-pane>
                    <el-tab-pane label="索引" name="2">
                        <el-table :data="tableData.indexs.res" :max-height="tableData.height">
                            <el-table-column :prop="item.prop" :label="item.label" v-for="item in tableData.indexs.colNames" :key="item.prop">
                                <template #default="scope">
                                    <el-input v-if="item.prop === 'indexName'" size="small" disabled v-model="scope.row.indexName"></el-input>

                                    <el-select
                                        v-if="item.prop === 'columnNames'"
                                        v-model="scope.row.columnNames"
                                        multiple
                                        collapse-tags
                                        collapse-tags-tooltip
                                        filterable
                                        placeholder="请选择字段"
                                        @change="indexChanges(scope.row)"
                                        style="width: 100%"
                                    >
                                        <el-option v-for="cl in tableData.indexs.columns" :key="cl.name" :label="cl.name" :value="cl.name">
                                            {{ cl.name + ' - ' + (cl.remark || '') }}
                                        </el-option>
                                    </el-select>

                                    <el-checkbox v-if="item.prop === 'unique'" size="small" v-model="scope.row.unique" @change="indexChanges(scope.row)">
                                    </el-checkbox>

                                    <el-select v-if="item.prop === 'indexType'" filterable size="small" v-model="scope.row.indexType">
                                        <el-option v-for="typeValue in indexTypeList" :key="typeValue" :value="typeValue">{{ typeValue }}</el-option>
                                    </el-select>

                                    <el-input v-if="item.prop === 'indexComment'" size="small" v-model="scope.row.indexComment"> </el-input>

                                    <el-link
                                        v-if="item.prop === 'action'"
                                        type="danger"
                                        plain
                                        size="small"
                                        :underline="false"
                                        @click.prevent="deleteIndex(scope.$index)"
                                        >删除</el-link
                                    >
                                </template>
                            </el-table-column>
                        </el-table>

                        <el-row style="margin-top: 20px">
                            <el-button @click="addIndex()" link type="primary" icon="plus">添加索引</el-button>
                        </el-row>
                    </el-tab-pane>
                </el-tabs>
            </el-form>
            <template #footer>
                <el-button :loading="btnloading" @click="submit()" type="primary">保存</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { watch, toRefs, reactive, ref } from 'vue';
import { TYPE_LIST, CHARACTER_SET_NAME_LIST, COLLATION_SUFFIX_LIST } from './service';
import { ElMessage } from 'element-plus';
import SqlExecBox from './component/SqlExecBox';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    title: {
        type: String,
    },
    data: {
        type: Object,
    },
    dbId: {
        type: Number,
    },
    db: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change', 'submit-sql']);

const formRef: any = ref();
const state = reactive({
    dialogVisible: false,
    btnloading: false,
    activeName: '1',
    columnTypeList: TYPE_LIST,
    indexTypeList: ['BTREE'], // mysql索引类型详解 http://c.biancheng.net/view/7897.html
    characterSetNameList: CHARACTER_SET_NAME_LIST,
    collationNameList: COLLATION_SUFFIX_LIST,
    tableData: {
        fields: {
            colNames: [
                {
                    prop: 'name',
                    label: '字段名称',
                },
                {
                    prop: 'type',
                    label: '字段类型',
                },
                {
                    prop: 'length',
                    label: '长度',
                },
                {
                    prop: 'value',
                    label: '默认值',
                },

                {
                    prop: 'notNull',
                    label: '非空',
                },
                {
                    prop: 'pri',
                    label: '主键',
                },
                {
                    prop: 'auto_increment',
                    label: '自增',
                },
                {
                    prop: 'remark',
                    label: '备注',
                },
                {
                    prop: 'action',
                    label: '操作',
                },
            ],
            res: [
                {
                    name: '',
                    type: '',
                    value: '',
                    length: '',
                    notNull: false,
                    pri: false,
                    auto_increment: false,
                    remark: '',
                },
            ],
        },
        indexs: {
            colNames: [
                {
                    prop: 'indexName',
                    label: '索引名',
                },
                {
                    prop: 'columnNames',
                    label: '列名',
                },
                {
                    prop: 'unique',
                    label: '唯一',
                },
                {
                    prop: 'indexType',
                    label: '类型',
                },
                {
                    prop: 'indexComment',
                    label: '备注',
                },
                {
                    prop: 'action',
                    label: '操作',
                },
            ],
            columns: [{ name: '', remark: '' }],
            res: [
                {
                    indexName: '',
                    columnNames: [],
                    unique: false,
                    indexType: 'BTREE',
                    indexComment: '',
                },
            ],
        },
        characterSet: 'utf8mb4',
        collation: 'utf8mb4_general_ci',
        tableName: '',
        tableComment: '',
        height: 550,
    },
});

const { dialogVisible, btnloading, activeName, columnTypeList, indexTypeList, characterSetNameList, collationNameList, tableData } = toRefs(state);

watch(props, async (newValue) => {
    state.dialogVisible = newValue.visible;
});

const cancel = () => {
    emit('update:visible', false);
    reset();
};

const addRow = () => {
    state.tableData.fields.res.push({
        name: '',
        type: '',
        value: '',
        length: '',
        notNull: false,
        pri: false,
        auto_increment: false,
        remark: '',
    });
};

const addIndex = () => {
    state.tableData.indexs.res.push({
        indexName: '',
        columnNames: [],
        unique: false,
        indexType: 'BTREE',
        indexComment: '',
    });
};

const addDefaultRows = () => {
    state.tableData.fields.res.push(
        { name: 'id', type: 'bigint', length: '20', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
        { name: 'creator_id', type: 'bigint', length: '20', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
        { name: 'creator', type: 'varchar', length: '100', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人姓名' },
        { name: 'create_time', type: 'datetime', length: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '创建时间' },
        { name: 'updator_id', type: 'bigint', length: '20', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
        { name: 'updator', type: 'varchar', length: '100', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人姓名' },
        { name: 'update_time', type: 'datetime', length: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '修改时间' }
    );
};

const deleteRow = (index: any) => {
    state.tableData.fields.res.splice(index, 1);
};

const deleteIndex = (index: any) => {
    state.tableData.indexs.res.splice(index, 1);
};

const submit = async () => {
    let sql = genSql();
    if (!sql) {
        ElMessage.warning('没有更改');
        return;
    }
    SqlExecBox({
        sql: sql,
        dbId: props.dbId as any,
        db: props.db as any,
        runSuccessCallback: () => {
            emit('submit-sql', { tableName: state.tableData.tableName });
            // cancel();
        },
    });
};

/**
 * 对比两个数组，取出被修改过的对象数组
 * @param oldArr 原对象数组
 * @param nowArr 修改后的对象数组
 * @param key 标志对象唯一属性
 */
const filterChangedData = (oldArr: object[], nowArr: object[], key: string): { del: any[]; add: any[]; upd: any[] } => {
    let data = {
        del: [] as object[], // 删除的数据
        add: [] as object[], // 新增的数据
        upd: [] as object[], // 修改的数据
    };

    // 旧数据为空
    if (oldArr && Array.isArray(oldArr) && oldArr.length === 0 && nowArr && Array.isArray(nowArr) && nowArr.length > 0) {
        data.add = nowArr;
        return data;
    }

    // 新数据为空
    if (nowArr && Array.isArray(nowArr) && nowArr.length === 0 && oldArr && Array.isArray(oldArr) && oldArr.length > 0) {
        data.del = oldArr;
        return data;
    }

    let oldMap = {},
        newMap = {};
    oldArr.forEach((a) => (oldMap[a[key]] = a));

    nowArr.forEach((a) => {
        let k = a[key];
        newMap[k] = a;
        if (!oldMap.hasOwnProperty(k)) {
            // 新增
            data.add.push(a);
        }
    });

    oldArr.forEach((a) => {
        let k = a[key];
        let newData = newMap[k];
        if (!newData) {
            // 删除
            data.del.push(a);
        } else {
            // 判断每个字段是否相等，否则为修改
            for (let f in a) {
                let oldV = a[f];
                let newV = newData[f];
                if (oldV.toString() !== newV.toString()) {
                    data.upd.push(newData);
                    break;
                }
            }
        }
    });
    return data;
};

const genSql = () => {
    const genColumnBasicSql = (cl: any) => {
        let val = cl.value ? (cl.value === 'CURRENT_TIMESTAMP' ? cl.value : "'" + cl.value + "'") : '';
        let defVal = `${val ? 'DEFAULT ' + val : ''}`;
        let length = cl.length ? `(${cl.length})` : '';
        let onUpdate = 'update_time' === cl.name ? ' ON UPDATE CURRENT_TIMESTAMP ' : '';
        return ` ${cl.name} ${cl.type}${length} ${cl.notNull ? 'NOT NULL' : 'NULL'} ${
            cl.auto_increment ? 'AUTO_INCREMENT' : ''
        } ${defVal} ${onUpdate} comment '${cl.remark || ''}' `;
    };

    let data = state.tableData;
    // 创建表
    if (!props.data?.edit) {
        if (state.activeName === '1') {
            // 创建表结构
            let primary_key = '';
            let fields: string[] = [];
            data.fields.res.forEach((item) => {
                item.name && fields.push(genColumnBasicSql(item));
                if (item.pri) {
                    primary_key += `${item.name},`;
                }
            });

            return `CREATE TABLE ${data.tableName}
                  ( ${fields.join(',')}
                      ${primary_key ? `, PRIMARY KEY (${primary_key.slice(0, -1)})` : ''}
                  ) ENGINE=InnoDB DEFAULT CHARSET=${data.characterSet} COLLATE =${data.collation} COMMENT='${data.tableComment}';`;
        } else if (state.activeName === '2' && data.indexs.res.length > 0) {
            // 创建索引
            let sql = `ALTER TABLE ${data.tableName}`;
            state.tableData.indexs.res.forEach((a) => {
                sql += ` ADD ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')}) USING ${a.indexType} COMMENT '${a.indexComment}',`;
            });
            return sql.substring(0, sql.length - 1) + ';';
        }
    } else {
        // 修改
        let addSql = '',
            updSql = '',
            delSql = '';
        if (state.activeName === '1') {
            // 修改列
            let changeData = filterChangedData(oldData.fields, state.tableData.fields.res, 'name');
            if (changeData.add.length > 0) {
                addSql = `ALTER TABLE ${data.tableName}`;
                changeData.add.forEach((a) => {
                    addSql += ` ADD ${genColumnBasicSql(a)},`;
                });
                addSql = addSql.substring(0, addSql.length - 1);
                addSql += ';';
            }

            if (changeData.upd.length > 0) {
                updSql = `ALTER TABLE ${data.tableName}`;
                changeData.upd.forEach((a) => {
                    updSql += ` MODIFY ${genColumnBasicSql(a)},`;
                });
                updSql = updSql.substring(0, updSql.length - 1);
                updSql += ';';
            }

            if (changeData.del.length > 0) {
                changeData.del.forEach((a) => {
                    delSql += ` ALTER TABLE ${data.tableName} DROP COLUMN ${a.name}; `;
                });
            }
            return addSql + updSql + delSql;
        } else if (state.activeName === '2') {
            // 修改索引
            let changeData = filterChangedData(oldData.indexs, state.tableData.indexs.res, 'indexName');
            // 搜集修改和删除的索引，添加到drop index xx
            // 收集新增和修改的索引，添加到ADD xx
            // ALTER TABLE `test1`
            // DROP INDEX `test1_name_uindex`,
            // DROP INDEX `test1_column_name4_index`,
            // ADD UNIQUE INDEX `test1_name_uindex`(`id`) USING BTREE COMMENT 'ASDASD',
            // ADD INDEX `111`(`column_name4`) USING BTREE COMMENT 'zasf';

            let dropIndexNames: string[] = [];
            let addIndexs: any[] = [];

            if (changeData.upd.length > 0) {
                changeData.upd.forEach((a) => {
                    dropIndexNames.push(a.indexName);
                    addIndexs.push(a);
                });
            }

            if (changeData.del.length > 0) {
                changeData.del.forEach((a) => {
                    dropIndexNames.push(a.indexName);
                });
            }

            if (changeData.add.length > 0) {
                changeData.add.forEach((a) => {
                    addIndexs.push(a);
                });
            }

            if (dropIndexNames.length > 0 || addIndexs.length > 0) {
                let sql = `ALTER TABLE ${data.tableName} `;
                if (dropIndexNames.length > 0) {
                    dropIndexNames.forEach((a) => {
                        sql += `DROP INDEX ${a},`;
                    });
                    sql = sql.substring(0, sql.length - 1);
                }

                if (addIndexs.length > 0) {
                    if (dropIndexNames.length > 0) {
                        sql += ',';
                    }
                    addIndexs.forEach((a) => {
                        sql += ` ADD ${a.unique ? 'UNIQUE' : ''} INDEX ${a.indexName}(${a.columnNames.join(',')}) USING ${a.indexType} COMMENT '${
                            a.indexComment
                        }',`;
                    });
                    sql = sql.substring(0, sql.length - 1);
                }
                return sql;
            }
        }
    }
};

const reset = () => {
    state.activeName = '1';
    formRef.value.resetFields();
    state.tableData.tableName = '';
    state.tableData.tableComment = '';
    state.tableData.fields.res = [
        {
            name: '',
            type: '',
            value: '',
            length: '',
            notNull: false,
            pri: false,
            auto_increment: false,
            remark: '',
        },
    ];
    state.tableData.indexs.res = [
        {
            indexName: '',
            columnNames: [],
            unique: false,
            indexType: 'BTREE',
            indexComment: '',
        },
    ];
};

const indexChanges = (row: any) => {
    let name = '';
    if (row.columnNames && row.columnNames.length > 0) {
        for (const column of row.columnNames) {
            name += column.replace('_', '').toLowerCase() + '_';
        }
        name = name.substring(0, name.length - 1);
    } else {
        return;
    }

    let prefix = row.unique ? 'udx_' : 'idx_';
    row.indexName = prefix + name;
};

const oldData = { indexs: [] as any[], fields: [] as any[] };
watch(
    () => props.data,
    (newValue: any) => {
        const { row, indexs, columns } = newValue;
        // 回显表名表注释
        state.tableData.tableName = row.tableName;
        state.tableData.tableComment = row.tableComment;
        // 回显列
        if (columns && Array.isArray(columns) && columns.length > 0) {
            oldData.fields = [];
            state.tableData.fields.res = [];
            // 索引列下拉选
            state.tableData.indexs.columns = [];
            columns.forEach((a) => {
                let typeObj = a.columnType.replace(')', '').split('(');
                let type = typeObj[0];
                let length = (typeObj.length > 1 && typeObj[1]) || '';
                let data = {
                    name: a.columnName,
                    type,
                    value: a.columnDefault || '',
                    length,
                    notNull: a.nullable !== 'YES',
                    pri: a.columnKey === 'PRI',
                    auto_increment: a.columnKey === 'PRI' /*a.extra?.indexOf('auto_increment') > -1*/,
                    remark: a.columnComment,
                };
                state.tableData.fields.res.push(data);
                oldData.fields.push(JSON.parse(JSON.stringify(data)));
                // 索引字段下拉选项
                state.tableData.indexs.columns.push({ name: a.columnName, remark: a.columnComment });
            });
        }
        // 回显索引
        if (indexs && Array.isArray(indexs) && indexs.length > 0) {
            oldData.indexs = [];
            state.tableData.indexs.res = [];
            // 索引过滤掉主键
            indexs
                .filter((a) => a.indexName !== 'PRIMARY')
                .forEach((a) => {
                    let data = {
                        indexName: a.indexName,
                        columnNames: a.columnName?.split(','),
                        unique: a.nonUnique === 0 || false,
                        indexType: a.indexType,
                        indexComment: a.indexComment,
                    };
                    state.tableData.indexs.res.push(data);
                    oldData.indexs.push(JSON.parse(JSON.stringify(data)));
                });
        }
    }
);
</script>
