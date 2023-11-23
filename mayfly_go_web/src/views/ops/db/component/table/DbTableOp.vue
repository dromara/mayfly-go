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
                </el-row>

                <el-tabs v-model="activeName">
                    <el-tab-pane label="字段" name="1">
                        <el-table :data="tableData.fields.res" :max-height="tableData.height">
                            <el-table-column :prop="item.prop" :label="item.label" v-for="item in tableData.fields.colNames" :key="item.prop">
                                <template #default="scope">
                                    <el-input v-if="item.prop === 'name'" size="small" v-model="scope.row.name"> </el-input>

                                    <el-select v-else-if="item.prop === 'type'" filterable size="small" v-model="scope.row.type">
                                        <el-option v-for="pgsqlType in state.columnTypeList" :key="pgsqlType.dataType" :value="pgsqlType.udtName" :label="pgsqlType.dataType">
                                            <span v-if="pgsqlType.dataType === pgsqlType.udtName">{{ pgsqlType.dataType }}{{ pgsqlType.desc && '：' + pgsqlType.desc }}</span>
                                            <span v-else>{{ pgsqlType.dataType }}，别名：{{ pgsqlType.udtName }} {{ pgsqlType.desc }}</span>
                                        </el-option>
                                    </el-select>

                                    <el-input v-else-if="item.prop === 'value'" size="small" v-model="scope.row.value"> </el-input>

                                    <el-input v-else-if="item.prop === 'length'" size="small" v-model="scope.row.length"> </el-input>

                                    <el-input v-else-if="item.prop === 'numScale'" size="small" v-model="scope.row.numScale"> </el-input>

                                    <el-checkbox v-else-if="item.prop === 'notNull'" size="small" v-model="scope.row.notNull"> </el-checkbox>

                                    <el-checkbox v-else-if="item.prop === 'pri'" size="small" v-model="scope.row.pri"> </el-checkbox>

                                    <el-checkbox v-else-if="item.prop === 'auto_increment'" size="small" v-model="scope.row.auto_increment" :disabled="dbType === DbType.postgresql"> </el-checkbox>

                                    <el-input v-else-if="item.prop === 'remark'" size="small" v-model="scope.row.remark"> </el-input>

                                    <el-link v-else-if="item.prop === 'action'" type="danger" plain size="small" :underline="false" @click.prevent="deleteRow(scope.$index)">删除</el-link>
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

                                    <el-select v-if="item.prop === 'columnNames'" v-model="scope.row.columnNames" multiple collapse-tags collapse-tags-tooltip filterable placeholder="请选择字段" @change="indexChanges(scope.row)" style="width: 100%">
                                        <el-option v-for="cl in tableData.indexs.columns" :key="cl.name" :label="cl.name" :value="cl.name">
                                            {{ cl.name + ' - ' + (cl.remark || '') }}
                                        </el-option>
                                    </el-select>

                                    <el-checkbox v-if="item.prop === 'unique'" size="small" v-model="scope.row.unique" @change="indexChanges(scope.row)"> </el-checkbox>

                                    <el-select v-if="item.prop === 'indexType'" filterable size="small" v-model="scope.row.indexType">
                                        <el-option v-for="typeValue in indexTypeList" :key="typeValue" :value="typeValue">{{ typeValue }}</el-option>
                                    </el-select>

                                    <el-input v-if="item.prop === 'indexComment'" size="small" v-model="scope.row.indexComment"> </el-input>

                                    <el-link v-if="item.prop === 'action'" type="danger" plain size="small" :underline="false" @click.prevent="deleteIndex(scope.$index)">删除</el-link>
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
import { reactive, ref, toRefs, watch } from 'vue';
import { getDbOption } from './service';
import { ElMessage } from 'element-plus';
import SqlExecBox from '../sqleditor/SqlExecBox';
import { DbType } from '@/views/ops/db/component/table/dbs/db-option';

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
    dbType: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change', 'submit-sql']);

const dbOption = getDbOption(props.dbType);

const formRef: any = ref();
const state = reactive({
    dialogVisible: false,
    btnloading: false,
    activeName: '1',
    columnTypeList: dbOption.getTypeList(),
    indexTypeList: ['BTREE'], // mysql索引类型详解 http://c.biancheng.net/view/7897.html
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
                    prop: 'numScale',
                    label: '小数点',
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
            res: [] as {
                name: string;
                type: string;
                value: string;
                length: string;
                numScale: string;
                notNull: boolean;
                pri: boolean;
                auto_increment: boolean;
                remark: string;
            }[],
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
            res: [] as {
                indexName: string;
                columnNames: string[];
                unique: boolean;
                indexType: 'BTREE';
                indexComment: string;
            }[],
        },
        tableName: '',
        tableComment: '',
        height: 550,
    },
});

const { dialogVisible, btnloading, activeName, indexTypeList, tableData } = toRefs(state);

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
        numScale: '',
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
    if (props.dbType === DbType.mysql) {
        state.tableData.fields.res.push(
            { name: 'id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            { name: 'creator', type: 'varchar', length: '100', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人姓名' },
            { name: 'create_time', type: 'datetime', length: '', numScale: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '创建时间' },
            { name: 'updator_id', type: 'bigint', length: '20', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            { name: 'updator', type: 'varchar', length: '100', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人姓名' },
            { name: 'update_time', type: 'datetime', length: '', numScale: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '修改时间' }
        );
    } else if (props.dbType === DbType.postgresql) {
        state.tableData.fields.res.push(
            { name: 'id', type: 'bigserial', length: '', numScale: '', value: '', notNull: true, pri: true, auto_increment: true, remark: '主键ID' },
            { name: 'creator_id', type: 'int8', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人id' },
            { name: 'creator', type: 'varchar', length: '100', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '创建人姓名' },
            { name: 'create_time', type: 'timestamp', length: '', numScale: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '创建时间' },
            { name: 'updator_id', type: 'int8', length: '', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人id' },
            { name: 'updator', type: 'varchar', length: '100', numScale: '', value: '', notNull: true, pri: false, auto_increment: false, remark: '修改人姓名' },
            { name: 'update_time', type: 'timestamp', length: '', numScale: '', value: 'CURRENT_TIMESTAMP', notNull: true, pri: false, auto_increment: false, remark: '修改时间' }
        );
    }
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
    let data = state.tableData;
    // 创建表
    if (!props.data?.edit) {
        if (state.activeName === '1') {
            return dbOption.getCreateTableSql(data);
        } else if (state.activeName === '2' && data.indexs.res.length > 0) {
            return dbOption.getCreateIndexSql(data);
        }
    } else {
        // 修改
        if (state.activeName === '1') {
            // 修改列
            let changeData = filterChangedData(oldData.fields, state.tableData.fields.res, 'name');
            return dbOption.getModifyColumnSql(data.tableName, changeData);
        } else if (state.activeName === '2') {
            // 修改索引
            let changeData = filterChangedData(oldData.indexs, state.tableData.indexs.res, 'indexName');
            return dbOption.getModifyIndexSql(data.tableName, changeData);
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
            numScale: '',
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

    let suffix = row.unique ? 'udx' : 'idx';
    let commentSuffix = row.unique ? '唯一索引' : '普通索引';
    // 以表名为前缀
    row.indexName = `${tableData.value.tableName}_${name}_${suffix}`.replaceAll(' ', '');
    row.indexComment = `${tableData.value.tableName}表(${name.replaceAll('_', ',')})${commentSuffix}`;
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
                    numScale: a.numScale,
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
