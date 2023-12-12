<template>
    <div class="file-manage">
        <el-dialog
            @open="getScripts()"
            :title="title"
            v-model="dialogVisible"
            :destroy-on-close="true"
            :show-close="true"
            :before-close="handleClose"
            width="55%"
        >
            <page-table
                ref="pageTableRef"
                :page-api="machineApi.scripts"
                :before-query-fn="checkScriptType"
                :lazy="true"
                :search-items="state.searchItems"
                v-model:query-form="query"
                :columns="columns"
                :show-selection="true"
                v-model:selection-data="selectionData"
            >
                <template #tableHeader>
                    <el-button v-auth="'machine:script:save'" type="primary" @click="editScript(null)" icon="plus" plain>添加</el-button>
                    <el-button
                        v-auth="'machine:script:del'"
                        :disabled="selectionData.length < 1"
                        type="danger"
                        @click="deleteRow(selectionData)"
                        icon="delete"
                        plain
                        >删除</el-button
                    >
                </template>

                <template #action="{ data }">
                    <el-button v-auth="'machine:script:run'" v-if="data.id != null" @click="runScript(data)" type="primary" icon="video-play" link
                        >执行
                    </el-button>

                    <el-button @click="editScript(data)" type="primary" icon="tickets" link>查看 </el-button>
                </template>
            </page-table>
        </el-dialog>

        <dynamic-form-dialog
            title="脚本参数"
            width="400px"
            v-model:visible="scriptParamsDialog.visible"
            ref="paramsForm"
            :form-items="scriptParamsDialog.paramsFormItem"
            v-model="scriptParamsDialog.params"
            @confirm="hasParamsRun"
        >
        </dynamic-form-dialog>

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
            draggable
            append-to-body
        >
            <TerminalBody ref="terminal" :cmd="terminalDialog.cmd" :socket-url="getMachineTerminalSocketUrl(terminalDialog.machineId)" height="560px" />
        </el-dialog>

        <script-edit
            v-model:visible="editDialog.visible"
            v-model:data="editDialog.data"
            :title="editDialog.title"
            v-model:machineId="editDialog.machineId"
            :isCommon="state.query.type == ScriptTypeEnum.Public.value"
            @submitSuccess="submitSuccess"
        />
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch, Ref } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import TerminalBody from '@/components/terminal/TerminalBody.vue';
import { getMachineTerminalSocketUrl, machineApi } from './api';
import { ScriptResultEnum, ScriptTypeEnum } from './enums';
import ScriptEdit from './ScriptEdit.vue';
import PageTable from '@/components/pagetable/PageTable.vue';
import { TableColumn } from '@/components/pagetable';
import { DynamicFormDialog } from '@/components/dynamic-form';
import { SearchItem } from '@/components/SearchForm';

const props = defineProps({
    visible: { type: Boolean },
    machineId: { type: Number },
    title: { type: String },
});

const emit = defineEmits(['update:visible', 'cancel', 'update:machineId']);

const paramsForm: any = ref(null);
const pageTableRef: Ref<any> = ref(null);

const state = reactive({
    dialogVisible: false,
    selectionData: [],
    searchItems: [SearchItem.select('type', '类型').withEnum(ScriptTypeEnum)],
    columns: [
        TableColumn.new('name', '名称'),
        TableColumn.new('description', '描述'),
        TableColumn.new('type', '类型').isEnum(ScriptResultEnum),
        TableColumn.new('action', '操作').isSlot().setMinWidth(130).alignCenter(),
    ],
    query: {
        machineId: 0 as any,
        type: ScriptTypeEnum.Private.value,
        pageNum: 1,
        pageSize: 6,
    },
    editDialog: {
        visible: false,
        data: null as any,
        title: '',
        machineId: 9999999,
    },
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

const { dialogVisible, columns, selectionData, query, editDialog, scriptParamsDialog, resultDialog, terminalDialog } = toRefs(state);

watch(props, async (newValue) => {
    state.dialogVisible = newValue.visible;
});

const getScripts = async () => {
    pageTableRef.value.search();
};

const checkScriptType = (query: any) => {
    if (!query.type) {
        query.machineId = props.machineId;
        query.type = ScriptTypeEnum.Private.value;
    } else {
        query.machineId = query.type == ScriptTypeEnum.Private.value ? props.machineId : 9999999;
    }

    return query;
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
    await run(state.scriptParamsDialog.script);
    state.scriptParamsDialog.visible = false;
    state.scriptParamsDialog.script = null;
};

const run = async (script: any) => {
    const noResult = script.type == ScriptResultEnum.NoResult.value;
    // 如果脚本类型为有结果类型，则显示结果信息
    if (script.type == ScriptResultEnum.Result.value || noResult) {
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

    if (script.type == ScriptResultEnum.RealTime.value) {
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
    ElMessageBox.confirm(`此操作将删除【${rows.map((x: any) => x.name).join(', ')}】脚本信息, 是否继续?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    }).then(() => {
        machineApi.deleteScript
            .request({
                machineId: props.machineId,
                scriptId: rows.map((x: any) => x.id).join(','),
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
    state.query.type = ScriptTypeEnum.Private.value;
    state.scriptParamsDialog.paramsFormItem = [];
};
</script>
<style lang="scss"></style>
