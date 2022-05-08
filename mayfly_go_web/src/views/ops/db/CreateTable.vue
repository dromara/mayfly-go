<template>
    <div>
        <el-dialog title="创建表" v-model="dialogVisible" :before-close="cancel" width="90%">
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
                    <el-col style="margin-top: 20px" :span="12">
                        <el-form-item prop="characterSet" label="字符集">
                            <el-select filterable style="width: 80%" v-model="tableData.characterSet" size="small">
                                <el-option v-for="item in characterSetNameList" :key="item" :label="item" :value="item"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                </el-row>

                <el-tabs v-model="activeName">
                    <el-tab-pane label="字段" name="1">
                        <el-table :data="tableData.fields.res">
                            <el-table-column :prop="item.prop" :label="item.label" v-for="item in tableData.fields.colNames" :key="item.prop">
                                <template #default="scope">
                                    <el-input v-if="item.prop === 'name'" size="small" v-model="scope.row.name"></el-input>

                                    <el-select v-if="item.prop === 'type'" filterable size="small" v-model="scope.row.type">
                                        <el-option v-for="typeValue in typeList" :key="typeValue" :value="typeValue">{{ typeValue }}</el-option>
                                    </el-select>

                                    <el-input v-if="item.prop === 'value'" size="small" v-model="scope.row.value"> </el-input>

                                    <el-input v-if="item.prop === 'length'" size="small" v-model="scope.row.length"> </el-input>

                                    <el-checkbox v-if="item.prop === 'notNull'" size="small" v-model="scope.row.notNull"> </el-checkbox>

                                    <el-checkbox v-if="item.prop === 'pri'" size="small" v-model="scope.row.pri"> </el-checkbox>

                                     <el-checkbox v-if="item.prop === 'auto_increment'" size="small" v-model="scope.row.auto_increment"> </el-checkbox>

                                     <el-input v-if="item.prop === 'remark'" size="small" v-model="scope.row.remark"> </el-input>

                                     <el-button v-if="item.prop === 'action'" type="text" size="small" @click.prevent="deleteRow(scope.$index)">删除</el-button>
                                </template>
                            </el-table-column>
                        </el-table>
                        <el-row style="margin-top: 20px">
                            <el-button @click="addRow()" type="text" icon="plus"></el-button>
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


<script lang="ts">
import { watch, toRefs, reactive, defineComponent, ref, getCurrentInstance } from 'vue';
import { TYPE_LIST, CHARACTER_SET_NAME_LIST } from './service.ts';
import { ElMessage } from 'element-plus';
import SqlExecBox from './component/SqlExecBox.ts';
export default defineComponent({
    name: 'createTable',
    props: {
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
        }
    },
    setup(props: any, { emit }) {
        const formRef: any = ref();
        const { proxy } = getCurrentInstance() as any;
        const state = reactive({
            dialogVisible: false,
            btnloading: false,
            activeName: '1',
            typeList: TYPE_LIST,
            characterSetNameList: CHARACTER_SET_NAME_LIST,
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
                characterSet: 'utf8mb4',
                tableName: '',
                tableComment: '',
            },
        });

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
        const deleteRow = (index: any) => {
            state.tableData.fields.res.splice(index, 1);
        };
        const submit = async () => {
            let data = state.tableData;
            let primary_key = '';
            let fields: string[] = [];
            data.fields.res.forEach((item) => {
                fields.push(
                    `${item.name} ${item.type}${+item.length > 0 ? `(${item.length})` : ''} ${item.notNull ? 'NOT NULL' : ''} ${
                        item.auto_increment ? 'AUTO_INCREMENT' : ''
                    } ${item.value ? 'DEFAULT ' + item.value : item.notNull ? '' : 'DEFAULT NULL'} ${
                        item.remark ? `COMMENT '${item.remark}'` : ''
                    } \n`
                );
                if (item.pri) {
                    primary_key += `${item.name},`;
                }
            });

            let sql = `
                CREATE TABLE ${data.tableName} (
                ${fields.join(',')}
                ${primary_key ? `, PRIMARY KEY (${primary_key.slice(0, -1)})` : ''}
                ) ENGINE=InnoDB DEFAULT CHARSET=${data.characterSet} COLLATE=utf8mb4_bin COMMENT='${data.tableComment}';`;

            SqlExecBox({
                sql: sql,
                dbId: props.dbId as any,
                db: props.db,
                runSuccessCallback: () => {
                    ElMessage.success('创建成功');
                    proxy.$parent.tableInfo({ id: props.dbId });
                    cancel();
                },
            });
        };
        const reset = () => {
            formRef.value.resetFields();
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
        };
        return {
            ...toRefs(state),
            formRef,
            cancel,
            reset,
            addRow,
            deleteRow,
            submit,
        };
    },
});
</script>

