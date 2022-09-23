<template>
    <div class="file-manage">
        <el-dialog :title="title" v-model="dialogVisible" :destroy-on-close="true" :show-close="true" :before-close="handleClose" width="60%">
            <div class="toolbar">
                <div style="float: left">
                    <el-select v-model="type" @change="getScripts" size="small" placeholder="请选择">
                        <el-option :key="0" label="私有" :value="0"> </el-option>
                        <el-option :key="1" label="公共" :value="1"> </el-option>
                    </el-select>
                </div>
                <div style="float: right">
                    <el-button @click="editScript(currentData)" :disabled="currentId == null" type="primary" icon="tickets" size="small" plain
                        >查看</el-button
                    >
                    <el-button v-auth="'machine:script:save'" type="primary" @click="editScript(null)" icon="plus" size="small" plain>添加</el-button>
                    <el-button
                        v-auth="'machine:script:del'"
                        :disabled="currentId == null"
                        type="danger"
                        @click="deleteRow(currentData)"
                        icon="delete"
                        size="small"
                        plain
                        >删除</el-button
                    >
                </div>
            </div>

            <el-table :data="scriptTable" @current-change="choose" stripe border size="small" style="width: 100%">
                <el-table-column label="选择" width="55px">
                    <template #default="scope">
                        <el-radio v-model="currentId" :label="scope.row.id">
                            <i></i>
                        </el-radio>
                    </template>
                </el-table-column>
                <el-table-column prop="name" label="名称" :min-width="70"> </el-table-column>
                <el-table-column prop="description" label="描述" :min-width="100" show-overflow-tooltip></el-table-column>
                <el-table-column prop="name" label="类型" :min-width="50">
                    <template #default="scope">
                        {{ enums.scriptTypeEnum.getLabelByValue(scope.row.type) }}
                    </template>
                </el-table-column>
                <el-table-column label="操作">
                    <template #default="scope">
                        <el-button v-if="scope.row.id == null" @click="addFiles(scope.row)" type="success" icon="el-icon-success" size="small" plain
                            >确定</el-button
                        >

                        <el-button
                            v-auth="'machine:script:run'"
                            v-if="scope.row.id != null"
                            @click="runScript(scope.row)"
                            type="primary"
                            icon="video-play"
                            size="small"
                            plain
                            >执行</el-button
                        >
                    </template>
                </el-table-column>
            </el-table>
            <el-row style="margin-top: 10px" type="flex" justify="end">
                <el-pagination
                    small
                    style="text-align: center"
                    :total="total"
                    layout="prev, pager, next, total, jumper"
                    v-model:current-page="query.pageNum"
                    :page-size="query.pageSize"
                    @current-change="handlePageChange"
                ></el-pagination>
            </el-row>
        </el-dialog>

        <el-dialog title="脚本参数" v-model="scriptParamsDialog.visible" width="400px">
            <el-form ref="paramsForm" :model="scriptParamsDialog.params" label-width="70px" size="small">
                <el-form-item v-for="item in scriptParamsDialog.paramsFormItem" :key="item.name" :prop="item.model" :label="item.name" required>
                    <el-input
                        v-if="!item.options"
                        v-model="scriptParamsDialog.params[item.model]"
                        :placeholder="item.placeholder"
                        autocomplete="off"
                        clearable
                    ></el-input>
                    <el-select
                        v-else
                        v-model="scriptParamsDialog.params[item.model]"
                        :placeholder="item.placeholder"
                        filterable
                        autocomplete="off"
                        clearable
                        style="width: 100%"
                    >
                        <el-option v-for="option in item.options.split(',')" :key="option" :label="option" :value="option" />
                    </el-select>
                </el-form-item>
            </el-form>
            <template #footer>
                <span class="dialog-footer">
                    <el-button type="primary" @click="hasParamsRun(currentData)" size="small">确 定</el-button>
                </span>
            </template>
        </el-dialog>

        <el-dialog title="执行结果" v-model="resultDialog.visible" width="50%">
            <div style="white-space: pre-line; padding: 10px; color: #000000">
                <el-input v-model="resultDialog.result" :rows="20" type="textarea" />
            </div>
        </el-dialog>

        <el-dialog
            v-if="terminalDialog.visible"
            title="终端"
            v-model="terminalDialog.visible"
            width="80%"
            :close-on-click-modal="false"
            :modal="false"
            @close="closeTermnial"
        >
            <ssh-terminal ref="terminal" :cmd="terminalDialog.cmd" :machineId="terminalDialog.machineId" height="560px" />
        </el-dialog>

        <script-edit
            v-model:visible="editDialog.visible"
            v-model:data="editDialog.data"
            :title="editDialog.title"
            v-model:machineId="editDialog.machineId"
            :isCommon="type == 1"
            @submitSuccess="submitSuccess"
        />
    </div>
</template>

<script lang="ts">
import { ref, toRefs, reactive, watch, defineComponent } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import SshTerminal from './SshTerminal.vue';
import { machineApi } from './api';
import enums from './enums';
import ScriptEdit from './ScriptEdit.vue';

export default defineComponent({
    name: 'ServiceManage',
    components: {
        ScriptEdit,
        SshTerminal,
    },
    props: {
        visible: { type: Boolean },
        machineId: { type: Number },
        title: { type: String },
    },
    setup(props: any, context) {
        const paramsForm: any = ref(null);
        const state = reactive({
            dialogVisible: false,
            type: 0,
            currentId: null,
            currentData: null,
            query: {
                machineId: 0,
                pageNum: 1,
                pageSize: 8,
            },
            editDialog: {
                visible: false,
                data: null,
                title: '',
                machineId: 9999999,
            },
            total: 0,
            scriptTable: [],
            scriptParamsDialog: {
                visible: false,
                params: {},
                paramsFormItem: [],
            },
            resultDialog: {
                visible: false,
                result: '',
            },
            terminalDialog: {
                visible: false,
                cmd: '',
                machineId: 0,
            },
        });

        watch(props, async (newValue) => {
            if (props.machineId && newValue.visible) {
                await getScripts();
            }
            state.dialogVisible = newValue.visible;
        });

        const getScripts = async () => {
            state.currentId = null;
            state.currentData = null;
            state.query.machineId = state.type == 0 ? props.machineId : 9999999;
            const res = await machineApi.scripts.request(state.query);
            state.scriptTable = res.list;
            state.total = res.total;
        };

        const handlePageChange = (curPage: number) => {
            state.query.pageNum = curPage;
            getScripts();
        };

        const runScript = async (script: any) => {
            // 如果存在参数，则弹窗输入参数后执行
            if (script.params) {
                state.scriptParamsDialog.paramsFormItem = JSON.parse(script.params);
                if (state.scriptParamsDialog.paramsFormItem && state.scriptParamsDialog.paramsFormItem.length > 0) {
                    state.scriptParamsDialog.visible = true;
                    return;
                }
            }

            run(script);
        };

        // 有参数的脚本执行函数
        const hasParamsRun = async (script: any) => {
            // 如果脚本参数弹窗显示，则校验参数表单数据通过后执行
            if (state.scriptParamsDialog.visible) {
                paramsForm.value.validate((valid: any) => {
                    if (valid) {
                        run(script);
                        state.scriptParamsDialog.params = {};
                        state.scriptParamsDialog.visible = false;
                        paramsForm.value.resetFields();
                    } else {
                        return false;
                    }
                });
            }
        };

        const run = async (script: any) => {
            const noResult = script.type == enums.scriptTypeEnum['NO_RESULT'].value;
            // 如果脚本类型为有结果类型，则显示结果信息
            if (script.type == enums.scriptTypeEnum['RESULT'].value || noResult) {
                const res = await machineApi.runScript.request({
                    machineId: props.machineId,
                    scriptId: script.id,
                    params: state.scriptParamsDialog.params,
                });

                if (noResult) {
                    ElMessage.success('执行完成');
                    return;
                }
                state.resultDialog.result = res;
                state.resultDialog.visible = true;
                return;
            }

            if (script.type == enums.scriptTypeEnum['REAL_TIME'].value) {
                script = script.script;
                if (state.scriptParamsDialog.params) {
                    script = templateResolve(script, state.scriptParamsDialog.params);
                }
                state.terminalDialog.cmd = script;
                state.terminalDialog.visible = true;
                state.terminalDialog.machineId = props.machineId;
                return;
            }
        };

        /**
         * 解析 {{.param}} 形式模板字符串
         */
        function templateResolve(template: string, param: any) {
            return template.replace(/\{{.\w+\}}/g, (word) => {
                const key = word.substring(3, word.length - 2);
                const value = param[key];
                if (value != null || value != undefined) {
                    return value;
                }
                return '';
            });
        }

        const closeTermnial = () => {
            state.terminalDialog.visible = false;
            state.terminalDialog.machineId = 0;
        };

        /**
         * 选择数据
         */
        const choose = (item: any) => {
            if (!item) {
                return;
            }
            state.currentId = item.id;
            state.currentData = item;
        };

        const editScript = (data: any) => {
            state.editDialog.machineId = props.machineId;
            state.editDialog.data = data;
            if (data) {
                state.editDialog.title = '查看编辑脚本';
            } else {
                state.editDialog.title = '新增脚本';
            }
            state.editDialog.visible = true;
        };

        const submitSuccess = () => {
            getScripts();
        };

        const deleteRow = (row: any) => {
            ElMessageBox.confirm(`此操作将删除 [${row.name}], 是否继续?`, '提示', {
                confirmButtonText: '确定',
                cancelButtonText: '取消',
                type: 'warning',
            }).then(() => {
                machineApi.deleteScript
                    .request({
                        machineId: props.machineId,
                        scriptId: row.id,
                    })
                    .then(() => {
                        getScripts();
                    });
                // 删除配置文件
            });
        };

        /**
         * 关闭取消按钮触发的事件
         */
        const handleClose = () => {
            context.emit('update:visible', false);
            context.emit('update:machineId', null);
            context.emit('cancel');
            state.scriptTable = [];
            state.scriptParamsDialog.paramsFormItem = [];
        };

        return {
            ...toRefs(state),
            paramsForm,
            enums,
            getScripts,
            handlePageChange,
            runScript,
            hasParamsRun,
            closeTermnial,
            choose,
            editScript,
            submitSuccess,
            deleteRow,
            handleClose,
        };
    },
});
</script>
<style lang="sass">
</style>
