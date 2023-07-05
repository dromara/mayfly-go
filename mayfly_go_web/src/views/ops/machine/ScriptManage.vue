<template>
    <div class="file-manage">
        <el-dialog @open="getScripts()" :title="title" v-model="dialogVisible" :destroy-on-close="true" :show-close="true"
            :before-close="handleClose" width="55%">

            <page-table ref="pageTableRef" :query="queryConfig" v-model:query-form="query" :data="scriptTable"
                :columns="columns" :total="total" v-model:page-size="query.pageSize" v-model:page-num="query.pageNum"
                @pageChange="getScripts()" :show-selection="true" v-model:selection-data="selectionData">

                <template #typeSelect>
                    <el-select v-model="type" placeholder="请选择">
                        <el-option :key="0" label="私有" :value="0"> </el-option>
                        <el-option :key="1" label="公共" :value="1"> </el-option>
                    </el-select>
                </template>

                <template #type="{ data }">
                    {{ enums.scriptTypeEnum.getLabelByValue(data.type) }}
                </template>

                <template #queryRight>
                    <el-button v-auth="'machine:script:save'" type="primary" @click="editScript(null)" icon="plus"
                        plain>添加</el-button>
                    <el-button v-auth="'machine:script:del'" :disabled="selectionData.length < 1" type="danger"
                        @click="deleteRow(selectionData)" icon="delete" plain>删除</el-button>
                </template>

                <template #action="{ data }">
                    <el-button v-auth="'machine:script:run'" v-if="data.id != null" @click="runScript(data)" type="primary"
                        icon="video-play" link>执行
                    </el-button>

                    <el-button @click="editScript(data)" type="primary" icon="tickets" link>查看
                    </el-button>
                </template>
            </page-table>

        </el-dialog>

        <el-dialog title="脚本参数" v-model="scriptParamsDialog.visible" width="400px">
            <el-form ref="paramsForm" :model="scriptParamsDialog.params" label-width="auto">
                <el-form-item v-for="item in scriptParamsDialog.paramsFormItem as any" :key="item.name" :prop="item.model"
                    :label="item.name" required>
                    <el-input v-if="!item.options" v-model="scriptParamsDialog.params[item.model]"
                        :placeholder="item.placeholder" autocomplete="off" clearable></el-input>
                    <el-select v-else v-model="scriptParamsDialog.params[item.model]" :placeholder="item.placeholder"
                        filterable autocomplete="off" clearable style="width: 100%">
                        <el-option v-for="option in item.options.split(',')" :key="option" :label="option"
                            :value="option" />
                    </el-select>
                </el-form-item>
            </el-form>
            <template #footer>
                <span class="dialog-footer">
                    <el-button type="primary" @click="hasParamsRun()">确 定</el-button>
                </span>
            </template>
        </el-dialog>

        <el-dialog title="执行结果" v-model="resultDialog.visible" width="50%">
            <div style="white-space: pre-line; padding: 10px; color: #000000">
                <el-input v-model="resultDialog.result" :rows="20" type="textarea" />
            </div>
        </el-dialog>

        <el-dialog v-if="terminalDialog.visible" title="终端" v-model="terminalDialog.visible" width="80%"
            :close-on-click-modal="false" :modal="false" @close="closeTermnial">
            <ssh-terminal ref="terminal" :cmd="terminalDialog.cmd" :machineId="terminalDialog.machineId" height="560px" />
        </el-dialog>

        <script-edit v-model:visible="editDialog.visible" v-model:data="editDialog.data" :title="editDialog.title"
            v-model:machineId="editDialog.machineId" :isCommon="type == 1" @submitSuccess="submitSuccess" />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import SshTerminal from './SshTerminal.vue';
import { machineApi } from './api';
import enums from './enums';
import ScriptEdit from './ScriptEdit.vue';
import PageTable from '@/components/pagetable/PageTable.vue'
import { TableColumn, TableQuery } from '@/components/pagetable';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
})

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId'])

const paramsForm: any = ref(null);
const pageTableRef: any = ref(null);

const state = reactive({
    dialogVisible: false,
    type: 0,
    selectionData: [],
    queryConfig: [
        TableQuery.slot("type", "类型", "typeSelect"),
    ],
    columns: [
        TableColumn.new("name", "名称"),
        TableColumn.new("description", "描述"),
        TableColumn.new("type", "类型").isSlot().setAddWidth(5),
        TableColumn.new("action", "操作").isSlot().setMinWidth(130).alignCenter(),
    ],
    query: {
        machineId: 0 as any,
        pageNum: 1,
        pageSize: 6,
    },
    editDialog: {
        visible: false,
        data: null as any,
        title: '',
        machineId: 9999999,
    },
    total: 0,
    scriptTable: [],
    scriptParamsDialog: {
        script: null,
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

const {
    dialogVisible,
    queryConfig,
    columns,
    type,
    selectionData,
    query,
    editDialog,
    total,
    scriptTable,
    scriptParamsDialog,
    resultDialog,
    terminalDialog,
} = toRefs(state)

watch(props, async (newValue) => {
    state.dialogVisible = newValue.visible;
});

const getScripts = async () => {
    try {
        // 通过open事件才开获取到pageTableRef值
        pageTableRef.value.loading(true)
        state.query.machineId = state.type == 0 ? props.machineId : 9999999;
        const res = await machineApi.scripts.request(state.query);
        state.scriptTable = res.list;
        state.total = res.total;
    } finally {
        pageTableRef.value.loading(false)
    }
};

const runScript = async (script: any) => {
    // 如果存在参数，则弹窗输入参数后执行
    if (script.params) {
        state.scriptParamsDialog.paramsFormItem = JSON.parse(script.params);
        if (state.scriptParamsDialog.paramsFormItem && state.scriptParamsDialog.paramsFormItem.length > 0) {
            state.scriptParamsDialog.visible = true;
            state.scriptParamsDialog.script = script;
            return;
        }
    }

    run(script);
};

// 有参数的脚本执行函数
const hasParamsRun = async () => {
    // 如果脚本参数弹窗显示，则校验参数表单数据通过后执行
    if (state.scriptParamsDialog.visible) {
        paramsForm.value.validate((valid: any) => {
            if (valid) {
                run(state.scriptParamsDialog.script);
                state.scriptParamsDialog.params = {};
                state.scriptParamsDialog.visible = false;
                state.scriptParamsDialog.script = null;
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
            params: JSON.stringify(state.scriptParamsDialog.params),
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
        state.terminalDialog.machineId = props.machineId as any;
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


const editScript = (data: any) => {
    state.editDialog.machineId = props.machineId as any;
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

const deleteRow = (rows: any) => {
    ElMessageBox.confirm(`此操作将删除【${rows.map((x: any) => x.name).join(", ")}】脚本信息, 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(() => {
        machineApi.deleteScript
            .request({
                machineId: props.machineId,
                scriptId: rows.map((x: any) => x.id).join(","),
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
    emit('update:visible', false);
    emit('update:machineId', null);
    emit('cancel');
    state.type = 0;
    state.scriptTable = [];
    state.scriptParamsDialog.paramsFormItem = [];
};
</script>
<style lang="sass">
</style>
