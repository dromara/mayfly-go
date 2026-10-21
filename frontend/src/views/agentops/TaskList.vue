<template>
    <div class="h-full">
        <data-table
            ref="dataTableRef"
            :page-api="agentTaskApi.list"
            :search-items="searchItems"
            v-model:query-form="params"
            :show-selection="true"
            v-model:selection-data="state.selectionData"
            :columns="columns"
        >
            <template #tableHeader>
                <el-button v-auth="perms.save" type="primary" icon="plus" plain @click="openFormDialog(false)">{{ $t('common.create') }}</el-button>
                <el-button v-auth="perms.del" :disabled="selectionData.length < 1" type="danger" icon="delete" @click="deleteTask()">
                    {{ $t('common.delete') }}
                </el-button>
            </template>

            <!-- cron 调度运行态：running=已注册进调度器（会被自动触发），否则仅可手动触发 -->
            <template #running="{ data }">
                <el-tag v-if="data.running" type="success" effect="plain">{{ $t('agentops.running') }}</el-tag>
                <el-tag v-else type="info" effect="plain">{{ $t('agentops.notRunning') }}</el-tag>
            </template>
        </data-table>

        <TaskEdit v-model:visible="taskEdit.visible" v-model:data="taskEdit.data" :title="taskEdit.title" @submit-success="search" />

        <!-- 任务执行记录弹窗：与任务强关联，弹窗内直达该任务的运行记录 -->
        <RunHistoryDialog v-model:visible="runHistory.visible" :task-key="runHistory.taskKey" :task-name="runHistory.taskName" />
    </div>
</template>

<script lang="ts" setup>
import { TableColumn } from '@/components/data-table';
import DataTable from '@/components/data-table/DataTable.vue';
import { SearchItem } from '@/components/data-table/SearchForm';
import { Msg, useI18nConfirm, useI18nCreateTitle, useI18nDeleteConfirm, useI18nEditTitle } from '@/hooks/useI18n';
import { defineAsyncComponent, reactive, ref, toRefs, useTemplateRef } from 'vue';
import { agentTaskApi } from './api';
import { AgentTaskStatusEnum, ProfileEnum } from './enums';
import { profileOptionsApi } from './useAgentProfiles';
import { AgentopsPerms as P } from './perms';
import type { AgentTask } from './types';

const TaskEdit = defineAsyncComponent(() => import('./TaskEdit.vue'));
const RunHistoryDialog = defineAsyncComponent(() => import('./RunHistoryDialog.vue'));

// 权限码取自集中声明（与后端路由、迁移资源三方同源）
const perms = {
    save: P.taskSave,
    del: P.taskDel,
    run: P.taskRun,
};

const searchItems = [
    SearchItem.input('name', 'common.name'),
    SearchItem.select('status', 'common.status').withEnum(AgentTaskStatusEnum),
    // 档位筛选项来自后端目录（新增档位无需改前端即可筛）；列渲染仍用枚举的 i18n 名与标签色
    SearchItem.select('profileId', 'agentops.profile').withOptionsApi(profileOptionsApi()),
];

const columns = ref([
    TableColumn.new('name', 'common.name'),
    TableColumn.new('cron', 'agentops.cron'),
    TableColumn.new('profileId', 'agentops.profile').typeTag(ProfileEnum),
    TableColumn.new('status', 'common.status').typeTag(AgentTaskStatusEnum),
    // 调度注册态（cron 是否已入调度器）：用独立 key 而非 runStatus，否则与上一列「状态」（启用/禁用）同名
    TableColumn.new('running', 'agentops.scheduleStatus').isSlot(),
    TableColumn.new('ownerUsername', 'agentops.owner'),
    TableColumn.new('lastFireTime', 'agentops.lastFireTime').isTime(),
    TableColumn.new('remark', 'common.remark'),
    TableColumn.new('action', 'common.operation')
        .setActions<AgentTask>([
            {
                key: 'run',
                label: 'agentops.run',
                type: 'primary',
                perm: perms.run,
                disabled: (row) => row.status === AgentTaskStatusEnum.Disable.value,
                onClick: (row) => runTask(row),
            },
            { key: 'edit', label: 'common.edit', type: 'primary', perm: perms.save, onClick: (row) => openFormDialog(row) },
            { key: 'execRecord', label: 'agentops.execRecord', type: 'primary', onClick: (row) => showExecRecord(row) },
        ])
        .fixedRight()
        .noShowOverflowTooltip()
        .alignCenter(),
]);

const dataTableRef = useTemplateRef<InstanceType<typeof DataTable>>('dataTableRef');

const state = reactive({
    params: {
        pageNum: 1,
        pageSize: 0,
        name: null,
        status: null,
        profileId: null,
    },
    selectionData: [] as AgentTask[],
    taskEdit: {
        visible: false,
        data: null as AgentTask | null,
        title: '',
    },
    runHistory: {
        visible: false,
        taskKey: '',
        taskName: '',
    },
});

const { selectionData, params, taskEdit, runHistory } = toRefs(state);

const openFormDialog = (data: AgentTask | false) => {
    if (data) {
        state.taskEdit.data = data;
        state.taskEdit.title = useI18nEditTitle('menu.aiopsTask');
    } else {
        state.taskEdit.data = null;
        state.taskEdit.title = useI18nCreateTitle('menu.aiopsTask');
    }
    state.taskEdit.visible = true;
};

/**
 * 手动触发一次运行（trigger=manual）：先二次确认再入队。
 *
 * 该动作会立即以负责人身份派生一次真实 Agent 运行（消耗 LLM 配额、可访问真实资源），
 * 与同页「删除」「急停」同级，不能一键直达（旧写法直接发请求，误触即产生一次真实运行与计费）
 */
const runTask = async (data: AgentTask) => {
    if (!(await useI18nConfirm('agentops.runTriggerConfirm'))) {
        return; // 取消触发
    }
    await agentTaskApi.run.request({ id: data.id });
    Msg.success('agentops.runTriggered');
};

const deleteTask = async () => {
    try {
        if (!(await useI18nDeleteConfirm(state.selectionData.map((x) => x.name).join('、')))) {
            return;
        }
        await agentTaskApi.del.request({ ids: state.selectionData.map((x) => x.id).join(',') });
        Msg.deleteSuccess();
        search();
    } catch (err) {
        // 取消删除
    }
};

/**
 * 查看该任务的执行历史：弹窗内按 triggerRef（任务 key）过滤直连展示，
 * cron 与 manual 触发的运行 triggerRef 均为任务 key。
 * 取消/回放/反馈等深度操作经弹窗内「在执行历史中查看」跳转全局页。
 */
const showExecRecord = (data: AgentTask) => {
    state.runHistory.taskKey = data.key;
    state.runHistory.taskName = data.name;
    state.runHistory.visible = true;
};

const search = () => {
    dataTableRef.value?.search();
};
</script>
