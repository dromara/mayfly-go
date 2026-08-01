<template>
    <div>
        <el-dialog
            @open="getScripts()"
            :title="title"
            v-model="dialogVisible"
            :destroy-on-close="true"
            :show-close="true"
            :before-close="handleClose"
            width="60%"
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
                    <el-button v-auth="'machine:script:save'" type="primary" @click="editScript(null)" icon="plus" plain>{{ $t('common.create') }}</el-button>
                    <el-button
                        v-auth="'machine:script:del'"
                        :disabled="selectionData.length < 1"
                        type="danger"
                        @click="deleteRow(selectionData)"
                        icon="delete"
                        plain
                        >{{ $t('common.delete') }}</el-button
                    >
                </template>

                <template #action="{ data }">
                    <el-button v-auth="'machine:script:run'" v-if="data.id != null" @click="runScript(data)" type="primary" icon="video-play" link
                        >{{ $t('machine.execute') }}
                    </el-button>

                    <el-button @click="editScript(data)" type="primary" icon="tickets" link>{{ $t('common.detail') }}</el-button>
                </template>
            </page-table>
        </el-dialog>

        <dynamic-form-dialog
            :title="$t('machine.scriptParam')"
            width="400px"
            v-model:visible="scriptParamsDialog.visible"
            ref="paramsForm"
            :form-items="scriptParamsDialog.paramsFormItem"
            v-model="scriptParamsDialog.params"
            @confirm="hasParamsRun"
        >
        </dynamic-form-dialog>

        <el-dialog :title="$t('machine.execResult')" v-model="resultDialog.visible" width="50%">
            <div style="white-space: pre-line; padding: 10px; color: #000000">
                <el-input v-model="resultDialog.result" :rows="20" type="textarea" />
            </div>
        </el-dialog>

        <el-dialog
            v-if="terminalDialog.visible"
            title="Terminal"
            v-model="terminalDialog.visible"
            width="80%"
            :close-on-click-modal="false"
            :modal="false"
            @close="closeTerminal"
            body-class="h-[65vh]"
            draggable
            append-to-body
        >
            <TerminalBody
                ref="terminal"
                :cmd="terminalDialog.cmd"
                :socket-url="getMachineTerminalSocketUrl(props.authCertName ?? '')"
                :machine-id="machineId ?? undefined"
                :auth-cert-name="props.authCertName"
                :file-id="0"
                :protocol="1"
            />
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
import { DynamicFormDialog } from '@/components/dynamic-form';
import { TableColumn } from '@/components/page-table';
import PageTable from '@/components/page-table/PageTable.vue';
import { SearchItem, OptionsApi } from '@/components/page-table/SearchForm';
import { Msg, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, nextTick, onMounted, reactive, ref, toRefs, watch } from 'vue';
import { getMachineTerminalSocketUrl, machineApi } from './api';
import { ScriptResultEnum, ScriptTypeEnum } from './enums';
import type { MachineScriptVO } from './types';

const ScriptEdit = defineAsyncComponent(() => import('./ScriptEdit.vue'));
const TerminalBody = defineAsyncComponent(() => import('@/components/terminal/TerminalBody.vue'));

const props = defineProps({
    authCertName: { type: String },
    title: { type: String },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });
const machineId = defineModel<number | null>('machineId');

const emit = defineEmits(['cancel']);

const paramsForm = ref<InstanceType<typeof DynamicFormDialog> | null>(null);
const pageTableRef = ref<InstanceType<typeof PageTable> | null>(null);

const state = reactive({
    selectionData: [],
    searchItems: [
        SearchItem.select('type', 'common.type').withEnum(ScriptTypeEnum),
        SearchItem.select('category', 'machine.category').withOptionsApi(
            OptionsApi.new(machineApi.scriptCategorys, {}).withConvertFn((res) => {
                return res.map((x: string) => {
                    return {
                        label: x,
                        value: x,
                    };
                });
            })
        ),
    ],
    columns: [
        TableColumn.new('name', 'common.name'),
        TableColumn.new('description', 'common.remark'),
        TableColumn.new('type', 'common.type').typeTag(ScriptResultEnum),
        TableColumn.new('category', 'machine.category'),
        TableColumn.new('action', 'common.operation').isSlot().setMinWidth(140).alignCenter(),
    ],
    query: {
        machineId: null as number | null,
        type: ScriptTypeEnum.Private.value as number,
        pageNum: 1,
        pageSize: 6,
    },
    editDialog: {
        visible: false,
        data: null as MachineScriptVO | null,
        title: '',
        machineId: 9999999 as number | undefined,
    },
    scriptParamsDialog: {
        script: null as MachineScriptVO | null,
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
    },
});

const { columns, selectionData, query, editDialog, scriptParamsDialog, resultDialog, terminalDialog } = toRefs(state);

const getScripts = async () => {
    pageTableRef.value?.search();
};

const checkScriptType = (query: Record<string, unknown>) => {
    if (!query.type) {
        query.machineId = machineId.value;
        query.type = ScriptTypeEnum.Private.value;
    } else {
        query.machineId = query.type == ScriptTypeEnum.Private.value ? machineId.value : 9999999;
    }

    return query;
};

const runScript = async (script: MachineScriptVO) => {
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
    if (state.scriptParamsDialog.script) {
        await run(state.scriptParamsDialog.script);
    }
    state.scriptParamsDialog.visible = false;
    state.scriptParamsDialog.script = null;
};

const run = async (script: MachineScriptVO) => {
    const noResult = script.type == ScriptResultEnum.NoResult.value;
    // 如果脚本类型为有结果类型，则显示结果信息
    if (script.type == ScriptResultEnum.Result.value || noResult) {
        const res = await machineApi.runScript.request({
            machineId: machineId.value,
            acName: props.authCertName,
            scriptId: script.id,
            params: JSON.stringify(state.scriptParamsDialog.params),
        });

        if (noResult) {
            Msg.success('machine.execCompleted');
            return;
        }
        state.resultDialog.result = res;
        state.resultDialog.visible = true;
        return;
    }

    if (script.type == ScriptResultEnum.RealTime.value) {
        let cmd = script.script ?? '';
        if (state.scriptParamsDialog.params) {
            cmd = templateResolve(cmd, state.scriptParamsDialog.params);
        }
        state.terminalDialog.cmd = cmd;
        state.terminalDialog.visible = true;
        return;
    }
};

/**
 * 解析 {{.param}} 形式模板字符串
 */
function templateResolve(template: string, param: Record<string, unknown>) {
    return template.replace(/\{{.\w+\}}/g, (word) => {
        const key = word.substring(3, word.length - 2);
        const value = param[key];
        if (value != null || value != undefined) {
            return String(value);
        }
        return '';
    });
}

const closeTerminal = () => {
    state.terminalDialog.visible = false;
};

const editScript = (data: MachineScriptVO | null) => {
    state.editDialog.machineId = machineId.value ?? undefined;
    state.editDialog.data = data;
    if (data) {
        state.editDialog.title = useI18nEditTitle('machine.script');
    } else {
        state.editDialog.title = useI18nCreateTitle('machine.script');
    }
    state.editDialog.visible = true;
};

const submitSuccess = () => {
    getScripts();
};

const deleteRow = async (rows: MachineScriptVO[]) => {
    await useI18nDeleteConfirm(rows.map((x: MachineScriptVO) => x.name).join('、'));
    await machineApi.deleteScript.request({
        machineId: machineId.value,
        scriptId: rows.map((x: MachineScriptVO) => x.id).join(','),
    });
    Msg.deleteSuccess();
    getScripts();
};

/**
 * 关闭取消按钮触发的事件
 */
const handleClose = () => {
    dialogVisible.value = false;
    machineId.value = null;
    emit('cancel');
    state.query.type = ScriptTypeEnum.Private.value;
    state.scriptParamsDialog.paramsFormItem = [];
};

onMounted(() => {
    nextTick(getScripts);
});
</script>
<style lang="scss"></style>
